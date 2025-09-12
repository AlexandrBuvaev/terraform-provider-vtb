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
	_ datasource.DataSource = ScyllaDbClusterImageDataSource{}
)

type ScyllaDbClusterImageDataSource struct {
	provider *VTBCloudProvider
}

func NewScyllaDbClusterImageDataSource(p *VTBCloudProvider) ScyllaDbClusterImageDataSource {
	return ScyllaDbClusterImageDataSource{
		provider: p,
	}
}

func (d ScyllaDbClusterImageDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = "vtb_scylla_db_cluster_image_data"
}

func (t ScyllaDbClusterImageDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes:          dataImageStandardSchema,
		Description:         "Данные образа из сервиса справочников необходимые для оформления заказа.",
		MarkdownDescription: "Данные образа из сервиса справочников необходимые для оформления заказа.",
	}
}

func (d ScyllaDbClusterImageDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data ImageStandardModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var image *references.ScyllaDbClusterImageData
	var err error
	if strings.EqualFold(d.provider.EnvironmentName, "lt") {
		image, err = references.GetScyllaDbClusterImageData(
			d.provider.Creds,
			data.Distribution.ValueString(),
			d.provider.Organization,
			d.provider.EnvironmentName,
			data.OsVersion.ValueString(),
		)
	} else {
		image, err = references.GetScyllaDbClusterImageData(
			d.provider.Creds,
			data.Distribution.ValueString(),
			d.provider.Organization,
			d.provider.Environment,
			data.OsVersion.ValueString(),
		)
	}

	if err != nil {
		resp.Diagnostics.AddError("Get Image data from references", err.Error())
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

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
