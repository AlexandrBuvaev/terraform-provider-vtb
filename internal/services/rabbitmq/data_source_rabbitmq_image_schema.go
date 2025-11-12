package rabbitmq

import (
	"context"
	"terraform-provider-vtb/internal/common"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

func (d *RabbitMQImageDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes:          common.DataImageWithGeoDistributionSchema,
		MarkdownDescription: "Type of rabbitmq instance",
	}
}
