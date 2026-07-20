// Copyright (c) Gigamon, Inc.
// Licensed under the Mozilla Public License v. 2.0

// Implements the PCapNG Application resource for Gigamon Terraform Provider

package commonresources

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

	"terraform-provider-gigamon/internal/commonutils"
	"terraform-provider-gigamon/internal/fmclient"
)

var pcapNGNameAliasRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

const pcapNGAppName = "pcapng"

var _ resource.Resource = &AppPCapNG{}
var _ resource.ResourceWithConfigure = &AppPCapNG{}
var _ resource.ResourceWithImportState = &AppPCapNG{}

func NewPCapNG() resource.Resource {
	return &AppPCapNG{}
}

type AppPCapNG struct {
	fmClient *fmclient.FmClient
}

type AppPCapNGModel struct {
	Id                   types.String `tfsdk:"id"`
	MonitoringSessionId  types.String `tfsdk:"monitoring_session_id"`
	Alias                types.String `tfsdk:"alias"`
	Name                 types.String `tfsdk:"name"`
	AppMode              types.String `tfsdk:"app_mode"`
	DomainClassification types.Bool   `tfsdk:"domain_classification"`
	DomainTableAlias     types.String `tfsdk:"domain_table_alias"`
	FlowTimeout          types.Int32  `tfsdk:"flow_timeout"`
}

type FMPCapNG struct {
	AppType              string                 `json:"app_type"`
	Alias                string                 `json:"alias"`
	Name                 string                 `json:"name"`
	AppMode              string                 `json:"appMode"`
	DomainClassification *bool                  `json:"domainClassification"`
	DomainTableAlias     string                 `json:"domainTableAlias"`
	FlowTimeout          *int32                 `json:"flowTimeout"`
	AppConfig            map[string]interface{} `json:"app_config"`
}

func (r *AppPCapNG) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_pcapng"
}

func (r *AppPCapNG) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a PCapNG application instance on a monitoring session.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The unique identifier for the PCapNG app resource",
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
			"alias": schema.StringAttribute{
				Description: "Alias for the PCapNG application.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.RegexMatches(pcapNGNameAliasRegex, "only alphanumeric, '-' and '_' are allowed"),
				},
			},
			"name": schema.StringAttribute{
				Description: "Internal FM application name. Always pcapng.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"app_mode": schema.StringAttribute{
				Description: "App mode. Either primary or secondary.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("secondary"),
				Validators: []validator.String{
					stringvalidator.OneOf("primary", "secondary"),
				},
			},
			"domain_classification": schema.BoolAttribute{
				Description: "Configurable only when app_mode is primary.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"domain_table_alias": schema.StringAttribute{
				Description: "Configurable only when app_mode is primary and domain_classification is true.",
				Optional:    true,
			},
			"flow_timeout": schema.Int32Attribute{
				Description: "Configurable only when app_mode is primary and domain_classification is true. Range: 360-1860.",
				Optional:    true,
				Computed:    true,
				Default:     int32default.StaticInt32(660),
			},
		},
	}
}

func (r *AppPCapNG) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AppPCapNG) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppPCapNGModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	setPCapNGSystemFields(&data)

	validatePCapNGModel(&resp.Diagnostics, data)
	if resp.Diagnostics.HasError() {
		return
	}

	sessionID := data.MonitoringSessionId.ValueString()
	payload := buildFMPCapNGPayload(data)
	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{{
			EntityType:  "application",
			Operation:   "create",
			Application: payload,
		}},
	}
	id, err := commonutils.UpdateMonSess(ctx, &updateReq, sessionID, r.fmClient)
	if err != nil {
		resp.Diagnostics.AddError("Error creating PCapNG app", err.Error())
		return
	}

	typedID, err := commonutils.MakeTypedID(commonutils.ModuleApp, commonutils.TypePCapNG, id)
	if err != nil {
		resp.Diagnostics.AddError("Error creating typed ID", err.Error())
		return
	}

	fmData := FMPCapNG{}
	err = GetMSAppData(ctx, sessionID, id, "pcapng", "", &fmData, r.fmClient)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Failed to fetch created PCapNG app: %v", err))
	} else {
		data = mapFMPCapNGToState(fmData, sessionID, typedID)
	}
	data.Id = types.StringValue(typedID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppPCapNG) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppPCapNGModel
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

	fmData := FMPCapNG{}
	err = GetMSAppData(ctx, sessionID, rawID, "pcapng", "", &fmData, r.fmClient)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading PCapNG app", err.Error())
		return
	}

	data = mapFMPCapNGToState(fmData, sessionID, typedID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppPCapNG) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData AppPCapNGModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}
	setPCapNGSystemFields(&planData)

	validatePCapNGModel(&resp.Diagnostics, planData)
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

	payload := buildFMPCapNGPayload(planData)
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
		resp.Diagnostics.AddError("Error updating PCapNG app", err.Error())
		return
	}

	fmData := FMPCapNG{}
	err = GetMSAppData(ctx, sessionID, rawID, "pcapng", "", &fmData, r.fmClient)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Failed to fetch updated PCapNG app: %v", err))
	} else {
		planData = mapFMPCapNGToState(fmData, sessionID, typedID)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
}

func (r *AppPCapNG) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppPCapNGModel
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
				"name":     pcapNGAppName,
				"app_type": "PCapNG",
			},
		}},
	}
	_, err = commonutils.UpdateMonSess(ctx, &updateReq, sessionID, r.fmClient)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting PCapNG app", err.Error())
		return
	}
}

func (r *AppPCapNG) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "::")
	if len(parts) != 2 {
		resp.Diagnostics.AddError("Invalid import ID format", fmt.Sprintf("Expected session_id::app_id, got %s", req.ID))
		return
	}

	sessionID := parts[0]
	rawID := parts[1]
	fmData := FMPCapNG{}
	err := GetMSAppData(ctx, sessionID, rawID, "pcapng", "", &fmData, r.fmClient)
	if err != nil {
		resp.Diagnostics.AddError("Error reading PCapNG app for import", err.Error())
		return
	}

	typedID, err := commonutils.MakeTypedID(commonutils.ModuleApp, commonutils.TypePCapNG, rawID)
	if err != nil {
		resp.Diagnostics.AddError("Error creating typed ID", err.Error())
		return
	}

	data := mapFMPCapNGToState(fmData, sessionID, typedID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func buildFMPCapNGPayload(model AppPCapNGModel) map[string]interface{} {
	appMode := model.AppMode.ValueString()
	if appMode == "" {
		appMode = "secondary"
	}

	payload := map[string]interface{}{
		"name":    pcapNGAppName,
		"alias":   model.Alias.ValueString(),
		"appMode": appMode,
	}

	if appMode == "primary" {
		domainClassification := model.DomainClassification.ValueBool()
		payload["domainClassification"] = domainClassification
		if domainClassification {
			if !model.DomainTableAlias.IsNull() && !model.DomainTableAlias.IsUnknown() {
				payload["domainTableAlias"] = model.DomainTableAlias.ValueString()
			}
			payload["flowTimeout"] = model.FlowTimeout.ValueInt32()
		}
	}

	return payload
}

func mapFMPCapNGToState(fmData FMPCapNG, sessionID string, typedID string) AppPCapNGModel {
	model := AppPCapNGModel{
		Id:                   types.StringValue(typedID),
		MonitoringSessionId:  types.StringValue(sessionID),
		Alias:                types.StringNull(),
		Name:                 types.StringValue(pcapNGAppName),
		AppMode:              types.StringValue("secondary"),
		DomainClassification: types.BoolValue(false),
		DomainTableAlias:     types.StringNull(),
		FlowTimeout:          types.Int32Value(660),
	}

	if fmData.Alias != "" {
		model.Alias = types.StringValue(fmData.Alias)
	}
	if fmData.AppMode != "" {
		model.AppMode = types.StringValue(fmData.AppMode)
	}
	if fmData.DomainClassification != nil {
		model.DomainClassification = types.BoolValue(*fmData.DomainClassification)
	}
	if fmData.DomainTableAlias != "" {
		model.DomainTableAlias = types.StringValue(fmData.DomainTableAlias)
	}
	if fmData.FlowTimeout != nil {
		model.FlowTimeout = types.Int32Value(*fmData.FlowTimeout)
	}

	if fmData.AppConfig != nil {
		if alias, ok := fmData.AppConfig["alias"].(string); ok {
			model.Alias = types.StringValue(alias)
		}
		if appMode, ok := fmData.AppConfig["appMode"].(string); ok {
			model.AppMode = types.StringValue(appMode)
		}
		if domainClassification, ok := fmData.AppConfig["domainClassification"].(bool); ok {
			model.DomainClassification = types.BoolValue(domainClassification)
		}
		if domainTableAlias, ok := fmData.AppConfig["domainTableAlias"].(string); ok {
			model.DomainTableAlias = types.StringValue(domainTableAlias)
		}

		if flowTimeout, ok := fmData.AppConfig["flowTimeout"].(float64); ok {
			model.FlowTimeout = types.Int32Value(int32(flowTimeout))
		} else if flowTimeout, ok := fmData.AppConfig["flowTimeout"].(int32); ok {
			model.FlowTimeout = types.Int32Value(flowTimeout)
		} else if flowTimeout, ok := fmData.AppConfig["flowTimeout"].(int); ok {
			model.FlowTimeout = types.Int32Value(int32(flowTimeout))
		}
	}

	return model
}

func setPCapNGSystemFields(model *AppPCapNGModel) {
	model.Name = types.StringValue(pcapNGAppName)
}

func validatePCapNGModel(diags *diag.Diagnostics, model AppPCapNGModel) {
	appMode := model.AppMode.ValueString()
	if appMode == "" {
		appMode = "secondary"
	}

	if appMode != "primary" {
		if !model.DomainClassification.IsNull() && !model.DomainClassification.IsUnknown() && model.DomainClassification.ValueBool() {
			diags.AddError("Invalid domain_classification", "domain_classification can be set to true only when app_mode is primary")
		}
		if !model.DomainTableAlias.IsNull() && !model.DomainTableAlias.IsUnknown() && model.DomainTableAlias.ValueString() != "" {
			diags.AddError("Invalid domain_table_alias", "domain_table_alias can be configured only when app_mode is primary and domain_classification is true")
		}
		if !model.FlowTimeout.IsNull() && !model.FlowTimeout.IsUnknown() && model.FlowTimeout.ValueInt32() != 660 {
			diags.AddError("Invalid flow_timeout", "flow_timeout can be configured only when app_mode is primary and domain_classification is true")
		}
		return
	}

	if !model.DomainClassification.IsNull() && !model.DomainClassification.IsUnknown() && !model.DomainClassification.ValueBool() {
		if !model.DomainTableAlias.IsNull() && !model.DomainTableAlias.IsUnknown() && model.DomainTableAlias.ValueString() != "" {
			diags.AddError("Invalid domain_table_alias", "domain_table_alias can be configured only when domain_classification is true")
		}
		if !model.FlowTimeout.IsNull() && !model.FlowTimeout.IsUnknown() && model.FlowTimeout.ValueInt32() != 660 {
			diags.AddError("Invalid flow_timeout", "flow_timeout can be configured only when domain_classification is true")
		}
		return
	}

	if !model.FlowTimeout.IsNull() && !model.FlowTimeout.IsUnknown() {
		flowTimeout := model.FlowTimeout.ValueInt32()
		if flowTimeout < 360 || flowTimeout > 1860 {
			diags.AddError("Invalid flow_timeout", "flow_timeout must be between 360 and 1860")
		}
	}
}
