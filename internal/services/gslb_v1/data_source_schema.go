package gslbv1

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func (d *GSLBV1ImageDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes:          GSLBV1ImageDataSourceSchema,
		Description:         "Данные образа GSLB v1",
		MarkdownDescription: "Данные образа GSLB v1",
	}
}

var GSLBV1ImageDataSourceSchema = map[string]schema.Attribute{
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
	"geo_distribution": schema.BoolAttribute{
		Computed:            true,
		Description:         "Флаг для использования геораспределения узлов кластера",
		MarkdownDescription: "Флаг для использования геораспределения узлов кластера",
	},
	"product_version": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Версия продукта GSLB v1",
		Validators: []validator.String{
			stringvalidator.OneOf("gslb_cluster_v1_5"),
		},
	},
}
