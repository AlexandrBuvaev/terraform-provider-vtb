package common

import (
	"terraform-provider-vtb/internal/consts"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var CoreSchema = map[string]schema.Attribute{
	"platform": schema.StringAttribute{
		MarkdownDescription: "Название платформы заказа Пример: OpenStack",
		Required:            true,
		Validators: []validator.String{
			stringvalidator.OneOf(consts.PLATFORMS...),
		},
	},
	"domain": schema.StringAttribute{
		MarkdownDescription: "Название сетевого сегмента для заказа. Пример: dev-srv-app",
		Required:            true,
		Validators: []validator.String{
			stringvalidator.OneOf(consts.DOMAINS...),
		},
	},
	"net_segment": schema.StringAttribute{
		MarkdownDescription: "Network segment name of order",
		Required:            true,
	},
	"zone": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Зона доступности, где будет размещен заказ. Пример: msk-north",
		Validators: []validator.String{
			stringvalidator.OneOf(consts.AVAILABILITY_ZONES...),
		},
	},
}

var FlavorSchema = map[string]schema.Attribute{
	"cores": schema.Int64Attribute{
		Required:            true,
		MarkdownDescription: "Количество ядер ЦПУ.",
	},
	"memory": schema.Int64Attribute{
		Required:            true,
		MarkdownDescription: "Количество оперативной памяти RAM (указывается в ГБ).",
	},
	"uuid": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Уникальный идентификатор на странице справочников.",
	},
	"name": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Кодовое название конфигурации.",
	},
}

var ImageStandardSchema = map[string]schema.Attribute{
	"distribution": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Название дистрибутива (RHEL or Astra)",
		Validators: []validator.String{
			stringvalidator.OneOf("rhel", "astra", "linux_preprom"),
		},
	},
	"os_version": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Версия образа дистрибутива",
	},
	"on_support": schema.BoolAttribute{
		Required:            true,
		MarkdownDescription: "Флаг означает, что продукт будет поддерживаться группой поддержки",
	},
	"product_id": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Идентификатор продукта вычислительного экземпляра",
	},
	"ad_integration": schema.BoolAttribute{
		Required:            true,
		MarkdownDescription: "Флаг для интеграции вычислительного экземпляра с Active Directory",
	},
}

var ClusterProductImageSchema = map[string]schema.Attribute{
	"geo_distribution": schema.BoolAttribute{
		Required:            true,
		MarkdownDescription: "Механизм для включения/выключения геораспределения.",
	},
	"distribution": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Название дистрибутива (RHEL или Astra).",
	},
	"os_version": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Версия образа дистрибутива.",
	},
	"on_support": schema.BoolAttribute{
		Required:            true,
		MarkdownDescription: "Флаг регулирущий режим работы группы сопровождения.",
	},
	"product_id": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Идентификатор продукта.",
	},
	"ad_integration": schema.BoolAttribute{
		Required:            true,
		MarkdownDescription: "Флаг для интеграции виртуальной машины в Active Directory",
	},
}
