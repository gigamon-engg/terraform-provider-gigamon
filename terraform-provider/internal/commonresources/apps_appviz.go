// Copyright (c) Gigamon, Inc.
// Licensed under the Mozilla Public License v. 2.0

// Implements the AppViz application resource for Gigamon Terraform Provider.

package commonresources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
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

var _ resource.Resource = &AppViz{}
var _ resource.ResourceWithConfigure = &AppViz{}
var _ resource.ResourceWithImportState = &AppViz{}
var _ resource.ResourceWithValidateConfig = &AppViz{}


func NewAppViz() resource.Resource {
	return &AppViz{}
}

type AppViz struct {
	fmClient *fmclient.FmClient
}

type AppVizModel struct {
	Id                  types.String `tfsdk:"id"`
	MonitoringSessionId types.String `tfsdk:"monitoring_session_id"`
	Alias               types.String `tfsdk:"alias"`
	Description         types.String `tfsdk:"description"`
	Action              types.Bool   `tfsdk:"action"`
	ExporterConfig      types.Object `tfsdk:"exporter_config"`
	MgmtInterface       types.String `tfsdk:"mgmt_interface"`
}

type AppVizExporterConfigModel struct {
	Monitor types.Object `tfsdk:"monitor"`
}

type AppVizMonitorModel struct {
	Timeout types.Int32 `tfsdk:"timeout"`
}

type FMAppVizMonitor struct {
	Timeout int32 `json:"timeout,omitempty"`
}

type FMAppVizExporterConfig struct {
	Monitor *FMAppVizMonitor `json:"monitor,omitempty"`
}

type FMAppViz struct {
	Id             string                  `json:"id,omitempty"`
	Name           string                  `json:"name,omitempty"`
	Alias          string                  `json:"alias,omitempty"`
	Description    string                  `json:"description,omitempty"`
	Action         bool                    `json:"action"`
	ExporterConfig *FMAppVizExporterConfig `json:"exporterConfig,omitempty"`
	MgmtInterface  string                  `json:"mgmtInterface,omitempty"`
}

func (r *AppViz) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_viz"
}

func (r *AppViz) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an App Viz application instance on a monitoring session.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the App Viz app resource",
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
				Description: "Alias for the App Viz application.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Description: "Description for the App Viz application.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"action": schema.BoolAttribute{
				Description: "Enable or disable action for App Viz.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"exporter_config": schema.SingleNestedAttribute{
				Description: "Exporter configuration for App Viz. This is required.",
				Required:    true,
				Attributes: map[string]schema.Attribute{
					"monitor": schema.SingleNestedAttribute{
						Description: "Monitor-specific export behavior.",
						Optional:    true,
						Computed:    true,
						Default: objectdefault.StaticValue(
							types.ObjectValueMust(
								map[string]attr.Type{"timeout": types.Int32Type},
								map[string]attr.Value{"timeout": types.Int32Value(300)},
							),
						),
						Attributes: map[string]schema.Attribute{
							"timeout": schema.Int32Attribute{
								Description: "Monitor timeout in seconds. Fixed at 300 for App Viz; not user-configurable.",
								Optional:    true,
								Computed:    true,
								Default:     int32default.StaticInt32(300),
								Validators: []validator.Int32{
									int32validator.OneOf(300),
								},
							},
						},
					},
				},
			},

			"mgmt_interface": schema.StringAttribute{
				Description: "Management interface used by App Viz. Valid values: internal, external.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("internal"),
				Validators: []validator.String{
					stringvalidator.OneOf("internal", "external"),
				},
			},
		},
	}
}

func (r *AppViz) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AppViz) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppVizModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sessionID := data.MonitoringSessionId.ValueString()
	payload := buildFMAppVizPayload(ctx, data)

	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{{
			EntityType:  "application",
			Operation:   "create",
			Application: payload,
		}},
	}

	id, err := commonutils.UpdateMonSess(ctx, &updateReq, sessionID, r.fmClient)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Failed to create App Viz app: %v", err))
		resp.Diagnostics.AddError(
			"Error creating App Viz app",
			fmt.Sprintf("Could not create app on session %s: %s", sessionID, err.Error()),
		)
		return
	}

	typedID, err := commonutils.MakeTypedID(commonutils.ModuleApp, commonutils.TypeAppViz, id)
	if err != nil {
		resp.Diagnostics.AddError("Error creating typed ID", err.Error())
		return
	}

	fmData := FMAppViz{}
	err = GetMSAppData(ctx, sessionID, id, "appviz", "", &fmData, r.fmClient)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Failed to fetch created App Viz app: %v", err))
	} else {
		data = mapFMAppVizToState(ctx, fmData, sessionID, typedID)
	}

	data.Id = types.StringValue(typedID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppViz) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppVizModel

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

	fmData := FMAppViz{}
	err = GetMSAppData(ctx, sessionID, rawID, "appviz", "", &fmData, r.fmClient)
	if err != nil {
		var fmErr *fmclient.FMErrors
		if errors.As(err, &fmErr) && fmErr.ErrorCode() == fmclient.ObjectNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading App Viz app",
			fmt.Sprintf("Could not read app %s: %s", rawID, err.Error()),
		)
		return
	}

	data = mapFMAppVizToState(ctx, fmData, sessionID, typedID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppViz) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData AppVizModel

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

	payload := buildFMAppVizPayload(ctx, planData)
	payload.Id = rawID

	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{{
			EntityType:  "application",
			Operation:   "update",
			Application: payload,
		}},
	}

	_, err = commonutils.UpdateMonSess(ctx, &updateReq, sessionID, r.fmClient)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Failed to update App Viz app: %v", err))
		resp.Diagnostics.AddError(
			"Error updating App Viz app",
			fmt.Sprintf("Could not update app on session %s: %s", sessionID, err.Error()),
		)
		return
	}

	fmData := FMAppViz{}
	err = GetMSAppData(ctx, sessionID, rawID, "appviz", "", &fmData, r.fmClient)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Failed to fetch updated App Viz app: %v", err))
	} else {
		planData = mapFMAppVizToState(ctx, fmData, sessionID, typedID)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
}

func (r *AppViz) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppVizModel

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
			Application: FMAppViz{
				Id:   rawID,
				Name: "appviz",
			},
		}},
	}

	_, err = commonutils.UpdateMonSess(ctx, &updateReq, sessionID, r.fmClient)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Failed to delete App Viz app: %v", err))
		resp.Diagnostics.AddError(
			"Error deleting App Viz app",
			fmt.Sprintf("Could not delete app on session %s: %s", sessionID, err.Error()),
		)
		return
	}
}

func (r *AppViz) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

	fmData := FMAppViz{}
	err := GetMSAppData(ctx, sessionID, rawID, "appviz", "", &fmData, r.fmClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading App Viz app for import",
			fmt.Sprintf("Could not read app %s: %s", rawID, err.Error()),
		)
		return
	}

	typedID, err := commonutils.MakeTypedID(commonutils.ModuleApp, commonutils.TypeAppViz, rawID)
	if err != nil {
		resp.Diagnostics.AddError("Error creating typed ID", err.Error())
		return
	}

	data := mapFMAppVizToState(ctx, fmData, sessionID, typedID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppViz) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
    var data AppVizModel

    resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }

    if data.ExporterConfig.IsNull() {
        resp.Diagnostics.AddAttributeError(
            path.Root("exporter_config"),
            "Missing Required Attribute",
            "exporter_config is required and cannot be null. Please provide an exporter_config block (e.g. with a monitor.timeout setting) for the App Viz application.",
        )
        return
    }
}

func buildFMAppVizPayload(ctx context.Context, model AppVizModel) FMAppViz {
    fmData := FMAppViz{
        Name:          "appviz",
        Alias:         model.Alias.ValueString(),
        Description:   model.Description.ValueString(),
        Action:        model.Action.ValueBool(), // defaults to true via schema Default
        MgmtInterface: model.MgmtInterface.ValueString(),
    }

    if model.ExporterConfig.IsNull() {
        panic("exporter_config is required and cannot be null")
    }

    // Always default the monitor timeout to 300, regardless of whether the
    // "monitor" sub-block was supplied in config, since it is fixed/non-configurable.
    timeout := int32(300)

    if !model.ExporterConfig.IsNull() && !model.ExporterConfig.IsUnknown() {
        var exporterCfg AppVizExporterConfigModel
        _ = model.ExporterConfig.As(ctx, &exporterCfg, basetypes.ObjectAsOptions{})

        if !exporterCfg.Monitor.IsNull() && !exporterCfg.Monitor.IsUnknown() {
            var monitorCfg AppVizMonitorModel
            _ = exporterCfg.Monitor.As(ctx, &monitorCfg, basetypes.ObjectAsOptions{})
            if !monitorCfg.Timeout.IsNull() && !monitorCfg.Timeout.IsUnknown() {
                timeout = monitorCfg.Timeout.ValueInt32()
            }
        }
    }

    fmData.ExporterConfig = &FMAppVizExporterConfig{
        Monitor: &FMAppVizMonitor{
            Timeout: timeout,
        },
    }

    return fmData
}


func mapFMAppVizToState(ctx context.Context, fmData FMAppViz, sessionID string, typedID string) AppVizModel {
	model := AppVizModel{
		Id:                  types.StringValue(typedID),
		MonitoringSessionId: types.StringValue(sessionID),
		Alias:               types.StringValue(""),
		Description:         types.StringValue(""),
		Action:              types.BoolValue(true),
		MgmtInterface:       types.StringValue("internal"),
	}

	if fmData.Alias != "" {
		model.Alias = types.StringValue(fmData.Alias)
	}

	model.Description = types.StringValue(fmData.Description)
	model.Action = types.BoolValue(fmData.Action)

	if fmData.MgmtInterface != "" {
		model.MgmtInterface = types.StringValue(fmData.MgmtInterface)
	}

	if fmData.ExporterConfig != nil && fmData.ExporterConfig.Monitor != nil {
		exporterModel := AppVizExporterConfigModel{}
		monitorModel := AppVizMonitorModel{Timeout: types.Int32Value(fmData.ExporterConfig.Monitor.Timeout)}

		monitorObj, _ := types.ObjectValueFrom(ctx,
			map[string]attr.Type{
				"timeout": types.Int32Type,
			},
			monitorModel,
		)
		exporterModel.Monitor = monitorObj

		exporterObj, _ := types.ObjectValueFrom(ctx,
			map[string]attr.Type{
				"monitor": types.ObjectType{
					AttrTypes: map[string]attr.Type{
						"timeout": types.Int32Type,
					},
				},
			},
			exporterModel,
		)
		model.ExporterConfig = exporterObj
	}

	return model
}
