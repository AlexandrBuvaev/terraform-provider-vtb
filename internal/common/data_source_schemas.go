package common

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var DataImageWithGeoDistributionSchema = map[string]schema.Attribute{
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

var DataImageStandardSchema = map[string]schema.Attribute{
	"distribution": schema.StringAttribute{
		Required:            true,
		Description:         "Название диструбутива (RHEL or Astra).",
		MarkdownDescription: "Название диструбутива (RHEL or Astra).",
		Validators: []validator.String{
			stringvalidator.OneOf("rhel", "astra", "linux_preprom"),
		},
	},
	"os_version": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Версия образа дистрибутива",
	},
	"on_support": schema.BoolAttribute{
		Computed:            true,
		MarkdownDescription: "Флаг означает, что продукт будет поддерживаться группой поддержки",
	},
	"product_id": schema.StringAttribute{
		Computed:            true,
		MarkdownDescription: "Идентификатор продукта вычислительного экземпляра",
	},
	"ad_integration": schema.BoolAttribute{
		Computed:            true,
		MarkdownDescription: "Флаг для интеграции вычислительного экземпляра с Active Directory",
	},
}
