package elasticsearch

import (
	"context"
	"fmt"
	"strings"
	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/common"
	"terraform-provider-vtb/internal/consts"
	"terraform-provider-vtb/pkg/client/references"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource = &ElasticSearchImageDataSource{}
)

type ElasticSearchImageDataSource struct {
	client *client.CloudClient
}

func NewElasticSearchImageDataSource() datasource.DataSource {
	return &ElasticSearchImageDataSource{}
}

func (d ElasticSearchImageDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_elasticsearch_image_data"
}

func (d *ElasticSearchImageDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d ElasticSearchImageDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes:          common.DataImageWithGeoDistributionSchema,
		Description:         "Данные образа из сервиса справочников для оформления заказа.",
		MarkdownDescription: "Данные образа из сервиса справочников для оформления заказа.",
	}
}

func (d ElasticSearchImageDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data common.ClusterProductImageDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	image, err := references.GetElasticSearchImageData(
		d.client.Creds,
		data.Distribution.ValueString(),
		d.client.Organization,
		d.client.Environment,
		data.OsVersion.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Get ElasticSearch Image Data from references ended with error.\nError: %v", err.Error()),
		)
		return
	}

	if !strings.Contains(image.Version, data.OsVersion.ValueString()) {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
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
