package clusterlayout

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func (d *ClusterProductLayoutDataSource) Schema(
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
				Optional:            true,
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
