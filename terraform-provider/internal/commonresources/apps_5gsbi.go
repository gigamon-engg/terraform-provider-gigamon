// Copyright (c) Gigamon, Inc.
// Licensed under the Mozilla Public License v. 2.0

// Implements the 5G-SBI Application resource for Gigamon Terraform Provider

package commonresources

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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

// App5GSBIModel represents the Terraform configuration and state for 5G-SBI app
type App5GSBIModel struct {
	Id                  types.String `tfsdk:"id"`
	MonitoringSessionId types.String `tfsdk:"monitoring_session_id"`

	// Legacy abstraction fields (kept for backward compatibility)
	SBIMode          types.String `tfsdk:"sbi_mode"`
	ProtocolHandlers types.List   `tfsdk:"protocol_handlers"`
	Authentication   types.Object `tfsdk:"authentication"`

	// FM API direct fields
	Alias                            types.String `tfsdk:"alias"`
	Name                             types.String `tfsdk:"name"`
	Type                             types.String `tfsdk:"type"`
	IpMappingAlias                   types.String `tfsdk:"ipMappingAlias"`
	Http2SynthesizeToolMtuPacketSize types.Int64  `tfsdk:"http2SynthesizeToolMtuPacketSize"`
	Http2SynthesizeIndexedHeaders    types.Bool   `tfsdk:"http2SynthesizeIndexedHeaders"`
	Http2SynthesizeCompressedHeaders types.Bool   `tfsdk:"http2SynthesizeCompressedHeaders"`
	TransactionLog                   types.Bool   `tfsdk:"transactionLog"`
	TransactionLogFileInterval       types.Int64  `tfsdk:"transactionLogFileInterval"`
	LogFolderSize                    types.Int64  `tfsdk:"logFolderSize"`
	StatsLog                         types.Bool   `tfsdk:"statsLog"`
	LogFolderLoc                     types.String `tfsdk:"logFolderLoc"`
	EricssonVTapConfig               types.Object `tfsdk:"ericssonVTapConfig"`
}

// AuthenticationModel represents the authentication nested block
type AuthenticationModel struct {
	Enabled  types.Bool   `tfsdk:"enabled"`
	CertPath types.String `tfsdk:"cert_path"`
	KeyPath  types.String `tfsdk:"key_path"`
}

// EricssonVTapConfigModel represents the ericssonVTapConfig nested block
type EricssonVTapConfigModel struct {
	Mode                 types.String `tfsdk:"mode"`
	EevtapVersion        types.String `tfsdk:"eevtapVersion"`
	NumTCPFlows          types.Int64  `tfsdk:"numTCPFlows"`
	TcpFlowTimeout       types.Int64  `tfsdk:"tcpFlowTimeout"`
	NumStreamsPerFlow    types.Int64  `tfsdk:"numStreamsPerFlow"`
	Http2RequestTimeout  types.Int64  `tfsdk:"http2RequestTimeout"`
	Http2ResponseTimeout types.Int64  `tfsdk:"http2ResponseTimeout"`
	DestinationIP        types.String `tfsdk:"destinationIP"`
	FqdnMappingAlias     types.String `tfsdk:"fqdnMappingAlias"`
}

var ericssonVTapConfigAttrTypes = map[string]attr.Type{
	"mode":                 types.StringType,
	"eevtapVersion":        types.StringType,
	"numTCPFlows":          types.Int64Type,
	"tcpFlowTimeout":       types.Int64Type,
	"numStreamsPerFlow":    types.Int64Type,
	"http2RequestTimeout":  types.Int64Type,
	"http2ResponseTimeout": types.Int64Type,
	"destinationIP":        types.StringType,
	"fqdnMappingAlias":     types.StringType,
}

var authenticationAttrTypes = map[string]attr.Type{
	"enabled":   types.BoolType,
	"cert_path": types.StringType,
	"key_path":  types.StringType,
}

// FM5GSBI represents the wire format for 5G-SBI application in the FM API
type FM5GSBI struct {
	AppType   string                 `json:"app_type"`
	AppConfig map[string]interface{} `json:"app_config"`
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

			// Legacy abstraction fields
			"sbi_mode": schema.StringAttribute{
				Description: "SBI mode: nrf, udm, or amf",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("nrf"),
				Validators: []validator.String{
					stringvalidator.OneOf("nrf", "udm", "amf"),
				},
			},
			"protocol_handlers": schema.ListAttribute{
				Description: "Protocol handlers: http, https, grpc",
				ElementType: types.StringType,
				Optional:    true,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(
						stringvalidator.OneOf("http", "https", "grpc"),
					),
				},
			},
			"authentication": schema.SingleNestedAttribute{
				Description: "Authentication configuration",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Description: "Enable authentication",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"cert_path": schema.StringAttribute{
						Description: "Certificate file path",
						Optional:    true,
					},
					"key_path": schema.StringAttribute{
						Description: "Private key file path",
						Optional:    true,
					},
				},
			},

			// FM API direct fields
			"alias": schema.StringAttribute{
				Description: "Alias for the 5G-SBI application template",
				Optional:    true,
			},
			"name": schema.StringAttribute{
				Description: "Display name for the 5G-SBI application",
				Optional:    true,
			},
			"type": schema.StringAttribute{
				Description: "Application type variant (e.g. ericssonVTap)",
				Optional:    true,
			},
			"ipMappingAlias": schema.StringAttribute{
				Description: "IP mapping alias for NF instance resolution",
				Optional:    true,
			},
			"http2SynthesizeToolMtuPacketSize": schema.Int64Attribute{
				Description: "MTU packet size for HTTP/2 synthesize tool (0 = default)",
				Optional:    true,
			},
			"http2SynthesizeIndexedHeaders": schema.BoolAttribute{
				Description: "Enable indexed headers for HTTP/2 synthesize",
				Optional:    true,
			},
			"http2SynthesizeCompressedHeaders": schema.BoolAttribute{
				Description: "Enable compressed headers for HTTP/2 synthesize",
				Optional:    true,
			},
			"transactionLog": schema.BoolAttribute{
				Description: "Enable transaction logging",
				Optional:    true,
			},
			"transactionLogFileInterval": schema.Int64Attribute{
				Description: "Transaction log file rotation interval in seconds",
				Optional:    true,
			},
			"logFolderSize": schema.Int64Attribute{
				Description: "Maximum log folder size in MB (0 = unlimited)",
				Optional:    true,
			},
			"statsLog": schema.BoolAttribute{
				Description: "Enable statistics logging",
				Optional:    true,
			},
			"logFolderLoc": schema.StringAttribute{
				Description: "Log folder location path",
				Optional:    true,
			},
			"ericssonVTapConfig": schema.SingleNestedAttribute{
				Description: "Ericsson vTap specific configuration",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"mode": schema.StringAttribute{Description: "vTap mode (e.g. L7json)", Optional: true},
					"eevtapVersion": schema.StringAttribute{Description: "Ericsson vTap version", Optional: true},
					"numTCPFlows": schema.Int64Attribute{Description: "Number of TCP flows", Optional: true},
					"tcpFlowTimeout": schema.Int64Attribute{Description: "TCP flow timeout in seconds", Optional: true},
					"numStreamsPerFlow": schema.Int64Attribute{Description: "Number of streams per flow", Optional: true},
					"http2RequestTimeout": schema.Int64Attribute{Description: "HTTP/2 request timeout in seconds", Optional: true},
					"http2ResponseTimeout": schema.Int64Attribute{Description: "HTTP/2 response timeout in seconds", Optional: true},
					"destinationIP": schema.StringAttribute{Description: "Destination IP address or label (e.g. SCP)", Optional: true},
					"fqdnMappingAlias": schema.StringAttribute{Description: "FQDN mapping alias", Optional: true},
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

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
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
				"id":       rawID,
				"app_type": "5G-SBI",
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
	appConfig := map[string]interface{}{}

	// Legacy abstraction fields
	setStringFromConfig(appConfig, "sbi_mode", model.SBIMode)
	if !model.ProtocolHandlers.IsNull() && !model.ProtocolHandlers.IsUnknown() {
		var handlers []string
		_ = model.ProtocolHandlers.ElementsAs(ctx, &handlers, false)
		appConfig["protocol_handlers"] = handlers
	}
	if !model.Authentication.IsNull() && !model.Authentication.IsUnknown() {
		var authModel AuthenticationModel
		_ = model.Authentication.As(ctx, &authModel, basetypes.ObjectAsOptions{})
		authCfg := map[string]interface{}{}
		setBoolFromConfig(authCfg, "enabled", authModel.Enabled)
		setStringFromConfig(authCfg, "cert_path", authModel.CertPath)
		setStringFromConfig(authCfg, "key_path", authModel.KeyPath)
		appConfig["authentication"] = authCfg
	}

	// FM direct fields
	setStringFromConfig(appConfig, "alias", model.Alias)
	setStringFromConfig(appConfig, "name", model.Name)
	setStringFromConfig(appConfig, "type", model.Type)
	setStringFromConfig(appConfig, "ipMappingAlias", model.IpMappingAlias)
	setInt64FromConfig(appConfig, "http2SynthesizeToolMtuPacketSize", model.Http2SynthesizeToolMtuPacketSize)
	setBoolFromConfig(appConfig, "http2SynthesizeIndexedHeaders", model.Http2SynthesizeIndexedHeaders)
	setBoolFromConfig(appConfig, "http2SynthesizeCompressedHeaders", model.Http2SynthesizeCompressedHeaders)
	setBoolFromConfig(appConfig, "transactionLog", model.TransactionLog)
	setInt64FromConfig(appConfig, "transactionLogFileInterval", model.TransactionLogFileInterval)
	setInt64FromConfig(appConfig, "logFolderSize", model.LogFolderSize)
	setBoolFromConfig(appConfig, "statsLog", model.StatsLog)
	setStringFromConfig(appConfig, "logFolderLoc", model.LogFolderLoc)

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
		appConfig["ericssonVTapConfig"] = cfg
	}

	return map[string]interface{}{
		"app_type":   "5G-SBI",
		"app_config": appConfig,
	}
}

func readString(cfg map[string]interface{}, key string) types.String {
	if v, ok := cfg[key].(string); ok {
		return types.StringValue(v)
	}
	return types.StringNull()
}

func readBool(cfg map[string]interface{}, key string) types.Bool {
	if v, ok := cfg[key].(bool); ok {
		return types.BoolValue(v)
	}
	return types.BoolNull()
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

// mapFM5GSBIToState converts FM API response to Terraform model
func mapFM5GSBIToState(ctx context.Context, fmData FM5GSBI, sessionID string, typedID string) App5GSBIModel {
	model := App5GSBIModel{
		Id:                  types.StringValue(typedID),
		MonitoringSessionId: types.StringValue(sessionID),
		SBIMode:             types.StringValue("nrf"),
		ProtocolHandlers:    types.ListNull(types.StringType),
		Authentication:      types.ObjectNull(authenticationAttrTypes),
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

	if fmData.AppConfig == nil {
		return model
	}

	cfg := fmData.AppConfig

	if sbiMode, ok := cfg["sbi_mode"].(string); ok {
		model.SBIMode = types.StringValue(sbiMode)
	}
	if handlers, ok := cfg["protocol_handlers"].([]interface{}); ok {
		var out []string
		for _, h := range handlers {
			if hs, ok := h.(string); ok {
				out = append(out, hs)
			}
		}
		if list, diags := types.ListValueFrom(ctx, types.StringType, out); !diags.HasError() {
			model.ProtocolHandlers = list
		}
	}
	if authCfg, ok := cfg["authentication"].(map[string]interface{}); ok {
		authModel := AuthenticationModel{
			Enabled:  readBool(authCfg, "enabled"),
			CertPath: readString(authCfg, "cert_path"),
			KeyPath:  readString(authCfg, "key_path"),
		}
		if obj, diags := types.ObjectValueFrom(ctx, authenticationAttrTypes, authModel); !diags.HasError() {
			model.Authentication = obj
		}
	}

	model.Alias = readString(cfg, "alias")
	model.Name = readString(cfg, "name")
	model.Type = readString(cfg, "type")
	model.IpMappingAlias = readString(cfg, "ipMappingAlias")
	model.Http2SynthesizeToolMtuPacketSize = readInt64(cfg, "http2SynthesizeToolMtuPacketSize")
	model.Http2SynthesizeIndexedHeaders = readBool(cfg, "http2SynthesizeIndexedHeaders")
	model.Http2SynthesizeCompressedHeaders = readBool(cfg, "http2SynthesizeCompressedHeaders")
	model.TransactionLog = readBool(cfg, "transactionLog")
	model.TransactionLogFileInterval = readInt64(cfg, "transactionLogFileInterval")
	model.LogFolderSize = readInt64(cfg, "logFolderSize")
	model.StatsLog = readBool(cfg, "statsLog")
	model.LogFolderLoc = readString(cfg, "logFolderLoc")

	if vtapCfg, ok := cfg["ericssonVTapConfig"].(map[string]interface{}); ok {
		vtap := EricssonVTapConfigModel{
			Mode:                 readString(vtapCfg, "mode"),
			EevtapVersion:        readString(vtapCfg, "eevtapVersion"),
			NumTCPFlows:          readInt64(vtapCfg, "numTCPFlows"),
			TcpFlowTimeout:       readInt64(vtapCfg, "tcpFlowTimeout"),
			NumStreamsPerFlow:    readInt64(vtapCfg, "numStreamsPerFlow"),
			Http2RequestTimeout:  readInt64(vtapCfg, "http2RequestTimeout"),
			Http2ResponseTimeout: readInt64(vtapCfg, "http2ResponseTimeout"),
			DestinationIP:        readString(vtapCfg, "destinationIP"),
			FqdnMappingAlias:     readString(vtapCfg, "fqdnMappingAlias"),
		}
		if obj, diags := types.ObjectValueFrom(ctx, ericssonVTapConfigAttrTypes, vtap); !diags.HasError() {
			model.EricssonVTapConfig = obj
		}
	}

	return model
}
