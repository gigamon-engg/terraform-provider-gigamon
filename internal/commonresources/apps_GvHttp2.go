// Copyright (c) Gigamon, Inc.
// Licensed under the Mozilla Public License v.2.0
//
// Implements the GVHTTP2 Application resource for Gigamon Terraform Provider.
//
// Schema/validation rules implemented in this file follow the FM GVHTTP2
// application config contract:
//
//	alias, http2ListeningIpaddress, http2ListeningPort, tls,
//	locationCertificate, locationPrivateKey, maxConcurrentStream,
//	workerThread, mode, txTunnel[], csvEnable, pcapEnable,
//	logFolderLoc, logLevel.
package commonresources

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strings"

	"terraform-provider-gigamon/internal/commonutils"
	"terraform-provider-gigamon/internal/fmclient"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &AppGVHTTP2{}
var _ resource.ResourceWithConfigure = &AppGVHTTP2{}
var _ resource.ResourceWithImportState = &AppGVHTTP2{}
var _ resource.ResourceWithModifyPlan = &AppGVHTTP2{}

// NewGVHTTP2 creates a new resource instance for the GVHTTP2 application.
func NewGVHTTP2() resource.Resource {
	return &AppGVHTTP2{}
}

// AppGVHTTP2 manages the GVHTTP2 application resource.
type AppGVHTTP2 struct {
	fmClient *fmclient.FmClient
}

// AppGVHTTP2Model is the Terraform state/plan model.
type AppGVHTTP2Model struct {
	Id                      types.String `tfsdk:"id"`
	MonitoringSessionId     types.String `tfsdk:"monitoring_session_id"`
	Alias                   types.String `tfsdk:"alias"`
	Http2ListeningIpaddress types.String `tfsdk:"http2_listening_ipaddress"`
	Http2ListeningPort      types.Int64  `tfsdk:"http2_listening_port"`
	Tls                     types.String `tfsdk:"tls"`
	LocationCertificate     types.String `tfsdk:"location_certificate"`
	LocationPrivateKey      types.String `tfsdk:"location_private_key"`
	MaxConcurrentStream     types.Int64  `tfsdk:"max_concurrent_stream"`
	WorkerThread            types.Int64  `tfsdk:"worker_thread"`
	Mode                    types.String `tfsdk:"mode"`
	TxTunnel                types.List   `tfsdk:"tx_tunnel"`
	CsvEnable               types.Bool   `tfsdk:"csv_enable"`
	PcapEnable              types.Bool   `tfsdk:"pcap_enable"`
	LogFolderLoc            types.String `tfsdk:"log_folder_loc"`
	LogLevel                types.List   `tfsdk:"log_level"`
}

// GvHttp2TxTunnelModel is one tx_tunnel list element for GVHTTP2.
type GvHttp2TxTunnelModel struct {
	TxSrcIpaddress types.String `tfsdk:"tx_src_ipaddress"`
	TxSrcPort      types.Int64  `tfsdk:"tx_src_port"`
	TxDstIpaddress types.String `tfsdk:"tx_dst_ipaddress"`
	TxDstPort      types.Int64  `tfsdk:"tx_dst_port"`
	TxType         types.String `tfsdk:"tx_type"`
	TxThread       types.Int64  `tfsdk:"tx_thread"`
	TxVniId        types.Int64  `tfsdk:"tx_vni_id"`
}

// gvHttp2TxTunnelAttrTypes is the attr.Type map used to (de)serialize tx_tunnel list elements.
var gvHttp2TxTunnelAttrTypes = map[string]attr.Type{
	"tx_src_ipaddress": types.StringType,
	"tx_src_port":      types.Int64Type,
	"tx_dst_ipaddress": types.StringType,
	"tx_dst_port":      types.Int64Type,
	"tx_type":          types.StringType,
	"tx_thread":        types.Int64Type,
	"tx_vni_id":        types.Int64Type,
}

// FMGVHTTP2 is the JSON representation exchanged with FM for this app.
type FMGVHTTP2 struct {
	Name                    string                   `json:"name,omitempty"`
	Alias                   string                   `json:"alias,omitempty"`
	Id                      string                   `json:"id,omitempty"`
	Http2ListeningIpaddress string                   `json:"http2ListeningIpaddress,omitempty"`
	Http2ListeningPort      int64                    `json:"http2ListeningPort,omitempty"`
	Tls                     string                   `json:"tls,omitempty"`
	LocationCertificate     string                   `json:"locationCertificate,omitempty"`
	LocationPrivateKey      string                   `json:"locationPrivateKey,omitempty"`
	MaxConcurrentStream     int64                    `json:"maxConcurrentStream,omitempty"`
	WorkerThread            int64                    `json:"workerThread,omitempty"`
	Mode                    string                   `json:"mode,omitempty"`
	TxTunnel                []map[string]interface{} `json:"txTunnel,omitempty"`
	CsvEnable               bool                     `json:"csvEnable"`
	PcapEnable              *bool                    `json:"pcapEnable,omitempty"`
	LogFolderLoc            string                   `json:"logFolderLoc,omitempty"`
	LogLevel                []string                 `json:"logLevel,omitempty"`
}

const gvhttp2AppName = "gvhttp2"

var aliasRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// modesVxlan are the modes that require tx_tunnel.tx_type = vxlan and support tx_thread/tx_vni_id/pcap_enable.
var modesVxlan = map[string]bool{"casa": true, "nokia": true, "oracle": true}

// modesHep3 are the modes that require tx_tunnel.tx_type = tcp and disallow pcap_enable.
var modesHep3 = map[string]bool{"nokiaHEP3Stream": true, "nokiaHEP3Transaction": true}

// gvhttp2ModeImmutablePlanModifier prevents changing mode after creation.
type gvhttp2ModeImmutablePlanModifier struct{}

func (m gvhttp2ModeImmutablePlanModifier) Description(ctx context.Context) string {
	return "mode cannot be changed once the app is configured"
}

func (m gvhttp2ModeImmutablePlanModifier) MarkdownDescription(ctx context.Context) string {
	return "mode cannot be changed once the app is configured"
}

func (m gvhttp2ModeImmutablePlanModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() {
		return
	}

	if req.PlanValue.Equal(req.StateValue) {
		return
	}

	resp.Diagnostics.AddAttributeError(
		req.Path,
		"Mode Cannot Be Changed",
		fmt.Sprintf("mode cannot be changed once the app is configured. Previous value: %s, new value: %s", req.StateValue.ValueString(), req.PlanValue.ValueString()),
	)
}

func (r *AppGVHTTP2) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_gvhttp2"
}

func (r *AppGVHTTP2) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	defaultLogLevel := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("info")})

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a GVHTTP2 (HTTP/2 listener) application instance on a monitoring session.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The unique identifier for the GVHTTP2 app resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"monitoring_session_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the monitoring session.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"alias": schema.StringAttribute{
				MarkdownDescription: "Alias for the GVHTTP2 application. Only alphanumeric characters, \"-\", and \"_\" are allowed.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.RegexMatches(
						aliasRegex,
						`Invalid characters (only alphanumeric, "-" and "_" are allowed) in alias`,
					),
				},
			},
			"http2_listening_ipaddress": schema.StringAttribute{
				MarkdownDescription: "IP address the HTTP/2 listener binds to (IPv4 or IPv6).",
				Required:            true,
			},
			"http2_listening_port": schema.Int64Attribute{
				MarkdownDescription: "Port the HTTP/2 listener binds to (1-65535).",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.Between(1, 65535),
				},
			},
			"tls": schema.StringAttribute{
				MarkdownDescription: "Enable or disable TLS termination on the listener. Default: disable.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("disable"),
				Validators: []validator.String{
					stringvalidator.OneOf("enable", "disable"),
				},
			},
			"location_certificate": schema.StringAttribute{
				MarkdownDescription: "Path to the TLS certificate file. Required when tls = \"enable\"; must not be set when tls = \"disable\".",
				Optional:            true,
			},
			"location_private_key": schema.StringAttribute{
				MarkdownDescription: "Path to the TLS private key file. Required when tls = \"enable\"; must not be set when tls = \"disable\".",
				Optional:            true,
			},
			"max_concurrent_stream": schema.Int64Attribute{
				MarkdownDescription: "Maximum concurrent HTTP/2 streams (1-100). Default: 100.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(100),
				Validators: []validator.Int64{
					int64validator.Between(1, 100),
				},
			},
			"worker_thread": schema.Int64Attribute{
				MarkdownDescription: "Number of worker threads (1-16). Default: 4.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(4),
				Validators: []validator.Int64{
					int64validator.Between(1, 16),
				},
			},
			"mode": schema.StringAttribute{
				MarkdownDescription: "Correlation/parsing mode. Allowed values: casa, nokia, oracle, nokiaHEP3Stream, nokiaHEP3Transaction.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("casa", "nokia", "oracle", "nokiaHEP3Stream", "nokiaHEP3Transaction"),
				},
				PlanModifiers: []planmodifier.String{
					gvhttp2ModeImmutablePlanModifier{},
				},
			},
			"tx_tunnel": schema.ListNestedAttribute{
				MarkdownDescription: "One or more egress tunnels used to forward decoded traffic. At least one is required.",
				Required:            true,
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.SizeAtMost(1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"tx_src_ipaddress": schema.StringAttribute{
							MarkdownDescription: "Source IP address of the egress tunnel (IPv4 or IPv6).",
							Required:            true,
						},
						"tx_src_port": schema.Int64Attribute{
							MarkdownDescription: "Source port of the egress tunnel (1-65535).",
							Required:            true,
							Validators: []validator.Int64{
								int64validator.Between(1, 65535),
							},
						},
						"tx_dst_ipaddress": schema.StringAttribute{
							MarkdownDescription: "Destination IP address of the egress tunnel (IPv4 or IPv6).",
							Required:            true,
						},
						"tx_dst_port": schema.Int64Attribute{
							MarkdownDescription: "Destination port of the egress tunnel (1-65535).",
							Required:            true,
							Validators: []validator.Int64{
								int64validator.Between(1, 65535),
							},
						},
						"tx_type": schema.StringAttribute{
							MarkdownDescription: "Tunnel transport type. \"vxlan\" is valid only for mode casa/nokia/oracle. \"tcp\" is valid only for mode nokiaHEP3Stream/nokiaHEP3Transaction.",
							Required:            true,
							Validators: []validator.String{
								stringvalidator.OneOf("vxlan", "tcp"),
							},
						},
						"tx_thread": schema.Int64Attribute{
							MarkdownDescription: "Number of TX threads (1-16). Default: 4. Only applicable when mode is casa/nokia/oracle and tx_type = vxlan.",
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(4),
							Validators: []validator.Int64{
								int64validator.Between(1, 16),
							},
						},
						"tx_vni_id": schema.Int64Attribute{
							MarkdownDescription: "VXLAN VNI identifier (1-16777215). Default: 0. Only applicable when mode is casa/nokia/oracle and tx_type = vxlan.",
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(0),
							Validators: []validator.Int64{
								int64validator.Between(0, 16777215),
							},
						},
					},
				},
			},
			"csv_enable": schema.BoolAttribute{
				MarkdownDescription: "Enable CSV export of parsed data. Default: false.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"pcap_enable": schema.BoolAttribute{
				MarkdownDescription: "Enable packet capture. Default: false. Supported only for mode casa/nokia/oracle; configuring this while mode is nokiaHEP3Stream or nokiaHEP3Transaction raises an error.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"log_folder_loc": schema.StringAttribute{
				MarkdownDescription: "Folder location for application logs. Default: /var/log.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("/var/log"),
			},
			"log_level": schema.ListAttribute{
				MarkdownDescription: "Log verbosity flags. Allowed values: all, info, detail, fullparse. Default: [\"info\"].",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Default:             listdefault.StaticValue(defaultLogLevel),
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.OneOf("all", "info", "detail", "fullparse"),
					),
				},
			},
		},
	}
}

func (r *AppGVHTTP2) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*fmclient.FmClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *fmclient.FmClient, got: %T. Report the issue to Gigamon", req.ProviderData))
		return
	}
	r.fmClient = client
}

// ModifyPlan runs the cross-field semantic validation described in the GVHTTP2 spec:
//   - location_certificate / location_private_key <-> tls coupling
//   - tx_tunnel.tx_type <-> mode coupling
//   - tx_tunnel.tx_thread / tx_tunnel.tx_vni_id only valid for vxlan modes
//   - pcap_enable not supported for the HEP3 modes
func (r *AppGVHTTP2) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return // destroying
	}
	var data AppGVHTTP2Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateGVHTTP2Config(ctx, data, &resp.Diagnostics)
}

func validateGVHTTP2Config(ctx context.Context, data AppGVHTTP2Model, diags *diag.Diagnostics) {
	mode := data.Mode.ValueString()

	// --- IP literal checks ---
	if ip := data.Http2ListeningIpaddress.ValueString(); ip != "" && net.ParseIP(ip) == nil {
		diags.AddAttributeError(path.Root("http2_listening_ipaddress"),
			"Invalid http2_listening_ipaddress",
			fmt.Sprintf("http2_listening_ipaddress %q must be a valid IPv4 or IPv6 address", ip))
	}

	// --- TLS / certificate coupling ---
	tls := data.Tls.ValueString()
	certSet := !data.LocationCertificate.IsNull() && !data.LocationCertificate.IsUnknown() && data.LocationCertificate.ValueString() != ""
	keySet := !data.LocationPrivateKey.IsNull() && !data.LocationPrivateKey.IsUnknown() && data.LocationPrivateKey.ValueString() != ""
	if tls == "enable" {
		if !certSet {
			diags.AddAttributeError(path.Root("location_certificate"),
				"Missing location_certificate",
				"location_certificate is required when tls = \"enable\"")
		}
		if !keySet {
			diags.AddAttributeError(path.Root("location_private_key"),
				"Missing location_private_key",
				"location_private_key is required when tls = \"enable\"")
		}
	} else if tls == "disable" {
		if certSet {
			diags.AddAttributeError(path.Root("location_certificate"),
				"Invalid location_certificate",
				"location_certificate must not be configured when tls = \"disable\"")
		}
		if keySet {
			diags.AddAttributeError(path.Root("location_private_key"),
				"Invalid location_private_key",
				"location_private_key must not be configured when tls = \"disable\"")
		}
	}

	// --- pcap_enable / mode coupling ---
	if !data.PcapEnable.IsNull() && !data.PcapEnable.IsUnknown() && modesHep3[mode] {
		diags.AddAttributeError(path.Root("pcap_enable"),
			"Invalid pcap_enable",
			fmt.Sprintf("pcap_enable is not supported when mode is %q", mode))
	}

	// --- tx_tunnel entries ---
	if data.TxTunnel.IsNull() || data.TxTunnel.IsUnknown() {
		return
	}
	var tunnels []GvHttp2TxTunnelModel
	diags.Append(data.TxTunnel.ElementsAs(ctx, &tunnels, false)...)
	if diags.HasError() {
		return
	}
	for idx, tun := range tunnels {
		p := path.Root("tx_tunnel").AtListIndex(idx)

		if ip := tun.TxSrcIpaddress.ValueString(); ip != "" && net.ParseIP(ip) == nil {
			diags.AddAttributeError(p.AtName("tx_src_ipaddress"),
				"Invalid tx_src_ipaddress",
				fmt.Sprintf("tx_tunnel[%d].tx_src_ipaddress must be a valid IPv4 or IPv6 address", idx))
		}
		if ip := tun.TxDstIpaddress.ValueString(); ip != "" && net.ParseIP(ip) == nil {
			diags.AddAttributeError(p.AtName("tx_dst_ipaddress"),
				"Invalid tx_dst_ipaddress",
				fmt.Sprintf("tx_tunnel[%d].tx_dst_ipaddress must be a valid IPv4 or IPv6 address", idx))
		}

		txType := tun.TxType.ValueString()
		switch {
		case modesVxlan[mode]:
			if txType != "vxlan" {
				diags.AddAttributeError(p.AtName("tx_type"),
					"Invalid tx_tunnel.tx_type",
					fmt.Sprintf("tx_tunnel[%d].tx_type must be vxlan when mode is %q", idx, mode))
			}
		case modesHep3[mode]:
			if txType != "tcp" {
				diags.AddAttributeError(p.AtName("tx_type"),
					"Invalid tx_tunnel.tx_type",
					fmt.Sprintf("tx_tunnel[%d].tx_type must be tcp when mode is %q", idx, mode))
			}
			// tx_thread / tx_vni_id are not applicable in HEP3 modes.
			if !tun.TxThread.IsNull() && !tun.TxThread.IsUnknown() {
				diags.AddAttributeError(p.AtName("tx_thread"),
					"Invalid tx_tunnel.tx_thread",
					fmt.Sprintf("tx_tunnel[%d].tx_thread is not configurable when mode is %q", idx, mode))
			}
			if !tun.TxVniId.IsNull() && !tun.TxVniId.IsUnknown() && tun.TxVniId.ValueInt64() != 0 {
				diags.AddAttributeError(p.AtName("tx_vni_id"),
					"Invalid tx_tunnel.tx_vni_id",
					fmt.Sprintf("tx_tunnel[%d].tx_vni_id is not configurable when mode is %q", idx, mode))
			}
		}

		if txType == "tcp" && modesVxlan[mode] {
			diags.AddAttributeError(p.AtName("tx_type"),
				"Invalid tx_tunnel.tx_type",
				fmt.Sprintf("tx_tunnel[%d].tx_type cannot be tcp when mode is %q (casa/nokia/oracle require vxlan)", idx, mode))
		}
	}
}

func (r *AppGVHTTP2) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppGVHTTP2Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateGVHTTP2Config(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	sessionID := data.MonitoringSessionId.ValueString()
	payload, diags := buildFMGVHTTP2Payload(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{{
			EntityType:  "application",
			Operation:   "create",
			Application: payload,
		}},
	}
	id, err := commonutils.UpdateMonSess(ctx, &updateReq, sessionID, r.fmClient)
	if err != nil {
		resp.Diagnostics.AddError("Error creating GVHTTP2 app", err.Error())
		return
	}
	typedID, err := commonutils.MakeTypedID(commonutils.ModuleApp, commonutils.TypeGVHTTP2, id)
	if err != nil {
		resp.Diagnostics.AddError("Error creating typed ID", err.Error())
		return
	}

	fmData := FMGVHTTP2{}
	if err := GetMSAppData(ctx, sessionID, id, gvhttp2AppName, "", &fmData, r.fmClient); err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Failed to fetch created GVHTTP2 app: %v", err))
	} else {
		state, d := mapFMGVHTTP2ToState(ctx, fmData, sessionID, typedID)
		resp.Diagnostics.Append(d...)
		data = state
	}
	data.Id = types.StringValue(typedID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppGVHTTP2) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppGVHTTP2Model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sessionID := data.MonitoringSessionId.ValueString()
	typedID := data.Id.ValueString()
	rawID, err := commonutils.UUIDFromTypedID(typedID)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing app ID", err.Error())
		return
	}
	fmData := FMGVHTTP2{}
	if err := GetMSAppData(ctx, sessionID, rawID, gvhttp2AppName, "", &fmData, r.fmClient); err != nil {
		if strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading GVHTTP2 app", err.Error())
		return
	}
	state, diags := mapFMGVHTTP2ToState(ctx, fmData, sessionID, typedID)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AppGVHTTP2) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData AppGVHTTP2Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}
	validateGVHTTP2Config(ctx, planData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	sessionID := planData.MonitoringSessionId.ValueString()
	typedID := planData.Id.ValueString()
	rawID, err := commonutils.UUIDFromTypedID(typedID)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing app ID", err.Error())
		return
	}
	payload, diags := buildFMGVHTTP2Payload(ctx, planData)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	payload["id"] = rawID

	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{{
			EntityType:  "application",
			Operation:   "update",
			Application: payload,
		}},
	}
	if _, err := commonutils.UpdateMonSess(ctx, &updateReq, sessionID, r.fmClient); err != nil {
		resp.Diagnostics.AddError("Error updating GVHTTP2 app", err.Error())
		return
	}

	fmData := FMGVHTTP2{}
	if err := GetMSAppData(ctx, sessionID, rawID, gvhttp2AppName, "", &fmData, r.fmClient); err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Failed to fetch updated GVHTTP2 app: %v", err))
	} else {
		state, d := mapFMGVHTTP2ToState(ctx, fmData, sessionID, typedID)
		resp.Diagnostics.Append(d...)
		planData = state
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
}

func (r *AppGVHTTP2) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppGVHTTP2Model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sessionID := data.MonitoringSessionId.ValueString()
	typedID := data.Id.ValueString()
	rawID, err := commonutils.UUIDFromTypedID(typedID)
	if err != nil {
		resp.Diagnostics.AddError("Error parsing app ID", err.Error())
		return
	}
	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{{
			EntityType: "application",
			Operation:  "delete",
			Application: map[string]interface{}{
				"id":       rawID,
				"app_type": gvhttp2AppName,
			},
		}},
	}
	if _, err := commonutils.UpdateMonSess(ctx, &updateReq, sessionID, r.fmClient); err != nil {
		resp.Diagnostics.AddError("Error deleting GVHTTP2 app", err.Error())
		return
	}
}

func (r *AppGVHTTP2) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "::")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID format", fmt.Sprintf("Expected monitoring_session_id::raw_uuid, got %s", req.ID))
		return
	}
	sessionID := parts[0]
	rawID := parts[1]
	fmData := FMGVHTTP2{}
	if err := GetMSAppData(ctx, sessionID, rawID, gvhttp2AppName, "", &fmData, r.fmClient); err != nil {
		resp.Diagnostics.AddError("Error reading GVHTTP2 app for import", err.Error())
		return
	}
	typedID, err := commonutils.MakeTypedID(commonutils.ModuleApp, commonutils.TypeGVHTTP2, rawID)
	if err != nil {
		resp.Diagnostics.AddError("Error creating typed ID", err.Error())
		return
	}
	data, diags := mapFMGVHTTP2ToState(ctx, fmData, sessionID, typedID)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// buildFMGVHTTP2Payload converts the TF plan model into the FM JSON app payload.
func buildFMGVHTTP2Payload(ctx context.Context, model AppGVHTTP2Model) (map[string]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	appConfig := map[string]interface{}{
		"name":                    gvhttp2AppName,
		"alias":                   model.Alias.ValueString(),
		"http2ListeningIpaddress": model.Http2ListeningIpaddress.ValueString(),
		"http2ListeningPort":      model.Http2ListeningPort.ValueInt64(),
		"tls":                     model.Tls.ValueString(),
		"maxConcurrentStream":     model.MaxConcurrentStream.ValueInt64(),
		"workerThread":            model.WorkerThread.ValueInt64(),
		"mode":                    model.Mode.ValueString(),
		"csvEnable":               model.CsvEnable.ValueBool(),
		"logFolderLoc":            model.LogFolderLoc.ValueString(),
	}

	if model.Tls.ValueString() == "enable" {
		appConfig["locationCertificate"] = model.LocationCertificate.ValueString()
		appConfig["locationPrivateKey"] = model.LocationPrivateKey.ValueString()
	}

	if !modesHep3[model.Mode.ValueString()] {
		appConfig["pcapEnable"] = model.PcapEnable.ValueBool()
	}

	if !model.LogLevel.IsNull() && !model.LogLevel.IsUnknown() {
		var levels []string
		diags.Append(model.LogLevel.ElementsAs(ctx, &levels, false)...)
		appConfig["logLevel"] = levels
	}

	if !model.TxTunnel.IsNull() && !model.TxTunnel.IsUnknown() {
		var tunnels []GvHttp2TxTunnelModel
		diags.Append(model.TxTunnel.ElementsAs(ctx, &tunnels, false)...)
		txList := make([]map[string]interface{}, 0, len(tunnels))
		for _, t := range tunnels {
			entry := map[string]interface{}{
				"txSrcIpaddress": t.TxSrcIpaddress.ValueString(),
				"txSrcPort":      t.TxSrcPort.ValueInt64(),
				"txDstIpaddress": t.TxDstIpaddress.ValueString(),
				"txDstPort":      t.TxDstPort.ValueInt64(),
				"txType":         t.TxType.ValueString(),
			}
			if t.TxType.ValueString() == "vxlan" {
				entry["txThread"] = t.TxThread.ValueInt64()
				entry["txVniId"] = t.TxVniId.ValueInt64()
			}
			txList = append(txList, entry)
		}
		appConfig["txTunnel"] = txList
	}

	return appConfig, diags
}

// mapFMGVHTTP2ToState converts the FM JSON app payload back into the TF state model.
func mapFMGVHTTP2ToState(ctx context.Context, fmData FMGVHTTP2, sessionID string, typedID string) (AppGVHTTP2Model, diag.Diagnostics) {
	var diags diag.Diagnostics

	model := AppGVHTTP2Model{
		Id:                      types.StringValue(typedID),
		MonitoringSessionId:     types.StringValue(sessionID),
		Alias:                   types.StringValue(fmData.Alias),
		Http2ListeningIpaddress: types.StringValue(fmData.Http2ListeningIpaddress),
		Http2ListeningPort:      types.Int64Value(fmData.Http2ListeningPort),
		Tls:                     types.StringValue(fmData.Tls),
		MaxConcurrentStream:     types.Int64Value(fmData.MaxConcurrentStream),
		WorkerThread:            types.Int64Value(fmData.WorkerThread),
		Mode:                    types.StringValue(fmData.Mode),
		CsvEnable:               types.BoolValue(fmData.CsvEnable),
		LogFolderLoc:            types.StringValue(fmData.LogFolderLoc),
	}

	if fmData.Tls == "enable" {
		model.LocationCertificate = types.StringValue(fmData.LocationCertificate)
		model.LocationPrivateKey = types.StringValue(fmData.LocationPrivateKey)
	} else {
		model.LocationCertificate = types.StringNull()
		model.LocationPrivateKey = types.StringNull()
	}

	if fmData.PcapEnable != nil {
		model.PcapEnable = types.BoolValue(*fmData.PcapEnable)
	} else {
		model.PcapEnable = types.BoolValue(false)
	}

	logLevels := fmData.LogLevel
	if len(logLevels) == 0 {
		logLevels = []string{"info"}
	}
	levelList, d := types.ListValueFrom(ctx, types.StringType, logLevels)
	diags.Append(d...)
	model.LogLevel = levelList

	txModels := make([]GvHttp2TxTunnelModel, 0, len(fmData.TxTunnel))
	for _, raw := range fmData.TxTunnel {
		t := GvHttp2TxTunnelModel{
			TxThread: types.Int64Value(4),
			TxVniId:  types.Int64Value(0),
		}
		if v, ok := raw["txSrcIpaddress"].(string); ok {
			t.TxSrcIpaddress = types.StringValue(v)
		}
		if v, ok := raw["txSrcPort"].(float64); ok {
			t.TxSrcPort = types.Int64Value(int64(v))
		}
		if v, ok := raw["txDstIpaddress"].(string); ok {
			t.TxDstIpaddress = types.StringValue(v)
		}
		if v, ok := raw["txDstPort"].(float64); ok {
			t.TxDstPort = types.Int64Value(int64(v))
		}
		if v, ok := raw["txType"].(string); ok {
			t.TxType = types.StringValue(v)
		}
		if v, ok := raw["txThread"].(float64); ok {
			t.TxThread = types.Int64Value(int64(v))
		}
		if v, ok := raw["txVniId"].(float64); ok {
			t.TxVniId = types.Int64Value(int64(v))
		}
		txModels = append(txModels, t)
	}

	txListVal, d2 := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: gvHttp2TxTunnelAttrTypes}, txModels)
	diags.Append(d2...)
	model.TxTunnel = txListVal

	return model, diags
}
