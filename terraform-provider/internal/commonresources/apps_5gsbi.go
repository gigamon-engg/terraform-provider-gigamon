// Copyright (c) Gigamon, Inc.
// Licensed under the Mozilla Public License v. 2.0

// Implements the 5G-SBI Application resource for Gigamon Terraform Provider

package commonresources

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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

var appNameAliasRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &App5GSBI{}
var _ resource.ResourceWithConfigure = &App5GSBI{}
var _ resource.ResourceWithImportState = &App5GSBI{}

// New5GSBI creates a new resource instance for 5G-SBI application
func New5GSBI() resource.Resource {
    return &App5GSBI{}
}

// App5GSBI manages the 5G-SBI application resource
type App5GSBI struct {
    fmClient *fmclient.FmClient
}

// App5GSBIModel represents the Terraform configuration and state for 5G-SBI app.
// Fields map 1:1 to the FM API app_config wire format.
type App5GSBIModel struct {
    Id                  types.String `tfsdk:"id"`
    MonitoringSessionId types.String `tfsdk:"monitoring_session_id"`

    // FM API direct fields
    Alias                            types.String `tfsdk:"alias"`
    Name                             types.String `tfsdk:"name"`
    Type                             types.String `tfsdk:"type"`
    IpMappingAlias                   types.String `tfsdk:"ip_mapping_alias"`
    Http2SynthesizeToolMtuPacketSize types.Int64  `tfsdk:"http2_synthesize_tool_mtu_packet_size"`
    Http2SynthesizeIndexedHeaders    types.Bool   `tfsdk:"http2_synthesize_indexed_headers"`
    Http2SynthesizeCompressedHeaders types.Bool   `tfsdk:"http2_synthesize_compressed_headers"`
    TransactionLog                   types.Bool   `tfsdk:"transaction_log"`
    TransactionLogFileInterval       types.Int64  `tfsdk:"transaction_log_file_interval"`
    LogFolderSize                    types.Int64  `tfsdk:"log_folder_size"`
    StatsLog                         types.Bool   `tfsdk:"stats_log"`
    LogFolderLoc                     types.String `tfsdk:"log_folder_loc"`
    EricssonVTapConfig               types.Object `tfsdk:"ericsson_vtap_config"`
}

// EricssonVTapConfigModel represents the ericsson_vtap_config nested block
type EricssonVTapConfigModel struct {
    Mode                 types.String `tfsdk:"mode"`
    EevtapVersion        types.String `tfsdk:"eevtap_version"`
    NumTCPFlows          types.Int64  `tfsdk:"num_tcp_flows"`
    TcpFlowTimeout       types.Int64  `tfsdk:"tcp_flow_timeout"`
    NumStreamsPerFlow    types.Int64  `tfsdk:"num_streams_per_flow"`
    Http2RequestTimeout  types.Int64  `tfsdk:"http2_request_timeout"`
    Http2ResponseTimeout types.Int64  `tfsdk:"http2_response_timeout"`
    DestinationIP        types.String `tfsdk:"destination_ip"`
    FqdnMappingAlias     types.String `tfsdk:"fqdn_mapping_alias"`
}

var ericssonVTapConfigAttrTypes = map[string]attr.Type{
    "mode":                   types.StringType,
    "eevtap_version":         types.StringType,
    "num_tcp_flows":          types.Int64Type,
    "tcp_flow_timeout":       types.Int64Type,
    "num_streams_per_flow":   types.Int64Type,
    "http2_request_timeout":  types.Int64Type,
    "http2_response_timeout": types.Int64Type,
    "destination_ip":         types.StringType,
    "fqdn_mapping_alias":     types.StringType,
}

// FM5GSBI represents the wire format for 5G-SBI application in the FM API.
type FM5GSBI struct {
    Alias                            string                 `json:"alias,omitempty"`
    Name                             string                 `json:"name,omitempty"`
    Type                             string                 `json:"type,omitempty"`
    IpMappingAlias                   string                 `json:"ipMappingAlias,omitempty"`
    Http2SynthesizeToolMtuPacketSize int64                  `json:"http2SynthesizeToolMtuPacketSize,omitempty"`
    Http2SynthesizeIndexedHeaders    bool                   `json:"http2SynthesizeIndexedHeaders,omitempty"`
    Http2SynthesizeCompressedHeaders bool                   `json:"http2SynthesizeCompressedHeaders,omitempty"`
    TransactionLog                   bool                   `json:"transactionLog,omitempty"`
    TransactionLogFileInterval       int64                  `json:"transactionLogFileInterval,omitempty"`
    LogFolderSize                    int64                  `json:"logFolderSize,omitempty"`
    StatsLog                         bool                   `json:"statsLog,omitempty"`
    LogFolderLoc                     string                 `json:"logFolderLoc,omitempty"`
    EricssonVTapConfig               map[string]interface{} `json:"ericssonVTapConfig,omitempty"`
    AppConfig                        map[string]interface{} `json:"app_config,omitempty"`
}

// Metadata returns the resource type name
func (r *App5GSBI) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_app_5gsbi"
}

// Schema defines the resource schema
func (r *App5GSBI) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Manages a 5G-SBI (Service-based Interface) application instance on a monitoring session.",
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "The unique identifier for the 5G-SBI app resource",
                Computed:    true,
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
            "monitoring_session_id": schema.StringAttribute{
                Description: "The ID of the monitoring session to associate this app with",
                Required:    true,
                Validators: []validator.String{
                    stringvalidator.LengthAtLeast(1),
                },
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.RequiresReplace(),
                },
            },

            "alias": schema.StringAttribute{
                Description: "Alias for the 5G-SBI application template",
                Required:    true,
                Validators: []validator.String{
                    stringvalidator.LengthAtLeast(1),
                    stringvalidator.RegexMatches(appNameAliasRegex, "only alphanumeric, '-' and '_' are allowed"),
                },
            },
            "name": schema.StringAttribute{
                Description: "Display name for the 5G-SBI application",
                Required:    true,
                Validators: []validator.String{
                    stringvalidator.LengthAtLeast(1),
                    stringvalidator.RegexMatches(appNameAliasRegex, "only alphanumeric, '-' and '_' are allowed"),
                },
            },
            "type": schema.StringAttribute{
                Description: "Application type variant",
                Required:    true,
                Validators: []validator.String{
                    stringvalidator.OneOf("ericssonVTap"),
                },
            },
            "ip_mapping_alias": schema.StringAttribute{
                Description: "IP mapping alias for NF instance resolution",
                Required:    true,
                Validators: []validator.String{
                    stringvalidator.LengthAtLeast(1),
                },
            },
            "http2_synthesize_tool_mtu_packet_size": schema.Int64Attribute{
                Description: "MTU packet size for HTTP/2 synthesize tool",
                Required:    true,
                Validators: []validator.Int64{
                    int64validator.Any(
                        int64validator.OneOf(0),
                        int64validator.Between(1200, 1500),
                    ),
                },
            },
            "http2_synthesize_indexed_headers": schema.BoolAttribute{
                Description: "Enable indexed headers for HTTP/2 synthesize",
                Optional:    true,
                Computed:    true,
                Default:     booldefault.StaticBool(false),
            },
            "http2_synthesize_compressed_headers": schema.BoolAttribute{
                Description: "Enable compressed headers for HTTP/2 synthesize",
                Optional:    true,
                Computed:    true,
                Default:     booldefault.StaticBool(false),
            },
            "transaction_log": schema.BoolAttribute{
                Description: "Enable transaction logging",
                Optional:    true,
                Computed:    true,
                Default:     booldefault.StaticBool(false),
            },
            "transaction_log_file_interval": schema.Int64Attribute{
                Description: "Transaction log file rotation interval in seconds",
                Optional:    true,
                Computed:    true,
                Default:     int64default.StaticInt64(60),
                Validators: []validator.Int64{
                    int64validator.OneOf(5, 60),
                },
            },
            "log_folder_size": schema.Int64Attribute{
                Description: "Maximum log folder size in MB (0 = unlimited)",
                Optional:    true,
                Computed:    true,
                Default:     int64default.StaticInt64(0),
                Validators: []validator.Int64{
                    int64validator.Any(
                        int64validator.OneOf(0),
                        int64validator.Between(50, 40960),
                    ),
                },
            },
            "stats_log": schema.BoolAttribute{
                Description: "Enable statistics logging",
                Optional:    true,
                Computed:    true,
                Default:     booldefault.StaticBool(false),
            },
            "log_folder_loc": schema.StringAttribute{
                Description: "Log folder location path",
                Optional:    true,
            },
            "ericsson_vtap_config": schema.SingleNestedAttribute{
                Description: "Ericsson vTap specific configuration",
                Required:    true,
                Attributes: map[string]schema.Attribute{
                    "mode": schema.StringAttribute{
                        Description: "vTap mode",
                        Optional:    true,
                        Computed:    true,
                        Default:     stringdefault.StaticString("L7json"),
                        Validators: []validator.String{
                            stringvalidator.OneOf("L7json"),
                        },
                    },
                    "eevtap_version": schema.StringAttribute{
                        Description: "Ericsson vTap version",
                        Optional:    true,
                        Computed:    true,
                        Default:     stringdefault.StaticString("2"),
                        Validators: []validator.String{
                            stringvalidator.OneOf("1", "2"),
                        },
                    },
                    "num_tcp_flows": schema.Int64Attribute{
                        Description: "Number of TCP flows",
                        Required:    true,
                        Validators: []validator.Int64{
                            int64validator.Between(128, 16384),
                        },
                    },
                    "tcp_flow_timeout": schema.Int64Attribute{
                        Description: "TCP flow timeout in seconds",
                        Required:    true,
                        Validators: []validator.Int64{
                            int64validator.Between(0, 7200),
                        },
                    },
                    "num_streams_per_flow": schema.Int64Attribute{
                        Description: "Number of streams per flow",
                        Required:    true,
                        Validators: []validator.Int64{
                            int64validator.Between(1, 122880),
                        },
                    },
                    "http2_request_timeout": schema.Int64Attribute{
                        Description: "HTTP/2 request timeout in seconds",
                        Optional:    true,
                        Computed:    true,
                        Default:     int64default.StaticInt64(10),
                        Validators: []validator.Int64{
                            int64validator.Between(1, 300),
                        },
                    },
                    "http2_response_timeout": schema.Int64Attribute{
                        Description: "HTTP/2 response timeout in seconds",
                        Optional:    true,
                        Computed:    true,
                        Default:     int64default.StaticInt64(2),
                        Validators: []validator.Int64{
                            int64validator.Between(1, 300),
                        },
                    },
                    "destination_ip": schema.StringAttribute{
                        Description: "Destination IP address or label",
                        Optional:    true,
                        Computed:    true,
                        Default:     stringdefault.StaticString("SCP"),
                        Validators: []validator.String{
                            stringvalidator.OneOf("SCP", "destinationNF"),
                        },
                    },
                    "fqdn_mapping_alias": schema.StringAttribute{
                        Description: "FQDN mapping alias",
                        Optional:    true,
                    },
                },
            },
        },
    }
}

// Configure adds the provider configured client to the resource
func (r *App5GSBI) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }

    client, ok := req.ProviderData.(*fmclient.FmClient)
    if !ok {
        resp.Diagnostics.AddError(
            "Unexpected Resource Configure Type",
            fmt.Sprintf("Expected *fmclient.FmClient, got: %T", req.ProviderData),
        )
        return
    }

    r.fmClient = client
}

// Create creates a 5G-SBI application
func (r *App5GSBI) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var data App5GSBIModel

    resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }

    sessionID := data.MonitoringSessionId.ValueString()
    payload := buildFM5GSBIPayload(ctx, data)

    updateReq := commonutils.UpdateReq{
        Requests: []commonutils.UpdateObject{{
            EntityType:  "application",
            Operation:   "create",
            Application: payload,
        }},
    }

    id, err := commonutils.UpdateMonSess(ctx, &updateReq, sessionID, r.fmClient)
    if err != nil {
        tflog.Error(ctx, fmt.Sprintf("Failed to create 5G-SBI app: %v", err))
        resp.Diagnostics.AddError(
            "Error creating 5G-SBI app",
            fmt.Sprintf("Could not create app on session %s: %s", sessionID, err.Error()),
        )
        return
    }

    typedID, err := commonutils.MakeTypedID(commonutils.ModuleApp, commonutils.Type5GSBI, id)
    if err != nil {
        resp.Diagnostics.AddError("Error creating typed ID", err.Error())
        return
    }

    fmData := FM5GSBI{}
    err = GetMSAppData(ctx, sessionID, id, "5G-SBI", "", &fmData, r.fmClient)
    if err != nil {
        tflog.Warn(ctx, fmt.Sprintf("Failed to fetch created 5G-SBI app: %v", err))
    } else {
        data = mapFM5GSBIToState(ctx, fmData, sessionID, typedID)
    }

    data.Id = types.StringValue(typedID)
    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads a 5G-SBI application
func (r *App5GSBI) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var data App5GSBIModel

    resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }

    sessionID := data.MonitoringSessionId.ValueString()
    typedID := data.Id.ValueString()

    rawID, err := commonutils.UUIDFromTypedID(typedID)
    if err != nil {
        resp.Diagnostics.AddError(
            "Error parsing app ID",
            fmt.Sprintf("Could not parse ID %s: %s", typedID, err.Error()),
        )
        return
    }

    fmData := FM5GSBI{}
    err = GetMSAppData(ctx, sessionID, rawID, "5G-SBI", "", &fmData, r.fmClient)
    if err != nil {
        if strings.Contains(err.Error(), "not found") {
            resp.State.RemoveResource(ctx)
            return
        }
        resp.Diagnostics.AddError(
            "Error reading 5G-SBI app",
            fmt.Sprintf("Could not read app %s: %s", rawID, err.Error()),
        )
        return
    }

    data = mapFM5GSBIToState(ctx, fmData, sessionID, typedID)
    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates a 5G-SBI application
func (r *App5GSBI) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var planData App5GSBIModel

    resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
    if resp.Diagnostics.HasError() {
        return
    }

    sessionID := planData.MonitoringSessionId.ValueString()
    typedID := planData.Id.ValueString()

    rawID, err := commonutils.UUIDFromTypedID(typedID)
    if err != nil {
        resp.Diagnostics.AddError(
            "Error parsing app ID",
            fmt.Sprintf("Could not parse ID %s: %s", typedID, err.Error()),
        )
        return
    }

    payload := buildFM5GSBIPayload(ctx, planData)
    payload["id"] = rawID

    updateReq := commonutils.UpdateReq{
        Requests: []commonutils.UpdateObject{{
            EntityType:  "application",
            Operation:   "update",
            Application: payload,
        }},
    }

    _, err = commonutils.UpdateMonSess(ctx, &updateReq, sessionID, r.fmClient)
    if err != nil {
        tflog.Error(ctx, fmt.Sprintf("Failed to update 5G-SBI app: %v", err))
        resp.Diagnostics.AddError(
            "Error updating 5G-SBI app",
            fmt.Sprintf("Could not update app on session %s: %s", sessionID, err.Error()),
        )
        return
    }

    fmData := FM5GSBI{}
    err = GetMSAppData(ctx, sessionID, rawID, "5G-SBI", "", &fmData, r.fmClient)
    if err != nil {
        tflog.Warn(ctx, fmt.Sprintf("Failed to fetch updated 5G-SBI app: %v", err))
    } else {
        planData = mapFM5GSBIToState(ctx, fmData, sessionID, typedID)
    }

    resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
}

// Delete deletes a 5G-SBI application
func (r *App5GSBI) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var data App5GSBIModel

    resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }

    sessionID := data.MonitoringSessionId.ValueString()
    typedID := data.Id.ValueString()

    rawID, err := commonutils.UUIDFromTypedID(typedID)
    if err != nil {
        resp.Diagnostics.AddError(
            "Error parsing app ID",
            fmt.Sprintf("Could not parse ID %s: %s", typedID, err.Error()),
        )
        return
    }

    updateReq := commonutils.UpdateReq{
        Requests: []commonutils.UpdateObject{{
            EntityType: "application",
            Operation:  "delete",
            Application: map[string]interface{}{
                "id": rawID,
            },
        }},
    }

    _, err = commonutils.UpdateMonSess(ctx, &updateReq, sessionID, r.fmClient)
    if err != nil {
        tflog.Error(ctx, fmt.Sprintf("Failed to delete 5G-SBI app: %v", err))
        resp.Diagnostics.AddError(
            "Error deleting 5G-SBI app",
            fmt.Sprintf("Could not delete app on session %s: %s", sessionID, err.Error()),
        )
        return
    }
}

// ImportState imports a 5G-SBI application by session_id::app_id
func (r *App5GSBI) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    parts := strings.Split(req.ID, "::")
    if len(parts) != 2 {
        resp.Diagnostics.AddError(
            "Invalid import ID format",
            fmt.Sprintf("Expected session_id::app_id, got %s", req.ID),
        )
        return
    }

    sessionID := parts[0]
    rawID := parts[1]

    fmData := FM5GSBI{}
    err := GetMSAppData(ctx, sessionID, rawID, "5G-SBI", "", &fmData, r.fmClient)
    if err != nil {
        resp.Diagnostics.AddError(
            "Error reading 5G-SBI app for import",
            fmt.Sprintf("Could not read app %s: %s", rawID, err.Error()),
        )
        return
    }

    typedID, err := commonutils.MakeTypedID(commonutils.ModuleApp, commonutils.Type5GSBI, rawID)
    if err != nil {
        resp.Diagnostics.AddError("Error creating typed ID", err.Error())
        return
    }

    data := mapFM5GSBIToState(ctx, fmData, sessionID, typedID)
    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func setStringFromConfig(cfg map[string]interface{}, key string, v types.String) {
    if !v.IsNull() && !v.IsUnknown() {
        cfg[key] = v.ValueString()
    }
}

func setBoolFromConfig(cfg map[string]interface{}, key string, v types.Bool) {
    if !v.IsNull() && !v.IsUnknown() {
        cfg[key] = v.ValueBool()
    }
}

func setInt64FromConfig(cfg map[string]interface{}, key string, v types.Int64) {
    if !v.IsNull() && !v.IsUnknown() {
        cfg[key] = v.ValueInt64()
    }
}

// buildFM5GSBIPayload converts the Terraform model to FM API payload format
func buildFM5GSBIPayload(ctx context.Context, model App5GSBIModel) map[string]interface{} {
    payload := map[string]interface{}{}

    setStringFromConfig(payload, "alias", model.Alias)
    setStringFromConfig(payload, "name", model.Name)
    setStringFromConfig(payload, "type", model.Type)
    setStringFromConfig(payload, "ipMappingAlias", model.IpMappingAlias)
    setInt64FromConfig(payload, "http2SynthesizeToolMtuPacketSize", model.Http2SynthesizeToolMtuPacketSize)
    setBoolFromConfig(payload, "http2SynthesizeIndexedHeaders", model.Http2SynthesizeIndexedHeaders)
    setBoolFromConfig(payload, "http2SynthesizeCompressedHeaders", model.Http2SynthesizeCompressedHeaders)
    setBoolFromConfig(payload, "transactionLog", model.TransactionLog)
    setInt64FromConfig(payload, "transactionLogFileInterval", model.TransactionLogFileInterval)
    setInt64FromConfig(payload, "logFolderSize", model.LogFolderSize)
    setBoolFromConfig(payload, "statsLog", model.StatsLog)
    setStringFromConfig(payload, "logFolderLoc", model.LogFolderLoc)

    if !model.EricssonVTapConfig.IsNull() && !model.EricssonVTapConfig.IsUnknown() {
        var vtap EricssonVTapConfigModel
        _ = model.EricssonVTapConfig.As(ctx, &vtap, basetypes.ObjectAsOptions{})
        cfg := map[string]interface{}{}
        setStringFromConfig(cfg, "mode", vtap.Mode)
        setStringFromConfig(cfg, "eevtapVersion", vtap.EevtapVersion)
        setInt64FromConfig(cfg, "numTCPFlows", vtap.NumTCPFlows)
        setInt64FromConfig(cfg, "tcpFlowTimeout", vtap.TcpFlowTimeout)
        setInt64FromConfig(cfg, "numStreamsPerFlow", vtap.NumStreamsPerFlow)
        setInt64FromConfig(cfg, "http2RequestTimeout", vtap.Http2RequestTimeout)
        setInt64FromConfig(cfg, "http2ResponseTimeout", vtap.Http2ResponseTimeout)
        setStringFromConfig(cfg, "destinationIP", vtap.DestinationIP)
        setStringFromConfig(cfg, "fqdnMappingAlias", vtap.FqdnMappingAlias)
        payload["ericssonVTapConfig"] = cfg
    }

    return payload
}

func readString(cfg map[string]interface{}, key string) types.String {
    if v, ok := cfg[key].(string); ok {
        return types.StringValue(v)
    }
    return types.StringNull()
}

func readStringDefault(cfg map[string]interface{}, key string, def string) types.String {
    if v, ok := cfg[key].(string); ok {
        return types.StringValue(v)
    }
    return types.StringValue(def)
}

func readBool(cfg map[string]interface{}, key string) types.Bool {
    if v, ok := cfg[key].(bool); ok {
        return types.BoolValue(v)
    }
    return types.BoolNull()
}

func readBoolDefault(cfg map[string]interface{}, key string, def bool) types.Bool {
    if v, ok := cfg[key].(bool); ok {
        return types.BoolValue(v)
    }
    return types.BoolValue(def)
}

func readInt64(cfg map[string]interface{}, key string) types.Int64 {
    if v, ok := cfg[key].(float64); ok {
        return types.Int64Value(int64(v))
    }
    if v, ok := cfg[key].(int64); ok {
        return types.Int64Value(v)
    }
    if v, ok := cfg[key].(int); ok {
        return types.Int64Value(int64(v))
    }
    return types.Int64Null()
}

func readInt64Default(cfg map[string]interface{}, key string, def int64) types.Int64 {
    if v, ok := cfg[key].(float64); ok {
        return types.Int64Value(int64(v))
    }
    if v, ok := cfg[key].(int64); ok {
        return types.Int64Value(v)
    }
    if v, ok := cfg[key].(int); ok {
        return types.Int64Value(int64(v))
    }
    return types.Int64Value(def)
}

// mapFM5GSBIToState converts FM API response to Terraform model
func mapFM5GSBIToState(ctx context.Context, fmData FM5GSBI, sessionID string, typedID string) App5GSBIModel {
    model := App5GSBIModel{
        Id:                               types.StringValue(typedID),
        MonitoringSessionId:              types.StringValue(sessionID),
        Alias:                            types.StringNull(),
        Name:                             types.StringNull(),
        Type:                             types.StringNull(),
        IpMappingAlias:                   types.StringNull(),
        Http2SynthesizeToolMtuPacketSize: types.Int64Null(),
        Http2SynthesizeIndexedHeaders:    types.BoolNull(),
        Http2SynthesizeCompressedHeaders: types.BoolNull(),
        TransactionLog:                   types.BoolNull(),
        TransactionLogFileInterval:       types.Int64Null(),
        LogFolderSize:                    types.Int64Null(),
        StatsLog:                         types.BoolNull(),
        LogFolderLoc:                     types.StringNull(),
        EricssonVTapConfig:               types.ObjectNull(ericssonVTapConfigAttrTypes),
    }

    cfg := fmData.AppConfig
    if cfg == nil {
        cfg = map[string]interface{}{
            "alias":                           fmData.Alias,
            "name":                            fmData.Name,
            "type":                            fmData.Type,
            "ipMappingAlias":                  fmData.IpMappingAlias,
            "http2SynthesizeToolMtuPacketSize": fmData.Http2SynthesizeToolMtuPacketSize,
            "http2SynthesizeIndexedHeaders":   fmData.Http2SynthesizeIndexedHeaders,
            "http2SynthesizeCompressedHeaders": fmData.Http2SynthesizeCompressedHeaders,
            "transactionLog":                  fmData.TransactionLog,
            "transactionLogFileInterval":      fmData.TransactionLogFileInterval,
            "logFolderSize":                   fmData.LogFolderSize,
            "statsLog":                        fmData.StatsLog,
            "logFolderLoc":                    fmData.LogFolderLoc,
        }
        if fmData.EricssonVTapConfig != nil {
            cfg["ericssonVTapConfig"] = fmData.EricssonVTapConfig
        }
    }

    model.Alias = readString(cfg, "alias")
    model.Name = readString(cfg, "name")
    model.Type = readString(cfg, "type")
    model.IpMappingAlias = readString(cfg, "ipMappingAlias")
    model.Http2SynthesizeToolMtuPacketSize = readInt64(cfg, "http2SynthesizeToolMtuPacketSize")
    model.Http2SynthesizeIndexedHeaders = readBoolDefault(cfg, "http2SynthesizeIndexedHeaders", false)
    model.Http2SynthesizeCompressedHeaders = readBoolDefault(cfg, "http2SynthesizeCompressedHeaders", false)
    model.TransactionLog = readBoolDefault(cfg, "transactionLog", false)
    model.TransactionLogFileInterval = readInt64Default(cfg, "transactionLogFileInterval", 60)
    model.LogFolderSize = readInt64Default(cfg, "logFolderSize", 0)
    model.StatsLog = readBoolDefault(cfg, "statsLog", false)
    model.LogFolderLoc = readString(cfg, "logFolderLoc")

    if vtapCfg, ok := cfg["ericssonVTapConfig"].(map[string]interface{}); ok {
        vtap := EricssonVTapConfigModel{
            Mode:                 readStringDefault(vtapCfg, "mode", "L7json"),
            EevtapVersion:        readStringDefault(vtapCfg, "eevtapVersion", "2"),
            NumTCPFlows:          readInt64(vtapCfg, "numTCPFlows"),
            TcpFlowTimeout:       readInt64(vtapCfg, "tcpFlowTimeout"),
            NumStreamsPerFlow:    readInt64(vtapCfg, "numStreamsPerFlow"),
            Http2RequestTimeout:  readInt64Default(vtapCfg, "http2RequestTimeout", 10),
            Http2ResponseTimeout: readInt64Default(vtapCfg, "http2ResponseTimeout", 2),
            DestinationIP:        readStringDefault(vtapCfg, "destinationIP", "SCP"),
            FqdnMappingAlias:     readString(vtapCfg, "fqdnMappingAlias"),
        }
        if obj, diags := types.ObjectValueFrom(ctx, ericssonVTapConfigAttrTypes, vtap); !diags.HasError() {
            model.EricssonVTapConfig = obj
        }
    }

    return model
}
