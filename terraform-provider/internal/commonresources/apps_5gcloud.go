// Copyright (c) Gigamon, Inc.
// Licensed under the Mozilla Public License v. 2.0

// Implements the 5G Cloud Application resource for Gigamon Terraform Provider

package commonresources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-gigamon/internal/commonutils"
	"terraform-provider-gigamon/internal/fmclient"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &App5GCloud{}
var _ resource.ResourceWithConfigure = &App5GCloud{}
var _ resource.ResourceWithImportState = &App5GCloud{}

// New5GCloud creates a new resource instance for 5G Cloud application
func New5GCloud() resource.Resource {
	return &App5GCloud{}
}

// App5GCloud manages the 5G Cloud application resource
// this file follows the provider's existing naming pattern.
//
//nolint:revive
type App5GCloud struct {
	fmClient *fmclient.FmClient
}

var app5GCloudNameRegex = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

const app5GCloudTypeID = "cloud5g"

var scpSupportedModes = map[string]bool{
	"oracleSCP":              true,
	"nokiaSCPInbound":        true,
	"nokiaSCPIn-Outbound":    true,
	"SBINF":                  true,
	"ericssonSCPOutbound":    true,
	"ericssonSCPIn-Outbound": true,
	"nokiaHEP3Inbound":       true,
	"nokiaHEP3IMS":           true,
}

var http2SupportedModes = map[string]bool{
	"SBINF":                  true,
	"ericssonSCPOutbound":    true,
	"ericssonSCPIn-Outbound": true,
}

var hep3SupportedModes = map[string]bool{
	"nokiaHEP3Inbound": true,
	"nokiaHEP3IMS":     true,
}

// App5GCloudModel represents the Terraform configuration and state for 5G Cloud app.
type App5GCloudModel struct {
	Id                  types.String `tfsdk:"id"`
	MonitoringSessionId types.String `tfsdk:"monitoring_session_id"`
	Mode                types.String `tfsdk:"mode"`
	RxTunnel            types.List   `tfsdk:"rx_tunnel"`
	TxTunnel            types.Object `tfsdk:"tx_tunnel"`
	ScpConfig           types.Object `tfsdk:"scp_config"`
	Hep3Config          types.Object `tfsdk:"hep3_config"`
	ToolMtu             types.Int64  `tfsdk:"tool_mtu"`
	LogFolderLoc        types.String `tfsdk:"log_folder_loc"`
	TunnelLogLevel      types.Int64  `tfsdk:"tunnel_log_level"`
	Alias               types.String `tfsdk:"alias"`
}

type RxTunnelModel struct {
	RxType          types.String `tfsdk:"rx_type"`
	ListenIpaddress types.String `tfsdk:"listen_ipaddress"`
	ListenPort      types.Int64  `tfsdk:"listen_port"`
	FromPort        types.Int64  `tfsdk:"from_port"`
	RxVNIId         types.Int64  `tfsdk:"rx_vni_id"`
	RxThread        types.Int64  `tfsdk:"rx_thread"`
}

type TxTunnelModel struct {
	TxType            types.String `tfsdk:"tx_type"`
	TxRemoteIpaddress types.String `tfsdk:"tx_remote_ipaddress"`
	TxSrcIpaddress    types.String `tfsdk:"tx_src_ipaddress"`
	TxSrcPort         types.Int64  `tfsdk:"tx_src_port"`
	TxDstPort         types.Int64  `tfsdk:"tx_dst_port"`
	L2GreKey          types.Int64  `tfsdk:"l2gre_key"`
	TxVNIId           types.Int64  `tfsdk:"tx_vni_id"`
}

type Http2MonitoredFlowsModel struct {
	NumMonitoredStreamFlows types.Int64 `tfsdk:"num_monitored_stream_flows"`
	Http2RequestTimeout     types.Int64 `tfsdk:"http2_request_timeout"`
}

type TcpMonitoredFlowsModel struct {
	NumMonitoredTCPFlows     types.Int64 `tfsdk:"num_monitored_tcp_flows"`
	TcpFlowTimeout           types.Int64 `tfsdk:"tcp_flow_timeout"`
	TcpFlowReassemblyTimeout types.Int64 `tfsdk:"tcp_flow_reassembly_timeout"`
}

type ScpConfigModel struct {
	NumTCPFlows                      types.Int64  `tfsdk:"num_tcp_flows"`
	NumTransactionFlows              types.Int64  `tfsdk:"num_transaction_flows"`
	TcpFlowTimeout                   types.Int64  `tfsdk:"tcp_flow_timeout"`
	ScpTransactionTimeout            types.Int64  `tfsdk:"scp_transaction_timeout"`
	HeaderIndex                      types.Bool   `tfsdk:"header_index"`
	HeaderCompressionCode            types.Bool   `tfsdk:"header_compression_code"`
	NrfDiscoveryEnabled              types.Bool   `tfsdk:"nrf_discovery_enabled"`
	AddGigamonHeader                 types.Bool   `tfsdk:"add_gigamon_header"`
	NfInstanceAlias                  types.String `tfsdk:"nf_instance_alias"`
	FqdnAlias                        types.String `tfsdk:"fqdn_alias"`
	UaAlias                          types.String `tfsdk:"ua_alias"`
	MinTcpFlowClientPort             types.Int64  `tfsdk:"min_tcp_flow_client_port"`
	MaxTcpFlowClientPort             types.Int64  `tfsdk:"max_tcp_flow_client_port"`
	PacketCaptureLogLevel            types.String `tfsdk:"packet_capture_log_level"`
	CsvLoggingLogLevel               types.String `tfsdk:"csv_logging_log_level"`
	NumSCPProcessingThreads          types.Int64  `tfsdk:"num_scp_processing_threads"`
	NumTCPFlowClientPortPerThread    types.Int64  `tfsdk:"num_tcp_flow_client_port_per_thread"`
	NokiaInboundUse3gppTargetApiRoot types.Bool   `tfsdk:"nokia_inbound_use_3gpp_target_api_root"`
	NokiaInboundReplaceAuthority     types.Bool   `tfsdk:"nokia_inbound_replace_authority"`
	Http2MonitoredFlows              types.Object `tfsdk:"http2_monitored_flows"`
	TcpMonitoredFlows                types.Object `tfsdk:"tcp_monitored_flows"`
}

type Hep3ConfigModel struct {
	NumIngressTCPconn     types.Int64  `tfsdk:"num_ingress_tcp_conn"`
	NumEgressTCPFlows     types.Int64  `tfsdk:"num_egress_tcp_flows"`
	IngressTCPTimeout     types.Int64  `tfsdk:"ingress_tcp_timeout"`
	EgressTCPFlowTimeout  types.Int64  `tfsdk:"egress_tcp_flow_timeout"`
	NumReceiveThread      types.Int64  `tfsdk:"num_receive_thread"`
	Mtls                  types.String `tfsdk:"mtls"`
	Hep3Timestamp         types.Bool   `tfsdk:"hep3_timestamp"`
	RecvTimestamp         types.Bool   `tfsdk:"recv_timestamp"`
	PrivateKeyPath        types.String `tfsdk:"private_key_path"`
	CertFilePath          types.String `tfsdk:"cert_file_path"`
	MtlsKeyAlias          types.String `tfsdk:"mtls_key_alias"`
	NumEgressSCTPFlows    types.Int64  `tfsdk:"num_egress_sctp_flows"`
	EgressSCTPFlowTimeout types.Int64  `tfsdk:"egress_sctp_flow_timeout"`
	ServiceMapTableAlias  types.String `tfsdk:"service_map_table_alias"`
}

var rxTunnelAttrTypes = map[string]attr.Type{
	"rx_type":          types.StringType,
	"listen_ipaddress": types.StringType,
	"listen_port":      types.Int64Type,
	"from_port":        types.Int64Type,
	"rx_vni_id":        types.Int64Type,
	"rx_thread":        types.Int64Type,
}

var txTunnelAttrTypes = map[string]attr.Type{
	"tx_type":             types.StringType,
	"tx_remote_ipaddress": types.StringType,
	"tx_src_ipaddress":    types.StringType,
	"tx_src_port":         types.Int64Type,
	"tx_dst_port":         types.Int64Type,
	"l2gre_key":           types.Int64Type,
	"tx_vni_id":           types.Int64Type,
}

var http2MonitoredFlowsAttrTypes = map[string]attr.Type{
	"num_monitored_stream_flows": types.Int64Type,
	"http2_request_timeout":      types.Int64Type,
}

var tcpMonitoredFlowsAttrTypes = map[string]attr.Type{
	"num_monitored_tcp_flows":     types.Int64Type,
	"tcp_flow_timeout":            types.Int64Type,
	"tcp_flow_reassembly_timeout": types.Int64Type,
}

var scpConfigAttrTypes = map[string]attr.Type{
	"num_tcp_flows":                          types.Int64Type,
	"num_transaction_flows":                  types.Int64Type,
	"tcp_flow_timeout":                       types.Int64Type,
	"scp_transaction_timeout":                types.Int64Type,
	"header_index":                           types.BoolType,
	"header_compression_code":                types.BoolType,
	"nrf_discovery_enabled":                  types.BoolType,
	"add_gigamon_header":                     types.BoolType,
	"nf_instance_alias":                      types.StringType,
	"fqdn_alias":                             types.StringType,
	"ua_alias":                               types.StringType,
	"min_tcp_flow_client_port":               types.Int64Type,
	"max_tcp_flow_client_port":               types.Int64Type,
	"packet_capture_log_level":               types.StringType,
	"csv_logging_log_level":                  types.StringType,
	"num_scp_processing_threads":             types.Int64Type,
	"num_tcp_flow_client_port_per_thread":    types.Int64Type,
	"nokia_inbound_use_3gpp_target_api_root": types.BoolType,
	"nokia_inbound_replace_authority":        types.BoolType,
	"http2_monitored_flows":                  types.ObjectType{AttrTypes: http2MonitoredFlowsAttrTypes},
	"tcp_monitored_flows":                    types.ObjectType{AttrTypes: tcpMonitoredFlowsAttrTypes},
}

var hep3ConfigAttrTypes = map[string]attr.Type{
	"num_ingress_tcp_conn":     types.Int64Type,
	"num_egress_tcp_flows":     types.Int64Type,
	"ingress_tcp_timeout":      types.Int64Type,
	"egress_tcp_flow_timeout":  types.Int64Type,
	"num_receive_thread":       types.Int64Type,
	"mtls":                     types.StringType,
	"hep3_timestamp":           types.BoolType,
	"recv_timestamp":           types.BoolType,
	"private_key_path":         types.StringType,
	"cert_file_path":           types.StringType,
	"mtls_key_alias":           types.StringType,
	"num_egress_sctp_flows":    types.Int64Type,
	"egress_sctp_flow_timeout": types.Int64Type,
	"service_map_table_alias":  types.StringType,
}

var rxTunnelObjectType = types.ObjectType{AttrTypes: rxTunnelAttrTypes}
var http2MonitoredFlowsObjectType = types.ObjectType{AttrTypes: http2MonitoredFlowsAttrTypes}
var tcpMonitoredFlowsObjectType = types.ObjectType{AttrTypes: tcpMonitoredFlowsAttrTypes}

// FM5GCloud represents the wire format for 5G Cloud application in the FM API
//
//nolint:revive
type FM5GCloud struct {
	AppType        string                 `json:"app_type"`
	Alias          string                 `json:"alias,omitempty"`
	Name           string                 `json:"name,omitempty"`
	AppConfig      map[string]interface{} `json:"app_config"`
	Mode           string                 `json:"mode,omitempty"`
	RxTunnel       []interface{}          `json:"rxTunnel,omitempty"`
	TxTunnel       map[string]interface{} `json:"txTunnel,omitempty"`
	ScpConfig      map[string]interface{} `json:"scpConfig,omitempty"`
	Hep3Config     map[string]interface{} `json:"hep3Config,omitempty"`
	ToolMtu        interface{}            `json:"toolMtu,omitempty"`
	LogFolderLoc   string                 `json:"logFolderLoc,omitempty"`
	TunnelLogLevel interface{}            `json:"tunnelLogLevel,omitempty"`
}

// Metadata returns the resource type name
func (r *App5GCloud) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_5gcloud"
}

// Schema defines the resource schema
func (r *App5GCloud) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a 5G Cloud application instance on a monitoring session.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the 5G Cloud app resource",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"monitoring_session_id": schema.StringAttribute{
				Description: "The ID of the monitoring session to associate this app with. Format: monitoringSession::vmware::<uuid>",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"mode": schema.StringAttribute{
				Description: "5G Cloud operating mode.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"casaVtap",
						"oracleSCP",
						"nokiaSCPInbound",
						"nokiaSCPIn-Outbound",
						"SBINF",
						"ericssonSCPOutbound",
						"ericssonSCPIn-Outbound",
						"nokiaHEP3Inbound",
						"nokiaHEP3IMS",
					),
				},
			},
			"rx_tunnel": schema.ListNestedAttribute{
				Description: "Receive tunnel configuration.",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"rx_type": schema.StringAttribute{
							Description: "Receive tunnel type.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.OneOf("vxlan", "tcp"),
							},
						},
						"listen_ipaddress": schema.StringAttribute{
							Description: "Listen IP address.",
							Required:    true,
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
							},
						},
						"listen_port": schema.Int64Attribute{
							Description: "Listen port.",
							Required:    true,
							Validators: []validator.Int64{
								int64validator.Between(1, 65535),
							},
						},
						"from_port": schema.Int64Attribute{
							Description: "Source port for the receive tunnel. Not configurable when rx_type is tcp.",
							Optional:    true,
							Validators: []validator.Int64{
								int64validator.Between(1, 65535),
							},
						},
						"rx_vni_id": schema.Int64Attribute{
							Description: "Receive VNI ID.",
							Optional:    true,
							Computed:    true,
							Default:     int64default.StaticInt64(0),
							Validators: []validator.Int64{
								int64validator.Between(0, 16777215),
							},
						},
						"rx_thread": schema.Int64Attribute{
							Description: "Receive thread count. casaVtap always uses 1; all other supported modes default to 8.",
							Optional:    true,
							Validators: []validator.Int64{
								int64validator.Between(1, 16),
							},
						},
					},
				},
			},
			"tx_tunnel": schema.SingleNestedAttribute{
				Description: "Transmit tunnel configuration.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"tx_type": schema.StringAttribute{
						Description: "Transmit tunnel type.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.OneOf("vxlan", "l2gre", "udpgre"),
						},
					},
					"tx_remote_ipaddress": schema.StringAttribute{
						Description: "Remote IP address for transmit tunnel.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"tx_src_ipaddress": schema.StringAttribute{
						Description: "Source IP address for transmit tunnel.",
						Required:    true,
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(1),
						},
					},
					"tx_src_port": schema.Int64Attribute{
						Description: "Source port for transmit tunnel.",
						Required:    true,
						Validators: []validator.Int64{
							int64validator.Between(1, 65535),
						},
					},
					"tx_dst_port": schema.Int64Attribute{
						Description: "Destination port for transmit tunnel.",
						Required:    true,
						Validators: []validator.Int64{
							int64validator.Between(1, 65535),
						},
					},
					"l2gre_key": schema.Int64Attribute{
						Description: "L2GRE key. Required only when tx_type is l2gre.",
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(0),
						Validators: []validator.Int64{
							int64validator.Between(0, 4294967295),
						},
					},
					"tx_vni_id": schema.Int64Attribute{
						Description: "Transmit VNI ID. Required only when tx_type is vxlan.",
						Optional:    true,
						Computed:    true,
						Default:     int64default.StaticInt64(0),
						Validators: []validator.Int64{
							int64validator.Between(0, 16777215),
						},
					},
				},
			},
			"scp_config": schema.SingleNestedAttribute{
				Description: "SCP configuration. Supported for oracleSCP, Nokia SCP modes, SBINF, Ericsson SCP modes, and Nokia HEP3 modes.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"num_tcp_flows":                          schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(1024), Validators: []validator.Int64{int64validator.Between(128, 2048)}},
					"num_transaction_flows":                  schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(2048), Validators: []validator.Int64{int64validator.Between(128, 26000)}},
					"tcp_flow_timeout":                       schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(900), Validators: []validator.Int64{int64validator.Between(0, 7200)}},
					"scp_transaction_timeout":                schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(10), Validators: []validator.Int64{int64validator.Between(1, 300)}},
					"header_index":                           schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
					"header_compression_code":                schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
					"nrf_discovery_enabled":                  schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
					"add_gigamon_header":                     schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
					"nf_instance_alias":                      schema.StringAttribute{Optional: true},
					"fqdn_alias":                             schema.StringAttribute{Optional: true},
					"ua_alias":                               schema.StringAttribute{Optional: true},
					"min_tcp_flow_client_port":               schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(32768), Validators: []validator.Int64{int64validator.Between(1023, 65535)}},
					"max_tcp_flow_client_port":               schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(36863), Validators: []validator.Int64{int64validator.Between(1023, 65535)}},
					"packet_capture_log_level":               schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("none"), Validators: []validator.String{stringvalidator.OneOf("all", "receive", "transmit", "none")}},
					"csv_logging_log_level":                  schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("none"), Validators: []validator.String{stringvalidator.OneOf("all", "flow", "message", "transaction", "none")}},
					"num_scp_processing_threads":             schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(8), Validators: []validator.Int64{int64validator.Between(1, 16)}},
					"num_tcp_flow_client_port_per_thread":    schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(1000), Validators: []validator.Int64{int64validator.Between(100, 8000)}},
					"nokia_inbound_use_3gpp_target_api_root": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
					"nokia_inbound_replace_authority":        schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
					"http2_monitored_flows": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"num_monitored_stream_flows": schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(1024), Validators: []validator.Int64{int64validator.Between(1024, 16384)}},
							"http2_request_timeout":      schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(15), Validators: []validator.Int64{int64validator.Between(1, 300)}},
						},
					},
					"tcp_monitored_flows": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"num_monitored_tcp_flows":     schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(2048), Validators: []validator.Int64{int64validator.Between(1024, 32768)}},
							"tcp_flow_timeout":            schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(60), Validators: []validator.Int64{int64validator.Between(1, 7200)}},
							"tcp_flow_reassembly_timeout": schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(500), Validators: []validator.Int64{int64validator.Between(1, 7200)}},
						},
					},
				},
			},
			"hep3_config": schema.SingleNestedAttribute{
				Description: "HEP3 configuration. Supported only for nokiaHEP3Inbound and nokiaHEP3IMS.",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"num_ingress_tcp_conn":     schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(1024), Validators: []validator.Int64{int64validator.Between(128, 2048)}},
					"num_egress_tcp_flows":     schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(4096), Validators: []validator.Int64{int64validator.Between(1024, 16000000)}},
					"ingress_tcp_timeout":      schema.Int64Attribute{Computed: true, Default: int64default.StaticInt64(60)},
					"egress_tcp_flow_timeout":  schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(900), Validators: []validator.Int64{int64validator.Between(30, 7200)}},
					"num_receive_thread":       schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(8), Validators: []validator.Int64{int64validator.Between(1, 128)}},
					"mtls":                     schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("enable"), Validators: []validator.String{stringvalidator.OneOf("enable", "disable")}},
					"hep3_timestamp":           schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
					"recv_timestamp":           schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
					"private_key_path":         schema.StringAttribute{Computed: true, Default: stringdefault.StaticString("/usr/lib/vseries-web/api/crypto/private/cloud5g/pvt_key")},
					"cert_file_path":           schema.StringAttribute{Computed: true, Default: stringdefault.StaticString("/usr/lib/vseries-web/api/crypto/private/cloud5g/cloud5G.crt")},
					"mtls_key_alias":           schema.StringAttribute{Optional: true},
					"num_egress_sctp_flows":    schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(1024), Validators: []validator.Int64{int64validator.Between(128, 2000000)}},
					"egress_sctp_flow_timeout": schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(900), Validators: []validator.Int64{int64validator.Between(30, 7200)}},
					"service_map_table_alias":  schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("5g-servicemap")},
				},
			},
			"tool_mtu": schema.Int64Attribute{
				Description: "Tool MTU.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.Between(1400, 8800),
				},
			},
			"log_folder_loc": schema.StringAttribute{
				Description: "Log folder location.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("/var/log"),
				Validators: []validator.String{
					stringvalidator.OneOf("/var/log"),
				},
			},
			"tunnel_log_level": schema.Int64Attribute{
				Description: "Tunnel log level.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2),
				Validators: []validator.Int64{
					int64validator.Between(0, 5),
				},
			},
			"alias": schema.StringAttribute{
				Description: "Alias for the 5G Cloud application.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 256),
					stringvalidator.RegexMatches(app5GCloudNameRegex, "only alphanumeric, '.', '-' and '_' are allowed"),
				},
			},
		},
	}
}

// Configure configures the resource with provider client
func (r *App5GCloud) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*fmclient.FmClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *fmclient.FmClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.fmClient = client
}

// Create creates the 5G Cloud app resource
func (r *App5GCloud) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Creating 5G Cloud app resource")

	var data App5GCloudModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validate5GCloudConfig(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	fmPayload := buildFM5GCloudPayload(ctx, data)
	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{{
			EntityType:  "application",
			Operation:   "create",
			Application: fmPayload,
		}},
	}

	sessionID := data.MonitoringSessionId.ValueString()
	id, err := commonutils.UpdateMonSess(ctx, &updateReq, sessionID, r.fmClient)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Failed to create 5G Cloud app: %v", err))
		resp.Diagnostics.AddError(
			"Error creating 5G Cloud app",
			fmt.Sprintf("Could not create app on session %s: %s", sessionID, err.Error()),
		)
		return
	}

	typedID, err := commonutils.MakeTypedID(commonutils.ModuleApp, commonutils.Type5GCloud, id)
	if err != nil {
		resp.Diagnostics.AddError("Error creating typed ID", err.Error())
		return
	}

	fmData := FM5GCloud{}
	err = GetMSAppData(ctx, sessionID, id, app5GCloudTypeID, "", &fmData, r.fmClient)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Failed to read created 5G Cloud app: %v", err))
		resp.Diagnostics.AddError("Error reading created app", err.Error())
		return
	}

	stateData := mapFM5GCloudToState(ctx, fmData, sessionID, typedID, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Successfully created 5G Cloud app: %s", stateData.Id.ValueString()))
}

// Read reads the 5G Cloud app resource from the FM API
func (r *App5GCloud) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Reading 5G Cloud app resource")

	var data App5GCloudModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rawID, err := commonutils.UUIDFromTypedID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing ID", err.Error())
		return
	}

	sessionID := data.MonitoringSessionId.ValueString()
	fmData := FM5GCloud{}
	err = GetMSAppData(ctx, sessionID, rawID, app5GCloudTypeID, "", &fmData, r.fmClient)
	if err != nil {
		var fmErr *fmclient.FMErrors
		if errors.As(err, &fmErr) && fmErr.ErrorCode() == fmclient.ObjectNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		tflog.Error(ctx, fmt.Sprintf("Failed to read 5G Cloud app: %v", err))
		resp.Diagnostics.AddError(
			"Error reading 5G Cloud app",
			fmt.Sprintf("Could not read app from session %s: %s", sessionID, err.Error()),
		)
		return
	}

	stateData := mapFM5GCloudToState(ctx, fmData, sessionID, data.Id.ValueString(), nil)
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Successfully read 5G Cloud app: %s", stateData.Id.ValueString()))
}

// Update updates the 5G Cloud app resource
func (r *App5GCloud) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Updating 5G Cloud app resource")

	var planData App5GCloudModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	validate5GCloudConfig(ctx, planData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	rawID, err := commonutils.UUIDFromTypedID(planData.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing ID", err.Error())
		return
	}

	fmPayload := buildFM5GCloudPayload(ctx, planData)
	fmPayload["id"] = rawID

	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{{
			EntityType:  "application",
			Operation:   "update",
			Application: fmPayload,
		}},
	}

	sessionID := planData.MonitoringSessionId.ValueString()
	_, err = commonutils.UpdateMonSess(ctx, &updateReq, sessionID, r.fmClient)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Failed to update 5G Cloud app: %v", err))
		resp.Diagnostics.AddError(
			"Error updating 5G Cloud app",
			fmt.Sprintf("Could not update app on session %s: %s", sessionID, err.Error()),
		)
		return
	}

	fmData := FM5GCloud{}
	err = GetMSAppData(ctx, sessionID, rawID, app5GCloudTypeID, "", &fmData, r.fmClient)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Failed to read updated 5G Cloud app: %v", err))
		resp.Diagnostics.AddError("Error reading updated app", err.Error())
		return
	}

	stateData := mapFM5GCloudToState(ctx, fmData, sessionID, planData.Id.ValueString(), &planData)
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Successfully updated 5G Cloud app: %s", stateData.Id.ValueString()))
}

// Delete deletes the 5G Cloud app resource
func (r *App5GCloud) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Deleting 5G Cloud app resource")

	var data App5GCloudModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rawID, err := commonutils.UUIDFromTypedID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing ID", err.Error())
		return
	}

	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{{
			EntityType: "application",
			Operation:  "delete",
			Application: map[string]interface{}{
				"id":       rawID,
				"app_type": app5GCloudTypeID,
				"name":     app5GCloudTypeID,
			},
		}},
	}

	sessionID := data.MonitoringSessionId.ValueString()
	_, err = commonutils.UpdateMonSess(ctx, &updateReq, sessionID, r.fmClient)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Failed to delete 5G Cloud app: %v", err))
		resp.Diagnostics.AddError(
			"Error deleting 5G Cloud app",
			fmt.Sprintf("Could not delete app from session %s: %s", sessionID, err.Error()),
		)
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Successfully deleted 5G Cloud app from session %s", sessionID))
}

// ImportState implements the import functionality for 5G Cloud app
func (r *App5GCloud) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, fmt.Sprintf("Importing 5G Cloud app resource: %s", req.ID))

	parts := strings.Split(req.ID, "::")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			fmt.Sprintf("Expected format: <sessionID>::<resourceID>, got: %s", req.ID),
		)
		return
	}

	sessionID := parts[0]
	rawID := parts[1]

	typedID, err := commonutils.MakeTypedID(commonutils.ModuleApp, commonutils.Type5GCloud, rawID)
	if err != nil {
		resp.Diagnostics.AddError("Error creating typed ID", err.Error())
		return
	}

	fmData := FM5GCloud{}
	err = GetMSAppData(ctx, sessionID, rawID, app5GCloudTypeID, "", &fmData, r.fmClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing 5G Cloud app",
			fmt.Sprintf("Could not read app from session %s: %s", sessionID, err.Error()),
		)
		return
	}

	stateData := mapFM5GCloudToState(ctx, fmData, sessionID, typedID, nil)
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Successfully imported 5G Cloud app: %s", stateData.Id.ValueString()))
}

// buildFM5GCloudPayload converts the Terraform model to FM API payload format.
func buildFM5GCloudPayload(ctx context.Context, model App5GCloudModel) map[string]interface{} {
	appConfig := map[string]interface{}{}

	setStringConfigValue(appConfig, "mode", model.Mode)
	setStringConfigValue(appConfig, "alias", model.Alias)
	setInt64ConfigValue(appConfig, "toolMtu", model.ToolMtu)
	setStringConfigValue(appConfig, "logFolderLoc", model.LogFolderLoc)
	setInt64ConfigValue(appConfig, "tunnelLogLevel", model.TunnelLogLevel)

	if !model.RxTunnel.IsNull() && !model.RxTunnel.IsUnknown() {
		var tunnels []RxTunnelModel
		_ = model.RxTunnel.ElementsAs(ctx, &tunnels, false)

		rxTunnels := make([]map[string]interface{}, 0, len(tunnels))
		for _, tunnel := range tunnels {
			tunnelMap := map[string]interface{}{}
			setStringConfigValue(tunnelMap, "rxType", tunnel.RxType)
			setStringConfigValue(tunnelMap, "listenIpaddress", tunnel.ListenIpaddress)
			setInt64ConfigValue(tunnelMap, "listenPort", tunnel.ListenPort)
			setInt64ConfigValue(tunnelMap, "fromPort", tunnel.FromPort)
			if tunnel.RxType.ValueString() != "tcp" {
				if tunnel.RxVNIId.IsNull() || tunnel.RxVNIId.IsUnknown() {
					tunnelMap["rxVNIId"] = int64(0)
				} else {
					setInt64ConfigValue(tunnelMap, "rxVNIId", tunnel.RxVNIId)
				}
				if tunnel.RxThread.IsNull() || tunnel.RxThread.IsUnknown() {
					if model.Mode.ValueString() == "casaVtap" {
						tunnelMap["rxThread"] = int64(1)
					} else {
						tunnelMap["rxThread"] = int64(8)
					}
				} else {
					setInt64ConfigValue(tunnelMap, "rxThread", tunnel.RxThread)
				}
			}
			rxTunnels = append(rxTunnels, tunnelMap)
		}
		appConfig["rxTunnel"] = rxTunnels
	}

	if !model.TxTunnel.IsNull() && !model.TxTunnel.IsUnknown() {
		var tunnel TxTunnelModel
		_ = model.TxTunnel.As(ctx, &tunnel, basetypes.ObjectAsOptions{})

		tunnelMap := map[string]interface{}{}
		setStringConfigValue(tunnelMap, "txType", tunnel.TxType)
		setStringConfigValue(tunnelMap, "txRemoteIpaddress", tunnel.TxRemoteIpaddress)
		setStringConfigValue(tunnelMap, "txSrcIpaddress", tunnel.TxSrcIpaddress)
		setInt64ConfigValue(tunnelMap, "txSrcPort", tunnel.TxSrcPort)
		setInt64ConfigValue(tunnelMap, "txDstPort", tunnel.TxDstPort)
		switch tunnel.TxType.ValueString() {
		case "l2gre":
			if tunnel.L2GreKey.IsNull() || tunnel.L2GreKey.IsUnknown() {
				tunnelMap["l2greKey"] = int64(0)
			} else {
				setInt64ConfigValue(tunnelMap, "l2greKey", tunnel.L2GreKey)
			}
		case "vxlan":
			if tunnel.TxVNIId.IsNull() || tunnel.TxVNIId.IsUnknown() {
				tunnelMap["txVNIId"] = int64(0)
			} else {
				setInt64ConfigValue(tunnelMap, "txVNIId", tunnel.TxVNIId)
			}
		}
		appConfig["txTunnel"] = tunnelMap
	}

	if !model.ScpConfig.IsNull() && !model.ScpConfig.IsUnknown() {
		var scp ScpConfigModel
		_ = model.ScpConfig.As(ctx, &scp, basetypes.ObjectAsOptions{})

		scpMap := map[string]interface{}{}
		setInt64ConfigValue(scpMap, "numTCPFlows", scp.NumTCPFlows)
		setInt64ConfigValue(scpMap, "numTransactionFlows", scp.NumTransactionFlows)
		setInt64ConfigValue(scpMap, "tcpFlowTimeout", scp.TcpFlowTimeout)
		setInt64ConfigValue(scpMap, "scpTransactionTimeout", scp.ScpTransactionTimeout)
		setBoolConfigValue(scpMap, "headerIndex", scp.HeaderIndex)
		setBoolConfigValue(scpMap, "headerCompressionCode", scp.HeaderCompressionCode)
		setBoolConfigValue(scpMap, "nrfDiscoveryEnabled", scp.NrfDiscoveryEnabled)
		setBoolConfigValue(scpMap, "addGigamonHeader", scp.AddGigamonHeader)
		setStringConfigValue(scpMap, "nfInstanceAlias", scp.NfInstanceAlias)
		setStringConfigValue(scpMap, "fqdnAlias", scp.FqdnAlias)
		setStringConfigValue(scpMap, "uaAlias", scp.UaAlias)
		setInt64ConfigValue(scpMap, "minTcpFlowClientPort", scp.MinTcpFlowClientPort)
		setInt64ConfigValue(scpMap, "maxTcpFlowClientPort", scp.MaxTcpFlowClientPort)
		setStringConfigValue(scpMap, "packetCaptureLogLevel", scp.PacketCaptureLogLevel)
		setStringConfigValue(scpMap, "csvLoggingLogLevel", scp.CsvLoggingLogLevel)
		setInt64ConfigValue(scpMap, "numSCPProcessingThreads", scp.NumSCPProcessingThreads)
		setInt64ConfigValue(scpMap, "numTCPFlowCLientPortPerThread", scp.NumTCPFlowClientPortPerThread)
		setBoolConfigValue(scpMap, "nokiaInboundUse3gppTargetApiRoot", scp.NokiaInboundUse3gppTargetApiRoot)
		setBoolConfigValue(scpMap, "nokiaInboundReplaceAuthority", scp.NokiaInboundReplaceAuthority)

		if !scp.Http2MonitoredFlows.IsNull() && !scp.Http2MonitoredFlows.IsUnknown() {
			var http2 Http2MonitoredFlowsModel
			_ = scp.Http2MonitoredFlows.As(ctx, &http2, basetypes.ObjectAsOptions{})
			http2Map := map[string]interface{}{}
			setInt64ConfigValue(http2Map, "numMonitoredStreamFlows", http2.NumMonitoredStreamFlows)
			setInt64ConfigValue(http2Map, "http2RequestTimeout", http2.Http2RequestTimeout)
			scpMap["http2MonitoredFlows"] = http2Map
		}

		if !scp.TcpMonitoredFlows.IsNull() && !scp.TcpMonitoredFlows.IsUnknown() {
			var tcp TcpMonitoredFlowsModel
			_ = scp.TcpMonitoredFlows.As(ctx, &tcp, basetypes.ObjectAsOptions{})
			tcpMap := map[string]interface{}{}
			setInt64ConfigValue(tcpMap, "numMonitoredTCPFlows", tcp.NumMonitoredTCPFlows)
			setInt64ConfigValue(tcpMap, "tcpFlowTimeout", tcp.TcpFlowTimeout)
			setInt64ConfigValue(tcpMap, "tcpFlowReassemblyTimeout", tcp.TcpFlowReassemblyTimeout)
			scpMap["tcpMonitoredFlows"] = tcpMap
		}

		appConfig["scpConfig"] = scpMap
	}

	if !model.Hep3Config.IsNull() && !model.Hep3Config.IsUnknown() {
		var hep3 Hep3ConfigModel
		_ = model.Hep3Config.As(ctx, &hep3, basetypes.ObjectAsOptions{})

		hep3Map := map[string]interface{}{}
		setInt64ConfigValue(hep3Map, "numIngressTCPconn", hep3.NumIngressTCPconn)
		setInt64ConfigValue(hep3Map, "numEgressTCPFlows", hep3.NumEgressTCPFlows)
		hep3Map["ingressTCPTimeout"] = int64(60)
		setInt64ConfigValue(hep3Map, "egressTCPFlowTimeout", hep3.EgressTCPFlowTimeout)
		setInt64ConfigValue(hep3Map, "numReceiveThread", hep3.NumReceiveThread)
		setStringConfigValue(hep3Map, "mtls", hep3.Mtls)
		setBoolConfigValue(hep3Map, "hep3Timestamp", hep3.Hep3Timestamp)
		setBoolConfigValue(hep3Map, "recvTimestamp", hep3.RecvTimestamp)
		hep3Map["privateKeyPath"] = "/usr/lib/vseries-web/api/crypto/private/cloud5g/pvt_key"
		hep3Map["certFilePath"] = "/usr/lib/vseries-web/api/crypto/private/cloud5g/cloud5G.crt"
		setStringConfigValue(hep3Map, "mtlsKeyAlias", hep3.MtlsKeyAlias)
		setInt64ConfigValue(hep3Map, "numEgressSCTPFlows", hep3.NumEgressSCTPFlows)
		setInt64ConfigValue(hep3Map, "egressSCTPFlowTimeout", hep3.EgressSCTPFlowTimeout)
		hep3Map["serviceMapTableAlias"] = "5g-servicemap"
		appConfig["hep3Config"] = hep3Map
	}

	return map[string]interface{}{
		"app_type":       app5GCloudTypeID,
		"alias":          model.Alias.ValueString(),
		"name":           app5GCloudTypeID,
		"mode":           appConfig["mode"],
		"toolMtu":        appConfig["toolMtu"],
		"logFolderLoc":   appConfig["logFolderLoc"],
		"tunnelLogLevel": appConfig["tunnelLogLevel"],
		"rxTunnel":       appConfig["rxTunnel"],
		"txTunnel":       appConfig["txTunnel"],
		"scpConfig":      appConfig["scpConfig"],
		"hep3Config":     appConfig["hep3Config"],
		"app_config":     appConfig,
	}
}

// mapFM5GCloudToState converts FM API response to Terraform model.
func mapFM5GCloudToState(ctx context.Context, fmData FM5GCloud, sessionID string, typedID string, base *App5GCloudModel) App5GCloudModel {
	model := App5GCloudModel{
		Id:                  types.StringValue(typedID),
		MonitoringSessionId: types.StringValue(sessionID),
		Mode:                types.StringNull(),
		RxTunnel:            types.ListNull(rxTunnelObjectType),
		TxTunnel:            types.ObjectNull(txTunnelAttrTypes),
		ScpConfig:           types.ObjectNull(scpConfigAttrTypes),
		Hep3Config:          types.ObjectNull(hep3ConfigAttrTypes),
		ToolMtu:             types.Int64Null(),
		LogFolderLoc:        types.StringValue("/var/log"),
		TunnelLogLevel:      types.Int64Value(2),
		Alias:               types.StringNull(),
	}

	if base != nil {
		model.Mode = base.Mode
		model.RxTunnel = base.RxTunnel
		model.TxTunnel = base.TxTunnel
		model.ScpConfig = base.ScpConfig
		model.Hep3Config = base.Hep3Config
		model.ToolMtu = base.ToolMtu
		model.LogFolderLoc = base.LogFolderLoc
		model.TunnelLogLevel = base.TunnelLogLevel
		model.Alias = base.Alias
	}

	cfg := fmData.AppConfig
	if cfg == nil {
		cfg = map[string]interface{}{}
	}

	if _, ok := cfg["mode"]; !ok && fmData.Mode != "" {
		cfg["mode"] = fmData.Mode
	}
	if _, ok := cfg["alias"]; !ok && fmData.Alias != "" {
		cfg["alias"] = fmData.Alias
	}
	if _, ok := cfg["toolMtu"]; !ok && fmData.ToolMtu != nil {
		cfg["toolMtu"] = fmData.ToolMtu
	}
	if _, ok := cfg["logFolderLoc"]; !ok && fmData.LogFolderLoc != "" {
		cfg["logFolderLoc"] = fmData.LogFolderLoc
	}
	if _, ok := cfg["tunnelLogLevel"]; !ok && fmData.TunnelLogLevel != nil {
		cfg["tunnelLogLevel"] = fmData.TunnelLogLevel
	}
	if _, ok := cfg["rxTunnel"]; !ok && fmData.RxTunnel != nil {
		cfg["rxTunnel"] = fmData.RxTunnel
	}
	if _, ok := cfg["txTunnel"]; !ok && fmData.TxTunnel != nil {
		cfg["txTunnel"] = fmData.TxTunnel
	}
	if _, ok := cfg["scpConfig"]; !ok && fmData.ScpConfig != nil {
		cfg["scpConfig"] = fmData.ScpConfig
	}
	if _, ok := cfg["hep3Config"]; !ok && fmData.Hep3Config != nil {
		cfg["hep3Config"] = fmData.Hep3Config
	}

	if mode, ok := cfg["mode"].(string); ok {
		model.Mode = types.StringValue(mode)
	}
	if alias, ok := cfg["alias"].(string); ok {
		model.Alias = types.StringValue(alias)
	}
	if model.Alias.IsNull() && fmData.Alias != "" {
		model.Alias = types.StringValue(fmData.Alias)
	}
	if toolMtu, ok := cfg["toolMtu"]; ok {
		model.ToolMtu = toInt64Value(toolMtu)
	}
	if logFolderLoc, ok := cfg["logFolderLoc"].(string); ok {
		model.LogFolderLoc = types.StringValue(logFolderLoc)
	}
	if tunnelLogLevel, ok := cfg["tunnelLogLevel"]; ok {
		model.TunnelLogLevel = toInt64Value(tunnelLogLevel)
	}

	if rawRxTunnels, ok := cfg["rxTunnel"].([]interface{}); ok {
		rxVals := make([]attr.Value, 0, len(rawRxTunnels))
		for _, rawTunnel := range rawRxTunnels {
			tunnelMap, _ := rawTunnel.(map[string]interface{})
			rx := RxTunnelModel{
				RxType:          types.StringNull(),
				ListenIpaddress: types.StringNull(),
				ListenPort:      types.Int64Null(),
				FromPort:        types.Int64Null(),
				RxVNIId:         types.Int64Value(0),
				RxThread:        types.Int64Null(),
			}
			if tunnelMap != nil {
				if v, ok := tunnelMap["rxType"].(string); ok {
					rx.RxType = types.StringValue(v)
				}
				if v, ok := tunnelMap["listenIpaddress"].(string); ok {
					rx.ListenIpaddress = types.StringValue(v)
				}
				if v, ok := tunnelMap["listenPort"]; ok {
					rx.ListenPort = toInt64Value(v)
				}
				if v, ok := tunnelMap["fromPort"]; ok {
					rx.FromPort = toInt64Value(v)
				}
				if v, ok := tunnelMap["rxVNIId"]; ok {
					rx.RxVNIId = toInt64Value(v)
				}
				if v, ok := tunnelMap["rxThread"]; ok {
					rx.RxThread = toInt64Value(v)
				} else if rx.RxType.ValueString() != "tcp" && model.Mode.ValueString() == "casaVtap" {
					rx.RxThread = types.Int64Value(1)
				} else if rx.RxType.ValueString() != "tcp" {
					rx.RxThread = types.Int64Value(8)
				}
			}
			rv, _ := types.ObjectValueFrom(ctx, rxTunnelAttrTypes, rx)
			rxVals = append(rxVals, rv)
		}
		rxList, _ := types.ListValue(rxTunnelObjectType, rxVals)
		model.RxTunnel = rxList
	}

	if rawTxTunnel, ok := cfg["txTunnel"].(map[string]interface{}); ok {
		tx := TxTunnelModel{
			TxType:            types.StringNull(),
			TxRemoteIpaddress: types.StringNull(),
			TxSrcIpaddress:    types.StringNull(),
			TxSrcPort:         types.Int64Null(),
			TxDstPort:         types.Int64Null(),
			L2GreKey:          types.Int64Value(0),
			TxVNIId:           types.Int64Value(0),
		}
		if v, ok := rawTxTunnel["txType"].(string); ok {
			tx.TxType = types.StringValue(v)
		}
		if v, ok := rawTxTunnel["txRemoteIpaddress"].(string); ok {
			tx.TxRemoteIpaddress = types.StringValue(v)
		}
		if v, ok := rawTxTunnel["txSrcIpaddress"].(string); ok {
			tx.TxSrcIpaddress = types.StringValue(v)
		}
		if v, ok := rawTxTunnel["txSrcPort"]; ok {
			tx.TxSrcPort = toInt64Value(v)
		}
		if v, ok := rawTxTunnel["txDstPort"]; ok {
			tx.TxDstPort = toInt64Value(v)
		}
		if v, ok := rawTxTunnel["l2greKey"]; ok {
			tx.L2GreKey = toInt64Value(v)
		}
		if v, ok := rawTxTunnel["txVNIId"]; ok {
			tx.TxVNIId = toInt64Value(v)
		}
		txObj, _ := types.ObjectValueFrom(ctx, txTunnelAttrTypes, tx)
		model.TxTunnel = txObj
	}

	if rawScpConfig, ok := cfg["scpConfig"].(map[string]interface{}); ok {
		scp := ScpConfigModel{
			NumTCPFlows:                      types.Int64Value(1024),
			NumTransactionFlows:              types.Int64Value(2048),
			TcpFlowTimeout:                   types.Int64Value(900),
			ScpTransactionTimeout:            types.Int64Value(10),
			HeaderIndex:                      types.BoolValue(false),
			HeaderCompressionCode:            types.BoolValue(false),
			NrfDiscoveryEnabled:              types.BoolValue(true),
			AddGigamonHeader:                 types.BoolValue(false),
			NfInstanceAlias:                  types.StringNull(),
			FqdnAlias:                        types.StringNull(),
			UaAlias:                          types.StringNull(),
			MinTcpFlowClientPort:             types.Int64Value(32768),
			MaxTcpFlowClientPort:             types.Int64Value(36863),
			PacketCaptureLogLevel:            types.StringValue("none"),
			CsvLoggingLogLevel:               types.StringValue("none"),
			NumSCPProcessingThreads:          types.Int64Value(8),
			NumTCPFlowClientPortPerThread:    types.Int64Value(1000),
			NokiaInboundUse3gppTargetApiRoot: types.BoolValue(false),
			NokiaInboundReplaceAuthority:     types.BoolValue(false),
			Http2MonitoredFlows:              types.ObjectNull(http2MonitoredFlowsAttrTypes),
			TcpMonitoredFlows:                types.ObjectNull(tcpMonitoredFlowsAttrTypes),
		}
		if v, ok := rawScpConfig["numTCPFlows"]; ok {
			scp.NumTCPFlows = toInt64Value(v)
		}
		if v, ok := rawScpConfig["numTransactionFlows"]; ok {
			scp.NumTransactionFlows = toInt64Value(v)
		}
		if v, ok := rawScpConfig["tcpFlowTimeout"]; ok {
			scp.TcpFlowTimeout = toInt64Value(v)
		}
		if v, ok := rawScpConfig["scpTransactionTimeout"]; ok {
			scp.ScpTransactionTimeout = toInt64Value(v)
		}
		if v, ok := rawScpConfig["headerIndex"]; ok {
			scp.HeaderIndex = toBoolValue(v)
		}
		if v, ok := rawScpConfig["headerCompressionCode"]; ok {
			scp.HeaderCompressionCode = toBoolValue(v)
		}
		if v, ok := rawScpConfig["nrfDiscoveryEnabled"]; ok {
			scp.NrfDiscoveryEnabled = toBoolValue(v)
		}
		if v, ok := rawScpConfig["addGigamonHeader"]; ok {
			scp.AddGigamonHeader = toBoolValue(v)
		}
		if v, ok := rawScpConfig["nfInstanceAlias"].(string); ok {
			scp.NfInstanceAlias = types.StringValue(v)
		}
		if v, ok := rawScpConfig["fqdnAlias"].(string); ok {
			scp.FqdnAlias = types.StringValue(v)
		}
		if v, ok := rawScpConfig["uaAlias"].(string); ok {
			scp.UaAlias = types.StringValue(v)
		}
		if v, ok := rawScpConfig["minTcpFlowClientPort"]; ok {
			scp.MinTcpFlowClientPort = toInt64Value(v)
		}
		if v, ok := rawScpConfig["maxTcpFlowClientPort"]; ok {
			scp.MaxTcpFlowClientPort = toInt64Value(v)
		}
		if v, ok := rawScpConfig["packetCaptureLogLevel"].(string); ok {
			scp.PacketCaptureLogLevel = types.StringValue(v)
		}
		if v, ok := rawScpConfig["csvLoggingLogLevel"].(string); ok {
			scp.CsvLoggingLogLevel = types.StringValue(v)
		}
		if v, ok := rawScpConfig["numSCPProcessingThreads"]; ok {
			scp.NumSCPProcessingThreads = toInt64Value(v)
		}
		if v, ok := rawScpConfig["numTCPFlowCLientPortPerThread"]; ok {
			scp.NumTCPFlowClientPortPerThread = toInt64Value(v)
		}
		if v, ok := rawScpConfig["nokiaInboundUse3gppTargetApiRoot"]; ok {
			scp.NokiaInboundUse3gppTargetApiRoot = toBoolValue(v)
		}
		if v, ok := rawScpConfig["nokiaInboundReplaceAuthority"]; ok {
			scp.NokiaInboundReplaceAuthority = toBoolValue(v)
		}

		if rawHTTP2, ok := rawScpConfig["http2MonitoredFlows"].(map[string]interface{}); ok {
			http2 := Http2MonitoredFlowsModel{
				NumMonitoredStreamFlows: types.Int64Value(1024),
				Http2RequestTimeout:     types.Int64Value(15),
			}
			if v, ok := rawHTTP2["numMonitoredStreamFlows"]; ok {
				http2.NumMonitoredStreamFlows = toInt64Value(v)
			}
			if v, ok := rawHTTP2["http2RequestTimeout"]; ok {
				http2.Http2RequestTimeout = toInt64Value(v)
			}
			http2Obj, _ := types.ObjectValueFrom(ctx, http2MonitoredFlowsAttrTypes, http2)
			scp.Http2MonitoredFlows = http2Obj
		}

		if rawTCP, ok := rawScpConfig["tcpMonitoredFlows"].(map[string]interface{}); ok {
			tcp := TcpMonitoredFlowsModel{
				NumMonitoredTCPFlows:     types.Int64Value(2048),
				TcpFlowTimeout:           types.Int64Value(60),
				TcpFlowReassemblyTimeout: types.Int64Value(500),
			}
			if v, ok := rawTCP["numMonitoredTCPFlows"]; ok {
				tcp.NumMonitoredTCPFlows = toInt64Value(v)
			}
			if v, ok := rawTCP["tcpFlowTimeout"]; ok {
				tcp.TcpFlowTimeout = toInt64Value(v)
			}
			if v, ok := rawTCP["tcpFlowReassemblyTimeout"]; ok {
				tcp.TcpFlowReassemblyTimeout = toInt64Value(v)
			}
			tcpObj, _ := types.ObjectValueFrom(ctx, tcpMonitoredFlowsAttrTypes, tcp)
			scp.TcpMonitoredFlows = tcpObj
		}

		scpObj, _ := types.ObjectValueFrom(ctx, scpConfigAttrTypes, scp)
		model.ScpConfig = scpObj
	}

	if rawHep3Config, ok := cfg["hep3Config"].(map[string]interface{}); ok {
		hep3 := Hep3ConfigModel{
			NumIngressTCPconn:     types.Int64Value(1024),
			NumEgressTCPFlows:     types.Int64Value(4096),
			IngressTCPTimeout:     types.Int64Value(60),
			EgressTCPFlowTimeout:  types.Int64Value(900),
			NumReceiveThread:      types.Int64Value(8),
			Mtls:                  types.StringValue("enable"),
			Hep3Timestamp:         types.BoolValue(false),
			RecvTimestamp:         types.BoolValue(false),
			PrivateKeyPath:        types.StringValue("/usr/lib/vseries-web/api/crypto/private/cloud5g/pvt_key"),
			CertFilePath:          types.StringValue("/usr/lib/vseries-web/api/crypto/private/cloud5g/cloud5G.crt"),
			MtlsKeyAlias:          types.StringNull(),
			NumEgressSCTPFlows:    types.Int64Value(1024),
			EgressSCTPFlowTimeout: types.Int64Value(900),
			ServiceMapTableAlias:  types.StringValue("5g-servicemap"),
		}
		if v, ok := rawHep3Config["numIngressTCPconn"]; ok {
			hep3.NumIngressTCPconn = toInt64Value(v)
		}
		if v, ok := rawHep3Config["numEgressTCPFlows"]; ok {
			hep3.NumEgressTCPFlows = toInt64Value(v)
		}
		if v, ok := rawHep3Config["ingressTCPTimeout"]; ok {
			hep3.IngressTCPTimeout = toInt64Value(v)
		}
		if v, ok := rawHep3Config["egressTCPFlowTimeout"]; ok {
			hep3.EgressTCPFlowTimeout = toInt64Value(v)
		}
		if v, ok := rawHep3Config["numReceiveThread"]; ok {
			hep3.NumReceiveThread = toInt64Value(v)
		}
		if v, ok := rawHep3Config["mtls"].(string); ok {
			hep3.Mtls = types.StringValue(v)
		}
		if v, ok := rawHep3Config["hep3Timestamp"]; ok {
			hep3.Hep3Timestamp = toBoolValue(v)
		}
		if v, ok := rawHep3Config["recvTimestamp"]; ok {
			hep3.RecvTimestamp = toBoolValue(v)
		}
		if v, ok := rawHep3Config["privateKeyPath"].(string); ok {
			hep3.PrivateKeyPath = types.StringValue(v)
		}
		if v, ok := rawHep3Config["certFilePath"].(string); ok {
			hep3.CertFilePath = types.StringValue(v)
		}
		if v, ok := rawHep3Config["mtlsKeyAlias"].(string); ok {
			hep3.MtlsKeyAlias = types.StringValue(v)
		}
		if v, ok := rawHep3Config["numEgressSCTPFlows"]; ok {
			hep3.NumEgressSCTPFlows = toInt64Value(v)
		}
		if v, ok := rawHep3Config["egressSCTPFlowTimeout"]; ok {
			hep3.EgressSCTPFlowTimeout = toInt64Value(v)
		}
		if v, ok := rawHep3Config["serviceMapTableAlias"].(string); ok {
			hep3.ServiceMapTableAlias = types.StringValue(v)
		}
		hep3Obj, _ := types.ObjectValueFrom(ctx, hep3ConfigAttrTypes, hep3)
		model.Hep3Config = hep3Obj
	}

	return model
}

func validate5GCloudConfig(ctx context.Context, model App5GCloudModel, diags *diag.Diagnostics) {
	mode := model.Mode.ValueString()

	if !model.RxTunnel.IsNull() && !model.RxTunnel.IsUnknown() {
		var rxTunnels []RxTunnelModel
		diags.Append(model.RxTunnel.ElementsAs(ctx, &rxTunnels, false)...)
		if diags.HasError() {
			return
		}
		for idx, tunnel := range rxTunnels {
			rxType := tunnel.RxType.ValueString()
			switch mode {
			case "casaVtap", "oracleSCP", "nokiaSCPInbound", "nokiaSCPIn-Outbound":
				if rxType != "vxlan" {
					diags.AddError("Invalid rx_tunnel.rx_type", fmt.Sprintf("rx_tunnel[%d].rx_type must be vxlan when mode is %s", idx, mode))
				}
			case "nokiaHEP3Inbound", "nokiaHEP3IMS":
				if rxType != "tcp" {
					diags.AddError("Invalid rx_tunnel.rx_type", fmt.Sprintf("rx_tunnel[%d].rx_type must be tcp when mode is %s", idx, mode))
				}
			}
			if ip := tunnel.ListenIpaddress.ValueString(); ip != "" && net.ParseIP(ip) == nil {
				diags.AddError("Invalid rx_tunnel.listen_ipaddress", fmt.Sprintf("rx_tunnel[%d].listen_ipaddress must be a valid IPv4 or IPv6 address", idx))
			}
			if rxType == "tcp" {
				if !tunnel.RxThread.IsNull() && !tunnel.RxThread.IsUnknown() {
					diags.AddError("Invalid rx_tunnel.rx_thread", fmt.Sprintf("rx_tunnel[%d].rx_thread is not configurable when rx_type is tcp", idx))
				}
			} else if tunnel.FromPort.IsNull() || tunnel.FromPort.IsUnknown() {
				diags.AddError("Missing rx_tunnel.from_port", fmt.Sprintf("rx_tunnel[%d].from_port is required when rx_type is %s", idx, rxType))
			}
			if mode == "casaVtap" && !tunnel.RxThread.IsNull() && !tunnel.RxThread.IsUnknown() && tunnel.RxThread.ValueInt64() != 1 {
				diags.AddError("Invalid rx_tunnel.rx_thread", fmt.Sprintf("rx_tunnel[%d].rx_thread must be 1 when mode is casaVtap", idx))
			}
		}
	}

	if !model.TxTunnel.IsNull() && !model.TxTunnel.IsUnknown() {
		var tx TxTunnelModel
		diags.Append(model.TxTunnel.As(ctx, &tx, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return
		}

		allowedTxTypes := []string{"vxlan", "l2gre"}
		if mode == "casaVtap" {
			allowedTxTypes = []string{"vxlan", "l2gre", "udpgre"}
		}
		if !stringInSlice(tx.TxType.ValueString(), allowedTxTypes) {
			diags.AddError("Invalid tx_tunnel.tx_type", fmt.Sprintf("tx_tunnel.tx_type %q is not allowed when mode is %s", tx.TxType.ValueString(), mode))
		}
		if ip := tx.TxRemoteIpaddress.ValueString(); ip != "" && net.ParseIP(ip) == nil {
			diags.AddError("Invalid tx_tunnel.tx_remote_ipaddress", "tx_tunnel.tx_remote_ipaddress must be a valid IPv4 or IPv6 address")
		}
		if ip := tx.TxSrcIpaddress.ValueString(); ip != "" && net.ParseIP(ip) == nil {
			diags.AddError("Invalid tx_tunnel.tx_src_ipaddress", "tx_tunnel.tx_src_ipaddress must be a valid IPv4 or IPv6 address")
		}
	}

	if !model.ScpConfig.IsNull() && !model.ScpConfig.IsUnknown() {
		if !scpSupportedModes[mode] {
			diags.AddError("Invalid scp_config for mode", fmt.Sprintf("scp_config is not supported when mode is %s", mode))
			return
		}

		var scp ScpConfigModel
		diags.Append(model.ScpConfig.As(ctx, &scp, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return
		}
		if scp.FqdnAlias.IsNull() || scp.FqdnAlias.IsUnknown() || strings.TrimSpace(scp.FqdnAlias.ValueString()) == "" {
			diags.AddError("Missing scp_config.fqdn_alias", "scp_config.fqdn_alias is required when scp_config is configured")
		}
		if http2SupportedModes[mode] && (scp.UaAlias.IsNull() || scp.UaAlias.IsUnknown() || strings.TrimSpace(scp.UaAlias.ValueString()) == "") {
			diags.AddError("Missing scp_config.ua_alias", fmt.Sprintf("scp_config.ua_alias is required when mode is %s", mode))
		}
		if (!scp.Http2MonitoredFlows.IsNull() && !scp.Http2MonitoredFlows.IsUnknown()) && !http2SupportedModes[mode] {
			diags.AddError("Invalid scp_config.http2_monitored_flows", fmt.Sprintf("http2_monitored_flows is not supported when mode is %s", mode))
		}
		if (!scp.TcpMonitoredFlows.IsNull() && !scp.TcpMonitoredFlows.IsUnknown()) && !http2SupportedModes[mode] {
			diags.AddError("Invalid scp_config.tcp_monitored_flows", fmt.Sprintf("tcp_monitored_flows is not supported when mode is %s", mode))
		}
	}

	if !model.Hep3Config.IsNull() && !model.Hep3Config.IsUnknown() {
		if !hep3SupportedModes[mode] {
			diags.AddError("Invalid hep3_config for mode", fmt.Sprintf("hep3_config is supported only when mode is nokiaHEP3Inbound or nokiaHEP3IMS, got %s", mode))
			return
		}
		var hep3 Hep3ConfigModel
		diags.Append(model.Hep3Config.As(ctx, &hep3, basetypes.ObjectAsOptions{})...)
		if diags.HasError() {
			return
		}
		if strings.EqualFold(hep3.Mtls.ValueString(), "disable") && !hep3.MtlsKeyAlias.IsNull() && !hep3.MtlsKeyAlias.IsUnknown() && strings.TrimSpace(hep3.MtlsKeyAlias.ValueString()) != "" {
			diags.AddError("Invalid hep3_config.mtls_key_alias", "hep3_config.mtls_key_alias can be configured only when mtls is enable")
		}
	}
}

func setStringConfigValue(payload map[string]interface{}, key string, value types.String) {
	if !value.IsNull() && !value.IsUnknown() {
		payload[key] = value.ValueString()
	}
}

func setInt64ConfigValue(payload map[string]interface{}, key string, value types.Int64) {
	if !value.IsNull() && !value.IsUnknown() {
		payload[key] = value.ValueInt64()
	}
}

func setBoolConfigValue(payload map[string]interface{}, key string, value types.Bool) {
	if !value.IsNull() && !value.IsUnknown() {
		payload[key] = value.ValueBool()
	}
}

func toInt64Value(value interface{}) types.Int64 {
	switch typed := value.(type) {
	case float64:
		return types.Int64Value(int64(typed))
	case int64:
		return types.Int64Value(typed)
	case int:
		return types.Int64Value(int64(typed))
	case json.Number:
		parsed, err := typed.Int64()
		if err == nil {
			return types.Int64Value(parsed)
		}
	}
	return types.Int64Null()
}

func toBoolValue(value interface{}) types.Bool {
	switch typed := value.(type) {
	case bool:
		return types.BoolValue(typed)
	case string:
		if strings.EqualFold(typed, "true") {
			return types.BoolValue(true)
		}
		if strings.EqualFold(typed, "false") {
			return types.BoolValue(false)
		}
	}
	return types.BoolNull()
}

func stringInSlice(value string, values []string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
