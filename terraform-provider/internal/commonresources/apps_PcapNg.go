// Copyright (c) Gigamon, Inc.
// Licensed under the Mozilla Public License v. 2.0

// Implements the PCapNG Application resource for Gigamon Terraform Provider

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-gigamon/internal/commonutils"
	"terraform-provider-gigamon/internal/fmclient"
)

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
	Id                  types.String `tfsdk:"id"`
	MonitoringSessionId types.String `tfsdk:"monitoring_session_id"`
	CaptureMode         types.String `tfsdk:"capture_mode"`
	PacketFilter        types.Object `tfsdk:"packet_filter"`
	OutputConfig        types.Object `tfsdk:"output_config"`
	Performance         types.Object `tfsdk:"performance"`
}

type PacketFilterModel struct {
	BPFSyntax  types.String `tfsdk:"bpf_syntax"`
	SourceIP   types.String `tfsdk:"source_ip"`
	DestIP     types.String `tfsdk:"dest_ip"`
	VLANFilter types.List   `tfsdk:"vlan_filter"`
}

type OutputConfigModel struct {
	FilePath    types.String `tfsdk:"file_path"`
	MaxFileSize types.Int32  `tfsdk:"max_file_size"`
	Rotation    types.Bool   `tfsdk:"rotation"`
	Compression types.String `tfsdk:"compression"`
}

type PerformanceModel2 struct {
	BufferSize    types.Int32 `tfsdk:"buffer_size"`
	ThreadCount   types.Int32 `tfsdk:"thread_count"`
	PacketSnaplen types.Int32 `tfsdk:"packet_snaplen"`
}

type FMPCapNG struct {
	AppType   string                 `json:"app_type"`
	AppConfig map[string]interface{} `json:"app_config"`
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
			"capture_mode": schema.StringAttribute{
				Description: "Capture mode: continuous, on-demand, or triggered",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("continuous", "on-demand", "triggered"),
				},
			},
			"packet_filter": schema.SingleNestedAttribute{
				Description: "Packet filter configuration",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"bpf_syntax": schema.StringAttribute{
						Description: "BPF filter syntax",
						Optional:    true,
					},
					"source_ip": schema.StringAttribute{
						Description: "Source IP filter",
						Optional:    true,
					},
					"dest_ip": schema.StringAttribute{
						Description: "Destination IP filter",
						Optional:    true,
					},
					"vlan_filter": schema.ListAttribute{
						Description: "VLAN IDs to filter",
						ElementType: types.Int32Type,
						Optional:    true,
					},
				},
			},
			"output_config": schema.SingleNestedAttribute{
				Description: "Output configuration",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"file_path": schema.StringAttribute{
						Description: "Output file path",
						Optional:    true,
					},
					"max_file_size": schema.Int32Attribute{
						Description: "Max file size in MB: 1-10000",
						Optional:    true,
						Computed:    true,
						Default:     int32default.StaticInt32(100),
						Validators: []validator.Int32{
							int32validator.Between(1, 10000),
						},
					},
					"rotation": schema.BoolAttribute{
						Description: "Enable file rotation",
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
					},
					"compression": schema.StringAttribute{
						Description: "Compression: none, gzip, or xz",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("none"),
						Validators: []validator.String{
							stringvalidator.OneOf("none", "gzip", "xz"),
						},
					},
				},
			},
			"performance": schema.SingleNestedAttribute{
				Description: "Performance configuration",
				Optional:    true,
				Attributes: map[string]schema.Attribute{
					"buffer_size": schema.Int32Attribute{
						Description: "Buffer size in MB: 4-1024",
						Optional:    true,
						Computed:    true,
						Default:     int32default.StaticInt32(64),
						Validators: []validator.Int32{
							int32validator.Between(4, 1024),
						},
					},
					"thread_count": schema.Int32Attribute{
						Description: "Thread count: 1-16",
						Optional:    true,
						Computed:    true,
						Default:     int32default.StaticInt32(4),
						Validators: []validator.Int32{
							int32validator.Between(1, 16),
						},
					},
					"packet_snaplen": schema.Int32Attribute{
						Description: "Packet snaplen: 64-65535",
						Optional:    true,
						Computed:    true,
						Default:     int32default.StaticInt32(65535),
						Validators: []validator.Int32{
							int32validator.Between(64, 65535),
						},
					},
				},
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
	sessionID := data.MonitoringSessionId.ValueString()
	payload := buildFMPCapNGPayload(ctx, data)
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
	err = GetMSAppData(ctx, sessionID, id, "PCapNG", "", &fmData, r.fmClient)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Failed to fetch created PCapNG app: %v", err))
	} else {
		data = mapFMPCapNGToState(ctx, fmData, sessionID, typedID)
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
	err = GetMSAppData(ctx, sessionID, rawID, "PCapNG", "", &fmData, r.fmClient)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading PCapNG app", err.Error())
		return
	}
	data = mapFMPCapNGToState(ctx, fmData, sessionID, typedID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AppPCapNG) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData AppPCapNGModel
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
	payload := buildFMPCapNGPayload(ctx, planData)
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
	err = GetMSAppData(ctx, sessionID, rawID, "PCapNG", "", &fmData, r.fmClient)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Failed to fetch updated PCapNG app: %v", err))
	} else {
		planData = mapFMPCapNGToState(ctx, fmData, sessionID, typedID)
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
	err := GetMSAppData(ctx, sessionID, rawID, "PCapNG", "", &fmData, r.fmClient)
	if err != nil {
		resp.Diagnostics.AddError("Error reading PCapNG app for import", err.Error())
		return
	}
	typedID, err := commonutils.MakeTypedID(commonutils.ModuleApp, commonutils.TypePCapNG, rawID)
	if err != nil {
		resp.Diagnostics.AddError("Error creating typed ID", err.Error())
		return
	}
	data := mapFMPCapNGToState(ctx, fmData, sessionID, typedID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func buildFMPCapNGPayload(ctx context.Context, model AppPCapNGModel) map[string]interface{} {
	appConfig := map[string]interface{}{
		"capture_mode": model.CaptureMode.ValueString(),
	}
	if !model.PacketFilter.IsNull() && !model.PacketFilter.IsUnknown() {
		var filterModel PacketFilterModel
		_ = model.PacketFilter.As(ctx, &filterModel, basetypes.ObjectAsOptions{})
		filterCfg := map[string]interface{}{}
		if !filterModel.BPFSyntax.IsNull() {
			filterCfg["bpf_syntax"] = filterModel.BPFSyntax.ValueString()
		}
		if !filterModel.SourceIP.IsNull() {
			filterCfg["source_ip"] = filterModel.SourceIP.ValueString()
		}
		if !filterModel.DestIP.IsNull() {
			filterCfg["dest_ip"] = filterModel.DestIP.ValueString()
		}
		if !filterModel.VLANFilter.IsNull() && !filterModel.VLANFilter.IsUnknown() {
			var vlans []int32
			filterModel.VLANFilter.ElementsAs(ctx, &vlans, false)
			filterCfg["vlan_filter"] = vlans
		}
		appConfig["packet_filter"] = filterCfg
	}
	if !model.OutputConfig.IsNull() && !model.OutputConfig.IsUnknown() {
		var outModel OutputConfigModel
		_ = model.OutputConfig.As(ctx, &outModel, basetypes.ObjectAsOptions{})
		appConfig["output_config"] = map[string]interface{}{
			"file_path":     outModel.FilePath.ValueString(),
			"max_file_size": outModel.MaxFileSize.ValueInt32(),
			"rotation":      outModel.Rotation.ValueBool(),
			"compression":   outModel.Compression.ValueString(),
		}
	}
	if !model.Performance.IsNull() && !model.Performance.IsUnknown() {
		var perfModel PerformanceModel2
		_ = model.Performance.As(ctx, &perfModel, basetypes.ObjectAsOptions{})
		appConfig["performance"] = map[string]interface{}{
			"buffer_size":    perfModel.BufferSize.ValueInt32(),
			"thread_count":   perfModel.ThreadCount.ValueInt32(),
			"packet_snaplen": perfModel.PacketSnaplen.ValueInt32(),
		}
	}
	return map[string]interface{}{
		"app_type":   "PCapNG",
		"app_config": appConfig,
	}
}

func mapFMPCapNGToState(ctx context.Context, fmData FMPCapNG, sessionID string, typedID string) AppPCapNGModel {
	model := AppPCapNGModel{
		Id:                  types.StringValue(typedID),
		MonitoringSessionId: types.StringValue(sessionID),
		CaptureMode:         types.StringValue("continuous"),
		PacketFilter: types.ObjectNull(map[string]attr.Type{
			"bpf_syntax": types.StringType,
			"source_ip":  types.StringType,
			"dest_ip":    types.StringType,
			"vlan_filter": types.ListType{
				ElemType: types.Int32Type,
			},
		}),
		OutputConfig: types.ObjectNull(map[string]attr.Type{
			"file_path":     types.StringType,
			"max_file_size": types.Int32Type,
			"rotation":      types.BoolType,
			"compression":   types.StringType,
		}),
		Performance: types.ObjectNull(map[string]attr.Type{
			"buffer_size":    types.Int32Type,
			"thread_count":   types.Int32Type,
			"packet_snaplen": types.Int32Type,
		}),
	}
	if fmData.AppConfig != nil {
		if captureMode, ok := fmData.AppConfig["capture_mode"].(string); ok {
			model.CaptureMode = types.StringValue(captureMode)
		}
		if filterCfg, ok := fmData.AppConfig["packet_filter"].(map[string]interface{}); ok {
			filterModel := PacketFilterModel{
				BPFSyntax: types.StringValue(""),
				SourceIP:  types.StringValue(""),
				DestIP:    types.StringValue(""),
				VLANFilter: types.ListNull(types.Int32Type),
			}
			if bpf, ok := filterCfg["bpf_syntax"].(string); ok {
				filterModel.BPFSyntax = types.StringValue(bpf)
			}
			if srcIP, ok := filterCfg["source_ip"].(string); ok {
				filterModel.SourceIP = types.StringValue(srcIP)
			}
			if dstIP, ok := filterCfg["dest_ip"].(string); ok {
				filterModel.DestIP = types.StringValue(dstIP)
			}
			if vlans, ok := filterCfg["vlan_filter"].([]interface{}); ok {
				var vlanInts []int32
				for _, v := range vlans {
					if vID, ok := v.(float64); ok {
						vlanInts = append(vlanInts, int32(vID))
					}
				}
				vlanObj, _ := types.ListValueFrom(ctx, types.Int32Type, vlanInts)
				filterModel.VLANFilter = vlanObj
			}
			filterObj, _ := types.ObjectValueFrom(ctx, map[string]attr.Type{
				"bpf_syntax":  types.StringType,
				"source_ip":   types.StringType,
				"dest_ip":     types.StringType,
				"vlan_filter": types.ListType{ElemType: types.Int32Type},
			}, filterModel)
			model.PacketFilter = filterObj
		}
		if outCfg, ok := fmData.AppConfig["output_config"].(map[string]interface{}); ok {
			outModel := OutputConfigModel{
				FilePath:    types.StringValue(""),
				MaxFileSize: types.Int32Value(100),
				Rotation:    types.BoolValue(false),
				Compression: types.StringValue("none"),
			}
			if filePath, ok := outCfg["file_path"].(string); ok {
				outModel.FilePath = types.StringValue(filePath)
			}
			if maxSize, ok := outCfg["max_file_size"].(float64); ok {
				outModel.MaxFileSize = types.Int32Value(int32(maxSize))
			}
			if rotation, ok := outCfg["rotation"].(bool); ok {
				outModel.Rotation = types.BoolValue(rotation)
			}
			if compression, ok := outCfg["compression"].(string); ok {
				outModel.Compression = types.StringValue(compression)
			}
			outObj, _ := types.ObjectValueFrom(ctx, map[string]attr.Type{
				"file_path":     types.StringType,
				"max_file_size": types.Int32Type,
				"rotation":      types.BoolType,
				"compression":   types.StringType,
			}, outModel)
			model.OutputConfig = outObj
		}
		if perfCfg, ok := fmData.AppConfig["performance"].(map[string]interface{}); ok {
			perfModel := PerformanceModel2{
				BufferSize:    types.Int32Value(64),
				ThreadCount:   types.Int32Value(4),
				PacketSnaplen: types.Int32Value(65535),
			}
			if bufSize, ok := perfCfg["buffer_size"].(float64); ok {
				perfModel.BufferSize = types.Int32Value(int32(bufSize))
			}
			if thrCount, ok := perfCfg["thread_count"].(float64); ok {
				perfModel.ThreadCount = types.Int32Value(int32(thrCount))
			}
			if snaplen, ok := perfCfg["packet_snaplen"].(float64); ok {
				perfModel.PacketSnaplen = types.Int32Value(int32(snaplen))
			}
			perfObj, _ := types.ObjectValueFrom(ctx, map[string]attr.Type{
				"buffer_size":    types.Int32Type,
				"thread_count":   types.Int32Type,
				"packet_snaplen": types.Int32Type,
			}, perfModel)
			model.Performance = perfObj
		}
	}
	return model
}
