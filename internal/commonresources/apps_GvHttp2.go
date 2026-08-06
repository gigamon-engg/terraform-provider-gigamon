// Copyright (c) Gigamon, Inc.
// Licensed under the Mozilla Public License v. 2.0

// Implements the GVHTTP2 Application resource for Gigamon Terraform Provider

package commonresources

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-gigamon/internal/commonutils"
	"terraform-provider-gigamon/internal/fmclient"
)

var _ resource.Resource = &AppGVHTTP2{}
var _ resource.ResourceWithConfigure = &AppGVHTTP2{}
var _ resource.ResourceWithImportState = &AppGVHTTP2{}

func NewGVHTTP2() resource.Resource {
	return &AppGVHTTP2{}
}

type AppGVHTTP2 struct {
	fmClient *fmclient.FmClient
}

type AppGVHTTP2Model struct {
	Id                  types.String `tfsdk:"id"`
	MonitoringSessionId types.String `tfsdk:"monitoring_session_id"`
	HTTP2Mode           types.String `tfsdk:"http2_mode"`
	ProtocolConfig      types.Object `tfsdk:"protocol_config"`
	Compression         types.Object `tfsdk:"compression"`
	FlowControl         types.Object `tfsdk:"flow_control"`
}

type ProtocolConfigModel struct {
	StreamMultiplexing types.Bool `tfsdk:"stream_multiplexing"`
	ServerPush         types.Bool `tfsdk:"server_push"`
}

type CompressionModel struct {
	Enabled types.Bool  `tfsdk:"enabled"`
	Level   types.Int32 `tfsdk:"level"`
}

type FlowControlModel struct {
	WindowSize        types.Int32 `tfsdk:"window_size"`
	InitialWindowSize types.Int32 `tfsdk:"initial_window_size"`
}

type FMGVHTTP2 struct {
	AppType   string                 `json:"app_type"`
	AppConfig map[string]interface{} `json:"app_config"`
}

func (r *AppGVHTTP2) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_gvhttp2"
}

func (r *AppGVHTTP2) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a GVHTTP2 application instance on a monitoring session.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the GVHTTP2 app resource",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"monitoring_session_id": schema.StringAttribute{
				Description: "The ID of the monitoring session",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"http2_mode": schema.StringAttribute{
				Description: "HTTP/2 mode: enabled or disabled",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("enabled", "disabled"),
				},
			},
			"protocol_config": schema.SingleNestedAttribute{
				Description: "Protocol configuration",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"stream_multiplexing": schema.BoolAttribute{
						Description: "Enable stream multiplexing",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(true),
					},
					"server_push": schema.BoolAttribute{
						Description: "Enable server push",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
				},
			},
			"compression": schema.SingleNestedAttribute{
				Description: "Compression configuration",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Description: "Enable compression",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(true),
					},
					"level": schema.Int32Attribute{
						Description: "Compression level: 1-9",
						Optional:    true,
						Computed:    true,
						Default:     int32default.StaticInt32(6),
						Validators: []validator.Int32{
							int32validator.Between(1, 9),
						},
					},
				},
			},
			"flow_control": schema.SingleNestedAttribute{
				Description: "Flow control configuration",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"window_size": schema.Int32Attribute{
						Description: "Window size: 16384-2147483647",
						Optional:    true,
						Computed:    true,
						Default:     int32default.StaticInt32(65535),
						Validators: []validator.Int32{
							int32validator.Between(16384, 2147483647),
						},
					},
					"initial_window_size": schema.Int32Attribute{
						Description: "Initial window size: 16384-2147483647",
						Optional:    true,
						Computed:    true,
						Default:     int32default.StaticInt32(65535),
						Validators: []validator.Int32{
							int32validator.Between(16384, 2147483647),
						},
					},
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
			fmt.Sprintf("Expected *fmclient.FmClient, got: %T", req.ProviderData))
		return
	}
	r.fmClient = client
}

func (r *AppGVHTTP2) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppGVHTTP2Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sessionID := data.MonitoringSessionId.ValueString()
	payload := buildFMGVHTTP2Payload(ctx, data)
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
	err = GetMSAppData(ctx, sessionID, id, "GVHTTP2", "", &fmData, r.fmClient)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Failed to fetch created GVHTTP2 app: %v", err))
	} else {
		data = mapFMGVHTTP2ToState(ctx, fmData, sessionID, typedID)
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
	err = GetMSAppData(ctx, sessionID, rawID, "GVHTTP2", "", &fmData, r.fmClient)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading GVHTTP2 app", err.Error())
		return
	}
	data = mapFMGVHTTP2ToState(ctx, fmData, sessionID, typedID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppGVHTTP2) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData AppGVHTTP2Model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
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
	payload := buildFMGVHTTP2Payload(ctx, planData)
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
		resp.Diagnostics.AddError("Error updating GVHTTP2 app", err.Error())
		return
	}
	fmData := FMGVHTTP2{}
	err = GetMSAppData(ctx, sessionID, rawID, "GVHTTP2", "", &fmData, r.fmClient)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Failed to fetch updated GVHTTP2 app: %v", err))
	} else {
		planData = mapFMGVHTTP2ToState(ctx, fmData, sessionID, typedID)
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
				"app_type": "GVHTTP2",
			},
		}},
	}
	_, err = commonutils.UpdateMonSess(ctx, &updateReq, sessionID, r.fmClient)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting GVHTTP2 app", err.Error())
		return
	}
}

func (r *AppGVHTTP2) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "::")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID format", fmt.Sprintf("Expected session_id::app_id, got %s", req.ID))
		return
	}
	sessionID := parts[0]
	rawID := parts[1]
	fmData := FMGVHTTP2{}
	err := GetMSAppData(ctx, sessionID, rawID, "GVHTTP2", "", &fmData, r.fmClient)
	if err != nil {
		resp.Diagnostics.AddError("Error reading GVHTTP2 app for import", err.Error())
		return
	}
	typedID, err := commonutils.MakeTypedID(commonutils.ModuleApp, commonutils.TypeGVHTTP2, rawID)
	if err != nil {
		resp.Diagnostics.AddError("Error creating typed ID", err.Error())
		return
	}
	data := mapFMGVHTTP2ToState(ctx, fmData, sessionID, typedID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func buildFMGVHTTP2Payload(ctx context.Context, model AppGVHTTP2Model) map[string]interface{} {
	appConfig := map[string]interface{}{
		"http2_mode": model.HTTP2Mode.ValueString(),
	}
	if !model.ProtocolConfig.IsNull() && !model.ProtocolConfig.IsUnknown() {
		var protoModel ProtocolConfigModel
		_ = model.ProtocolConfig.As(ctx, &protoModel, basetypes.ObjectAsOptions{})
		appConfig["protocol_config"] = map[string]interface{}{
			"stream_multiplexing": protoModel.StreamMultiplexing.ValueBool(),
			"server_push":         protoModel.ServerPush.ValueBool(),
		}
	}
	if !model.Compression.IsNull() && !model.Compression.IsUnknown() {
		var compModel CompressionModel
		_ = model.Compression.As(ctx, &compModel, basetypes.ObjectAsOptions{})
		appConfig["compression"] = map[string]interface{}{
			"enabled": compModel.Enabled.ValueBool(),
			"level":   compModel.Level.ValueInt32(),
		}
	}
	if !model.FlowControl.IsNull() && !model.FlowControl.IsUnknown() {
		var fcModel FlowControlModel
		_ = model.FlowControl.As(ctx, &fcModel, basetypes.ObjectAsOptions{})
		appConfig["flow_control"] = map[string]interface{}{
			"window_size":         fcModel.WindowSize.ValueInt32(),
			"initial_window_size": fcModel.InitialWindowSize.ValueInt32(),
		}
	}
	return map[string]interface{}{
		"app_type":   "GVHTTP2",
		"app_config": appConfig,
	}
}

func mapFMGVHTTP2ToState(ctx context.Context, fmData FMGVHTTP2, sessionID string, typedID string) AppGVHTTP2Model {
	model := AppGVHTTP2Model{
		Id:                  types.StringValue(typedID),
		MonitoringSessionId: types.StringValue(sessionID),
		HTTP2Mode:           types.StringValue("enabled"),
	}
	if fmData.AppConfig != nil {
		if http2Mode, ok := fmData.AppConfig["http2_mode"].(string); ok {
			model.HTTP2Mode = types.StringValue(http2Mode)
		}
		if protoConfig, ok := fmData.AppConfig["protocol_config"].(map[string]interface{}); ok {
			protoModel := ProtocolConfigModel{
				StreamMultiplexing: types.BoolValue(true),
				ServerPush:         types.BoolValue(false),
			}
			if sm, ok := protoConfig["stream_multiplexing"].(bool); ok {
				protoModel.StreamMultiplexing = types.BoolValue(sm)
			}
			if sp, ok := protoConfig["server_push"].(bool); ok {
				protoModel.ServerPush = types.BoolValue(sp)
			}
			protoObj, _ := types.ObjectValueFrom(ctx, map[string]attr.Type{
				"stream_multiplexing": types.BoolType,
				"server_push":         types.BoolType,
			}, protoModel)
			model.ProtocolConfig = protoObj
		}
		if compConfig, ok := fmData.AppConfig["compression"].(map[string]interface{}); ok {
			compModel := CompressionModel{
				Enabled: types.BoolValue(true),
				Level:   types.Int32Value(6),
			}
			if enabled, ok := compConfig["enabled"].(bool); ok {
				compModel.Enabled = types.BoolValue(enabled)
			}
			if level, ok := compConfig["level"].(float64); ok {
				compModel.Level = types.Int32Value(int32(level))
			}
			compObj, _ := types.ObjectValueFrom(ctx, map[string]attr.Type{
				"enabled": types.BoolType,
				"level":   types.Int32Type,
			}, compModel)
			model.Compression = compObj
		}
		if fcConfig, ok := fmData.AppConfig["flow_control"].(map[string]interface{}); ok {
			fcModel := FlowControlModel{
				WindowSize:        types.Int32Value(65535),
				InitialWindowSize: types.Int32Value(65535),
			}
			if ws, ok := fcConfig["window_size"].(float64); ok {
				fcModel.WindowSize = types.Int32Value(int32(ws))
			}
			if iws, ok := fcConfig["initial_window_size"].(float64); ok {
				fcModel.InitialWindowSize = types.Int32Value(int32(iws))
			}
			fcObj, _ := types.ObjectValueFrom(ctx, map[string]attr.Type{
				"window_size":         types.Int32Type,
				"initial_window_size": types.Int32Type,
			}, fcModel)
			model.FlowControl = fcObj
		}
	}
	return model
}
