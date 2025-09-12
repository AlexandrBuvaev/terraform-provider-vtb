package provider

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-vtb/pkg/client/references"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource = ClusterProductLayoutDataSource{}
)

type ClusterProductLayoutDataSource struct {
	provider *VTBCloudProvider
}

func NewClusterProductLayoutDataSource(p *VTBCloudProvider) ClusterProductLayoutDataSource {
	return ClusterProductLayoutDataSource{
		provider: p,
	}
}

func (d ClusterProductLayoutDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = "vtb_cluster_layout"
}

type ClusterLayoutModel struct {
	ID             types.String `tfsdk:"id"`
	Layout         types.String `tfsdk:"layout"`
	NetSegmentCode types.String `tfsdk:"net_segment"`
}

func (t ClusterProductLayoutDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Type of rabbitmq instance",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				Description:         "Layout ID from reference.",
				MarkdownDescription: "Layout ID from reference.",
			},
			"net_segment": schema.StringAttribute{
				Required:            true,
				Description:         "Network segment name of order.",
				MarkdownDescription: "Network segment name of order.",
			},
			"layout": schema.StringAttribute{
				Required:            true,
				Description:         "String represent cluster layout. Example: `kafka-2:zookeeper-2`.",
				MarkdownDescription: "String represent cluster layout. Example: `kafka-2:zookeeper-2`.",
			},
		},
	}
}

func (d ClusterProductLayoutDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var layout types.String
	req.Config.GetAttribute(ctx, path.Root("layout"), &layout)

	var netSegment types.String
	req.Config.GetAttribute(ctx, path.Root("net_segment"), &netSegment)

	product, err := selectClusterProductType(layout.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("layout"), READ_RES_FAIL, err.Error())
		return
	}
	var layoutID string
	layoutID, err = references.GetGeoDistributionLayoutID(
		d.provider.Creds,
		layout.ValueString(),
		product,
		d.provider.Organization,
		netSegment.ValueString(),
	)

	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("layout"),
			fmt.Sprintln(READ_RES_FAIL, "geo_distribution reference fetch error"),
			err.Error(),
		)
		return
	}

	data := ClusterLayoutModel{
		ID:     types.StringValue(layoutID),
		Layout: layout,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func selectClusterProductType(layout string) (string, error) {

	products := []string{
		"kafka",
		"debezium",
		"rabbitmq",
		"artemis",
		"balancer_v3",
		"balancer",
		"airflow", // in airflow layout is like one_dc:webserver-2:scheduler-2:worker-2
		"tarantool",
	}

	for _, product := range products {
		if product == "tarantool" {
			return "tarantool_v2", nil
		}
		if strings.Contains(layout, product) {
			return product, nil
		}
		if product == "airflow" && strings.Contains(layout, "worker") && strings.Contains(layout, "scheduler") {
			return product, nil
		}
	}

	return "", fmt.Errorf(
		"layout invalid: there is no suitable product for %s.\nAvailable cluster products: [%s]",
		layout, strings.Join(products, ", "),
	)
}
