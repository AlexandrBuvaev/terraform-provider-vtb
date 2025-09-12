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
	_ datasource.DataSource = DebeziumImageDataSource{}
)

type DebeziumImageDataSource struct {
	provider *VTBCloudProvider
}

func NewDebeziumImageDataSource(p *VTBCloudProvider) DebeziumImageDataSource {
	return DebeziumImageDataSource{
		provider: p,
	}
}

func (d DebeziumImageDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = "vtb_debezium_image_data"
}

func (t DebeziumImageDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes:          dataImageWithGeoDistributionSchema,
		Description:         "Image data from reference service needed to create order",
		MarkdownDescription: "Image data from reference service needed to create order",
	}
}

func (d DebeziumImageDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data ClusterProductImageDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	image, err := references.GetDebeziumImageData(
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
