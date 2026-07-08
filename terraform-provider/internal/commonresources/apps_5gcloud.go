// Copyright (c) Gigamon, Inc.
// Licensed under the Mozilla Public License v. 2.0

// Implements the 5G Cloud Application resource for Gigamon Terraform Provider

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
var _ resource.Resource = &App5GCloud{}
var _ resource.ResourceWithConfigure = &App5GCloud{}
var _ resource.ResourceWithImportState = &App5GCloud{}

// New5GCloud creates a new resource instance for 5G Cloud application
func New5GCloud() resource.Resource {
	return &App5GCloud{}
}

// App5GCloud manages the 5G Cloud application resource
type App5GCloud struct {
	fmClient *fmclient.FmClient
}

// App5GCloudModel represents the Terraform configuration and state for 5G Cloud app
type App5GCloudModel struct {
	// Resource identifiers
	Id                  types.String `tfsdk:"id"`
	MonitoringSessionId types.String `tfsdk:"monitoring_session_id"`

	// 5G Cloud specific fields
	Enabled      types.Bool   `tfsdk:"enabled"`
	Profile      types.String `tfsdk:"profile"`
	FilterConfig types.Object `tfsdk:"filter_config"`
	ExportConfig types.Object `tfsdk:"export_config"`
}

// FilterConfigModel represents the filter configuration
type FilterConfigModel struct {
	ProtocolFilter types.String `tfsdk:"protocol_filter"`
	PortRange      types.Object `tfsdk:"port_range"`
}

// PortRangeModel represents the port range
type PortRangeModel struct {
	Min types.Int32 `tfsdk:"min"`
	Max types.Int32 `tfsdk:"max"`
}

// ExportConfigModel represents the export configuration
type ExportConfigModel struct {
	Format   types.String `tfsdk:"format"`
	Interval types.Int32  `tfsdk:"interval"`
}

// FM5GCloud represents the wire format for 5G Cloud application in the FM API
type FM5GCloud struct {
	AppType   string                 `json:"app_type"`
	AppConfig map[string]interface{} `json:"app_config"`
}

// Metadata returns the resource type name
func (r *App5GCloud) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_5gcloud"
}

// Schema defines the resource schema
func (r *App5GCloud) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a 5G Cloud application instance on a monitoring session. " +
			"The 5G Cloud app provides deep packet inspection and analytics for 5G cloud traffic.",

		Attributes: map[string]schema.Attribute{
			// Resource ID
			"id": schema.StringAttribute{
				Description: "The unique identifier for the 5G Cloud app resource",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			// Monitoring Session Reference
			"monitoring_session_id": schema.StringAttribute{
				Description: "The ID of the monitoring session to associate this app with. " +
					"Format: monitoringSession::vmware::<uuid>",
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			// 5G Cloud specific attributes
			"enabled": schema.BoolAttribute{
				Description: "Enable or disable the 5G Cloud application. Defaults to true",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},

			"profile": schema.StringAttribute{
				Description: "Configuration profile for the 5G Cloud app. Valid values: 'default', 'custom'. Defaults to 'default'",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("default"),
				Validators: []validator.String{
					stringvalidator.OneOf("default", "custom"),
				},
			},

			"filter_config": schema.SingleNestedAttribute{
				Description: "Traffic filter configuration",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"protocol_filter": schema.StringAttribute{
						Description: "Protocol filter type. Valid values: 'all', 'udp', 'tcp'. Defaults to 'all'",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("all"),
						Validators: []validator.String{
							stringvalidator.OneOf("all", "udp", "tcp"),
						},
					},

					"port_range": schema.SingleNestedAttribute{
						Description: "Port range configuration",
						Optional:    true,
						Attributes: map[string]schema.Attribute{
							"min": schema.Int32Attribute{
								Description: "Minimum port number. Range: 0-65535. Defaults to 0",
								Optional:    true,
								Computed:    true,
								Default:     int32default.StaticInt32(0),
								Validators: []validator.Int32{
									int32validator.Between(0, 65535),
								},
							},

							"max": schema.Int32Attribute{
								Description: "Maximum port number. Range: 0-65535. Defaults to 65535",
								Optional:    true,
								Computed:    true,
								Default:     int32default.StaticInt32(65535),
								Validators: []validator.Int32{
									int32validator.Between(0, 65535),
								},
							},
						},
					},
				},
			},

			"export_config": schema.SingleNestedAttribute{
				Description: "Export configuration for captured data",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"format": schema.StringAttribute{
						Description: "Export format. Valid values: 'netflow'. Defaults to 'netflow'",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("netflow"),
						Validators: []validator.String{
							stringvalidator.OneOf("netflow"),
						},
					},

					"interval": schema.Int32Attribute{
						Description: "Export interval in seconds. Range: 1-3600. Defaults to 60",
						Optional:    true,
						Computed:    true,
						Default:     int32default.StaticInt32(60),
						Validators: []validator.Int32{
							int32validator.Between(1, 3600),
						},
					},
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

	// Extract the configuration from the plan
	var data App5GCloudModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build the FM API payload from the Terraform model
	fmPayload := buildFM5GCloudPayload(ctx, data)

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
	typedID, err := commonutils.MakeTypedID(commonutils.ModuleApp, commonutils.Type5GCloud, id)
	if err != nil {
		resp.Diagnostics.AddError("Error creating typed ID", err.Error())
		return
	}

	// Fetch the created resource to get full state
	fmData := FM5GCloud{}
	err = GetMSAppData(ctx, sessionID, id, "5GCloud", "", &fmData, r.fmClient)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Failed to read created 5G Cloud app: %v", err))
		resp.Diagnostics.AddError("Error reading created app", err.Error())
		return
	}

	// Map to state
	stateData := mapFM5GCloudToState(ctx, fmData, sessionID, typedID)

	// Save the state
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Successfully created 5G Cloud app: %s", stateData.Id.ValueString()))
}

// Read reads the 5G Cloud app resource from the FM API
func (r *App5GCloud) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Info(ctx, "Reading 5G Cloud app resource")

	// Get the current state from Terraform
	var data App5GCloudModel
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
	fmData := FM5GCloud{}
	err = GetMSAppData(ctx, sessionID, rawID, "5GCloud", "", &fmData, r.fmClient)
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
	stateData := mapFM5GCloudToState(ctx, fmData, sessionID, data.Id.ValueString())

	// Save the refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Successfully read 5G Cloud app: %s", stateData.Id.ValueString()))
}

// Update updates the 5G Cloud app resource
func (r *App5GCloud) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Info(ctx, "Updating 5G Cloud app resource")

	// Get the plan (desired state)
	var planData App5GCloudModel
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
	fmPayload := buildFM5GCloudPayload(ctx, planData)
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
	fmData := FM5GCloud{}
	err = GetMSAppData(ctx, sessionID, rawID, "5GCloud", "", &fmData, r.fmClient)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("Failed to read updated 5G Cloud app: %v", err))
		resp.Diagnostics.AddError("Error reading updated app", err.Error())
		return
	}

	// Map to state
	stateData := mapFM5GCloudToState(ctx, fmData, sessionID, planData.Id.ValueString())

	// Save the updated state
	resp.Diagnostics.Append(resp.State.Set(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, fmt.Sprintf("Successfully updated 5G Cloud app: %s", stateData.Id.ValueString()))
}

// Delete deletes the 5G Cloud app resource
func (r *App5GCloud) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Info(ctx, "Deleting 5G Cloud app resource")

	// Get the current state to extract the session ID and resource ID
	var data App5GCloudModel
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
				"app_type": "5GCloud",
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
func (r *App5GCloud) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
	typedID, err := commonutils.MakeTypedID(commonutils.ModuleApp, commonutils.Type5GCloud, rawID)
	if err != nil {
		resp.Diagnostics.AddError("Error creating typed ID", err.Error())
		return
	}

	// Fetch the app data from FM
	fmData := FM5GCloud{}
	err = GetMSAppData(ctx, sessionID, rawID, "5GCloud", "", &fmData, r.fmClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error importing 5G Cloud app",
			fmt.Sprintf("Could not read app from session %s: %s", sessionID, err.Error()),
		)
		return
	}

	// Map to state
	stateData := mapFM5GCloudToState(ctx, fmData, sessionID, typedID)

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

// buildFM5GCloudPayload converts the Terraform model to FM API payload format
func buildFM5GCloudPayload(ctx context.Context, model App5GCloudModel) map[string]interface{} {
	appConfig := map[string]interface{}{
		"enabled": model.Enabled.ValueBool(),
		"profile": model.Profile.ValueString(),
	}

	// Add filter_config if provided
	if !model.FilterConfig.IsNull() && !model.FilterConfig.IsUnknown() {
		var filterCfg FilterConfigModel
		_ = model.FilterConfig.As(ctx, &filterCfg, basetypes.ObjectAsOptions{})

		filterConfigMap := map[string]interface{}{
			"protocol_filter": filterCfg.ProtocolFilter.ValueString(),
		}

		// Add port_range if provided
		if !filterCfg.PortRange.IsNull() && !filterCfg.PortRange.IsUnknown() {
			var portRange PortRangeModel
			_ = filterCfg.PortRange.As(ctx, &portRange, basetypes.ObjectAsOptions{})

			filterConfigMap["port_range"] = map[string]interface{}{
				"min": portRange.Min.ValueInt32(),
				"max": portRange.Max.ValueInt32(),
			}
		}

		appConfig["filter_config"] = filterConfigMap
	}

	// Add export_config if provided
	if !model.ExportConfig.IsNull() && !model.ExportConfig.IsUnknown() {
		var exportCfg ExportConfigModel
		_ = model.ExportConfig.As(ctx, &exportCfg, basetypes.ObjectAsOptions{})

		appConfig["export_config"] = map[string]interface{}{
			"format":   exportCfg.Format.ValueString(),
			"interval": exportCfg.Interval.ValueInt32(),
		}
	}

	return map[string]interface{}{
		"app_type":   "5GCloud",
		"app_config": appConfig,
	}
}

// mapFM5GCloudToState converts FM API response to Terraform model
func mapFM5GCloudToState(ctx context.Context, fmData FM5GCloud, sessionID string, typedID string) App5GCloudModel {
	model := App5GCloudModel{
		Id:                  types.StringValue(typedID),
		MonitoringSessionId: types.StringValue(sessionID),
		Enabled:             types.BoolValue(true),
		Profile:             types.StringValue("default"),
	}

	// Parse FM response data from AppConfig
	if fmData.AppConfig != nil {
		// Extract enabled
		if enabled, ok := fmData.AppConfig["enabled"].(bool); ok {
			model.Enabled = types.BoolValue(enabled)
		}

		// Extract profile
		if profile, ok := fmData.AppConfig["profile"].(string); ok {
			model.Profile = types.StringValue(profile)
		}

		// Extract filter_config
		if filterCfg, ok := fmData.AppConfig["filter_config"].(map[string]interface{}); ok {
			filterModel := FilterConfigModel{
				ProtocolFilter: types.StringValue("all"),
			}

			if protocolFilter, ok := filterCfg["protocol_filter"].(string); ok {
				filterModel.ProtocolFilter = types.StringValue(protocolFilter)
			}

			// Extract port_range
			if portRange, ok := filterCfg["port_range"].(map[string]interface{}); ok {
				portModel := PortRangeModel{
					Min: types.Int32Value(0),
					Max: types.Int32Value(65535),
				}

				if min, ok := portRange["min"].(float64); ok {
					portModel.Min = types.Int32Value(int32(min))
				}

				if max, ok := portRange["max"].(float64); ok {
					portModel.Max = types.Int32Value(int32(max))
				}

				// Create PortRange object
				portObj, _ := types.ObjectValueFrom(ctx,
					map[string]attr.Type{
						"min": types.Int32Type,
						"max": types.Int32Type,
					},
					portModel,
				)
				filterModel.PortRange = portObj
			}

			// Create FilterConfig object
			filterObj, _ := types.ObjectValueFrom(ctx,
				map[string]attr.Type{
					"protocol_filter": types.StringType,
					"port_range": types.ObjectType{
						AttrTypes: map[string]attr.Type{
							"min": types.Int32Type,
							"max": types.Int32Type,
						},
					},
				},
				filterModel,
			)
			model.FilterConfig = filterObj
		}

		// Extract export_config
		if exportCfg, ok := fmData.AppConfig["export_config"].(map[string]interface{}); ok {
			exportModel := ExportConfigModel{
				Format:   types.StringValue("netflow"),
				Interval: types.Int32Value(60),
			}

			if format, ok := exportCfg["format"].(string); ok {
				exportModel.Format = types.StringValue(format)
			}

			if interval, ok := exportCfg["interval"].(float64); ok {
				exportModel.Interval = types.Int32Value(int32(interval))
			}

			// Create ExportConfig object
			exportObj, _ := types.ObjectValueFrom(ctx,
				map[string]attr.Type{
					"format":   types.StringType,
					"interval": types.Int32Type,
				},
				exportModel,
			)
			model.ExportConfig = exportObj
		}
	}

	return model
}
