package gslbv1

import (
	"context"
	"fmt"
	"strings"
	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/consts"
	"terraform-provider-vtb/pkg/client/references"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource = &GSLBV1ImageDataSource{}
)

type GSLBV1ImageDataSource struct {
	client *client.CloudClient
}

func NewGSLBV1ImageDataSource() datasource.DataSource {
	return &GSLBV1ImageDataSource{}
}

func (d *GSLBV1ImageDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_gslb_v1_cluster_image_data"
}

func (d *GSLBV1ImageDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *GSLBV1ImageDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data GSLBV1ImageDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	image, err := references.GetGSLBV1ImageData(
		d.client.Creds,
		data.Distribution.ValueString(),
		d.client.Organization,
		d.client.Environment,
		data.OsVersion.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Get GSLB Image Data from portal ended with error.\nError: %s", err.Error()),
		)
		return
	}

	value, exists := image.Products[data.ProductVersion.ValueString()]

	if !exists {
		resp.Diagnostics.AddAttributeError(
			path.Root("product_version"),
			consts.READ_RES_FAIL,
			fmt.Errorf("not found %s in allowed to order `gslb_cluster_v1` versions", data.ProductVersion.ValueString()).Error(),
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
	data.ProductID = types.StringValue(value.Id)
	data.GeoDistribution = types.BoolValue(image.GeoDistribution)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
