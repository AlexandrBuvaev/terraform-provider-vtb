package gslbv1

import (
	"context"
	"regexp"
	"terraform-provider-vtb/internal/common"
	"terraform-provider-vtb/internal/custommodifires"
	"terraform-provider-vtb/internal/customvalidators"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *GSLBV1Resource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			// basic schema
			"core": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.CoreSchema,
				MarkdownDescription: "Основные параметры для вирутальных машин в заказе",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"image": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Образ вычислителього ресусра",
				Attributes:          GSLBV1ImageSchema,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"lifetime": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Время жизни заказа в днях (2, 7, 14, 30)",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
			},
			"label": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Метка заказа",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"order_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Идентификатор заказа, известен после создания заказа",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"item_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Идентификатор сущностей ВМ, известен после создания заказа",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"access": schema.MapAttribute{
				Required: true,
				MarkdownDescription: "Словарь, где ключом является роль, а значением список групп, " +
					"которые предоставляют доступ для входа в Active Directory",
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
				Validators: []validator.Map{
					customvalidators.EmptyAccessGroupListValidator{},
				},
			},
			"financial_project": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Источник финансирования",
			},
			"flavor": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.FlavorSchema,
				MarkdownDescription: "Кол-во CPU/RAM для вирутальных машин в заказе",
			},
			"extra_mounts": schema.MapNestedAttribute{
				Required:            true,
				MarkdownDescription: "Точки монтирования",
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.OneOf("/app")),
					mapvalidator.SizeBetween(1, 1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"size": schema.Int64Attribute{
							Required:            true,
							MarkdownDescription: "Размер точки монтирования в ГБ",
							Validators: []validator.Int64{
								int64validator.AtMost(2048),
								int64validator.AtLeast(10),
							},
							PlanModifiers: []planmodifier.Int64{
								custommodifires.GrowOnly(),
								custommodifires.MininalGrow(10),
							},
						},
						"file_system": schema.StringAttribute{
							Computed:            true,
							Default:             stringdefault.StaticString("xfs"),
							MarkdownDescription: "Тип файловой системы",
						},
					},
				},
			},
			"layout": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Идентификатор конфигурации из справочника `geo_distribution`",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"desired_version": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Версия инсталяции",
				Validators: []validator.String{
					stringvalidator.OneOf(AvailaiableGSLBV1InstallationVersions...),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"api_password": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
				MarkdownDescription: "API Password. " +
					"Должен быть длиной от 6 до 128 символов и содержать следующие значения: " +
					"1 буквенное значение в нижнем регистре, 1 буквенное значение в верхнем, 1 число, и 1 спецальный знак (!@#$%^&*)",
				Validators: []validator.String{
					stringvalidator.LengthBetween(6, 128),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`[a-z]`),
						"Must contain at least one lowercase letter",
					),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`[A-Z]`),
						"Must contain at least one uppercase letter",
					),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`[0-9]`),
						"Must contain at least one number",
					),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`[!@#$%^&*]`),
						"Must contain at least one special character (!@#$%^&*)",
					),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9!@#$%^&*]+$`),
						"Contains invalid characters. Only letters, numbers and !@#$%^&* are allowed",
					),
				},
			},
			"nginx_password": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
				MarkdownDescription: "Nginx Password. " +
					"Должен быть длиной от 6 до 128 символов и содержать следующие значения: " +
					"1 буквенное значение в нижнем регистре, 1 буквенное значение в верхнем, 1 число, и 1 спецальный знак (!@#$%^&*)",
				Validators: []validator.String{
					stringvalidator.LengthBetween(6, 128),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`[a-z]`),
						"Must contain at least one lowercase letter",
					),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`[A-Z]`),
						"Must contain at least one uppercase letter",
					),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`[0-9]`),
						"Must contain at least one number",
					),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`[!@#$%^&*]`),
						"Must contain at least one special character (!@#$%^&*)",
					),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9!@#$%^&*]+$`),
						"Contains invalid characters. Only letters, numbers and !@#$%^&* are allowed",
					),
				},
			},
			"dns_zone": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Префикс доменной зоны",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"bgpaas": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"gslb_hosts": schema.SetNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Hostname вирутальной машины",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"maintenance_mode": schema.StringAttribute{
							Computed:    true,
							Optional:    true,
							Description: "Статус Maintenance Mode",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
							Validators: []validator.String{
								stringvalidator.OneOf("on", "off"),
							},
						},
					},
				},
				Computed: true,
				Optional: true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "Хосты GSLB, с указанием статуса простановки в ММ",
			},
		},
	}
}

var GSLBV1ImageSchema = map[string]schema.Attribute{
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
		Required:            true,
		Description:         "Флаг означает, что продукт будет поддерживаться группой поддержки.",
		MarkdownDescription: "Флаг означает, что продукт будет поддерживаться группой поддержки.",
	},
	"product_id": schema.StringAttribute{
		Required:            true,
		Description:         "Product ID из сервиса product_catalog.",
		MarkdownDescription: "Product ID из сервиса product_catalog.",
	},
	"geo_distribution": schema.BoolAttribute{
		Required:            true,
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
