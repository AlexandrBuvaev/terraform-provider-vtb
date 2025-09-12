package provider

import (
	"context"

	"terraform-provider-vtb/pkg/client/references"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource = FlavorDataSource{}
)

type FlavorDataSource struct {
	provider *VTBCloudProvider
}

func NewFlavorDataSource(p *VTBCloudProvider) FlavorDataSource {
	return FlavorDataSource{
		provider: p,
	}
}

func (d FlavorDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = "vtb_flavor_data"
}

type FlavorModel struct {
	Cores  types.Int64  `tfsdk:"cores"`
	Memory types.Int64  `tfsdk:"memory"`
	UUID   types.String `tfsdk:"uuid"`
	Name   types.String `tfsdk:"name"`
}

func (t FlavorDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "CPU/оперативная память для вычислительного экземпляра.",
		Attributes: map[string]schema.Attribute{

			"cores": schema.Int64Attribute{
				Required:            true,
				Description:         "Количество ядер CPU.",
				MarkdownDescription: "Количество ядер CPU.",
			},

			"memory": schema.Int64Attribute{
				Required:            true,
				Description:         "Количество оперативной памяти (указвается в ГБ).",
				MarkdownDescription: "Количество оперативной памяти (указвается в ГБ).",
			},

			"uuid": schema.StringAttribute{
				Computed:            true,
				Description:         "UUID на странице справочников в КП.",
				MarkdownDescription: "UUID на странице справочников в КП.",
			},

			"name": schema.StringAttribute{
				Computed:            true,
				Description:         "Кодовое название конфигурации(flavor).",
				MarkdownDescription: "Кодовое название конфигурации(flavor).",
			},
		},
	}
}

func (d FlavorDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data FlavorModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	flavor, err := references.GetFlavor(
		d.provider.Creds,
		data.Memory.ValueInt64(),
		data.Cores.ValueInt64(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Flavor fetch error", err.Error())
		return
	}

	data.Name = types.StringValue(flavor.Name)
	data.UUID = types.StringValue(flavor.ReferenceResponse.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
