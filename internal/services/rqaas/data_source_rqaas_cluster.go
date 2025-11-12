package rqaas

import (
	"context"
	"fmt"
	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/consts"
	"terraform-provider-vtb/pkg/client/references"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &RQaaSClusterDataSource{}

type RQaaSClusterDataSource struct {
	client *client.CloudClient
}

func NewRQaaSClusterDataSource() datasource.DataSource {
	return &RQaaSClusterDataSource{}
}

func (d RQaaSClusterDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_rqaas_cluster_data"
}

func (d *RQaaSClusterDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.CloudClient)

	if !ok {
		resp.Diagnostics.AddError(
			"unexpected resource configure type",
			fmt.Sprintf("Expected *client.CloudClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = client
}

type RQaaSClusterDataSourceModel struct {
	Name       types.String `tfsdk:"name"`
	Hosts      types.List   `tfsdk:"hosts"`
	Platform   types.String `tfsdk:"platform"`
	Domain     types.String `tfsdk:"domain"`
	NetSegment types.String `tfsdk:"net_segment"`
	Zone       types.String `tfsdk:"zone"`
}

func (d RQaaSClusterDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Основные параметры кластера для RQaaS",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "Имя кластера",
				MarkdownDescription: "Имя кластера",
			},
			"hosts": schema.ListAttribute{
				Computed:            true,
				Description:         "Список хостов кластера",
				MarkdownDescription: "Список хостов кластера",
				ElementType:         types.StringType,
			},
			"platform": schema.StringAttribute{
				Computed:            true,
				Description:         "Название платформы",
				MarkdownDescription: "Название платформы",
			},
			"domain": schema.StringAttribute{
				Computed:            true,
				Description:         "Название доменной зоны",
				MarkdownDescription: "Название доменной зоны",
			},
			"net_segment": schema.StringAttribute{
				Computed:            true,
				Description:         "Название сетевого сегмента",
				MarkdownDescription: "Название сетевого сегмента",
			},
			"zone": schema.StringAttribute{
				Computed:            true,
				Description:         "Зона доступности",
				MarkdownDescription: "Зона доступности",
			},
		},
	}
}

func (d RQaaSClusterDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data RQaaSClusterDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rqaasCluster, err := references.GetRQaasCluster(
		d.client.Creds,
		d.client.Environment,
		data.Name.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("error for fetching rqaas cluster.\nError: %v", err.Error()),
		)
		return
	}

	data.Hosts, _ = types.ListValueFrom(ctx, types.StringType, rqaasCluster.Hosts)
	data.Domain = types.StringValue(rqaasCluster.Domain)
	data.NetSegment = types.StringValue(rqaasCluster.NetSegment)
	data.Platform = types.StringValue(rqaasCluster.Platform)
	data.Zone = types.StringValue(rqaasCluster.AvailabilityZone)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}
