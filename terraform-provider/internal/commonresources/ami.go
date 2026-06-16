// Copyright (c) Gigamon, Inc.

// Implements AMI application resource support.

package commonresources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int32validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int32default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-gigamon/internal/commonutils"
	"terraform-provider-gigamon/internal/fmclient"
)

var _ resource.Resource = &Ami{}

// NewAmi creates a new AMI application resource.
func NewAmi() resource.Resource {
	return &Ami{}
}

// Ami manages AMI (appmetadata) application instances in a monitoring session.
type Ami struct {
	fmClient *fmclient.FmClient
}

type AmiModel struct {
	MonitoringSessionId types.String         `tfsdk:"monitoring_session_id"`
	Alias               types.String         `tfsdk:"alias"`
	Description         types.String         `tfsdk:"description"`
	AppMetadata         *AmiAppMetadataModel `tfsdk:"app_metadata"`
	Id                  types.String         `tfsdk:"id"`
}

type AmiTimeoutModel struct {
	Idle types.Int32 `tfsdk:"idle"`
}

type AmiPrefixMatchModel struct {
	PrefixMinMask types.String `tfsdk:"prefix_min_mask"`
}

type AmiIPMatchModel struct {
	Destination *AmiPrefixMatchModel `tfsdk:"destination"`
	Protocol    types.Bool           `tfsdk:"protocol"`
	NextHeader  types.Bool           `tfsdk:"next_header"`
	Source      *AmiPrefixMatchModel `tfsdk:"source"`
}

type AmiTCPModel struct {
	Flags types.Bool `tfsdk:"flags"`
}

type AmiTransportMatchModel struct {
	DstPort types.Bool   `tfsdk:"dst_port"`
	SrcPort types.Bool   `tfsdk:"src_port"`
	TCP     *AmiTCPModel `tfsdk:"tcp"`
}

type AmiDatalinkMatchModel struct {
	Vlan types.Bool `tfsdk:"vlan"`
}

type AmiCounterModel struct {
	Bytes         types.Bool `tfsdk:"bytes"`
	BytesLong     types.Bool `tfsdk:"bytes_long"`
	Packets       types.Bool `tfsdk:"packets"`
	PacketsLong   types.Bool `tfsdk:"packets_long"`
	InnerByte     types.Bool `tfsdk:"inner_byte"`
	InnerByteLong types.Bool `tfsdk:"inner_byte_long"`
}

type AmiMatchModel struct {
	Ipv4      *AmiIPMatchModel        `tfsdk:"ipv4"`
	Ipv6      *AmiIPMatchModel        `tfsdk:"ipv6"`
	Transport *AmiTransportMatchModel `tfsdk:"transport"`
	Datalink  *AmiDatalinkMatchModel  `tfsdk:"datalink"`
}

type AmiAttributeModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

type AmiApplicationModel struct {
	Name       types.String        `tfsdk:"name"`
	Attributes []AmiAttributeModel `tfsdk:"attributes"`
}

type AmiAppProfileConfigModel struct {
	ApplicationID types.Bool              `tfsdk:"application_id"`
	FamilyID      types.Bool              `tfsdk:"family_id"`
	TagID         types.Bool              `tfsdk:"tag_id"`
	Applications  []AmiApplicationModel   `tfsdk:"applications"`
	Ipv4          *AmiIPMatchModel        `tfsdk:"ipv4"`
	Ipv6          *AmiIPMatchModel        `tfsdk:"ipv6"`
	Transport     *AmiTransportMatchModel `tfsdk:"transport"`
	Counter       *AmiCounterModel        `tfsdk:"counter"`
	Type          types.String            `tfsdk:"type"`
}

type AmiCefModel struct {
	ActiveTimeout   types.Int32  `tfsdk:"active_timeout"`
	InactiveTimeout types.Int32  `tfsdk:"inactive_timeout"`
	RecordType      types.String `tfsdk:"record_type"`
}

type AmiExporterConfigModel struct {
	Type             types.String               `tfsdk:"type"`
	MaxPktSize       types.Int32                `tfsdk:"max_pkt_size"`
	AppProfileConfig []AmiAppProfileConfigModel `tfsdk:"app_profile_config"`
	Cef              *AmiCefModel               `tfsdk:"cef"`
}

type AmiExporterModel struct {
	AepID          types.Int32             `tfsdk:"aep_id"`
	Name           types.String            `tfsdk:"name"`
	ExporterConfig *AmiExporterConfigModel `tfsdk:"exporter_config"`
}

type AmiPersistProfileConfigModel struct {
	Alias        types.String          `tfsdk:"alias"`
	Applications []AmiApplicationModel `tfsdk:"applications"`
	Type         types.String          `tfsdk:"type"`
}

type AmiAppMetadataModel struct {
	FlowBehavior         types.String                  `tfsdk:"flow_behavior"`
	Timeout              *AmiTimeoutModel              `tfsdk:"timeout"`
	MultiCollect         types.Bool                    `tfsdk:"multi_collect"`
	AggregateMode        types.Bool                    `tfsdk:"aggregate_mode"`
	ObservDomainId       types.Int32                   `tfsdk:"observ_domain_id"`
	Match                *AmiMatchModel                `tfsdk:"match"`
	DpiInjectLimit       types.Int32                   `tfsdk:"dpi_inject_limit"`
	Exporters            []AmiExporterModel            `tfsdk:"exporters"`
	PersistProfileConfig *AmiPersistProfileConfigModel `tfsdk:"persist_profile_config"`
}

type fmAmiTimeout struct {
	Idle *int32 `json:"idle,omitempty"`
}

type fmAmiPrefixMatch struct {
	PrefixMinMask string `json:"prefixMinMask,omitempty"`
}

type fmAmiIPMatch struct {
	Destination *fmAmiPrefixMatch `json:"destination,omitempty"`
	Protocol    *bool             `json:"protocol,omitempty"`
	NextHeader  *bool             `json:"nextHeader,omitempty"`
	Source      *fmAmiPrefixMatch `json:"source,omitempty"`
}

type fmAmiTCP struct {
	Flags bool `json:"flags"`
}

type fmAmiTransportMatch struct {
	DstPort bool      `json:"dstPort"`
	SrcPort bool      `json:"srcPort"`
	TCP     *fmAmiTCP `json:"tcp,omitempty"`
}

type fmAmiDatalinkMatch struct {
	Vlan bool `json:"vlan"`
}

type fmAmiCounter struct {
	Bytes         bool `json:"bytes"`
	BytesLong     bool `json:"bytesLong"`
	Packets       bool `json:"packets"`
	PacketsLong   bool `json:"packetsLong"`
	InnerByte     bool `json:"innerByte"`
	InnerByteLong bool `json:"innerByteLong"`
}

type fmAmiMatch struct {
	Ipv4      *fmAmiIPMatch        `json:"ipv4,omitempty"`
	Ipv6      *fmAmiIPMatch        `json:"ipv6,omitempty"`
	Transport *fmAmiTransportMatch `json:"transport,omitempty"`
	Datalink  *fmAmiDatalinkMatch  `json:"datalink,omitempty"`
}

type fmAmiAttribute struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type fmAmiApplication struct {
	Name       string           `json:"name"`
	Attributes []fmAmiAttribute `json:"attributes,omitempty"`
}

type fmAmiAppProfileConfig struct {
	ApplicationID bool                 `json:"applicationId"`
	FamilyID      bool                 `json:"familyId"`
	TagID         bool                 `json:"tagId"`
	Applications  []fmAmiApplication   `json:"applications,omitempty"`
	Ipv4          *fmAmiIPMatch        `json:"ipv4,omitempty"`
	Ipv6          *fmAmiIPMatch        `json:"ipv6,omitempty"`
	Transport     *fmAmiTransportMatch `json:"transport,omitempty"`
	Counter       *fmAmiCounter        `json:"counter,omitempty"`
	Type          string               `json:"type,omitempty"`
}

type fmAmiCef struct {
	ActiveTimeout   int32  `json:"activeTimeout,omitempty"`
	InactiveTimeout int32  `json:"inactiveTimeout,omitempty"`
	RecordType      string `json:"recordType,omitempty"`
}

type fmAmiExporterConfig struct {
	Type             string                  `json:"type,omitempty"`
	MaxPktSize       int32                   `json:"maxPktSize,omitempty"`
	AppProfileConfig []fmAmiAppProfileConfig `json:"appProfileConfig,omitempty"`
	Cef              *fmAmiCef               `json:"cef,omitempty"`
}

type fmAmiExporter struct {
	AepID          int32                `json:"aepid"`
	Name           string               `json:"name,omitempty"`
	ExporterConfig *fmAmiExporterConfig `json:"exporterConfig,omitempty"`
}

type fmAmiPersistProfileConfig struct {
	Alias        string             `json:"alias,omitempty"`
	Applications []fmAmiApplication `json:"applications,omitempty"`
	Type         string             `json:"type,omitempty"`
}

type fmAmiAppMetadata struct {
	FlowBehavior         string                     `json:"flowBehavior,omitempty"`
	Timeout              *fmAmiTimeout              `json:"timeout,omitempty"`
	MultiCollect         *bool                      `json:"multiCollect,omitempty"`
	AggregateMode        *bool                      `json:"aggregateMode,omitempty"`
	ObservDomainId       *int32                     `json:"observDomainId,omitempty"`
	Match                *fmAmiMatch                `json:"match,omitempty"`
	DpiInjectLimit       *int32                     `json:"dpiInjectLimit,omitempty"`
	Exporters            []fmAmiExporter            `json:"exporters,omitempty"`
	PersistProfileConfig *fmAmiPersistProfileConfig `json:"persistProfileConfig,omitempty"`
}

type FMAmi struct {
	Id          string `json:"id,omitempty"`
	Alias       string `json:"alias,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	AppMetadata any    `json:"appMetadata,omitempty"`
}

func (a *Ami) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_ami"
}

func (a *Ami) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Gigamon AMI application schema.",
		Attributes: map[string]schema.Attribute{
			"monitoring_session_id": schema.StringAttribute{
				MarkdownDescription: "Monitoring Session ID on which this AMI application is created.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"alias": schema.StringAttribute{
				MarkdownDescription: "Alias for the AMI application.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description for the AMI application.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"app_metadata": schema.SingleNestedAttribute{
				MarkdownDescription: "Typed AMI appMetadata configuration.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"flow_behavior": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("bidir")},
					"timeout": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"idle": schema.Int32Attribute{Optional: true, Computed: true, Default: int32default.StaticInt32(300), Validators: []validator.Int32{int32validator.AtLeast(1)}},
						},
					},
					"multi_collect":    schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
					"aggregate_mode":   schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
					"observ_domain_id": schema.Int32Attribute{Optional: true, Computed: true, Default: int32default.StaticInt32(0), Validators: []validator.Int32{int32validator.AtLeast(0)}},
					"match": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"ipv4":      amiIPMatchSchema(),
							"ipv6":      amiIPv6MatchSchema(),
							"transport": amiTransportSchema(),
							"datalink":  schema.SingleNestedAttribute{Optional: true, Attributes: map[string]schema.Attribute{"vlan": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)}}},
						},
					},
					"dpi_inject_limit":       schema.Int32Attribute{Optional: true, Computed: true, Default: int32default.StaticInt32(30), Validators: []validator.Int32{int32validator.AtLeast(0)}},
					"exporters":              schema.ListNestedAttribute{Optional: true, NestedObject: amiExporterNestedObject()},
					"persist_profile_config": amiPersistProfileConfigSchema(),
				},
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Typed ID of this AMI app instance.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func amiPrefixSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"prefix_min_mask": schema.StringAttribute{Optional: true},
		},
	}
}

func amiIPMatchSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"destination": amiPrefixSchema(),
			"protocol":    schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
			"source":      amiPrefixSchema(),
		},
	}
}

func amiIPv6MatchSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"destination": amiPrefixSchema(),
			"next_header": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
			"source":      amiPrefixSchema(),
		},
	}
}

func amiTransportSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"dst_port": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
			"src_port": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
			"tcp": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"flags": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
				},
			},
		},
	}
}

func amiCounterSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"bytes":           schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
			"bytes_long":      schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
			"packets":         schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
			"packets_long":    schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
			"inner_byte":      schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
			"inner_byte_long": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
		},
	}
}

func amiAttributeNestedObject() schema.NestedAttributeObject {
	return schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
		"name":  schema.StringAttribute{Required: true},
		"value": schema.StringAttribute{Required: true},
	}}
}

func amiApplicationNestedObject() schema.NestedAttributeObject {
	return schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
		"name": schema.StringAttribute{Required: true},
		"attributes": schema.ListNestedAttribute{
			Optional:     true,
			NestedObject: amiAttributeNestedObject(),
		},
	}}
}

func amiAppProfileConfigNestedObject() schema.NestedAttributeObject {
	return schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
		"application_id": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
		"family_id":      schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
		"tag_id":         schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
		"applications": schema.ListNestedAttribute{
			Optional:     true,
			NestedObject: amiApplicationNestedObject(),
		},
		"ipv4":      amiIPMatchSchema(),
		"ipv6":      amiIPv6MatchSchema(),
		"transport": amiTransportSchema(),
		"counter":   amiCounterSchema(),
		"type":      schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("export")},
	}}
}

func amiExporterNestedObject() schema.NestedAttributeObject {
	return schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
		"aep_id": schema.Int32Attribute{Required: true, Validators: []validator.Int32{int32validator.AtLeast(2), int32validator.AtMost(63)}},
		"name":   schema.StringAttribute{Required: true, Validators: []validator.String{stringvalidator.LengthAtLeast(1)}},
		"exporter_config": schema.SingleNestedAttribute{
			Required: true,
			Attributes: map[string]schema.Attribute{
				"type":               schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("cef")},
				"max_pkt_size":       schema.Int32Attribute{Optional: true, Computed: true, Default: int32default.StaticInt32(0), Validators: []validator.Int32{int32validator.AtLeast(0)}},
				"app_profile_config": schema.ListNestedAttribute{Optional: true, NestedObject: amiAppProfileConfigNestedObject()},
				"cef": schema.SingleNestedAttribute{
					Optional: true,
					Attributes: map[string]schema.Attribute{
						"active_timeout":   schema.Int32Attribute{Optional: true, Computed: true, Default: int32default.StaticInt32(60), Validators: []validator.Int32{int32validator.AtLeast(1)}},
						"inactive_timeout": schema.Int32Attribute{Optional: true, Computed: true, Default: int32default.StaticInt32(15), Validators: []validator.Int32{int32validator.AtLeast(1)}},
						"record_type":      schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("segregated")},
					},
				},
			},
		},
	}}
}

func amiPersistProfileConfigSchema() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"alias": schema.StringAttribute{Required: true},
			"applications": schema.ListNestedAttribute{
				Optional:     true,
				NestedObject: amiApplicationNestedObject(),
			},
			"type": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString("persist")},
		},
	}
}

func (a *Ami) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	a.fmClient = fmClient
}

func buildFMAmiIPMatch(m *AmiIPMatchModel, isIPv6 bool) *fmAmiIPMatch {
	if m == nil {
		return nil
	}
	fmMatch := &fmAmiIPMatch{}
	if m.Destination != nil {
		fmMatch.Destination = &fmAmiPrefixMatch{PrefixMinMask: m.Destination.PrefixMinMask.ValueString()}
	}
	if !m.Protocol.IsNull() && !m.Protocol.IsUnknown() {
		v := m.Protocol.ValueBool()
		fmMatch.Protocol = &v
	}
	if isIPv6 && !m.NextHeader.IsNull() && !m.NextHeader.IsUnknown() {
		v := m.NextHeader.ValueBool()
		fmMatch.NextHeader = &v
	}
	if m.Source != nil {
		fmMatch.Source = &fmAmiPrefixMatch{PrefixMinMask: m.Source.PrefixMinMask.ValueString()}
	}
	return fmMatch
}

func fmAmiApplicationsFromModel(modelApps []AmiApplicationModel) []fmAmiApplication {
	apps := make([]fmAmiApplication, 0, len(modelApps))
	for _, app := range modelApps {
		fmApp := fmAmiApplication{Name: app.Name.ValueString()}
		for _, attr := range app.Attributes {
			fmApp.Attributes = append(fmApp.Attributes, fmAmiAttribute{Name: attr.Name.ValueString(), Value: attr.Value.ValueString()})
		}
		apps = append(apps, fmApp)
	}
	return apps
}

func modelApplicationsFromFm(fmApps []fmAmiApplication) []AmiApplicationModel {
	apps := make([]AmiApplicationModel, 0, len(fmApps))
	for _, app := range fmApps {
		modelApp := AmiApplicationModel{Name: types.StringValue(app.Name), Attributes: make([]AmiAttributeModel, 0, len(app.Attributes))}
		for _, attr := range app.Attributes {
			modelApp.Attributes = append(modelApp.Attributes, AmiAttributeModel{Name: types.StringValue(attr.Name), Value: types.StringValue(attr.Value)})
		}
		apps = append(apps, modelApp)
	}
	return apps
}

func appMetadataModelToAny(model *AmiAppMetadataModel) (any, error) {
	if model == nil {
		return nil, nil
	}

	fmModel := fmAmiAppMetadata{}

	if !model.FlowBehavior.IsNull() && !model.FlowBehavior.IsUnknown() {
		fmModel.FlowBehavior = model.FlowBehavior.ValueString()
	}
	if model.Timeout != nil && !model.Timeout.Idle.IsNull() && !model.Timeout.Idle.IsUnknown() {
		idle := model.Timeout.Idle.ValueInt32()
		fmModel.Timeout = &fmAmiTimeout{Idle: &idle}
	}
	if !model.MultiCollect.IsNull() && !model.MultiCollect.IsUnknown() {
		v := model.MultiCollect.ValueBool()
		fmModel.MultiCollect = &v
	}
	if !model.AggregateMode.IsNull() && !model.AggregateMode.IsUnknown() {
		v := model.AggregateMode.ValueBool()
		fmModel.AggregateMode = &v
	}
	if !model.ObservDomainId.IsNull() && !model.ObservDomainId.IsUnknown() {
		v := model.ObservDomainId.ValueInt32()
		fmModel.ObservDomainId = &v
	}
	if !model.DpiInjectLimit.IsNull() && !model.DpiInjectLimit.IsUnknown() {
		v := model.DpiInjectLimit.ValueInt32()
		fmModel.DpiInjectLimit = &v
	}

	if model.Match != nil {
		fmModel.Match = &fmAmiMatch{
			Ipv4: buildFMAmiIPMatch(model.Match.Ipv4, false),
			Ipv6: buildFMAmiIPMatch(model.Match.Ipv6, true),
		}
		if model.Match.Transport != nil {
			fmModel.Match.Transport = &fmAmiTransportMatch{}
			if !model.Match.Transport.DstPort.IsNull() && !model.Match.Transport.DstPort.IsUnknown() {
				fmModel.Match.Transport.DstPort = model.Match.Transport.DstPort.ValueBool()
			}
			if !model.Match.Transport.SrcPort.IsNull() && !model.Match.Transport.SrcPort.IsUnknown() {
				fmModel.Match.Transport.SrcPort = model.Match.Transport.SrcPort.ValueBool()
			}
			if model.Match.Transport.TCP != nil {
				fmModel.Match.Transport.TCP = &fmAmiTCP{}
				if !model.Match.Transport.TCP.Flags.IsNull() && !model.Match.Transport.TCP.Flags.IsUnknown() {
					fmModel.Match.Transport.TCP.Flags = model.Match.Transport.TCP.Flags.ValueBool()
				}
			}
		}
		if model.Match.Datalink != nil {
			fmModel.Match.Datalink = &fmAmiDatalinkMatch{}
			if !model.Match.Datalink.Vlan.IsNull() && !model.Match.Datalink.Vlan.IsUnknown() {
				fmModel.Match.Datalink.Vlan = model.Match.Datalink.Vlan.ValueBool()
			}
		}
	}

	if len(model.Exporters) > 0 {
		fmModel.Exporters = make([]fmAmiExporter, 0, len(model.Exporters))
		for _, exporter := range model.Exporters {
			fmExporter := fmAmiExporter{AepID: exporter.AepID.ValueInt32(), Name: exporter.Name.ValueString()}
			if exporter.ExporterConfig != nil {
				cfg := &fmAmiExporterConfig{Type: exporter.ExporterConfig.Type.ValueString(), MaxPktSize: exporter.ExporterConfig.MaxPktSize.ValueInt32()}
				for _, apc := range exporter.ExporterConfig.AppProfileConfig {
					fmAPC := fmAmiAppProfileConfig{
						ApplicationID: apc.ApplicationID.ValueBool(),
						FamilyID:      apc.FamilyID.ValueBool(),
						TagID:         apc.TagID.ValueBool(),
						Applications:  fmAmiApplicationsFromModel(apc.Applications),
						Ipv4:          buildFMAmiIPMatch(apc.Ipv4, false),
						Ipv6:          buildFMAmiIPMatch(apc.Ipv6, true),
						Type:          apc.Type.ValueString(),
					}
					if apc.Transport != nil {
						fmAPC.Transport = &fmAmiTransportMatch{DstPort: apc.Transport.DstPort.ValueBool(), SrcPort: apc.Transport.SrcPort.ValueBool()}
						if apc.Transport.TCP != nil {
							fmAPC.Transport.TCP = &fmAmiTCP{Flags: apc.Transport.TCP.Flags.ValueBool()}
						}
					}
					if apc.Counter != nil {
						fmAPC.Counter = &fmAmiCounter{
							Bytes:         apc.Counter.Bytes.ValueBool(),
							BytesLong:     apc.Counter.BytesLong.ValueBool(),
							Packets:       apc.Counter.Packets.ValueBool(),
							PacketsLong:   apc.Counter.PacketsLong.ValueBool(),
							InnerByte:     apc.Counter.InnerByte.ValueBool(),
							InnerByteLong: apc.Counter.InnerByteLong.ValueBool(),
						}
					}
					cfg.AppProfileConfig = append(cfg.AppProfileConfig, fmAPC)
				}
				if exporter.ExporterConfig.Cef != nil {
					cfg.Cef = &fmAmiCef{ActiveTimeout: exporter.ExporterConfig.Cef.ActiveTimeout.ValueInt32(), InactiveTimeout: exporter.ExporterConfig.Cef.InactiveTimeout.ValueInt32(), RecordType: exporter.ExporterConfig.Cef.RecordType.ValueString()}
				}
				fmExporter.ExporterConfig = cfg
			}
			fmModel.Exporters = append(fmModel.Exporters, fmExporter)
		}
	}

	if model.PersistProfileConfig != nil {
		fmModel.PersistProfileConfig = &fmAmiPersistProfileConfig{
			Alias:        model.PersistProfileConfig.Alias.ValueString(),
			Applications: fmAmiApplicationsFromModel(model.PersistProfileConfig.Applications),
			Type:         model.PersistProfileConfig.Type.ValueString(),
		}
	}

	return fmModel, nil
}

func anyToAmiIPMatchModel(fmMatch *fmAmiIPMatch) *AmiIPMatchModel {
	if fmMatch == nil {
		return nil
	}
	m := &AmiIPMatchModel{Protocol: types.BoolNull(), NextHeader: types.BoolNull()}
	if fmMatch.Destination != nil {
		m.Destination = &AmiPrefixMatchModel{PrefixMinMask: types.StringValue(fmMatch.Destination.PrefixMinMask)}
	}
	if fmMatch.Protocol != nil {
		m.Protocol = types.BoolValue(*fmMatch.Protocol)
	}
	if fmMatch.NextHeader != nil {
		m.NextHeader = types.BoolValue(*fmMatch.NextHeader)
	}
	if fmMatch.Source != nil {
		m.Source = &AmiPrefixMatchModel{PrefixMinMask: types.StringValue(fmMatch.Source.PrefixMinMask)}
	}
	return m
}

func anyToAppMetadataModel(raw any) *AmiAppMetadataModel {
	b, err := json.Marshal(raw)
	if err != nil {
		return nil
	}

	var fmModel fmAmiAppMetadata
	if err := json.Unmarshal(b, &fmModel); err != nil {
		return nil
	}

	model := &AmiAppMetadataModel{
		FlowBehavior:   types.StringNull(),
		MultiCollect:   types.BoolNull(),
		AggregateMode:  types.BoolNull(),
		ObservDomainId: types.Int32Null(),
		DpiInjectLimit: types.Int32Null(),
		Exporters:      make([]AmiExporterModel, 0),
	}

	if fmModel.FlowBehavior != "" {
		model.FlowBehavior = types.StringValue(fmModel.FlowBehavior)
	}
	if fmModel.Timeout != nil && fmModel.Timeout.Idle != nil {
		model.Timeout = &AmiTimeoutModel{Idle: types.Int32Value(*fmModel.Timeout.Idle)}
	}
	if fmModel.MultiCollect != nil {
		model.MultiCollect = types.BoolValue(*fmModel.MultiCollect)
	}
	if fmModel.AggregateMode != nil {
		model.AggregateMode = types.BoolValue(*fmModel.AggregateMode)
	}
	if fmModel.ObservDomainId != nil {
		model.ObservDomainId = types.Int32Value(*fmModel.ObservDomainId)
	}
	if fmModel.DpiInjectLimit != nil {
		model.DpiInjectLimit = types.Int32Value(*fmModel.DpiInjectLimit)
	}

	if fmModel.Match != nil {
		model.Match = &AmiMatchModel{
			Ipv4: anyToAmiIPMatchModel(fmModel.Match.Ipv4),
			Ipv6: anyToAmiIPMatchModel(fmModel.Match.Ipv6),
		}
		if fmModel.Match.Transport != nil {
			model.Match.Transport = &AmiTransportMatchModel{DstPort: types.BoolValue(fmModel.Match.Transport.DstPort), SrcPort: types.BoolValue(fmModel.Match.Transport.SrcPort)}
			if fmModel.Match.Transport.TCP != nil {
				model.Match.Transport.TCP = &AmiTCPModel{Flags: types.BoolValue(fmModel.Match.Transport.TCP.Flags)}
			}
		}
		if fmModel.Match.Datalink != nil {
			model.Match.Datalink = &AmiDatalinkMatchModel{Vlan: types.BoolValue(fmModel.Match.Datalink.Vlan)}
		}
	}

	for _, exp := range fmModel.Exporters {
		modelExporter := AmiExporterModel{AepID: types.Int32Value(exp.AepID), Name: types.StringValue(exp.Name)}
		if exp.ExporterConfig != nil {
			cfg := &AmiExporterConfigModel{Type: types.StringValue(exp.ExporterConfig.Type), MaxPktSize: types.Int32Value(exp.ExporterConfig.MaxPktSize), AppProfileConfig: make([]AmiAppProfileConfigModel, 0, len(exp.ExporterConfig.AppProfileConfig))}
			for _, apc := range exp.ExporterConfig.AppProfileConfig {
				modelAPC := AmiAppProfileConfigModel{
					ApplicationID: types.BoolValue(apc.ApplicationID),
					FamilyID:      types.BoolValue(apc.FamilyID),
					TagID:         types.BoolValue(apc.TagID),
					Applications:  modelApplicationsFromFm(apc.Applications),
					Ipv4:          anyToAmiIPMatchModel(apc.Ipv4),
					Ipv6:          anyToAmiIPMatchModel(apc.Ipv6),
					Type:          types.StringValue(apc.Type),
				}
				if apc.Transport != nil {
					modelAPC.Transport = &AmiTransportMatchModel{DstPort: types.BoolValue(apc.Transport.DstPort), SrcPort: types.BoolValue(apc.Transport.SrcPort)}
					if apc.Transport.TCP != nil {
						modelAPC.Transport.TCP = &AmiTCPModel{Flags: types.BoolValue(apc.Transport.TCP.Flags)}
					}
				}
				if apc.Counter != nil {
					modelAPC.Counter = &AmiCounterModel{
						Bytes:         types.BoolValue(apc.Counter.Bytes),
						BytesLong:     types.BoolValue(apc.Counter.BytesLong),
						Packets:       types.BoolValue(apc.Counter.Packets),
						PacketsLong:   types.BoolValue(apc.Counter.PacketsLong),
						InnerByte:     types.BoolValue(apc.Counter.InnerByte),
						InnerByteLong: types.BoolValue(apc.Counter.InnerByteLong),
					}
				}
				cfg.AppProfileConfig = append(cfg.AppProfileConfig, modelAPC)
			}
			if exp.ExporterConfig.Cef != nil {
				cfg.Cef = &AmiCefModel{
					ActiveTimeout:   types.Int32Value(exp.ExporterConfig.Cef.ActiveTimeout),
					InactiveTimeout: types.Int32Value(exp.ExporterConfig.Cef.InactiveTimeout),
					RecordType:      types.StringValue(exp.ExporterConfig.Cef.RecordType),
				}
			}
			modelExporter.ExporterConfig = cfg
		}
		model.Exporters = append(model.Exporters, modelExporter)
	}

	if fmModel.PersistProfileConfig != nil {
		model.PersistProfileConfig = &AmiPersistProfileConfigModel{
			Alias:        types.StringValue(fmModel.PersistProfileConfig.Alias),
			Applications: modelApplicationsFromFm(fmModel.PersistProfileConfig.Applications),
			Type:         types.StringValue(fmModel.PersistProfileConfig.Type),
		}
	}

	return model
}

func (a *Ami) buildFMAmi(data *AmiModel) (*FMAmi, error) {
	fmData := &FMAmi{
		Alias:       data.Alias.ValueString(),
		Name:        "appmetadata",
		Description: data.Description.ValueString(),
	}

	if data.AppMetadata != nil {
		parsed, err := appMetadataModelToAny(data.AppMetadata)
		if err != nil {
			return nil, err
		}
		fmData.AppMetadata = parsed
	}

	return fmData, nil
}

func (a *Ami) updateTFStruct(data *AmiModel, fmData *FMAmi) {
	data.Alias = types.StringValue(fmData.Alias)

	if fmData.Description != "" {
		data.Description = types.StringValue(fmData.Description)
	} else {
		data.Description = types.StringNull()
	}

	if fmData.AppMetadata != nil {
		data.AppMetadata = anyToAppMetadataModel(fmData.AppMetadata)
	} else {
		data.AppMetadata = nil
	}
}

func (a *Ami) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AmiModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fmData, err := a.buildFMAmi(&data)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create AMI app", err.Error())
		return
	}

	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{{
			EntityType:  "application",
			Operation:   "create",
			Application: fmData,
		}},
	}

	id, err := commonutils.UpdateMonSess(ctx, &updateReq, data.MonitoringSessionId.ValueString(), a.fmClient)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create AMI app", fmt.Sprintf("app creation failed: %s", err))
		return
	}

	typedID, err := commonutils.MakeTypedID(commonutils.ModuleApp, commonutils.TypeAmi, id)
	if err != nil {
		return
	}

	a.updateTFStruct(&data, fmData)
	data.Id = types.StringValue(typedID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (a *Ami) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AmiModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rawID, err := commonutils.UUIDFromTypedID(data.Id.ValueString())
	if err != nil {
		return
	}

	fmData := FMAmi{}
	err = GetMSAppData(
		ctx,
		data.MonitoringSessionId.ValueString(),
		rawID,
		"appmetadata",
		"",
		&fmData,
		a.fmClient,
	)
	if err != nil {
		var fmErr *fmclient.FMErrors
		if errors.As(err, &fmErr) && fmErr.ErrorCode() == fmclient.ObjectNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Unable to get AMI app details",
			fmt.Sprintf("unable to get AMI app details. error is %v", err),
		)
		return
	}

	a.updateTFStruct(&data, &fmData)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (a *Ami) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData AmiModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	fmData, err := a.buildFMAmi(&planData)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update AMI app", err.Error())
		return
	}

	rawID, err := commonutils.UUIDFromTypedID(planData.Id.ValueString())
	if err != nil {
		return
	}
	fmData.Id = rawID

	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{{
			EntityType:  "application",
			Operation:   "update",
			Application: fmData,
		}},
	}

	_, err = commonutils.UpdateMonSess(ctx, &updateReq, planData.MonitoringSessionId.ValueString(), a.fmClient)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update AMI app", fmt.Sprintf("app update failed: %s", err))
		return
	}

	a.updateTFStruct(&planData, fmData)
	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
}

func (a *Ami) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AmiModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rawID, err := commonutils.UUIDFromTypedID(data.Id.ValueString())
	if err != nil {
		return
	}

	updateReq := commonutils.UpdateReq{
		Requests: []commonutils.UpdateObject{{
			EntityType: "application",
			Operation:  "delete",
			Application: FMAmi{
				Id:   rawID,
				Name: "Application",
			},
		}},
	}

	_, err = commonutils.UpdateMonSess(ctx, &updateReq, data.MonitoringSessionId.ValueString(), a.fmClient)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete AMI app", fmt.Sprintf("app deletion failed: %s", err))
	}
}
