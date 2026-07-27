// Copyright (c) Gigamon, Inc.
// Licensed under the Mozilla Public License v. 2.0

// Implements the 5G Apps SBI IP mappings upload resource for Gigamon Terraform Provider.

package commonresources

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-gigamon/internal/fmclient"
)

var sbiIPMappingsNameRegex = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

var sbiIPMappingsTypeEnum = []string{
	"ericssonVTap",
	"ericssonVTapFqdnTable",
	"domainTable",
	"cloud5gNfInstanceTable",
	"cloud5gFqdnTable",
	"cloud5gUaTable",
	"cloud5gServiceMapTable",
}

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &App5GSBIIpMappings{}
var _ resource.ResourceWithConfigure = &App5GSBIIpMappings{}
var _ resource.ResourceWithImportState = &App5GSBIIpMappings{}
var _ resource.ResourceWithModifyPlan = &App5GSBIIpMappings{}

// New5GSBIIpMappings creates a new resource instance for SBI IP mappings upload.
func New5GSBIIpMappings() resource.Resource {
	return &App5GSBIIpMappings{}
}

// App5GSBIIpMappings manages the 5G Apps SBI IP mappings upload resource.
type App5GSBIIpMappings struct {
	fmClient *fmclient.FmClient
}

// App5GSBIIpMappingsModel represents Terraform configuration and state.
type App5GSBIIpMappingsModel struct {
	Name            types.String `tfsdk:"name"`
	Type            types.String `tfsdk:"type"`
	AllowDuplicates types.Bool   `tfsdk:"allow_duplicates"`
	CsvPath         types.String `tfsdk:"csv_path"`
}

func (r *App5GSBIIpMappings) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_5g_apps_sbi_ip_mappings"
}

func (r *App5GSBIIpMappings) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages 5G Apps SBI IP mappings by uploading a CSV file to FM.",

		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "SBI IP mappings object name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.RegexMatches(sbiIPMappingsNameRegex, "only alphanumeric, '-' and '_' are allowed"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"type": schema.StringAttribute{
				MarkdownDescription: "SBI IP mappings type.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(sbiIPMappingsTypeEnum...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"allow_duplicates": schema.BoolAttribute{
				MarkdownDescription: "Allow duplicate records while importing mappings.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},

			"csv_path": schema.StringAttribute{
				MarkdownDescription: "Path to the CSV file to upload.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.RegexMatches(regexp.MustCompile(`(?i)^.*\.csv$`), "file must use .csv extension"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r *App5GSBIIpMappings) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan enforces immutable parameters after create.
// Any change to configured arguments must be done via delete + recreate.
func (r *App5GSBIIpMappings) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Delete operation
	if req.Plan.Raw.IsNull() {
		return
	}

	// Create operation
	if req.State.Raw.IsNull() {
		return
	}

	var stateData App5GSBIIpMappingsModel
	var planData App5GSBIIpMappingsModel

	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !planData.Name.IsUnknown() && !planData.Name.Equal(stateData.Name) {
		resp.Diagnostics.AddAttributeError(
			path.Root("name"),
			"Updating params is not allowed",
			"Changing 'name' is not supported for gigamon_5g_apps_sbi_ip_mappings. Delete and recreate the resource.",
		)
	}

	if !planData.Type.IsUnknown() && !planData.Type.Equal(stateData.Type) {
		resp.Diagnostics.AddAttributeError(
			path.Root("type"),
			"Updating params is not allowed",
			"Changing 'type' is not supported for gigamon_5g_apps_sbi_ip_mappings. Delete and recreate the resource.",
		)
	}

	if !planData.AllowDuplicates.IsUnknown() && !planData.AllowDuplicates.Equal(stateData.AllowDuplicates) {
		resp.Diagnostics.AddAttributeError(
			path.Root("allow_duplicates"),
			"Updating params is not allowed",
			"Changing 'allow_duplicates' is not supported for gigamon_5g_apps_sbi_ip_mappings. Delete and recreate the resource.",
		)
	}

	if !planData.CsvPath.IsUnknown() && !planData.CsvPath.Equal(stateData.CsvPath) {
		resp.Diagnostics.AddAttributeError(
			path.Root("csv_path"),
			"Updating params is not allowed",
			"Changing 'csv_path' is not supported for gigamon_5g_apps_sbi_ip_mappings. Delete and recreate the resource.",
		)
	}
}

func (r *App5GSBIIpMappings) endpoint(name, mappingType string) string {
	return fmt.Sprintf("api/v1.3/cloud/apps/config/sbiIpmappings/%s/%s", name, mappingType)
}

func (r *App5GSBIIpMappings) readFromFM(ctx context.Context, data *App5GSBIIpMappingsModel) error {
	query := map[string]string{
		"persist": "true",
	}

	_, err := r.fmClient.DoRequest(
		ctx,
		"GET",
		r.endpoint(data.Name.ValueString(), data.Type.ValueString()),
		query,
		nil,
		nil,
		"",
	)
	if err != nil {
		var fmErr *fmclient.FMErrors
		if errors.As(err, &fmErr) {
			return fmErr
		}
		return fmt.Errorf("failed to read SBI IP mappings name=%q type=%q: %w", data.Name.ValueString(), data.Type.ValueString(), err)
	}

	return nil
}

func isReadNotSupported(err error) bool {
	var fmErr *fmclient.FMErrors
	return errors.As(err, &fmErr) && fmErr.ErrorCode() == http.StatusMethodNotAllowed
}

func (r *App5GSBIIpMappings) uploadFile(ctx context.Context, name, mappingType, csvPath string, allowDuplicates bool) error {
	body, contentType, err := r.fmClient.PrepareFileUpload(
		ctx,
		csvPath,
		"file",
		nil,
	)
	if err != nil {
		return err
	}

	query := map[string]string{
		"persist":         "true",
		"allowDuplicates": fmt.Sprintf("%t", allowDuplicates),
	}

	tflog.Info(ctx, "Uploading SBI IP mappings CSV via multipart", map[string]any{
		"name":             name,
		"type":             mappingType,
		"allow_duplicates": allowDuplicates,
	})

	_, err = r.fmClient.DoRequest(
		ctx,
		"POST",
		r.endpoint(name, mappingType),
		query,
		nil,
		body,
		contentType,
	)
	if err != nil {
		return fmt.Errorf("SBI IP mappings multipart upload failed for name=%q type=%q: %w", name, mappingType, err)
	}

	return nil
}

func (r *App5GSBIIpMappings) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data App5GSBIIpMappingsModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Name.IsNull() || data.Name.IsUnknown() || strings.TrimSpace(data.Name.ValueString()) == "" {
		resp.Diagnostics.AddError("Missing name", "Cannot create SBI IP mappings because 'name' is null/unknown/empty.")
		return
	}

	if data.Type.IsNull() || data.Type.IsUnknown() || strings.TrimSpace(data.Type.ValueString()) == "" {
		resp.Diagnostics.AddError("Missing type", "Cannot create SBI IP mappings because 'type' is null/unknown/empty.")
		return
	}

	csvPath := ""
	if !data.CsvPath.IsNull() && !data.CsvPath.IsUnknown() {
		csvPath = strings.TrimSpace(data.CsvPath.ValueString())
	}
	if csvPath == "" {
		resp.Diagnostics.AddError(
			"Missing csv_path",
			"'csv_path' must be provided when creating/replacing the SBI IP mappings resource.",
		)
		return
	}
	if !strings.HasSuffix(strings.ToLower(csvPath), ".csv") {
		resp.Diagnostics.AddError("Invalid csv_path", "'csv_path' must point to a .csv file.")
		return
	}

	allowDuplicates := false
	if !data.AllowDuplicates.IsNull() && !data.AllowDuplicates.IsUnknown() {
		allowDuplicates = data.AllowDuplicates.ValueBool()
	}

	if err := r.uploadFile(ctx, data.Name.ValueString(), data.Type.ValueString(), csvPath, allowDuplicates); err != nil {
		resp.Diagnostics.AddError("Unable to upload SBI IP mappings CSV", err.Error())
		return
	}

	if err := r.readFromFM(ctx, &data); err != nil {
		if isReadNotSupported(err) {
			tflog.Warn(ctx, "GET for SBI IP mappings is not supported by FM; keeping create-time state without refresh", map[string]any{
				"name": data.Name.ValueString(),
				"type": data.Type.ValueString(),
			})
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to read SBI IP mappings after create",
			fmt.Sprintf("name=%q type=%q error: %v", data.Name.ValueString(), data.Type.ValueString(), err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *App5GSBIIpMappings) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data App5GSBIIpMappingsModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Name.IsNull() || data.Name.IsUnknown() || strings.TrimSpace(data.Name.ValueString()) == "" {
		resp.Diagnostics.AddError("Missing name", "Cannot read SBI IP mappings because 'name' is null/unknown/empty.")
		return
	}

	if data.Type.IsNull() || data.Type.IsUnknown() || strings.TrimSpace(data.Type.ValueString()) == "" {
		resp.Diagnostics.AddError("Missing type", "Cannot read SBI IP mappings because 'type' is null/unknown/empty.")
		return
	}

	err := r.readFromFM(ctx, &data)
	if err != nil {
		if isReadNotSupported(err) {
			tflog.Warn(ctx, "GET for SBI IP mappings is not supported by FM; keeping prior Terraform state", map[string]any{
				"name": data.Name.ValueString(),
				"type": data.Type.ValueString(),
			})
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		var fmErr *fmclient.FMErrors
		if errors.As(err, &fmErr) && fmErr.ErrorCode() == fmclient.ObjectNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Could not read SBI IP mappings from FM",
			fmt.Sprintf("name=%q type=%q error: %v", data.Name.ValueString(), data.Type.ValueString(), err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *App5GSBIIpMappings) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// csv_path and identity changes trigger replace; Update has nothing to do.
}

func (r *App5GSBIIpMappings) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data App5GSBIIpMappingsModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Name.IsNull() || data.Name.IsUnknown() || strings.TrimSpace(data.Name.ValueString()) == "" ||
		data.Type.IsNull() || data.Type.IsUnknown() || strings.TrimSpace(data.Type.ValueString()) == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	query := map[string]string{
		"persist": "true",
	}

	tflog.Info(ctx, "Deleting SBI IP mappings", map[string]any{
		"name": data.Name.ValueString(),
		"type": data.Type.ValueString(),
	})

	_, err := r.fmClient.DoRequest(
		ctx,
		"DELETE",
		r.endpoint(data.Name.ValueString(), data.Type.ValueString()),
		query,
		nil,
		nil,
		"",
	)
	if err != nil {
		var fmErr *fmclient.FMErrors
		if errors.As(err, &fmErr) && fmErr.ErrorCode() == fmclient.ObjectNotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			"Unable to delete SBI IP mappings",
			fmt.Sprintf("name=%q type=%q error: %v", data.Name.ValueString(), data.Type.ValueString(), err),
		)
		return
	}

	resp.State.RemoveResource(ctx)
}

func (r *App5GSBIIpMappings) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data App5GSBIIpMappingsModel

	parts := strings.Split(strings.TrimSpace(req.ID), "::")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import id",
			"Import id must be in the format '<name>::<type>' for SBI IP mappings.",
		)
		return
	}

	data.Name = types.StringValue(strings.TrimSpace(parts[0]))
	data.Type = types.StringValue(strings.TrimSpace(parts[1]))
	data.AllowDuplicates = types.BoolValue(false)
	data.CsvPath = types.StringNull()

	if data.Name.ValueString() == "" || data.Type.ValueString() == "" {
		resp.Diagnostics.AddError("Invalid import id", "Import id must contain non-empty '<name>::<type>' values.")
		return
	}

	err := r.readFromFM(ctx, &data)
	if err != nil {
		if isReadNotSupported(err) {
			resp.Diagnostics.AddWarning(
				"SBI IP mappings import completed without FM read",
				"FM does not support GET on this endpoint (HTTP 405). Import state is set from import id with default allow_duplicates=false and csv_path unknown.",
			)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to import SBI IP mappings from FM",
			fmt.Sprintf("Failed to import SBI IP mappings with name=%q type=%q: %v", data.Name.ValueString(), data.Type.ValueString(), err),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
