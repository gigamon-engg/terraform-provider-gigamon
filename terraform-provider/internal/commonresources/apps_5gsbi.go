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
	SBIMode             types.String `tfsdk:"sbi_mode"`
	ProtocolHandlers    types.List   `tfsdk:"protocol_handlers"`
	Authentication      types.Object `tfsdk:"authentication"`
}

type AuthenticationModel struct {
	Enabled  types.Bool   `tfsdk:"enabled"`
	CertPath types.String `tfsdk:"cert_path"`
	KeyPath  types.String `tfsdk:"key_path"`
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

			"sbi_mode": schema.StringAttribute{
				Description: "SBI mode: nrf, udm, or amf",
				Required:    true,
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

	// Build FM API payload
	payload := buildFM5GSBIPayload(ctx, data)

	// Create the application
	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{
			{
				EntityType:  "application",
				Operation:   "create",
				Application: payload,
			},
		},
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

	// Create typed ID
	typedID, err := commonutils.MakeTypedID(commonutils.ModuleApp, commonutils.Type5GSBI, id)
	if err != nil {
		resp.Diagnostics.AddError("Error creating typed ID", err.Error())
		return
	}

	// Fetch the created resource to get full state
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

	// Fetch the resource
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

	// Build FM API payload with id
	payload := buildFM5GSBIPayload(ctx, planData)
	payload["id"] = rawID

	// Update the application
	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{
			{
				EntityType:  "application",
				Operation:   "update",
				Application: payload,
			},
		},
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

	// Fetch the updated resource to get full state
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

	// Delete the application
	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{
			{
				EntityType: "application",
				Operation:  "delete",
				Application: map[string]interface{}{
					"id":       rawID,
					"app_type": "5G-SBI",
				},
			},
		},
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
	// Expected format: session_id::app_id
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

	// Fetch the resource
	fmData := FM5GSBI{}
	err := GetMSAppData(ctx, sessionID, rawID, "5G-SBI", "", &fmData, r.fmClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error reading 5G-SBI app for import",
			fmt.Sprintf("Could not read app %s: %s", rawID, err.Error()),
		)
		return
	}

	// Create typed ID
	typedID, err := commonutils.MakeTypedID(commonutils.ModuleApp, commonutils.Type5GSBI, rawID)
	if err != nil {
		resp.Diagnostics.AddError("Error creating typed ID", err.Error())
		return
	}

	// Map to state
	data := mapFM5GSBIToState(ctx, fmData, sessionID, typedID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// buildFM5GSBIPayload converts the Terraform model to FM API payload format
func buildFM5GSBIPayload(ctx context.Context, model App5GSBIModel) map[string]interface{} {
	appConfig := map[string]interface{}{
		"sbi_mode": model.SBIMode.ValueString(),
	}

	if !model.ProtocolHandlers.IsNull() && !model.ProtocolHandlers.IsUnknown() {
		var handlers []string
		model.ProtocolHandlers.ElementsAs(ctx, &handlers, false)
		appConfig["protocol_handlers"] = handlers
	}

	if !model.Authentication.IsNull() && !model.Authentication.IsUnknown() {
		var authModel AuthenticationModel
		_ = model.Authentication.As(ctx, &authModel, basetypes.ObjectAsOptions{})
		appConfig["authentication"] = map[string]interface{}{
			"enabled":   authModel.Enabled.ValueBool(),
			"cert_path": authModel.CertPath.ValueString(),
			"key_path":  authModel.KeyPath.ValueString(),
		}
	}

	return map[string]interface{}{
		"app_type":   "5G-SBI",
		"app_config": appConfig,
	}
}

// mapFM5GSBIToState converts FM API response to Terraform model
func mapFM5GSBIToState(ctx context.Context, fmData FM5GSBI, sessionID string, typedID string) App5GSBIModel {
	model := App5GSBIModel{
		Id:                  types.StringValue(typedID),
		MonitoringSessionId: types.StringValue(sessionID),
		SBIMode:             types.StringValue("nrf"),
	}

	if fmData.AppConfig != nil {
		if sbiMode, ok := fmData.AppConfig["sbi_mode"].(string); ok {
			model.SBIMode = types.StringValue(sbiMode)
		}

		if handlers, ok := fmData.AppConfig["protocol_handlers"].([]interface{}); ok {
			var handlerStrs []string
			for _, h := range handlers {
				if hStr, ok := h.(string); ok {
					handlerStrs = append(handlerStrs, hStr)
				}
			}
			handlersObj, _ := types.ListValueFrom(ctx, types.StringType, handlerStrs)
			model.ProtocolHandlers = handlersObj
		}

		if authCfg, ok := fmData.AppConfig["authentication"].(map[string]interface{}); ok {
			authModel := AuthenticationModel{
				Enabled:  types.BoolValue(false),
				CertPath: types.StringValue(""),
				KeyPath:  types.StringValue(""),
			}
			if enabled, ok := authCfg["enabled"].(bool); ok {
				authModel.Enabled = types.BoolValue(enabled)
			}
			if certPath, ok := authCfg["cert_path"].(string); ok {
				authModel.CertPath = types.StringValue(certPath)
			}
			if keyPath, ok := authCfg["key_path"].(string); ok {
				authModel.KeyPath = types.StringValue(keyPath)
			}
			authObj, _ := types.ObjectValueFrom(ctx, map[string]attr.Type{
				"enabled":   types.BoolType,
				"cert_path": types.StringType,
				"key_path":  types.StringType,
			}, authModel)
			model.Authentication = authObj
		}
	}

	return model
}
