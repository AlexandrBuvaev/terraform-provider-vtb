package provider

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-vtb/pkg/client/references"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource = ArtemisImageDataSource{}
)

type ArtemisImageDataSource struct {
	provider *VTBCloudProvider
}

func NewArtemisImageDataSource(p *VTBCloudProvider) ArtemisImageDataSource {
	return ArtemisImageDataSource{
		provider: p,
	}
}

func (d ArtemisImageDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = "vtb_artemis_image_data"
}

type ClusterProductImageDataSourceModel struct {
	Distribution  types.String `tfsdk:"distribution"`
	OsVersion     types.String `tfsdk:"os_version"`
	OnSupport     types.Bool   `tfsdk:"on_support"`
	ProductID     types.String `tfsdk:"product_id"`
	ADIntegration types.Bool   `tfsdk:"ad_integration"`

	GeoDistribution types.Bool `tfsdk:"geo_distribution"`
}

var dataImageWithGeoDistributionSchema = map[string]schema.Attribute{
	"distribution": schema.StringAttribute{
		Required:            true,
		Description:         "Название дистрибутива (RHEL or Astra).",
		MarkdownDescription: "Название дистрибутива (RHEL or Astra).",
	},
	"os_version": schema.StringAttribute{
		Required:            true,
		Description:         "Версия образа дистрибутива.",
		MarkdownDescription: "Версия образа дистрибутива.",
	},
	"on_support": schema.BoolAttribute{
		Computed:            true,
		Description:         "Флаг означает, что продукт будет поддерживаться группой поддержки.",
		MarkdownDescription: "Флаг означает, что продукт будет поддерживаться группой поддержки.",
	},
	"product_id": schema.StringAttribute{
		Computed:            true,
		Description:         "Product ID из сервиса product_catalog.",
		MarkdownDescription: "Product ID из сервиса product_catalog.",
	},
	"ad_integration": schema.BoolAttribute{
		Computed:            true,
		Description:         "Флаг для интеграции вычислительного экземпляра с Active Directory",
		MarkdownDescription: "Флаг для интеграции вычислительного экземпляра с Active Directory",
	},
	"geo_distribution": schema.BoolAttribute{
		Computed:            true,
		Description:         "Флаг для использования геораспределения узлов кластера",
		MarkdownDescription: "Флаг для использования геораспределения узлов кластера",
	},
}

func (t ArtemisImageDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes:          dataImageWithGeoDistributionSchema,
		Description:         "Данные образа из сервиса справочников необходимые для оформления заказа.",
		MarkdownDescription: "Данные образа из сервиса справочников необходимые для оформления заказа.",
	}
}

func (d ArtemisImageDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data ClusterProductImageDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	image, err := references.GetArtemisImageData(
		d.provider.Creds,
		data.Distribution.ValueString(),
		d.provider.Organization,
		d.provider.Environment,
		data.OsVersion.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Get defaults data from references", err.Error())
		return
	}

	if !strings.Contains(image.Version, data.OsVersion.ValueString()) {
		resp.Diagnostics.AddError(
			"Validate OS Version",
			fmt.Sprintf("Unknown OS version %s", data.OsVersion.ValueString()),
		)
		return
	}

	data.OsVersion = types.StringValue(image.Version)
	data.OnSupport = types.BoolValue(image.OnSupport)
	data.ProductID = types.StringValue(image.ProductID)
	data.ADIntegration = types.BoolValue(image.ADIntegration)
	data.GeoDistribution = types.BoolValue(image.GeoDistribution)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
