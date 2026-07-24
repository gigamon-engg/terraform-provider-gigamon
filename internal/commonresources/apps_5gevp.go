// Copyright (c) Gigamon, Inc.
// Licensed under the Mozilla Public License v. 2.0

// Implements the 5G-EVP Application resource for Gigamon Terraform Provider

package commonresources

import (
	"context"
	"errors"
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
var _ resource.Resource = &App5GEvp{}
var _ resource.ResourceWithConfigure = &App5GEvp{}
var _ resource.ResourceWithImportState = &App5GEvp{}

// New5GEvp creates a new resource instance for 5G Cloud application
func New5GEvp() resource.Resource {
	return &App5GEvp{}
}

// App5GEvp manages the 5G Cloud application resource
type App5GEvp struct {
	fmClient *fmclient.FmClient
}

// App5GEvpModel represents the Terraform configuration and state for 5G-EVP app
type App5GEvpModel struct {
	Id                  types.String `tfsdk:"id"`
	MonitoringSessionId types.String `tfsdk:"monitoring_session_id"`
	NetworkMode         types.String `tfsdk:"network_mode"`
	TrafficOptimization types.Object `tfsdk:"traffic_optimization"`
	PerformanceTuning   types.Object `tfsdk:"performance_tuning"`
}

type TrafficOptimizationModel struct {
	Enabled types.Bool  `tfsdk:"enabled"`
	Level   types.Int32 `tfsdk:"level"`
}

type PerformanceTuningModel struct {
	CacheSize   types.Int32 `tfsdk:"cache_size"`
	BufferDepth types.Int32 `tfsdk:"buffer_depth"`
}

// FM5GEvp represents the wire format for 5G-EVP application in the FM API
type FM5GEvp struct {
	AppType   string                 `json:"app_type"`
	AppConfig map[string]interface{} `json:"app_config"`
}

// Metadata returns the resource type name
func (r *App5GEvp) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_5gevp"
}

// Schema defines the resource schema
func (r *App5GEvp) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a 5G-EVP (Enhanced Visibility Platform) application instance on a monitoring session.",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the 5G-EVP app resource",
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

			"network_mode": schema.StringAttribute{
				Description: "Network mode configuration. Valid values: 'standalone', 'distributed'. Defaults to 'standalone'",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("standalone"),
				Validators: []validator.String{
					stringvalidator.OneOf("standalone", "distributed"),
				},
			},

			"traffic_optimization": schema.SingleNestedAttribute{
				Description: "Traffic optimization settings",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"enabled": schema.BoolAttribute{
						Description: "Enable traffic optimization. Defaults to true",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(true),
					},

					"level": schema.Int32Attribute{
						Description: "Optimization level (1-10). Defaults to 5",
						Optional:    true,
						Computed:    true,
						Default:     int32default.StaticInt32(5),
						Validators: []validator.Int32{
							int32validator.Between(1, 10),
						},
					},
				},
			},

			"performance_tuning": schema.SingleNestedAttribute{
				Description: "Performance tuning configuration",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"cache_size": schema.Int32Attribute{
						Description: "Cache size in MB (64-1024). Defaults to 256",
						Optional:    true,
						Computed:    true,
						Default:     int32default.StaticInt32(256),
						Validators: []validator.Int32{
							int32validator.Between(64, 1024),
						},
					},

					"buffer_depth": schema.Int32Attribute{
						Description: "Buffer depth in packets (100-10000). Defaults to 1000",
						Optional:    true,
						Computed:    true,
						Default:     int32default.StaticInt32(1000),
						Validators: []validator.Int32{
							int32validator.Between(100, 10000),
						},
					},
				},
			},
		},
	}
}

// Configure configures the resource with provider client
func (r *App5GEvp) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *App5GEvp) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Info(ctx, "Creating 5G Cloud app resource")

	// Extract the configuration from the plan
	var data App5GEvpModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the FM API payload from the Terraform model
	fmPayload := buildFM5GEvpPayload(ctx, data)

	// Create update request for monitoring session
	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{{
			EntityType:  "application",
			Operation:   "create",
			Application: fmPayload,
		}},
	}

	// Call the FM API to create the app
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

	// Create typed ID
	typedID, err := commonutils.MakeTypedID(commonutils.ModuleApp, commonutils.Type5GEvp, id)
	if err != nil {
		resp.Diagnostics.AddError("Error creating typed ID", err.Error())
		return
	}

	// Fetch the created resource to get full state
	fmData := FM5GEvp{}
	err = GetMSAppData(ctx, sessionID, id, "5GEvp", "", &fmData, r.fmClient)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Failed to read created 5G Cloud app: %v", err))
		resp.Diagnostics.AddError("Error reading created app", err.Error())
		return
	}

	// Map to state
	stateData := mapFM5GEvpToState(ctx, fmData, sessionID, typedID)

	// Save the state
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Successfully created 5G Cloud app: %s", stateData.Id.ValueString()))
}

// Read reads the 5G Cloud app resource from the FM API
func (r *App5GEvp) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Reading 5G Cloud app resource")

	// Get the current state from Terraform
	var data App5GEvpModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract UUID from typed ID
	rawID, err := commonutils.UUIDFromTypedID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing ID", err.Error())
		return
	}

	// Fetch the app data from FM API
	sessionID := data.MonitoringSessionId.ValueString()
	fmData := FM5GEvp{}
	err = GetMSAppData(ctx, sessionID, rawID, "5GEvp", "", &fmData, r.fmClient)
	if err != nil {
		var fmErr *fmclient.FMErrors
		if errors.As(err, &fmErr) && fmErr.ErrorCode() == fmclient.ObjectNotFound {
			// Resource no longer exists
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

	// Map the FM response back to the Terraform state model
	stateData := mapFM5GEvpToState(ctx, fmData, sessionID, data.Id.ValueString())

	// Save the refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Successfully read 5G Cloud app: %s", stateData.Id.ValueString()))
}

// Update updates the 5G Cloud app resource
func (r *App5GEvp) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Updating 5G Cloud app resource")

	// Get the plan (desired state)
	var planData App5GEvpModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract UUID from typed ID
	rawID, err := commonutils.UUIDFromTypedID(planData.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing ID", err.Error())
		return
	}

	// Build the FM API update payload
	fmPayload := buildFM5GEvpPayload(ctx, planData)
	fmPayload["id"] = rawID

	// Create update request for monitoring session
	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{{
			EntityType:  "application",
			Operation:   "update",
			Application: fmPayload,
		}},
	}

	// Call the FM API to update the app
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

	// Fetch the updated resource to get full state
	fmData := FM5GEvp{}
	err = GetMSAppData(ctx, sessionID, rawID, "5GEvp", "", &fmData, r.fmClient)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Failed to read updated 5G Cloud app: %v", err))
		resp.Diagnostics.AddError("Error reading updated app", err.Error())
		return
	}

	// Map to state
	stateData := mapFM5GEvpToState(ctx, fmData, sessionID, planData.Id.ValueString())

	// Save the updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Successfully updated 5G Cloud app: %s", stateData.Id.ValueString()))
}

// Delete deletes the 5G Cloud app resource
func (r *App5GEvp) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Deleting 5G Cloud app resource")

	// Get the current state to extract the session ID and resource ID
	var data App5GEvpModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract UUID from typed ID
	rawID, err := commonutils.UUIDFromTypedID(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error parsing ID", err.Error())
		return
	}

	// Create delete request for monitoring session
	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{{
			EntityType: "application",
			Operation:  "delete",
			Application: map[string]interface{}{
				"id":       rawID,
				"app_type": "5GEvp",
			},
		}},
	}

	// Call the FM API to delete the app
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
func (r *App5GEvp) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	tflog.Info(ctx, fmt.Sprintf("Importing 5G Cloud app resource: %s", req.ID))

	// The import ID format is: <sessionID>::<resourceID>
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

	// Create typed ID
	typedID, err := commonutils.MakeTypedID(commonutils.ModuleApp, commonutils.Type5GEvp, rawID)
	if err != nil {
		resp.Diagnostics.AddError("Error creating typed ID", err.Error())
		return
	}

	// Fetch the app data from FM
	fmData := FM5GEvp{}
	err = GetMSAppData(ctx, sessionID, rawID, "5GEvp", "", &fmData, r.fmClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing 5G Cloud app",
			fmt.Sprintf("Could not read app from session %s: %s", sessionID, err.Error()),
		)
		return
	}

	// Map to state
	stateData := mapFM5GEvpToState(ctx, fmData, sessionID, typedID)

	// Save state
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Successfully imported 5G Cloud app: %s", stateData.Id.ValueString()))
}

// ============================================================================
// Helper Functions
// ============================================================================

// buildFM5GEvpPayload converts the Terraform model to FM API payload format
func buildFM5GEvpPayload(ctx context.Context, model App5GEvpModel) map[string]interface{} {
	appConfig := map[string]interface{}{
		"network_mode": model.NetworkMode.ValueString(),
	}

	if !model.TrafficOptimization.IsNull() && !model.TrafficOptimization.IsUnknown() {
		var toModel TrafficOptimizationModel
		_ = model.TrafficOptimization.As(ctx, &toModel, basetypes.ObjectAsOptions{})
		appConfig["traffic_optimization"] = map[string]interface{}{
			"enabled": toModel.Enabled.ValueBool(),
			"level":   toModel.Level.ValueInt32(),
		}
	}

	if !model.PerformanceTuning.IsNull() && !model.PerformanceTuning.IsUnknown() {
		var ptModel PerformanceTuningModel
		_ = model.PerformanceTuning.As(ctx, &ptModel, basetypes.ObjectAsOptions{})
		appConfig["performance_tuning"] = map[string]interface{}{
			"cache_size":   ptModel.CacheSize.ValueInt32(),
			"buffer_depth": ptModel.BufferDepth.ValueInt32(),
		}
	}

	return map[string]interface{}{
		"app_type":   "5G-EVP",
		"app_config": appConfig,
	}
}

// mapFM5GEvpToState converts FM API response to Terraform model
func mapFM5GEvpToState(ctx context.Context, fmData FM5GEvp, sessionID string, typedID string) App5GEvpModel {
	model := App5GEvpModel{
		Id:                  types.StringValue(typedID),
		MonitoringSessionId: types.StringValue(sessionID),
		NetworkMode:         types.StringValue("standalone"),
	}

	if fmData.AppConfig != nil {
		if networkMode, ok := fmData.AppConfig["network_mode"].(string); ok {
			model.NetworkMode = types.StringValue(networkMode)
		}

		if toCfg, ok := fmData.AppConfig["traffic_optimization"].(map[string]interface{}); ok {
			toModel := TrafficOptimizationModel{
				Enabled: types.BoolValue(true),
				Level:   types.Int32Value(5),
			}
			if enabled, ok := toCfg["enabled"].(bool); ok {
				toModel.Enabled = types.BoolValue(enabled)
			}
			if level, ok := toCfg["level"].(float64); ok {
				toModel.Level = types.Int32Value(int32(level))
			}
			toObj, _ := types.ObjectValueFrom(ctx, map[string]attr.Type{
				"enabled": types.BoolType,
				"level":   types.Int32Type,
			}, toModel)
			model.TrafficOptimization = toObj
		}

		if ptCfg, ok := fmData.AppConfig["performance_tuning"].(map[string]interface{}); ok {
			ptModel := PerformanceTuningModel{
				CacheSize:   types.Int32Value(256),
				BufferDepth: types.Int32Value(1000),
			}
			if cacheSize, ok := ptCfg["cache_size"].(float64); ok {
				ptModel.CacheSize = types.Int32Value(int32(cacheSize))
			}
			if bufDepth, ok := ptCfg["buffer_depth"].(float64); ok {
				ptModel.BufferDepth = types.Int32Value(int32(bufDepth))
			}
			ptObj, _ := types.ObjectValueFrom(ctx, map[string]attr.Type{
				"cache_size":   types.Int32Type,
				"buffer_depth": types.Int32Type,
			}, ptModel)
			model.PerformanceTuning = ptObj
		}
	}

	return model
}
