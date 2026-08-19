//  Copyright (c) 2017-2026 Gigamon, Inc. All rights reserved.
//
//  Author: Gigamon Terraform Team (gigamon-terraform-team@gigamon.com)
//
//  This program is free software: you can redistribute it and/or modify
//  it under the terms of the GNU General Public License as published by
//  the Free Software Foundation, version 3 of the License.
//
//  This program is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of
//  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
//  GNU General Public License for more details.
//
//  You should have received a copy of the GNU General Public License
//  along with this program. If not, see <https://www.gnu.org/licenses/>

package commonresources

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"

	"terraform-provider-gigamon/internal/commonutils"
	"terraform-provider-gigamon/internal/fmclient"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Fixed / non-configurable FM values for the EVP5G app.
// These are never surfaced as Terraform attributes; the provider always
// sends these constants to FM regardless of user input.
const (
	evp5gAppName        = "evp5g"
	evp5gPrivateKeyPath = "/usr/lib/vseries-web/api/crypto/private/cloud5g/pvt_key"
	evp5gCertFilePath   = "/usr/lib/vseries-web/api/crypto/private/cloud5g/cloud5G.crt"
	evp5gTxDstPort      = 4754 // only supported value
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &Evp5g{}

var _ resource.ResourceWithModifyPlan = &Evp5g{}
// Evp5g app resource, which manages the EVP5G (Ericsson vTAP / 5G Cloud)
// application instances deployed on a Monitoring Session.
func NewEvp5g() resource.Resource {
	return &Evp5g{}
}

// Evp5g manages the evp5g app instance on a monitoring session
type Evp5g struct {
	fmClient *fmclient.FmClient // Instance to our FM http client instance
}

// ---- Nested TF models ----

type Evp5gRxTunnelModel struct {
	ListenIPAddress types.String `tfsdk:"listen_ip_address"`
	ListenPort      types.Int32  `tfsdk:"listen_port"`
	RxThread        types.Int32  `tfsdk:"rx_thread"`
	Dtls            types.String `tfsdk:"dtls"`
	DtlsKeyAlias    types.String `tfsdk:"dtls_key_alias"`
}

type Evp5gTxTunnelModel struct {
	TxRemoteIPAddress types.String `tfsdk:"tx_remote_ip_address"`
	TxSrcPort         types.Int32  `tfsdk:"tx_src_port"`
	TxDstPort         types.Int32  `tfsdk:"tx_dst_port"`
	TxThread          types.Int32  `tfsdk:"tx_thread"`
	TxSrcIPAddress    types.List   `tfsdk:"tx_src_ip_address"` // list(string)
}

type Evp5gTimeServerConfigModel struct {
	PrimaryServer   types.String `tfsdk:"primary_server"`
	SecondaryServer types.String `tfsdk:"secondary_server"`
}

type Evp5gPacketOrderingConfigModel struct {
	NumEgressFlows             types.Int32  `tfsdk:"num_egress_flows"`
	EgressFlowTimeoutValue     types.Int32  `tfsdk:"egress_flow_timeout_value"`
	NumBuckets                 types.Int32  `tfsdk:"num_buckets"`
	PktsPerBucket              types.Int32  `tfsdk:"pkts_per_bucket"`
	BucketInterval             types.Int32  `tfsdk:"bucket_interval"`
	PktRxOutsideBucketInterval types.String `tfsdk:"pkt_rx_outside_bucket_interval"`
}

type Evp5gDiagnosticOptionsModel struct {
	PctDisable types.List `tfsdk:"pct_disable"` // list(number), 0-12
	TxDisable  types.Bool `tfsdk:"tx_disable"`
}

type Evp5gLoggingModel struct {
	PacketCaptureLogLevel types.String `tfsdk:"packet_capture_log_level"`
	CsvLoggingLevel       types.String `tfsdk:"csv_logging_level"`
	MsgLogLevel           types.String `tfsdk:"msg_log_level"`
	LogFolderLoc          types.String `tfsdk:"log_folder_loc"`
}

// Evp5g App Model (top-level TF resource model)
type Evp5gModel struct {
	MonitoringSessionId types.String `tfsdk:"monitoring_session_id"`
	Alias                types.String `tfsdk:"alias"`
	Id                   types.String `tfsdk:"id"`

	RxTunnel             *Evp5gRxTunnelModel             `tfsdk:"rx_tunnel"`
	TxTunnel             *Evp5gTxTunnelModel             `tfsdk:"tx_tunnel"`
	TimeServerConfig     *Evp5gTimeServerConfigModel     `tfsdk:"time_server_config"`
	PacketOrderingConfig *Evp5gPacketOrderingConfigModel `tfsdk:"packet_ordering_config"`
	DiagnosticOptions    *Evp5gDiagnosticOptionsModel    `tfsdk:"diagnostic_options"`
	Logging              *Evp5gLoggingModel              `tfsdk:"logging"`
}

// ---- FM payload structs ----

type FMEvp5gRxTunnel struct {
	ListenIPAddress string `json:"listenIpaddress"`
	ListenPort      int32  `json:"listenPort,omitempty"`
	RxThread        int32  `json:"rxThread,omitempty"`
	Dtls            string `json:"dtls,omitempty"`
	PrivateKeyPath  string `json:"privateKeyPath"` // fixed, provider-owned
	CertFilePath    string `json:"certFilePath"`   // fixed, provider-owned
	DtlsKeyAlias    string `json:"dtlsKeyAlias,omitempty"`
}

type FMEvp5gTxTunnel struct {
	TxRemoteIPAddress string   `json:"txRemoteIpaddress"`
	TxSrcPort         int32    `json:"txSrcPort,omitempty"`
	TxDstPort         int32    `json:"txDstPort"`
	TxThread          int32    `json:"txThread,omitempty"`
	TxSrcIPAddress    []string `json:"txSrcIpaddress"`
}

type FMEvp5gTimeServerConfig struct {
	PrimaryServer   string `json:"primaryServer"`
	SecondaryServer string `json:"secondaryServer,omitempty"`
}

type FMEvp5gPacketOrderingConfig struct {
	NumEgressFlows             int32  `json:"numEgressFlows,omitempty"`
	EgressFlowTimeoutValue     int32  `json:"egressFlowTimeoutValue,omitempty"`
	PktOrderingEnable          bool   `json:"pktOrderingEnable"` // always true, provider-owned
	NumBuckets                 int32  `json:"numBuckets,omitempty"`
	PktsPerBucket              int32  `json:"pktsPerBucket,omitempty"`
	BucketInterval             int32  `json:"bucketInterval,omitempty"`
	PktRxOutSideBucketInterval string `json:"pktRxOutSideBucketInterval,omitempty"`
}

type FMEvp5gDiagnosticOptions struct {
	PctDisable []int32 `json:"pctDisable,omitempty"`
	TxDisable  bool    `json:"txDisable"`
}

type FMEvp5gLogging struct {
	PacketCaptureLogLevel string `json:"packetCaptureLogLevel,omitempty"`
	CsvLoggingLevel       string `json:"csvLoggingLevel,omitempty"`
	MsgLogLevel           string `json:"msgLogLevel,omitempty"`
	LogFolderLoc          string `json:"logFolderLoc,omitempty"`
}

// FM payload struct for EVP5G (used in /monitoringSessions/{id}/update)
type FMEvp5g struct {
	Alias                string                       `json:"alias,omitempty"`
	Name                 string                       `json:"name,omitempty"`
	RxTunnel             *FMEvp5gRxTunnel             `json:"rxTunnel,omitempty"`
	TxTunnel             *FMEvp5gTxTunnel             `json:"txTunnel,omitempty"`
	TimeServerConfig     *FMEvp5gTimeServerConfig     `json:"timeServerConfig,omitempty"`
	PacketOrderingConfig *FMEvp5gPacketOrderingConfig `json:"packetOrderingConfig,omitempty"`
	DiagnosticOptions    *FMEvp5gDiagnosticOptions    `json:"diagnosticOptions,omitempty"`
	Logging              *FMEvp5gLogging              `json:"logging,omitempty"`
	Id                   string                       `json:"id,omitempty"`
}

// ---- TF Hooks ----

func (e *Evp5g) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_evp5g"
}

func (e *Evp5g) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Gigamon APP EVP5G (Ericsson vTAP / 5G Cloud) Schema",
		Attributes: map[string]schema.Attribute{
			"alias": schema.StringAttribute{
				MarkdownDescription: "Name for this EVP5G application. Only alphanumeric, '-' and '_' are allowed.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9_-]+$`),
						"alias must contain only alphanumeric characters, '-' and '_'",
					),
				},
			},
			"monitoring_session_id": schema.StringAttribute{
				MarkdownDescription: "Monitoring session ID on which to deploy this APP",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ID of this App instance for later use",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"rx_tunnel": schema.SingleNestedBlock{
				MarkdownDescription: "RX tunnel configuration for EVP5G.",
				Attributes: map[string]schema.Attribute{
					"listen_ip_address": schema.StringAttribute{
						MarkdownDescription: "IPv4 or IPv6 address to listen on.",
						Required:            true,
					},
					"listen_port": schema.Int32Attribute{
						MarkdownDescription: "Listening port (1-65535).",
						Optional:            true,
						Computed:            true,
						Default:             int32default.StaticInt32(4754),
						Validators: []validator.Int32{
							int32validator.Between(1, 65535),
						},
					},
					"rx_thread": schema.Int32Attribute{
						MarkdownDescription: "Number of RX threads (1-16).",
						Optional:            true,
						Computed:            true,
						Default:             int32default.StaticInt32(8),
						Validators: []validator.Int32{
							int32validator.Between(1, 16),
						},
					},
					"dtls": schema.StringAttribute{
						MarkdownDescription: "Enable or disable DTLS.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("disable"),
						Validators: []validator.String{
							stringvalidator.OneOf("enable", "disable"),
						},
					},
					"dtls_key_alias": schema.StringAttribute{
						MarkdownDescription: "DTLS key alias.",
						Optional:            true,
					},
					// NOTE: privateKeyPath and certFilePath are fixed FM constants
					// and are intentionally not exposed as Terraform attributes.
				},
			},
			"tx_tunnel": schema.SingleNestedBlock{
				MarkdownDescription: "TX tunnel configuration for EVP5G.",
				Attributes: map[string]schema.Attribute{
					"tx_remote_ip_address": schema.StringAttribute{
						MarkdownDescription: "IPv4 or IPv6 remote address for the TX tunnel.",
						Required:            true,
					},
					"tx_src_port": schema.Int32Attribute{
						MarkdownDescription: "TX source port (1-65535).",
						Optional:            true,
						Computed:            true,
						Default:             int32default.StaticInt32(4754),
						Validators: []validator.Int32{
							int32validator.Between(1, 65535),
						},
					},
					"tx_dst_port": schema.Int32Attribute{
						MarkdownDescription: "TX destination port. Only supported value is 4754.",
						Required:            true,
						Validators: []validator.Int32{
							int32validator.OneOf(evp5gTxDstPort),
						},
					},
					"tx_thread": schema.Int32Attribute{
						MarkdownDescription: "Number of TX threads (1-16).",
						Optional:            true,
						Computed:            true,
						Default:             int32default.StaticInt32(4),
						Validators: []validator.Int32{
							int32validator.Between(1, 16),
						},
					},
					"tx_src_ip_address": schema.ListAttribute{
						ElementType:         types.StringType,
						MarkdownDescription: "List of IPv4 or IPv6 source addresses for the TX tunnel.",
						Required:            true,
						Validators: []validator.List{
							listvalidator.SizeAtLeast(1),
						},
					},
				},
			},
			"time_server_config": schema.SingleNestedBlock{
				MarkdownDescription: "Time server (NTP) configuration for EVP5G.",
				Attributes: map[string]schema.Attribute{
					"primary_server": schema.StringAttribute{
						MarkdownDescription: "IPv4 or IPv6 address of the primary time server.",
						Required:            true,
					},
					"secondary_server": schema.StringAttribute{
						MarkdownDescription: "IPv4 or IPv6 address of the secondary time server.",
						Optional:            true,
					},
				},
			},
			"packet_ordering_config": schema.SingleNestedBlock{
				MarkdownDescription: "Packet ordering configuration for EVP5G.",
				Attributes: map[string]schema.Attribute{
					"num_egress_flows": schema.Int32Attribute{
						MarkdownDescription: "Number of egress flows (32-16384).",
						Optional:            true,
						Computed:            true,
						Default:             int32default.StaticInt32(512),
						Validators: []validator.Int32{
							int32validator.Between(32, 16384),
						},
					},
					"egress_flow_timeout_value": schema.Int32Attribute{
						MarkdownDescription: "Egress flow timeout value in seconds (360-1860).",
						Optional:            true,
						Computed:            true,
						Default:             int32default.StaticInt32(660),
						Validators: []validator.Int32{
							int32validator.Between(360, 1860),
						},
					},
					"num_buckets": schema.Int32Attribute{
						MarkdownDescription: "Number of buckets (10-200).",
						Optional:            true,
						Computed:            true,
						Default:             int32default.StaticInt32(50),
						Validators: []validator.Int32{
							int32validator.Between(10, 200),
						},
					},
					"pkts_per_bucket": schema.Int32Attribute{
						MarkdownDescription: "Packets per bucket (10000-100000).",
						Optional:            true,
						Computed:            true,
						Default:             int32default.StaticInt32(20000),
						Validators: []validator.Int32{
							int32validator.Between(10000, 100000),
						},
					},
					"bucket_interval": schema.Int32Attribute{
						MarkdownDescription: "Bucket interval in seconds (1-5).",
						Optional:            true,
						Computed:            true,
						Default:             int32default.StaticInt32(1),
						Validators: []validator.Int32{
							int32validator.Between(1, 5),
						},
					},
					"pkt_rx_outside_bucket_interval": schema.StringAttribute{
						MarkdownDescription: "Action for packets received outside the bucket interval: forward or discard.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("forward"),
						Validators: []validator.String{
							stringvalidator.OneOf("forward", "discard"),
						},
					},
					// NOTE: pktOrderingEnable is a fixed FM constant (always true)
					// and is intentionally not exposed as a Terraform attribute.
				},
			},
			"diagnostic_options": schema.SingleNestedBlock{
				MarkdownDescription: "Diagnostic options for EVP5G.",
				Attributes: map[string]schema.Attribute{
					"pct_disable": schema.ListAttribute{
						ElementType:         types.Int32Type,
						MarkdownDescription: "List of PCT indices to disable (each 0-12).",
						Optional:            true,
						Validators: []validator.List{
							listvalidator.ValueInt32sAre(int32validator.Between(0, 12)),
						},
					},
					"tx_disable": schema.BoolAttribute{
						MarkdownDescription: "Disable TX.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
				},
			},
			"logging": schema.SingleNestedBlock{
				MarkdownDescription: "Logging configuration for EVP5G.",
				Attributes: map[string]schema.Attribute{
					"packet_capture_log_level": schema.StringAttribute{
						MarkdownDescription: "Packet capture log level.",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("all", "receive", "transmit", "none"),
						},
					},
					"csv_logging_level": schema.StringAttribute{
						MarkdownDescription: "Enable or disable CSV logging.",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("enable", "disable"),
						},
					},
					"msg_log_level": schema.StringAttribute{
						MarkdownDescription: "Message log level.",
						Optional:            true,
						Validators: []validator.String{
							stringvalidator.OneOf("none", "fatal", "error", "info", "detail", "full-parse"),
						},
					},
					"log_folder_loc": schema.StringAttribute{
						MarkdownDescription: "Folder path where logs will be stored.",
						Optional:            true,
						Computed:            true,
						Default:             stringdefault.StaticString("/var/log"),
					},
				},
			},
		},
	}
}

func (e *Evp5g) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return // destroying
	}
	var data Evp5gModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateEvp5gIPs(ctx, &data, resp)
}

func validateEvp5gIPs(ctx context.Context, data *Evp5gModel, resp *resource.ModifyPlanResponse) {
	if data.RxTunnel != nil {
		if ip := data.RxTunnel.ListenIPAddress.ValueString(); ip != "" && net.ParseIP(ip) == nil {
			resp.Diagnostics.AddAttributeError(
				// path reference is approximate for block attributes
				evp5gPath("rx_tunnel", "listen_ip_address"),
				"Invalid listen_ip_address",
				fmt.Sprintf("rx_tunnel.listen_ip_address %q must be a valid IPv4 or IPv6 address", ip),
			)
		}
	}

	if data.TxTunnel != nil {
		if ip := data.TxTunnel.TxRemoteIPAddress.ValueString(); ip != "" && net.ParseIP(ip) == nil {
			resp.Diagnostics.AddAttributeError(
				evp5gPath("tx_tunnel", "tx_remote_ip_address"),
				"Invalid tx_remote_ip_address",
				fmt.Sprintf("tx_tunnel.tx_remote_ip_address %q must be a valid IPv4 or IPv6 address", ip),
			)
		}

		if !data.TxTunnel.TxSrcIPAddress.IsNull() && !data.TxTunnel.TxSrcIPAddress.IsUnknown() {
			var srcIPs []string
			_ = data.TxTunnel.TxSrcIPAddress.ElementsAs(ctx, &srcIPs, false)
			for i, ip := range srcIPs {
				if ip != "" && net.ParseIP(ip) == nil {
					resp.Diagnostics.AddAttributeError(
						evp5gPath("tx_tunnel", "tx_src_ip_address"),
						"Invalid tx_src_ip_address",
						fmt.Sprintf("tx_tunnel.tx_src_ip_address[%d] %q must be a valid IPv4 or IPv6 address", i, ip),
					)
				}
			}
		}
	}

	if data.TimeServerConfig != nil {
		if ip := data.TimeServerConfig.PrimaryServer.ValueString(); ip != "" && net.ParseIP(ip) == nil {
			resp.Diagnostics.AddAttributeError(
				evp5gPath("time_server_config", "primary_server"),
				"Invalid primary_server",
				fmt.Sprintf("time_server_config.primary_server %q must be a valid IPv4 or IPv6 address", ip),
			)
		}
		if ip := data.TimeServerConfig.SecondaryServer.ValueString(); ip != "" && net.ParseIP(ip) == nil {
			resp.Diagnostics.AddAttributeError(
				evp5gPath("time_server_config", "secondary_server"),
				"Invalid secondary_server",
				fmt.Sprintf("time_server_config.secondary_server %q must be a valid IPv4 or IPv6 address", ip),
			)
		}
	}
}

// evp5gPath builds a simple attribute path for a nested block field.
// The framework path API requires matching the schema structure; for blocks
// we use AtName on both the block and the attribute within it.
func evp5gPath(block, attr string) path.Path {
	return path.Root(block).AtName(attr)
}

// Initial Configure call, to initialize the Provider
func (e *Evp5g) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	fmClient, ok := req.ProviderData.(*fmclient.FmClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *fmclient.FmClient, got: %T. Report the issue to Gigamon", req.ProviderData),
		)
		return
	}
	e.fmClient = fmClient
}

// Create a FM DS from the TF DS and return the same
func (e *Evp5g) createFMStruct(ctx context.Context, data *Evp5gModel) *FMEvp5g {
	fm := &FMEvp5g{
		Alias: data.Alias.ValueString(),
		Name:  evp5gAppName,
		Id:    data.Id.ValueString(),
	}

	if data.RxTunnel != nil {
		fm.RxTunnel = &FMEvp5gRxTunnel{
			ListenIPAddress: data.RxTunnel.ListenIPAddress.ValueString(),
			ListenPort:      data.RxTunnel.ListenPort.ValueInt32(),
			RxThread:        data.RxTunnel.RxThread.ValueInt32(),
			Dtls:            data.RxTunnel.Dtls.ValueString(),
			PrivateKeyPath:  evp5gPrivateKeyPath,
			CertFilePath:    evp5gCertFilePath,
			DtlsKeyAlias:    data.RxTunnel.DtlsKeyAlias.ValueString(),
		}
	}

	if data.TxTunnel != nil {
		var srcIPs []string
		if !data.TxTunnel.TxSrcIPAddress.IsNull() && !data.TxTunnel.TxSrcIPAddress.IsUnknown() {
			var ss []types.String
			_ = data.TxTunnel.TxSrcIPAddress.ElementsAs(ctx, &ss, false)
			for _, s := range ss {
				srcIPs = append(srcIPs, s.ValueString())
			}
		}
		fm.TxTunnel = &FMEvp5gTxTunnel{
			TxRemoteIPAddress: data.TxTunnel.TxRemoteIPAddress.ValueString(),
			TxSrcPort:         data.TxTunnel.TxSrcPort.ValueInt32(),
			TxDstPort:         evp5gTxDstPort, // only supported value
			TxThread:          data.TxTunnel.TxThread.ValueInt32(),
			TxSrcIPAddress:    srcIPs,
		}
	}

	if data.TimeServerConfig != nil {
		fm.TimeServerConfig = &FMEvp5gTimeServerConfig{
			PrimaryServer:   data.TimeServerConfig.PrimaryServer.ValueString(),
			SecondaryServer: data.TimeServerConfig.SecondaryServer.ValueString(),
		}
	}

	if data.PacketOrderingConfig != nil {
		fm.PacketOrderingConfig = &FMEvp5gPacketOrderingConfig{
			NumEgressFlows:             data.PacketOrderingConfig.NumEgressFlows.ValueInt32(),
			EgressFlowTimeoutValue:     data.PacketOrderingConfig.EgressFlowTimeoutValue.ValueInt32(),
			PktOrderingEnable:          true, // fixed, always true
			NumBuckets:                 data.PacketOrderingConfig.NumBuckets.ValueInt32(),
			PktsPerBucket:              data.PacketOrderingConfig.PktsPerBucket.ValueInt32(),
			BucketInterval:             data.PacketOrderingConfig.BucketInterval.ValueInt32(),
			PktRxOutSideBucketInterval: data.PacketOrderingConfig.PktRxOutsideBucketInterval.ValueString(),
		}
	}

	if data.DiagnosticOptions != nil {
		var pct []int32
		if !data.DiagnosticOptions.PctDisable.IsNull() && !data.DiagnosticOptions.PctDisable.IsUnknown() {
			var vv []types.Int32
			_ = data.DiagnosticOptions.PctDisable.ElementsAs(ctx, &vv, false)
			for _, v := range vv {
				pct = append(pct, v.ValueInt32())
			}
		}
		fm.DiagnosticOptions = &FMEvp5gDiagnosticOptions{
			PctDisable: pct,
			TxDisable:  data.DiagnosticOptions.TxDisable.ValueBool(),
		}
	}

	if data.Logging != nil {
		fm.Logging = &FMEvp5gLogging{
			PacketCaptureLogLevel: data.Logging.PacketCaptureLogLevel.ValueString(),
			CsvLoggingLevel:       data.Logging.CsvLoggingLevel.ValueString(),
			MsgLogLevel:           data.Logging.MsgLogLevel.ValueString(),
			LogFolderLoc:          data.Logging.LogFolderLoc.ValueString(),
		}
	}

	return fm
}

// Update the TF Data from the FM struct
func (e *Evp5g) updateTFStruct(ctx context.Context, data *Evp5gModel, fmData *FMEvp5g) {
	if fmData.Alias != "" {
		data.Alias = types.StringValue(fmData.Alias)
	}

	if fmData.RxTunnel != nil {
		data.RxTunnel = &Evp5gRxTunnelModel{
			ListenIPAddress: types.StringValue(fmData.RxTunnel.ListenIPAddress),
			ListenPort:      types.Int32Value(fmData.RxTunnel.ListenPort),
			RxThread:        types.Int32Value(fmData.RxTunnel.RxThread),
			Dtls:            types.StringValue(fmData.RxTunnel.Dtls),
			DtlsKeyAlias:    stringOrNull(fmData.RxTunnel.DtlsKeyAlias),
		}
	}

	if fmData.TxTunnel != nil {
		var srcIPs types.List
		if len(fmData.TxTunnel.TxSrcIPAddress) == 0 {
			srcIPs = types.ListNull(types.StringType)
		} else {
			srcIPs, _ = types.ListValueFrom(ctx, types.StringType, fmData.TxTunnel.TxSrcIPAddress)
		}
		data.TxTunnel = &Evp5gTxTunnelModel{
			TxRemoteIPAddress: types.StringValue(fmData.TxTunnel.TxRemoteIPAddress),
			TxSrcPort:         types.Int32Value(fmData.TxTunnel.TxSrcPort),
			TxDstPort:         types.Int32Value(fmData.TxTunnel.TxDstPort),
			TxThread:          types.Int32Value(fmData.TxTunnel.TxThread),
			TxSrcIPAddress:    srcIPs,
		}
	}

	if fmData.TimeServerConfig != nil {
		data.TimeServerConfig = &Evp5gTimeServerConfigModel{
			PrimaryServer:   types.StringValue(fmData.TimeServerConfig.PrimaryServer),
			SecondaryServer: stringOrNull(fmData.TimeServerConfig.SecondaryServer),
		}
	}

	if fmData.PacketOrderingConfig != nil {
		data.PacketOrderingConfig = &Evp5gPacketOrderingConfigModel{
			NumEgressFlows:             types.Int32Value(fmData.PacketOrderingConfig.NumEgressFlows),
			EgressFlowTimeoutValue:     types.Int32Value(fmData.PacketOrderingConfig.EgressFlowTimeoutValue),
			NumBuckets:                 types.Int32Value(fmData.PacketOrderingConfig.NumBuckets),
			PktsPerBucket:              types.Int32Value(fmData.PacketOrderingConfig.PktsPerBucket),
			BucketInterval:             types.Int32Value(fmData.PacketOrderingConfig.BucketInterval),
			PktRxOutsideBucketInterval: types.StringValue(fmData.PacketOrderingConfig.PktRxOutSideBucketInterval),
		}
	}

	if fmData.DiagnosticOptions != nil {
		var pct types.List
		if len(fmData.DiagnosticOptions.PctDisable) == 0 {
			pct = types.ListNull(types.Int32Type)
		} else {
			pct, _ = types.ListValueFrom(ctx, types.Int32Type, fmData.DiagnosticOptions.PctDisable)
		}
		data.DiagnosticOptions = &Evp5gDiagnosticOptionsModel{
			PctDisable: pct,
			TxDisable:  types.BoolValue(fmData.DiagnosticOptions.TxDisable),
		}
	}

	if fmData.Logging != nil {
		data.Logging = &Evp5gLoggingModel{
			PacketCaptureLogLevel: stringOrNull(fmData.Logging.PacketCaptureLogLevel),
			CsvLoggingLevel:       stringOrNull(fmData.Logging.CsvLoggingLevel),
			MsgLogLevel:           stringOrNull(fmData.Logging.MsgLogLevel),
			LogFolderLoc:          types.StringValue(fmData.Logging.LogFolderLoc),
		}
	}
}

// Create call for new EVP5G App Instance
func (e *Evp5g) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data Evp5gModel
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fmData := e.createFMStruct(ctx, &data)
	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{
			{
				EntityType:  "application",
				Operation:   "create",
				Application: fmData,
			},
		},
	}

	id, err := commonutils.UpdateMonSess(
		ctx,
		&updateReq,
		data.MonitoringSessionId.ValueString(),
		e.fmClient,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create evp5g app",
			fmt.Sprintf("app creation failed: %s", err),
		)
		return
	}

	typedID, err := commonutils.MakeTypedID(
		commonutils.ModuleApp,
		commonutils.Type5GEvp, // NOTE: add TypeEvp5g to commonutils alongside TypeDedup/TypeSlicing/etc.
		id,
	)
	if err != nil {
		return
	}

	data.Id = types.StringValue(typedID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (e *Evp5g) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data Evp5gModel
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	evp5gData := FMEvp5g{}
	rawID, err := commonutils.UUIDFromTypedID(data.Id.ValueString())
	if err != nil {
		return
	}

	err = GetMSAppData(
		ctx,
		data.MonitoringSessionId.ValueString(),
		rawID,
		evp5gAppName,
		"",
		&evp5gData,
		e.fmClient,
	)
	if err != nil {
		var fmErr *fmclient.FMErrors
		if errors.As(err, &fmErr) {
			if fmErr.ErrorCode() == fmclient.ObjectNotFound {
				tflog.Info(ctx, "evp5g app not found, removing from state", nil)
				resp.State.RemoveResource(ctx)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Unable to Get EVP5G App details",
			fmt.Sprintf("unable to get EVP5G App details. error is %v", err),
		)
		return
	}

	// Save updated data into Terraform state
	e.updateTFStruct(ctx, &data, &evp5gData)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (e *Evp5g) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData Evp5gModel
	// Read desired values from the plan
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fmData := e.createFMStruct(ctx, &planData)
	rawID, err := commonutils.UUIDFromTypedID(planData.Id.ValueString())
	if err != nil {
		return
	}
	fmData.Id = rawID

	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{
			{
				EntityType:  "application",
				Operation:   "update",
				Application: fmData,
			},
		},
	}

	_, err = commonutils.UpdateMonSess(
		ctx,
		&updateReq,
		planData.MonitoringSessionId.ValueString(),
		e.fmClient,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update evp5g app",
			fmt.Sprintf("app update failed: %s", err),
		)
		return
	}

	// Let FM override computed/FM-owned fields
	e.updateTFStruct(ctx, &planData, fmData)
	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
}

func (e *Evp5g) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data Evp5gModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rawID, err := commonutils.UUIDFromTypedID(data.Id.ValueString())
	if err != nil {
		return
	}

	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{
			{
				EntityType: "application",
				Operation:  "delete",
				Application: FMEvp5g{
					Id:   rawID,
					Name: "Application",
				},
			},
		},
	}

	_, err = commonutils.UpdateMonSess(
		ctx,
		&updateReq,
		data.MonitoringSessionId.ValueString(),
		e.fmClient,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete evp5g app",
			fmt.Sprintf("app deletion failed: %s", err),
		)
	}
}
