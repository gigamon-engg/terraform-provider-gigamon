// Copyright (c) Gigamon, Inc.

// Implements the map Resrouces that are common across all environment

package commonresources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-gigamon/internal/commonutils"
	"terraform-provider-gigamon/internal/fmclient"
	//"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &TrafficMap{}
var _ resource.Resource = &InclusionMap{}
var _ resource.Resource = &ExclusionMap{}
var _ resource.ResourceWithValidateConfig = &TrafficMap{}

// TrafficMap resoruce, which manages the Maps for Traffic Handling
func NewTrafficMap() resource.Resource {
	return &TrafficMap{}
}

// InclusionMap resource, which manages Inclusion Maps
func NewInclusionMap() resource.Resource {
	return &InclusionMap{}
}

// ExclusionMap resource, which manages Exclusion Maps
func NewExclusionMap() resource.Resource {
	return &ExclusionMap{}
}

// TrafficMap - implements the maps for traffic handling
type TrafficMap struct {
	fmClient *fmclient.FmClient // Instance to our FM http client instance
}

// InclusionMap - implements inclusion maps (used for ATS target selection)
type InclusionMap struct {
	fmClient *fmclient.FmClient
}

// ExclusionMap - implements exclusion maps (used for ATS target selection)
type ExclusionMap struct {
	fmClient *fmclient.FmClient
}

func (tm *TrafficMap) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_traffic_map"
}

func (tm *TrafficMap) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = MapSchema()
}

func (tm *TrafficMap) ValidateConfig(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	if !req.Config.Raw.IsFullyKnown() {
		return
	}

	var cfg MapModel
	diags := req.Config.Get(ctx, &cfg)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if cfg.Asf != nil && cfg.Asf.AsfProfileConfig == nil {
		resp.Diagnostics.AddError("Invalid asf", "asf.asf_profile_config must be set when asf is provided")
		return
	}
	validateAsfMonitoringSessionPrereqs(ctx, tm.fmClient, cfg.MonitoringSessionId, cfg.Asf, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	applyAndValidateAsfSessionFields(cfg.Asf, &resp.Diagnostics)
	validateAppRulesRequireAsf(&cfg, &resp.Diagnostics)
	validateAppRulesRuleIDRequired(&cfg, &resp.Diagnostics)
}

// Initial Configure call, to initialize the Provider
func (tm *TrafficMap) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	fmClient, ok := req.ProviderData.(*fmclient.FmClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *fmclient.FmClient, got: %T. Report the issue to Gigamon", req.ProviderData),
		)
		return
	}
	tm.fmClient = fmClient
}

// Create call for new Traffic Map
func (tm *TrafficMap) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MapModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Asf != nil && data.Asf.AsfProfileConfig == nil {
		resp.Diagnostics.AddError("Invalid asf", "asf.asf_profile_config must be set when asf is provided")
		return
	}
	validateAsfMonitoringSessionPrereqs(ctx, tm.fmClient, data.MonitoringSessionId, data.Asf, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	applyAndValidateAsfSessionFields(data.Asf, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	validateAppRulesRequireAsf(&data, &resp.Diagnostics)
	validateAppRulesRuleIDRequired(&data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	trafficMap := ModelMapToGoMap(ctx, &data)
	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{
			{
				EntityType: "trafficMap",
				Operation:  "create",
				Map:        trafficMap,
			},
		},
	}

	id, err := commonutils.UpdateMonSess(
		ctx,
		&updateReq,
		data.MonitoringSessionId.ValueString(),
		tm.fmClient,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create traffic map",
			fmt.Sprintf("traffic map creation failed: %v", err),
		)
		return
	}

	typedID, err := commonutils.MakeTypedID(
		commonutils.ModuleMap,
		commonutils.TypeTrafficMap,
		id,
	)
	if err != nil {
		return
	}
	data.Id = types.StringValue(typedID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (tm *TrafficMap) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MapModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rawID, err := commonutils.UUIDFromTypedID(data.Id.ValueString())
	if err != nil {
		return
	}

	fmData, err := GetMSMapData(
		ctx,
		data.MonitoringSessionId.ValueString(),
		rawID,
		data.Name.ValueString(),
		"trafficMap",
		tm.fmClient,
	)
	if err != nil {
		var fmErr *fmclient.FMErrors
		if errors.As(err, &fmErr) {
			if fmErr.ErrorCode() == fmclient.ObjectNotFound {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		resp.Diagnostics.AddError(
			"Unable to Get Traffic Map details",
			fmt.Sprintf("unable to get Traffic Map details. error is %v", err),
		)
		return
	}

	// Preserve typed ID from state; FM only knows raw UUID
	fmData.Id = data.Id

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(resp.State.Set(ctx, &fmData)...)
}

func (tm *TrafficMap) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData, stateData MapModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	if planData.Asf != nil && planData.Asf.AsfProfileConfig == nil {
		resp.Diagnostics.AddError("Invalid asf", "asf.asf_profile_config must be set when asf is provided")
		return
	}
	validateAsfMonitoringSessionPrereqs(ctx, tm.fmClient, planData.MonitoringSessionId, planData.Asf, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	applyAndValidateAsfSessionFields(planData.Asf, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	validateAppRulesRequireAsf(&planData, &resp.Diagnostics)
	validateAppRulesRuleIDRequired(&planData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	rawID, err := commonutils.UUIDFromTypedID(stateData.Id.ValueString())
	if err != nil {
		return
	}

	// Read existing map from FM so we can preserve macFilterList (VM selection)
	existingMap, err := GetMSMapData(
		ctx,
		stateData.MonitoringSessionId.ValueString(),
		rawID,
		stateData.Name.ValueString(),
		"trafficMap",
		tm.fmClient,
	)
	if err != nil {
		var fmErr *fmclient.FMErrors
		if errors.As(err, &fmErr) && fmErr.ErrorCode() == fmclient.ObjectNotFound {
			resp.Diagnostics.AddError(
				"Traffic map not found during update",
				fmt.Sprintf("Map %q (%s) no longer exists in monitoring session %q",
					stateData.Name.ValueString(),
					stateData.Id.ValueString(),
					stateData.MonitoringSessionId.ValueString(),
				),
			)
		} else {
			resp.Diagnostics.AddError(
				"Unable to read existing traffic map",
				fmt.Sprintf("Error fetching map from FM before update: %v", err),
			)
		}
		return
	}

	// Preserve current macFilterList so map update does not clear VM selection
	planData.MacFilterList = existingMap.MacFilterList

	trafficMap := ModelMapToGoMap(ctx, &planData)
	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{
			{
				EntityType: "trafficMap",
				Operation:  "update",
				Map:        trafficMap,
			},
		},
	}

	_, err = commonutils.UpdateMonSess(
		ctx,
		&updateReq,
		planData.MonitoringSessionId.ValueString(),
		tm.fmClient,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to update Map",
			fmt.Sprintf("map updation failed: %v", err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
}

func (tm *TrafficMap) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MapModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rawID, err := commonutils.UUIDFromTypedID(data.Id.ValueString())
	if err != nil {
		return
	}

	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{
			{
				EntityType: "trafficMap",
				Operation:  "delete",
				Map: MapGo{
					Id: rawID,
				},
			},
		},
	}

	_, err = commonutils.UpdateMonSess(
		ctx,
		&updateReq,
		data.MonitoringSessionId.ValueString(),
		tm.fmClient,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete the map",
			fmt.Sprintf("traffic map deletion failed: %v", err),
		)
	}
}

func (im *InclusionMap) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_inclusion_map" // → gigamon_inclusion_map
}

func (im *InclusionMap) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = MapSchema()
}

func (im *InclusionMap) ValidateConfig(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	if !req.Config.Raw.IsFullyKnown() {
		return
	}

	var cfg MapModel
	diags := req.Config.Get(ctx, &cfg)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateNilDropRules(cfg.RuleSets, &resp.Diagnostics)
}

func (im *InclusionMap) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	fmClient, ok := req.ProviderData.(*fmclient.FmClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *fmclient.FmClient, got: %T. Report the issue to Gigamon", req.ProviderData),
		)
		return
	}
	im.fmClient = fmClient
}

func (im *InclusionMap) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MapModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(data.RuleSets) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("rule_sets"),
			"rule_sets must be defined",
			"An inclusion map requires at least one rule_set.",
		)
		return
	}
	validateNilDropRules(data.RuleSets, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	rawID, err := CreateMSMap(ctx, MapKindInclusion, &data, im.fmClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create inclusion map",
			fmt.Sprintf("inclusion map creation failed: %v", err),
		)
		return
	}

	typedID, err := commonutils.MakeTypedID(
		commonutils.ModuleMap,
		commonutils.TypeInclusionMap,
		rawID,
	)
	if err != nil {
		return
	}
	data.Id = types.StringValue(typedID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (im *InclusionMap) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MapModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rawID, err := commonutils.UUIDFromTypedID(data.Id.ValueString())
	if err != nil {
		return
	}

	fmData, err := GetMSMapData(
		ctx,
		data.MonitoringSessionId.ValueString(),
		rawID,
		data.Name.ValueString(),
		"inclusionMaps",
		im.fmClient,
	)
	if err != nil {
		var fmErr *fmclient.FMErrors
		if errors.As(err, &fmErr) && fmErr.ErrorCode() == fmclient.ObjectNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Get Inclusion Map details",
			fmt.Sprintf("unable to get Inclusion Map details. error is %v", err),
		)
		return
	}

	// Preserve typed ID from state
	fmData.Id = data.Id

	resp.Diagnostics.Append(resp.State.Set(ctx, fmData)...)
}

func (im *InclusionMap) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData MapModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(planData.RuleSets) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("rule_sets"),
			"rule_sets must be defined",
			"An inclusion map requires at least one rule_set.",
		)
		return
	}
	validateNilDropRules(planData.RuleSets, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := UpdateMSMap(ctx, MapKindInclusion, &planData, im.fmClient); err != nil {
		resp.Diagnostics.AddError(
			"Unable to update inclusion map",
			fmt.Sprintf("inclusion map update failed: %v", err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
}

func (im *InclusionMap) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MapModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rawID, err := commonutils.UUIDFromTypedID(data.Id.ValueString())
	if err != nil {
		return
	}

	if err := DeleteMSMap(ctx, MapKindInclusion, data.MonitoringSessionId.ValueString(), rawID, im.fmClient); err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete inclusion map",
			fmt.Sprintf("inclusion map deletion failed: %v", err),
		)
	}
}

func (em *ExclusionMap) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_exclusion_map" // → gigamon_exclusion_map
}

func (em *ExclusionMap) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = MapSchema()
}

func (em *ExclusionMap) ValidateConfig(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	if !req.Config.Raw.IsFullyKnown() {
		return
	}

	var cfg MapModel
	diags := req.Config.Get(ctx, &cfg)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	validateNilPassRules(cfg.RuleSets, &resp.Diagnostics)
}

// validateNilDropRules checks that no rule_set contains drop_rules, which are
// forbidden on inclusion maps.
func validateNilDropRules(ruleSets []RuleSetModel, diags *diag.Diagnostics) {
	for i, rs := range ruleSets {
		if rs.DropRules != nil {
			diags.AddAttributeError(
				path.Root("rule_sets").AtListIndex(i).AtName("drop_rules"),
				"drop_rules not allowed on inclusion maps",
				"Inclusion maps only support pass_rules for Automatic Target Selection (ATS). "+
					"Remove drop_rules from this rule_set.",
			)
		}
	}
}

// validateNilPassRules checks that no rule_set contains pass_rules, which are
// forbidden on exclusion maps.
func validateNilPassRules(ruleSets []RuleSetModel, diags *diag.Diagnostics) {
	for i, rs := range ruleSets {
		if rs.PassRules != nil {
			diags.AddAttributeError(
				path.Root("rule_sets").AtListIndex(i).AtName("pass_rules"),
				"pass_rules not allowed on exclusion maps",
				"Exclusion maps only support drop_rules for Automatic Target Selection (ATS). "+
					"Remove pass_rules from this rule_set.",
			)
		}
	}
}

func validateAppRulesRequireAsf(data *MapModel, diags *diag.Diagnostics) {
	if data == nil {
		return
	}

	hasAsf := data.Asf != nil && data.Asf.AsfProfileConfig != nil
	if hasAsf {
		return
	}

	for i, rs := range data.RuleSets {
		if rs.AppRules == nil {
			continue
		}
		if len(rs.AppRules.PassRules) == 0 && len(rs.AppRules.DropRules) == 0 {
			continue
		}

		diags.AddAttributeError(
			path.Root("rule_sets").AtListIndex(i).AtName("app_rules"),
			"app_rules require ASF",
			"rule_sets.app_rules is supported only when asf.asf_profile_config is configured on the traffic map.",
		)
	}
}

func validateAppRulesRuleIDRequired(data *MapModel, diags *diag.Diagnostics) {
	if data == nil {
		return
	}

	for i, rs := range data.RuleSets {
		if rs.AppRules == nil {
			continue
		}

		for j, pr := range rs.AppRules.PassRules {
			if pr.RuleId.IsNull() || pr.RuleId.IsUnknown() {
				diags.AddAttributeError(
					path.Root("rule_sets").AtListIndex(i).AtName("app_rules").AtName("pass_rules").AtListIndex(j).AtName("rule_id"),
					"app_rules.pass_rules.rule_id is required",
					"rule_id (integer) must be configured for each app_rules.pass_rules entry.",
				)
			}
		}

		for j, dr := range rs.AppRules.DropRules {
			if dr.RuleId.IsNull() || dr.RuleId.IsUnknown() {
				diags.AddAttributeError(
					path.Root("rule_sets").AtListIndex(i).AtName("app_rules").AtName("drop_rules").AtListIndex(j).AtName("rule_id"),
					"app_rules.drop_rules.rule_id is required",
					"rule_id (integer) must be configured for each app_rules.drop_rules entry.",
				)
			}
		}
	}
}

func applyAndValidateAsfSessionFields(asf *AsfModel, diags *diag.Diagnostics) {
	if asf == nil || asf.AsfProfileConfig == nil {
		return
	}

	if len(asf.AsfProfileConfig.SessionFields) == 0 {
		asf.AsfProfileConfig.SessionFields = []AsfSessionFieldModel{
			{
				Pos:  types.Int32Value(2),
				Type: types.StringValue("fiveTuple"),
			},
		}
	}

	fields := asf.AsfProfileConfig.SessionFields
	if len(fields) < 1 || len(fields) > 2 {
		diags.AddAttributeError(
			path.Root("asf").AtName("asf_profile_config").AtName("session_fields"),
			"Invalid ASF session fields length",
			"asf.asf_profile_config.session_fields must contain between 1 and 2 entries.",
		)
		return
	}

	fiveTupleCount := 0
	vlanIdCount := 0

	for i, sf := range fields {
		if sf.Pos.IsNull() || sf.Pos.IsUnknown() || sf.Pos.ValueInt32() != 2 {
			diags.AddAttributeError(
				path.Root("asf").AtName("asf_profile_config").AtName("session_fields").AtListIndex(i).AtName("pos"),
				"Invalid ASF session field pos",
				"asf.asf_profile_config.session_fields.pos must be 2.",
			)
		}

		if sf.Type.IsNull() || sf.Type.IsUnknown() {
			diags.AddAttributeError(
				path.Root("asf").AtName("asf_profile_config").AtName("session_fields").AtListIndex(i).AtName("type"),
				"Invalid ASF session field type",
				"asf.asf_profile_config.session_fields.type must be fiveTuple or vlanId.",
			)
			continue
		}

		t := sf.Type.ValueString()
		if t != "fiveTuple" && t != "vlanId" {
			diags.AddAttributeError(
				path.Root("asf").AtName("asf_profile_config").AtName("session_fields").AtListIndex(i).AtName("type"),
				"Invalid ASF session field type",
				"asf.asf_profile_config.session_fields.type must be fiveTuple or vlanId.",
			)
			continue
		}

		if t == "fiveTuple" {
			fiveTupleCount++
		}
		if t == "vlanId" {
			vlanIdCount++
		}
	}

	if fiveTupleCount != 1 {
		diags.AddAttributeError(
			path.Root("asf").AtName("asf_profile_config").AtName("session_fields"),
			"Invalid ASF session fields configuration",
			"asf.asf_profile_config.session_fields must contain exactly one entry with type fiveTuple.",
		)
	}

	if vlanIdCount > 1 || len(fields) != fiveTupleCount+vlanIdCount {
		diags.AddAttributeError(
			path.Root("asf").AtName("asf_profile_config").AtName("session_fields"),
			"Invalid ASF session fields configuration",
			"asf.asf_profile_config.session_fields may include at most one optional vlanId entry in addition to fiveTuple.",
		)
	}
}

func validateAsfMonitoringSessionPrereqs(ctx context.Context, fmClient *fmclient.FmClient, monitoringSessionID types.String, asf *AsfModel, diags *diag.Diagnostics) {
	if fmClient == nil || asf == nil || asf.AsfProfileConfig == nil {
		return
	}

	if monitoringSessionID.IsNull() || monitoringSessionID.IsUnknown() || monitoringSessionID.ValueString() == "" {
		return
	}

	ms, err := getMonitoringSessionForMap(ctx, fmClient, monitoringSessionID.ValueString())
	if err != nil {
		diags.AddAttributeError(
			path.Root("asf"),
			"Invalid monitoring session for ASF",
			fmt.Sprintf("failed to read monitoring session %q: %v", monitoringSessionID.ValueString(), err),
		)
		return
	}

	if ms.ScaleUnit <= 0 {
		diags.AddAttributeError(
			path.Root("asf"),
			"Invalid monitoring session for ASF",
			"ASF requires scale_unit to already be configured on the selected monitoring session.",
		)
	}
}

func getMonitoringSessionForMap(ctx context.Context, fmClient *fmclient.FmClient, monitoringSessionID string) (*FMMonSess, error) {
	rawID, err := commonutils.UUIDFromTypedID(monitoringSessionID)
	if err != nil {
		return nil, err
	}

	respData, err := fmClient.DoRequest(
		ctx,
		"GET",
		fmt.Sprintf("api/v1.3/cloud/monitoringSessions/%s", rawID),
		nil,
		nil,
		nil,
		"",
	)
	if err != nil {
		return nil, err
	}

	var fmResp struct {
		MonitoringSession FMMonSess `json:"monitoringSession"`
	}
	if err := json.Unmarshal(respData, &fmResp); err != nil {
		return nil, fmt.Errorf("unable to convert monitoring session response to struct: %s error is: %w", string(respData), err)
	}

	return &fmResp.MonitoringSession, nil
}

func (em *ExclusionMap) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	fmClient, ok := req.ProviderData.(*fmclient.FmClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *fmclient.FmClient, got: %T. Report the issue to Gigamon", req.ProviderData),
		)
		return
	}
	em.fmClient = fmClient
}

func (em *ExclusionMap) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data MapModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(data.RuleSets) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("rule_sets"),
			"rule_sets must be defined",
			"An exclusion map requires at least one rule_set.",
		)
		return
	}
	validateNilPassRules(data.RuleSets, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	rawID, err := CreateMSMap(ctx, MapKindExclusion, &data, em.fmClient)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create exclusion map",
			fmt.Sprintf("exclusion map creation failed: %v", err),
		)
		return
	}

	typedID, err := commonutils.MakeTypedID(
		commonutils.ModuleMap,
		commonutils.TypeExclusionMap,
		rawID,
	)
	if err != nil {
		return
	}
	data.Id = types.StringValue(typedID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (em *ExclusionMap) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data MapModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rawID, err := commonutils.UUIDFromTypedID(data.Id.ValueString())
	if err != nil {
		return
	}

	fmData, err := GetMSMapData(
		ctx,
		data.MonitoringSessionId.ValueString(),
		rawID,
		data.Name.ValueString(),
		"exclusionMaps",
		em.fmClient,
	)
	if err != nil {
		var fmErr *fmclient.FMErrors
		if errors.As(err, &fmErr) && fmErr.ErrorCode() == fmclient.ObjectNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to Get Exclusion Map details",
			fmt.Sprintf("unable to get Exclusion Map details. error is %v", err),
		)
		return
	}

	// Preserve typed ID from state
	fmData.Id = data.Id

	resp.Diagnostics.Append(resp.State.Set(ctx, fmData)...)
}

func (em *ExclusionMap) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData MapModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(planData.RuleSets) == 0 {
		resp.Diagnostics.AddAttributeError(
			path.Root("rule_sets"),
			"rule_sets must be defined",
			"An exclusion map requires at least one rule_set.",
		)
		return
	}
	validateNilPassRules(planData.RuleSets, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := UpdateMSMap(ctx, MapKindExclusion, &planData, em.fmClient); err != nil {
		resp.Diagnostics.AddError(
			"Unable to update exclusion map",
			fmt.Sprintf("exclusion map update failed: %v", err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
}

func (em *ExclusionMap) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data MapModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rawID, err := commonutils.UUIDFromTypedID(data.Id.ValueString())
	if err != nil {
		return
	}

	if err := DeleteMSMap(ctx, MapKindExclusion, data.MonitoringSessionId.ValueString(), rawID, em.fmClient); err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete exclusion map",
			fmt.Sprintf("exclusion map deletion failed: %v", err),
		)
	}
}
