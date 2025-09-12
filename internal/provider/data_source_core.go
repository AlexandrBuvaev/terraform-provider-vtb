package provider

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"terraform-provider-vtb/pkg/client/sources"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource = CoreDataSource{}
)

type CoreDataSource struct {
	provider *VTBCloudProvider
}

func NewCoreDataSource(p *VTBCloudProvider) CoreDataSource {
	return CoreDataSource{
		provider: p,
	}
}

func (d CoreDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = "vtb_core_data"
}

type CoreModel struct {
	Platform       types.String `tfsdk:"platform"`
	Domain         types.String `tfsdk:"domain"`
	NetSegmentCode types.String `tfsdk:"net_segment"`
	Zone           types.String `tfsdk:"zone"`
}

func (t CoreDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Основные параметры для виртуальных машин (ВМ) в заказе",
		Attributes: map[string]schema.Attribute{

			"platform": schema.StringAttribute{
				Required:            true,
				Description:         "Название платформы заказа Пример: OpenStack",
				MarkdownDescription: "Название платформы заказа Пример: OpenStack",
			},

			"domain": schema.StringAttribute{
				Required:            true,
				Description:         "Название доменной зоны для заказа. Пример: corp.dev.vtb",
				MarkdownDescription: "Название доменной зоны для заказа. Пример: corp.dev.vtb",
			},

			"net_segment": schema.StringAttribute{
				Required:            true,
				Description:         "Название сетевого сегмента для заказа. Пример: dev-srv-app",
				MarkdownDescription: "Название сетевого сегмента для заказа. Пример: dev-srv-app",
			},

			"zone": schema.StringAttribute{
				Required:            true,
				Description:         "Зона доступности, где будет размещен заказ. Пример: msk-north",
				MarkdownDescription: "Зона доступности, где будет размещен заказ. Пример: msk-north",
			},
		},
	}
}

func (d CoreDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data CoreModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	d.checkNetSegment(&data, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	d.checkAvailabilityZone(&data, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	d.checkDomain(&data, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	d.checkPlatform(&data, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (d CoreDataSource) checkNetSegment(data *CoreModel, resp *datasource.ReadResponse) {

	netSegments, err := sources.GetNetSegments(
		d.provider.Creds,
		d.provider.ProjectName,
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("net_segment"), READ_RES_FAIL, err.Error())
		return
	}

	var available bool
	var availableSegments []string
	for _, segment := range netSegments {
		if segment.Code == data.NetSegmentCode.ValueString() {
			available = true
			return
		}
		availableSegments = append(availableSegments, segment.Code)
	}

	if !available {
		resp.Diagnostics.AddAttributeError(
			path.Root("net_segment"),
			READ_RES_FAIL,
			fmt.Sprintf(
				"`net_segment=%s` not found.\nAvailable network segments is [%s]",
				data.NetSegmentCode.ValueString(),
				strings.Join(availableSegments, ", "),
			),
		)
	}
}

func (d CoreDataSource) checkAvailabilityZone(data *CoreModel, resp *datasource.ReadResponse) {

	zones, err := sources.GetAvailAbilityZones(
		d.provider.Creds,
		data.NetSegmentCode.ValueString(),
		d.provider.Organization,
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("zone"),
			fmt.Sprintln(READ_RES_FAIL, "can't get availability zones"),
			err.Error(),
		)
		return
	}

	var available bool
	var availableZones []string
	for _, zone := range zones {
		isZoneForEnvironment := slices.Contains(zone.EnvironmentTypes, d.provider.Environment)
		if isZoneForEnvironment && (zone.Code == data.Zone.ValueString()) {
			available = true
			break
		}
		availableZones = append(availableZones, zone.Code)
	}

	if !available {
		resp.Diagnostics.AddAttributeError(
			path.Root("zone"),
			"Availability zone not available",
			fmt.Sprintf(
				"you can't choose zone=`%s` for current service account environment=`%s`"+
					"\nAvailable zones: %v",
				data.Zone.ValueString(),
				d.provider.Environment,
				strings.Join(availableZones, ", "),
			),
		)
	}
}

func (d CoreDataSource) checkDomain(data *CoreModel, resp *datasource.ReadResponse) {

	domains, err := sources.GetDomains(
		d.provider.Creds,
		data.NetSegmentCode.ValueString(),
		d.provider.Organization,
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("domain"), READ_RES_FAIL, err.Error())
		return
	}

	var available bool
	var availableDomains []string
	for _, domain := range domains {
		if domain.Code == data.Domain.ValueString() {
			available = true
			break
		}
		availableDomains = append(availableDomains, domain.Code)
	}

	if !available {
		resp.Diagnostics.AddAttributeError(
			path.Root("domain"),
			READ_RES_FAIL,
			fmt.Sprintf(
				"You can't get domain with name '%s'.\nAvailable domains is [%s]",
				data.Domain.ValueString(),
				strings.Join(availableDomains, ", "),
			),
		)
	}
}

func (d CoreDataSource) checkPlatform(data *CoreModel, resp *datasource.ReadResponse) {

	platforms, err := sources.GetPlatforms(
		d.provider.Creds,
		data.NetSegmentCode.ValueString(),
		d.provider.Organization,
		data.Zone.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("platform"), READ_RES_FAIL, err.Error())
		return
	}

	var available bool
	var availablePlatforms []string
	for _, platform := range platforms {
		if platform.Code == data.Platform.ValueString() {
			available = true
			break
		}
		availablePlatforms = append(availablePlatforms, platform.Code)
	}

	if !available {
		resp.Diagnostics.AddAttributeError(
			path.Root("platform"),
			READ_RES_FAIL,
			fmt.Sprintf(
				"Platform '%s' for project '%s' in network segment '%s' not available. "+
					"\nAvailable platforms: %v",
				data.Platform.ValueString(),
				d.provider.ProjectName,
				data.NetSegmentCode.ValueString(),
				strings.Join(availablePlatforms, ", "),
			),
		)
	}
}
