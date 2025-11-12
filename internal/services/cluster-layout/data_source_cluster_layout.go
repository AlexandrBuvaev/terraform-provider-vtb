package clusterlayout

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
	_ datasource.DataSource                   = &ClusterProductLayoutDataSource{}
	_ datasource.DataSourceWithValidateConfig = &ClusterProductLayoutDataSource{}
)

type ClusterProductLayoutDataSource struct {
	client *client.CloudClient
}

func NewClusterProductLayoutDataSource() datasource.DataSource {
	return &ClusterProductLayoutDataSource{}
}

func (d *ClusterProductLayoutDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_cluster_layout"
}

func (d *ClusterProductLayoutDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ClusterProductLayoutDataSource) ValidateConfig(
	ctx context.Context,
	req datasource.ValidateConfigRequest,
	resp *datasource.ValidateConfigResponse,
) {
	if req.Config.Raw.IsNull() {
		return
	}

	var config ClusterLayoutModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)

	if config.Layout.IsNull() || config.Layout.IsUnknown() {
		return
	}

	if !strings.Contains(config.Layout.ValueString(), "gslb") && config.NetSegmentCode.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("net_segment"),
			consts.VALIDATION_FAIL,
			fmt.Sprintf("Field `net_segment` is required for this `layout`: %q", config.Layout.ValueString()),
		)
		return
	}
}

func (d *ClusterProductLayoutDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {

	var data ClusterLayoutModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	product, err := selectClusterProductType(data.Layout.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("layout"), consts.READ_RES_FAIL, err.Error())
		return
	}
	var layoutID string
	if product == "gslb" {
		layoutID, err = references.GetGeoDistributionLayoutID(
			d.client.Creds,
			data.Layout.ValueString(),
			product,
			d.client.Organization,
			d.client.Environment,
		)
	} else {
		layoutID, err = references.GetGeoDistributionLayoutID(
			d.client.Creds,
			data.Layout.ValueString(),
			product,
			d.client.Organization,
			data.NetSegmentCode.ValueString(),
		)
	}

	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("layout"),
			fmt.Sprintln(consts.READ_RES_FAIL, "geo_distribution reference fetch error"),
			err.Error(),
		)
		return
	}

	data.ID = types.StringValue(layoutID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
