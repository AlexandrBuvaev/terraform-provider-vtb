package kafka

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"time"

	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/common"
	"terraform-provider-vtb/internal/consts"
	"terraform-provider-vtb/internal/custommodifires"
	"terraform-provider-vtb/internal/customvalidators"
	"terraform-provider-vtb/internal/services/core"
	"terraform-provider-vtb/internal/services/flavor"
	"terraform-provider-vtb/internal/utils"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"
	"terraform-provider-vtb/pkg/client/references"
	"terraform-provider-vtb/pkg/client/sources"

	"github.com/Masterminds/semver"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	CREATE_ACLS_ATTEMPTS = 5
	CREATE_ACLS_DELAY    = 30
)

var (
	_ resource.Resource                = KafkaResource{}
	_ resource.ResourceWithImportState = KafkaResource{}
	_ resource.ResourceWithModifyPlan  = KafkaResource{}
)

type KafkaResource struct {
	client *client.CloudClient
}

func NewKafkaResource() resource.Resource {
	return &KafkaResource{}
}

func (r KafkaResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_kafka_instance"
}

func (r *KafkaResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = client
}

type KafkaClusterResourceModel struct {
	Lifetime    types.Int64                       `tfsdk:"lifetime"`
	Label       types.String                      `tfsdk:"label"`
	OrderID     types.String                      `tfsdk:"order_id"`
	ItemID      types.String                      `tfsdk:"item_id"`
	Core        core.CoreModel                    `tfsdk:"core"`
	Flavor      flavor.FlavorModel                `tfsdk:"flavor"`
	Access      map[string][]types.String         `tfsdk:"access"`
	ExtraMounts map[string]common.ExtraMountModel `tfsdk:"extra_mounts"`

	Image KafkaImageModel `tfsdk:"image"`

	LayoutID                types.String `tfsdk:"layout_id"`
	ClusterName             types.String `tfsdk:"cluster_name"`
	KafkaVersion            types.String `tfsdk:"kafka_version"`
	BuildVersion            types.String `tfsdk:"build_version"`
	RetentionMinutes        types.Int64  `tfsdk:"retention_minutes"`
	Topics                  types.Map    `tfsdk:"topics"`
	ACLs                    types.Map    `tfsdk:"acls"`
	Quotas                  types.Set    `tfsdk:"quotas"`
	FinancialProject        types.String `tfsdk:"financial_project"`
	UpgradeKafkaDistribMode types.String `tfsdk:"upgrade_kafka_distrib_mode"`
	ConnectionURL           types.String `tfsdk:"connection_url"`
}

type ClientACLsModel struct {
	Idempotent          types.Bool `tfsdk:"allow_idempotent"`
	ConsumerByName      types.Set  `tfsdk:"consumer_by_name"`
	ProducerByName      types.Set  `tfsdk:"producer_by_name"`
	ConsumerByMask      types.Set  `tfsdk:"consumer_by_mask"`
	ProducerByMask      types.Set  `tfsdk:"producer_by_mask"`
	TransactionalByName types.Set  `tfsdk:"transactional_by_name"`
	TransactionalByMask types.Set  `tfsdk:"transactional_by_mask"`
}

type TopicModel struct {
	CleanupPolicy   types.String `tfsdk:"cleanup_policy"`
	Partitions      types.Int64  `tfsdk:"partitions"`
	SegmentSizeMb   types.Int64  `tfsdk:"segment_size_mb"`
	RetentionMs     types.Int64  `tfsdk:"retention_ms"`
	RetentionBytes  types.Int64  `tfsdk:"retention_bytes"`
	CompressionType types.String `tfsdk:"compression_type"`
}

type QuotaModel struct {
	ClientCN         types.String `tfsdk:"client_cn"`
	Type             types.String `tfsdk:"type"`
	ProducerByteRate types.Int64  `tfsdk:"producer_byte_rate"`
}

func (m ClientACLsModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"allow_idempotent":      types.BoolType,
		"consumer_by_name":      basetypes.SetType{ElemType: types.StringType},
		"producer_by_name":      basetypes.SetType{ElemType: types.StringType},
		"consumer_by_mask":      basetypes.SetType{ElemType: types.StringType},
		"producer_by_mask":      basetypes.SetType{ElemType: types.StringType},
		"transactional_by_name": basetypes.SetType{ElemType: types.StringType},
		"transactional_by_mask": basetypes.SetType{ElemType: types.StringType},
	}
}

func (m TopicModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"cleanup_policy":   types.StringType,
		"partitions":       types.Int64Type,
		"segment_size_mb":  types.Int64Type,
		"retention_ms":     types.Int64Type,
		"retention_bytes":  types.Int64Type,
		"compression_type": types.StringType,
	}
}

func (m QuotaModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"client_cn":          types.StringType,
		"type":               types.StringType,
		"producer_byte_rate": types.Int64Type,
	}
}

func (r KafkaResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			// basic schema
			"lifetime": schema.Int64Attribute{
				Optional:            true,
				Description:         "Время жизни заказа в днях (2, 7, 14, 30)",
				MarkdownDescription: "Время жизни заказа в днях (2, 7, 14, 30)",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
			},
			"label": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Метка заказа.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"order_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Идентификатор заказа (известен после создания заказа).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"item_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Идентификатор элемента ВМ в заказе. (становиться известен после создания заказа.)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"core": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.CoreSchema,
				MarkdownDescription: "Основные параметры для ВМ в заказе.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"flavor": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.FlavorSchema,
				MarkdownDescription: "Параметры ЦПУ и оперативной памяти на ВМ.",
			},

			"access": schema.MapAttribute{
				Optional: true,
				Computed: true,
				Description: "Словарь,где ключом является роль, а значением список групп, " +
					"которые предоставляют доступ для входа в Active Directory",
				MarkdownDescription: "Словарь,где ключом является роль, а значением список групп, " +
					"которые предоставляют доступ для входа в Active Directory",
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
					custommodifires.EmptyAccessGroup(),
				},
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
			},
			"extra_mounts": schema.MapNestedAttribute{
				Required:            true,
				MarkdownDescription: "Дополнительные точки монтирования.",
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.OneOf("/app")),
					mapvalidator.SizeBetween(1, 1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{

						"size": schema.Int64Attribute{
							Required:            true,
							MarkdownDescription: "Размер точки монтирования. (в ГБ)",
							Validators: []validator.Int64{
								int64validator.AtLeast(25),
								int64validator.AtMost(2048),
							},
							PlanModifiers: []planmodifier.Int64{
								custommodifires.GrowOnly(),
								custommodifires.MininalGrow(10),
							},
						},
						"file_system": schema.StringAttribute{
							Computed:            true,
							Default:             stringdefault.StaticString("xfs"),
							Description:         "Тип файловой системы.",
							MarkdownDescription: "Тип файловой системы.",
						},
					},
				},
			},

			// image schema
			"image": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Тип вычислительного экземпляра.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"distribution": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Название дистрибутива (RHEL или Astra).",
						Validators: []validator.String{
							stringvalidator.OneOf("rhel", "astra"),
						},
					},
					"os_version": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Версия образа дистрибутива.",
					},
					"default_kafka_version": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Версия брокеров Кафка по умолчанию.",
						Validators: []validator.String{
							stringvalidator.OneOf(
								"2.13-2.4.1",
								"2.13-2.8.2",
								"2.13-3.6.2",
							),
						},
					},
					"geo_distribution": schema.BoolAttribute{
						Required:            true,
						MarkdownDescription: "Механизм включения/выключения геораспределения.",
					},
					"on_support": schema.BoolAttribute{
						Required:            true,
						MarkdownDescription: "Флаг означает, что продукт будет на поддержке у группы сопровождения.",
					},
					"product_id": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Идентификатор продукта экземпляра кластера Кафка.",
					},
					"ad_integration": schema.BoolAttribute{
						Required:            true,
						MarkdownDescription: "Флаг для интеграции вычислительного экземпляра с Active Directory.",
					},
				},
			},

			// extra product schema
			"layout_id": schema.StringAttribute{
				Required:            true,
				Description:         "Layout ID из справочника geo_distribution.",
				MarkdownDescription: "Layout ID из справочника geo_distribution.",
			},
			"cluster_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Имя кластера.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(3),
					stringvalidator.LengthAtMost(33),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9-]+$`),
						"may contain uppercase/lowercase Latin letters, numbers, "+
							"punctuation marks ['-'].",
					),
				},
			},
			"kafka_version": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Версия ядра Кафка.",
				Validators: []validator.String{
					stringvalidator.OneOf(
						"2.13-2.4.1",
						"2.13-2.8.2",
						"2.13-3.6.2",
					),
				},
			},
			"build_version": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Версия облачной сборки продукта.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"retention_minutes": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Сроки хранения указанные в минутах.",
				Default:             int64default.StaticInt64(30),
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64validator.AtLeast(30),
					int64validator.AtMost(20160),
				},
			},

			"topics": schema.MapNestedAttribute{
				MarkdownDescription: "Список топиков.",
				Optional:            true,
				Validators: []validator.Map{
					mapvalidator.KeysAre(
						stringvalidator.LengthAtLeast(1),
						stringvalidator.LengthAtMost(255),
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\.\-_]*$`),
							"may contain uppercase/lowercase Latin letters, numbers, "+
								"punctuation marks ['.' '-' '_']. (cannot start with punctuation)",
						),
					),
					customvalidators.AtLeastOneOfIfValueAre(
						"cleanup_policy",
						"delete",
						[]string{"retention_ms", "retention_bytes"},
					),
					customvalidators.AtLeastOneOfIfValueAre(
						"cleanup_policy",
						"delete,compact",
						[]string{"retention_ms", "retention_bytes"},
					),
					customvalidators.ConflictingIfValueAre(
						"cleanup_policy",
						"compact",
						[]string{"retention_ms", "retention_bytes"},
					),
				},

				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{

						"cleanup_policy": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Политика очистки для данного топика.",
							Validators: []validator.String{
								stringvalidator.OneOf(
									"delete",
									"compact",
									"delete,compact",
								),
							},
						},

						"partitions": schema.Int64Attribute{
							Required:            true,
							MarkdownDescription: "Количество разделов в топике.",
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
								custommodifires.GrowOnly(),
							},
							Validators: []validator.Int64{
								int64validator.AtLeast(1),
								int64validator.AtMost(50),
							},
						},

						"segment_size_mb": schema.Int64Attribute{
							Optional:            true,
							Computed:            true,
							Default:             int64default.StaticInt64(1024),
							MarkdownDescription: "Максимальный размер для хранения данных в разделах указан в Мб.",
							Validators: []validator.Int64{
								int64validator.OneOf(1024, 512, 256, 128),
							},
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
						},

						"retention_ms": schema.Int64Attribute{
							Optional:            true,
							MarkdownDescription: "Время удаления сегмента с данными, указанное в миллисекундах.",
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
							Validators: []validator.Int64{
								int64validator.AtLeast(1800000),
								int64validator.AtMost(7776000000),
							},
						},

						"retention_bytes": schema.Int64Attribute{
							Optional:            true,
							MarkdownDescription: "Максимальный размер для хранения данных в разделах указан в байтах.",
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
							Validators: []validator.Int64{
								int64validator.AtLeast(134217728),
								int64validator.AtMost(1000000000000),
							},
						},
						"compression_type": schema.StringAttribute{
							Computed:            true,
							Optional:            true,
							Default:             stringdefault.StaticString("default"),
							MarkdownDescription: "Тип сжатия",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
							Validators: []validator.String{
								stringvalidator.OneOf(
									"default",
									"uncompressed",
									"zstd",
									"lz4",
									"snappy",
									"gzip",
								),
							},
						},
					},
				},
			},

			"acls": schema.MapNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Список ACLS.",
				Validators: []validator.Map{
					mapvalidator.KeysAre(
						stringvalidator.LengthAtLeast(1),
						stringvalidator.LengthAtMost(64),
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^[a-zA-Z0-9_\-@\.]+$`),
							"may contain uppercase/lowercase Latin letters, numbers, "+
								"punctuation marks ['.' '-' '_'] and '@'",
						),
					),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{

						"allow_idempotent": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
							MarkdownDescription: "Добавленный для этого client_cn ACL поддерживает идемпотентные действия.",
						},

						"consumer_by_name": schema.SetAttribute{
							MarkdownDescription: "Список названий топиков, доступные клиенту для чтения/описания.",
							Optional:            true,
							Computed:            true,
							ElementType:         types.StringType,
							PlanModifiers: []planmodifier.Set{
								custommodifires.DefaultEmptyStringList(),
							},
							Validators: []validator.Set{
								setvalidator.ValueStringsAre(
									stringvalidator.LengthAtLeast(1),
									stringvalidator.LengthAtMost(255),
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\.\-_]*$`),
										"may contain uppercase/lowercase Latin letters, numbers, "+
											"punctuation marks ['.' '-' '_']. (cannot start with punctuation)",
									),
								),
							},
						},

						"producer_by_name": schema.SetAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Список названий топиков, доступные клиенту для записи/создания/описания.",
							ElementType:         types.StringType,
							PlanModifiers: []planmodifier.Set{
								custommodifires.DefaultEmptyStringList(),
							},
							Validators: []validator.Set{
								setvalidator.ValueStringsAre(
									stringvalidator.LengthAtLeast(1),
									stringvalidator.LengthAtMost(255),
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\.\-_]*$`),
										"may contain uppercase/lowercase Latin letters, numbers, "+
											"punctuation marks ['.' '-' '_']. (cannot start with punctuation)",
									),
								),
							},
						},

						"consumer_by_mask": schema.SetAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Список масок, указанных в форме префикса, доступные клиенту для чтения/описания.",
							ElementType:         types.StringType,
							PlanModifiers: []planmodifier.Set{
								custommodifires.DefaultEmptyStringList(),
							},
							Validators: []validator.Set{
								setvalidator.ValueStringsAre(
									stringvalidator.LengthAtLeast(1),
									stringvalidator.LengthAtMost(255),
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^[a-zA-Z0-9-_\.]+$`),
										"may contain uppercase/lowercase Latin letters, numbers, "+
											"punctuation marks ['.' '-' '_'].",
									),
								),
							},
						},

						"producer_by_mask": schema.SetAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Список масок, указанных в префиксной форме, доступные клиенту записывать/создавать/описывать",
							ElementType:         types.StringType,
							PlanModifiers: []planmodifier.Set{
								custommodifires.DefaultEmptyStringList(),
							},
							Validators: []validator.Set{
								setvalidator.ValueStringsAre(
									stringvalidator.LengthAtLeast(1),
									stringvalidator.LengthAtMost(255),
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^[a-zA-Z0-9-_\.]+$`),
										"may contain uppercase/lowercase Latin letters, numbers, "+
											"punctuation marks ['.' '-' '_'].",
									),
								),
							},
						},

						"transactional_by_name": schema.SetAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Список имен транзакций, разрешающих клиенту использовать его",
							ElementType:         types.StringType,
							PlanModifiers: []planmodifier.Set{
								custommodifires.DefaultEmptyStringList(),
							},
							Validators: []validator.Set{
								setvalidator.ValueStringsAre(
									stringvalidator.LengthAtLeast(1),
									stringvalidator.LengthAtMost(255),
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^[a-zA-Z0-9-_\.]+$`),
										"may contain uppercase/lowercase Latin letters, numbers, "+
											"punctuation marks ['.' '-' '_'].",
									),
								),
							},
						},

						"transactional_by_mask": schema.SetAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Список транзакционных масок, разрешающих клиенту использовать его.",
							ElementType:         types.StringType,
							PlanModifiers: []planmodifier.Set{
								custommodifires.DefaultEmptyStringList(),
							},
							Validators: []validator.Set{
								setvalidator.ValueStringsAre(
									stringvalidator.LengthAtLeast(1),
									stringvalidator.LengthAtMost(255),
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^[a-zA-Z0-9-_\.]+$`),
										"may contain uppercase/lowercase Latin letters, numbers, "+
											"punctuation marks ['.' '-' '_'].",
									),
								),
							},
						},
					},
				},
			},

			"quotas": schema.SetNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Список квот",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{

						"type": schema.StringAttribute{
							Required:            true,
							Description:         "Тип квоты (one of [default, personal])",
							MarkdownDescription: "Тип квоты (one of [default, personal])",
							Validators: []validator.String{
								stringvalidator.OneOf("default", "personal"),
							},
						},

						"client_cn": schema.StringAttribute{
							Optional:            true,
							Description:         "CN клиентского сертификата. Он может быть использован только для персональных квот.",
							MarkdownDescription: "CN клиентского сертификата. Он может быть использован только для персональных квот.",
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(3),
								stringvalidator.LengthAtMost(64),
								stringvalidator.NoneOf("default", "<default>"),
								stringvalidator.RegexMatches(
									regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\.\-_]*$`),
									"may contain uppercase/lowercase Latin letters, numbers, "+
										"punctuation marks ['.' '-' '_']. (cannot start with punctuation)",
								),
							},
						},

						"producer_byte_rate": schema.Int64Attribute{
							Required:            true,
							Description:         "Скорость передачи байтов квоты производителя.",
							MarkdownDescription: "Скорость передачи байтов квоты производителя.",
							PlanModifiers: []planmodifier.Int64{
								int64planmodifier.UseStateForUnknown(),
							},
							Validators: []validator.Int64{
								int64validator.AtLeast(131072),
								int64validator.AtMost(52428800),
							},
						},
					},
				},
			},

			"financial_project": schema.StringAttribute{
				Required:            true,
				Description:         "Источник финансирования.",
				MarkdownDescription: "Источник финансирования.",
			},
			"upgrade_kafka_distrib_mode": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("none"),
				MarkdownDescription: `Мод для регулировки политики обновлений версии дистрибутива ВТБ Кафка ("latest/none")
									если флаг проставлен в "latest", то будет запущено действие "Обновление версии дистрибутива"`,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("latest", "none"),
				},
			},
			"connection_url": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				MarkdownDescription: "Connection URL",
			},
		},
	}
}

func (r KafkaResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r KafkaResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan, state KafkaClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	}
	checkOrderIsDeleted := utils.CheckOrderIsDeleted(r.client, plan.OrderID, "cluster", "kafka")
	resp.Diagnostics.Append(checkOrderIsDeleted.Diagnostics...)

	utils.HandleExternalDeletionRecreate(ctx, resp, &plan, checkOrderIsDeleted.IsDeleted, checkOrderIsDeleted.Diagnostics)

	validateAccessForEnvDiags := utils.ValidateActiveDirectoryAccessForEnvironmentsV1(r.client, plan.Access, "DEV")
	resp.Diagnostics.Append(validateAccessForEnvDiags...)

	validateAccessRolesDiags := utils.ValidateAccessRolesV2(r.client, "cluster:kafka", plan.Access)
	resp.Diagnostics.Append(validateAccessRolesDiags...)

	if !plan.ACLs.IsNull() || plan.ACLs.IsUnknown() {
		planAcls := make(map[string]ClientACLsModel)
		plan.ACLs.ElementsAs(ctx, &planAcls, false)
		warnDeprecatedACLsFunction(planAcls, &resp.Diagnostics, plan.KafkaVersion.ValueString())
	}

	if !plan.Quotas.IsNull() {
		r.validateQuotas(ctx, &plan, &resp.Diagnostics)
	}

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r KafkaResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan KafkaClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	finProj, err := sources.GetFinancialProjectByName(
		r.client.Creds,
		r.client.ProjectName,
		plan.FinancialProject.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("financial_project"),
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Get financial source ended  with error.\nError: %s", err.Error()))
		return
	}

	layoutName, err := references.GetGeoDistributionLayoutNameByID(
		r.client.Creds,
		plan.LayoutID.ValueString(),
		"kafka",
		r.client.Organization,
		plan.Core.NetSegmentCode.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("layout_id"),
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Get layout_name ended with error.\nError: %s", err.Error()),
		)
		return
	}

	attrs := r.prepareAttrs(plan, layoutName)

	order := orders.NewKafka(
		r.client.Creds,
		r.client.ProjectName,
		plan.Image.ProductID.ValueString(),
		attrs,
	)

	err = order.Create(orders.CreateOrderPayload{
		Label:        plan.Label.ValueString(),
		Lifetime:     int(plan.Lifetime.ValueInt64()),
		FinProjectID: finProj.ID,
	})
	if err != nil {
		resp.Diagnostics.AddError(consts.CREATE_RES_FAIL, err.Error())
		return
	}

	orderItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Get order item ended with error.\nError: %s", err.Error()),
		)
		return
	}

	if !plan.Quotas.IsNull() {
		planQuotas := []QuotaModel{}
		plan.Quotas.ElementsAs(ctx, &planQuotas, false)

		applyQuotas(order, planQuotas, []QuotaModel{}, &resp.Diagnostics)
	}

	if !plan.Topics.IsNull() || !plan.Topics.IsUnknown() {
		pTopics := make(map[string]TopicModel)
		plan.Topics.ElementsAs(ctx, &pTopics, false)
		diags := applyTopics(pTopics, order)
		resp.Diagnostics.Append(diags...)
	}

	if !plan.ACLs.IsNull() || !plan.ACLs.IsUnknown() {
		planAcls := make(map[string]ClientACLsModel)
		plan.ACLs.ElementsAs(ctx, &planAcls, false)
		diags := applyACLs(ctx, planAcls, order)
		resp.Diagnostics.Append(diags...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(orderItem.ID)
	plan.KafkaVersion = types.StringValue(orderItem.Data.Config.(entities.KafkaItemConfig).KafkaVersion)
	plan.BuildVersion = types.StringValue(orderItem.Data.Build.SetupVersion)
	plan.ConnectionURL = types.StringValue(orderItem.Data.Config.(entities.KafkaItemConfig).ConnectionURL)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r KafkaResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var orderID types.String
	req.State.GetAttribute(ctx, path.Root("order_id"), &orderID)
	if orderID.IsNull() {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Resource hasn't order_id. Without order_id can't get order from portal",
		)
		return
	}

	// Get order data
	order, err := orders.GetKafkaOrder(
		r.client.Creds,
		r.client.ProjectName,
		orderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't read order with order_id equal '%s'.\nError:%s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}

	kafkaItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get cluster item from portal with order_id '%s'.\nError:%s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}

	kafkaConfig, ok := kafkaItem.Data.Config.(entities.KafkaItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Invalid type for kafka config",
		)
		return
	}

	kafkaVMs, err := order.GetKafkaVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Can't get kafka VMs items.\nError: %s", err.Error()),
		)
		return
	}

	vmItem := kafkaVMs[0]
	vmConfig, ok := vmItem.Data.Config.(entities.VMItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Invalid type for vmConfig",
		)
		return
	}

	prefix := r.getKafkaLayoutsPrefix(vmConfig.DefaultNic.NetSegment)

	kafkaLayouts := map[int]string{
		4:  fmt.Sprintf("%s:kafka-4:zookeeper-3", prefix),
		6:  fmt.Sprintf("%s:kafka-6:zookeeper-3", prefix),
		8:  fmt.Sprintf("%s:kafka-8:zookeeper-3", prefix),
		16: fmt.Sprintf("%s:kafka-16:zookeeper-3", prefix),
	}

	image, err := references.GetKafkaImageData(
		r.client.Creds,
		vmConfig.Image.Os.Distribution,
		r.client.Organization,
		r.client.Environment,
		vmConfig.Image.Os.Version,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Can't extract image about kafka.\nError: %s", err.Error()),
		)
		return
	}

	appMount, err := order.GetKafkaExtraMount("/app")
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Can't get kafka extra mounts.\nError: %s", err.Error()),
		)
		return
	}

	var layoutId string
	kafkaCount := len(strings.Split(kafkaConfig.ConnectionURL, ","))
	if kafkaCount < 4 {
		layoutId = order.Attrs.Layout
	} else {
		layoutId, err = references.GetGeoDistributionLayoutID(
			r.client.Creds,
			kafkaLayouts[kafkaCount],
			"kafka",
			r.client.Organization,
			vmConfig.DefaultNic.NetSegment,
		)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("layout_id"),
				consts.READ_RES_FAIL,
				fmt.Sprintf("Get 'layout_id' from portal ended with error.\nError: %s", err.Error()),
			)
			return
		}
	}

	state := KafkaClusterResourceModel{
		OrderID:          orderID,
		LayoutID:         types.StringValue(layoutId),
		ACLs:             readACLS(kafkaConfig, ctx, resp),
		Quotas:           readQuotas(ctx, kafkaConfig),
		Access:           utils.ReadAccessMapV2(vmItem.Data.ACLs),
		ItemID:           types.StringValue(kafkaItem.ID),
		Label:            types.StringValue(order.Label),
		BuildVersion:     types.StringValue(kafkaItem.Data.Build.SetupVersion),
		KafkaVersion:     types.StringValue(kafkaConfig.KafkaVersion),
		ClusterName:      types.StringValue(kafkaConfig.ClusterName),
		RetentionMinutes: types.Int64Value(int64(kafkaConfig.KafkaLogRetentionMinutes)),
		Flavor: flavor.FlavorModel{
			Cores:  types.Int64Value(vmConfig.Flavor.Cores),
			Memory: types.Int64Value(vmConfig.Flavor.Memory),
			Name:   types.StringValue(vmConfig.Flavor.Name),
			UUID:   types.StringValue(vmConfig.Flavor.UUID),
		},
		Core: core.CoreModel{
			Platform:       types.StringValue(utils.SelectPlatform(order.Attrs.Platform)),
			Domain:         types.StringValue(order.Attrs.Domain),
			NetSegmentCode: types.StringValue(vmConfig.DefaultNic.NetSegment),
			Zone:           types.StringValue(order.Attrs.AvailabilityZone),
		},
		Image: KafkaImageModel{
			GeoDistribution:     types.BoolValue(order.Attrs.GeoDistribution),
			Distribution:        types.StringValue(vmConfig.Image.Os.Distribution),
			OsVersion:           types.StringValue(vmConfig.Image.Os.Version),
			OnSupport:           types.BoolValue(order.Attrs.OnSupport),
			ProductID:           types.StringValue(order.ProductID),
			ADIntegration:       types.BoolValue(order.Attrs.ADIntegration),
			DefaultKafkaVersion: types.StringValue(image.DefaultKafkaVersion),
		},
		ExtraMounts: map[string]common.ExtraMountModel{
			"/app": {
				Size:       types.Int64Value(appMount.Size),
				FileSystem: types.StringValue(appMount.FileSystem),
			},
		},
		FinancialProject:        types.StringValue(order.FinancialSource.Name),
		UpgradeKafkaDistribMode: types.StringValue("none"),
		ConnectionURL:           types.StringValue(kafkaConfig.ConnectionURL),
	}

	// Get Topics
	orderTopics := kafkaConfig.Topics
	topics, diags := convertTopicsToTerraform(orderTopics)
	resp.Diagnostics.Append(diags...)

	var tfTopics basetypes.MapValue
	if len(topics) > 0 {
		tfTopics, _ = types.MapValueFrom(
			ctx,
			types.ObjectType{AttrTypes: TopicModel{}.AttributeTypes()},
			topics,
		)
	} else {
		tfTopics = types.MapNull(types.ObjectType{AttrTypes: TopicModel{}.AttributeTypes()})
	}

	state.Topics = tfTopics
	if resp.Diagnostics.HasError() {
		return
	}

	var lifetime types.Int64
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("lifetime"), &lifetime)...)
	if !lifetime.IsNull() {
		state.Lifetime = lifetime
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r KafkaResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state KafkaClusterResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	order, err := orders.GetKafkaOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("order_id"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Get order data from portal ended with error.\nError: %s", err.Error()),
		)
		return
	}
	kafkaImageData, err := references.GetKafkaImageData(
		r.client.Creds,
		state.Image.Distribution.ValueString(),
		r.client.Organization,
		r.client.Environment,
		state.Image.OsVersion.ValueString(),
	)

	if err != nil {
		resp.Diagnostics.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Get kafka image data ended with error.\nError: %v", err.Error()),
		)
		return
	}

	finProj, err := sources.GetFinancialProjectByName(
		r.client.Creds,
		r.client.ProjectName,
		plan.FinancialProject.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("financial_project"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Get financial source for project ended with error.\nError: %s", err.Error()),
		)
		return
	}
	labelChanged := plan.Label != state.Label
	mountChanged := utils.IsExtraMountChanged(state.ExtraMounts, plan.ExtraMounts)
	flavorChanged := plan.Flavor != state.Flavor
	if labelChanged {
		utils.ChangeOrderLabel(order, plan.Label.ValueString(), resp)
	}

	if !plan.FinancialProject.Equal(state.FinancialProject) {
		r.changeFinancialSource(order, finProj.ID, resp)
	}

	if !plan.Topics.Equal(state.Topics) {
		pTopics := make(map[string]TopicModel)
		plan.Topics.ElementsAs(ctx, &pTopics, false)

		updateTopics(order, pTopics, resp)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !plan.ACLs.Equal(state.ACLs) {
		planAcls := make(map[string]ClientACLsModel)
		plan.ACLs.ElementsAs(ctx, &planAcls, false)

		updateACLs(ctx, order, planAcls, resp)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !plan.Quotas.Equal(state.Quotas) {

		planQuotas := []QuotaModel{}
		plan.Quotas.ElementsAs(ctx, &planQuotas, false)

		stateQuotas := []QuotaModel{}
		state.Quotas.ElementsAs(ctx, &stateQuotas, false)

		deleteQuotas(order, planQuotas, stateQuotas, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}

		applyQuotas(order, planQuotas, stateQuotas, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if mountChanged {
		diags := changeKafkaExtraMounts(order, &plan)
		resp.Diagnostics.Append(diags...)
	}

	if flavorChanged {
		diags := changeKafkaFlavor(order, &plan)
		resp.Diagnostics.Append(diags...)
	}

	if !plan.KafkaVersion.Equal(state.KafkaVersion) {
		diags := upgradeKafkaVersion(order, &plan, &state)
		resp.Diagnostics.Append(diags...)
	}

	if !plan.LayoutID.Equal(state.LayoutID) {
		r.horizontalScaling(order, &plan, &state, resp)
	}

	if !plan.UpgradeKafkaDistribMode.IsNull() && plan.UpgradeKafkaDistribMode.ValueString() == "latest" {
		if state.KafkaVersion.Equal(types.StringValue("2.13-2.4.1")) {
			r.updateKafkaDistribVersion(order, state.BuildVersion.ValueString(), kafkaImageData.LatestReleaseVersionV1, resp)
		} else {
			r.updateKafkaDistribVersion(order, state.BuildVersion.ValueString(), kafkaImageData.LatestReleaseVersionV2, resp)
		}
	}

	if !plan.ClusterName.Equal(state.ClusterName) {
		r.editClusterName(order, &plan, resp)
	}

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r KafkaResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state KafkaClusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	order, err := orders.GetKafkaOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)

	if err != nil {
		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf("Get order from portal ended with error.\nError: %s", err.Error()),
		)
		return
	}

	orderState, err := order.GetState()
	if err != nil {
		resp.Diagnostics.AddWarning(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf("Can't get state from portal.\nError: %s", err.Error()),
		)
		return
	}

	if orderState == "deleted" || orderState == "deprovisioned" {
		resp.Diagnostics.AddWarning(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf(
				"Order with id '%s' was deleted by another way. Not from terraform. Resource was recreated",
				state.OrderID.ValueString(),
			),
		)
		return
	}

	if !order.Deletable {
		orderURL := fmt.Sprintf(
			"https://cloud.vtb.ru/all/orders/%s/main?context=%s&type=project&org=%s",
			order.ID,
			order.ProjectName,
			r.client.Organization,
		)

		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf(
				"Order with order_id '%s' isn't deletable. "+
					"Please, make your order is deletable by the portal VTB.Cloud, and repeat again."+
					"\nURL reference: %s",
				order.ID, orderURL,
			),
		)
		return
	}

	err = order.DeleteTwoLayer(false)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf("Delete order from portal ended with error.\nError: %s", err.Error()))
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}
}

// * Custom loigc *

func (r KafkaResource) validateQuotas(
	ctx context.Context,
	plan *KafkaClusterResourceModel,
	diags *diag.Diagnostics,
) {
	if plan.KafkaVersion.Equal(types.StringValue("2.13-2.4.1")) {
		diags.AddAttributeError(
			path.Root("quotas"),
			consts.MODIFY_PLAN_FAIL,
			fmt.Sprintf(
				"quotas is only can used with kafka versions `2.13-2.8.2` or `2.13-3.6.2`, but choosen version is `%s`",
				plan.KafkaVersion.ValueString(),
			),
		)
	}

	var quotas []QuotaModel
	diags.Append(plan.Quotas.ElementsAs(ctx, &quotas, false)...)

	defaultQuotas := 0
	var uniqueClientCNs []string

	for _, quota := range quotas {

		if slices.Contains(uniqueClientCNs, quota.ClientCN.ValueString()) {
			diags.AddAttributeError(
				path.Root("quotas"),
				fmt.Sprintln(consts.MODIFY_PLAN_FAIL, "non-unique client_cn occured"),
				fmt.Sprintf(
					"quota client_cn must be unique for all quotas.\n"+
						"Non-unique: %s", quota.ClientCN.ValueString(),
				),
			)
		}
		uniqueClientCNs = append(uniqueClientCNs, quota.ClientCN.ValueString())

		if quota.Type.ValueString() == "personal" && quota.ClientCN.IsNull() {
			diags.AddAttributeError(
				path.Root("quotas"),
				consts.MODIFY_PLAN_FAIL,
				"`client_cn` must be specified for `personal` quotas.",
			)
		}

		if quota.Type.ValueString() == "default" && !quota.ClientCN.IsNull() {
			diags.AddAttributeError(
				path.Root("quotas"),
				consts.MODIFY_PLAN_FAIL,
				"`client_cn` must not be specified for `default` quotas.",
			)
		}

		if quota.Type.ValueString() == "default" {
			defaultQuotas = defaultQuotas + 1
		}
	}

	if defaultQuotas > 1 {
		diags.AddError(
			fmt.Sprintln(consts.MODIFY_PLAN_FAIL, "too many `default` quotas"),
			"only one quota with type `default` must be specified."+
				"\nKeep only one `default` quota and remove all others. ",
		)
	}
}

func changeKafkaFlavor(
	order *orders.Kafka,
	plan *KafkaClusterResourceModel,
) diag.Diagnostics {
	var diags diag.Diagnostics
	flavor := entities.Flavor{
		Cores:  plan.Flavor.Cores.ValueInt64(),
		Memory: plan.Flavor.Memory.ValueInt64(),
		UUID:   string(plan.Flavor.UUID.ValueString()),
		Name:   string(plan.Flavor.Name.ValueString()),
	}
	err := order.ChangeFlavor(flavor)
	if err != nil {
		diags.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Change `flavor` ended with error.\nError: %s", err.Error()),
		)
	}
	return diags
}

func changeKafkaExtraMounts(
	order *orders.Kafka,
	plan *KafkaClusterResourceModel,
) diag.Diagnostics {
	var diags diag.Diagnostics
	for path, planEM := range plan.ExtraMounts {
		err := order.ExpandKafkaMountPoint(entities.ExtraMount{
			Path:       path,
			Size:       planEM.Size.ValueInt64(),
			FileSystem: planEM.FileSystem.ValueString(),
		}, false)
		if err != nil {
			diags.AddError(
				consts.UPDATE_RES_FAIL,
				fmt.Sprintf("Change `extra_mounts` ended with error.\nError: %s", err.Error()),
			)
		}
	}
	return diags
}

// #topics
func applyTopics(pTopics map[string]TopicModel, order *orders.Kafka) (diags diag.Diagnostics) {

	err := order.Sync()
	if err != nil {
		diags.AddWarning(
			"Sync Kafka Topics",
			fmt.Sprintf("Can't sync kafka cluster.\nError: %s", err.Error()),
		)
	}

	orderTopics, err := order.GetTopics()
	if err != nil {
		diags.AddWarning(
			"Get Kafka Topics",
			fmt.Sprintf("Get `topics` from portal ended with error.\nError: %s", err.Error()),
		)
	}

	var temporaryTopics map[string]TopicModel = pTopics
	planTopics, xdiags := converTerraformTopicsToEntities(temporaryTopics)
	diags.Append(xdiags...)

	var toCreate []entities.KafkaTopic
	var toRemove []entities.KafkaTopic
	var toEdit []entities.KafkaTopic
	var existsTopics []bool = make([]bool, len(orderTopics))

	for _, planTopic := range planTopics {

		exists := false
		for i, existTopic := range orderTopics {

			if planTopic.TopicName == existTopic.TopicName {

				if planTopic.Equal(existTopic) {
					existsTopics[i] = true
					exists = true
					break
				} else {
					toEdit = append(toEdit, planTopic)
					existsTopics[i] = true
					exists = true
				}
			}
		}

		if !exists {
			toCreate = append(toCreate, planTopic)
		}
	}

	for i, exists := range existsTopics {
		if !exists {
			toRemove = append(toRemove, orderTopics[i])
		}
	}

	if len(toCreate) > 0 {
		err := order.CreateTopics(toCreate, false)
		if err != nil {
			diags.AddWarning(
				"Create topics",
				fmt.Sprintf(
					"Kafka didn't created topics or created parted.\nError: %s",
					err.Error(),
				),
			)
		}
	}

	if len(toRemove) > 0 {
		err = order.DeleteTopics(toRemove, false)
		if err != nil {
			diags.AddWarning(
				"Delete topics",
				fmt.Sprintf(
					"Kafka didn't deleted topics or deleted parted.\nError: %s",
					err.Error(),
				),
			)
		}
	}

	if len(toEdit) > 0 {
		err = order.EditTopics(toEdit, false)
		if err != nil {
			diags.AddWarning(
				"Edit topics",
				fmt.Sprintf(
					"Kafka didn't edit topics or edited parted.\nError: %s",
					err.Error(),
				),
			)
		}
	}
	return
}

func updateTopics(
	order *orders.Kafka,
	pTopics map[string]TopicModel,
	resp *resource.UpdateResponse,
) {
	var tmpTopics map[string]TopicModel = make(map[string]TopicModel)
	tmpTopics = pTopics

	planTopics, diags := converTerraformTopicsToEntities(tmpTopics)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orderTopics, err := order.GetTopics()
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "can't get kafka topics"),
			err.Error(),
		)
		return
	}

	if isTopicsChanged(planTopics, orderTopics) {
		diags = applyTopics(pTopics, order)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}

func convertTopicsToTerraform(entityTopics []entities.KafkaTopic) (
	terraformTopics map[string]TopicModel,
	diags diag.Diagnostics,
) {
	var pointerTopics map[string]*TopicModel = make(map[string]*TopicModel)

	for _, topic := range entityTopics {

		pointerTopics[topic.TopicName] = &TopicModel{
			CleanupPolicy:   types.StringValue(topic.CleanupPolicy),
			Partitions:      types.Int64Value(topic.PartitionsNumber),
			SegmentSizeMb:   types.Int64Value(bytesToMb(topic.SegmentSize)),
			CompressionType: types.StringValue(topic.CompressionType),
		}
		topicModel := pointerTopics[topic.TopicName]

		if topic.RetentionMs == 0 {
			topicModel.RetentionMs = types.Int64Null()
		} else {
			topicModel.RetentionMs = types.Int64Value(topic.RetentionMs)
		}

		if topic.RetentionBytes == 0 {
			topicModel.RetentionBytes = types.Int64Null()
		} else {
			topicModel.RetentionBytes = types.Int64Value(topic.RetentionBytes)
		}
	}

	terraformTopics = make(map[string]TopicModel)
	for topicName, topicData := range pointerTopics {
		terraformTopics[topicName] = *topicData
	}
	return terraformTopics, diags
}

func converTerraformTopicsToEntities(
	terraformTopics map[string]TopicModel,
) (
	entityTopics []entities.KafkaTopic,
	diags diag.Diagnostics,
) {
	for topicName, topicData := range terraformTopics {
		entityTopics = append(entityTopics, entities.KafkaTopic{
			TopicName:        topicName,
			CleanupPolicy:    topicData.CleanupPolicy.ValueString(),
			PartitionsNumber: topicData.Partitions.ValueInt64(),
			RetentionMs:      topicData.RetentionMs.ValueInt64(),
			RetentionBytes:   topicData.RetentionBytes.ValueInt64(),
			SegmentSize:      mbToBytes(topicData.SegmentSizeMb.ValueInt64()),
			CompressionType:  topicData.CompressionType.ValueString(),
		})
	}
	return
}

// #acls
func applyACLs(
	ctx context.Context,
	planAcls map[string]ClientACLsModel,
	order *orders.Kafka,
) (diags diag.Diagnostics) {

	err := order.Sync()
	if err != nil {
		diags.AddWarning(
			"Sync kafka topics",
			fmt.Sprintf("Can't sync kafka cluster.\nError: %s", err.Error()),
		)
		return diags
	}

	orderAccessACLs, err := order.GetACLs()
	if err != nil {
		diags.AddWarning("Get Kafka access ACLs", err.Error())
		return diags
	}

	orderTransactionalACLs, err := order.GetTransactionalACLs()
	if err != nil {
		diags.AddWarning("Get Kafka transactional ACLs", err.Error())
		return diags
	}

	orderIdempotentACLs, err := order.GetIdempotentACLs()
	if err != nil {
		diags.AddWarning("Get Kafka idempotent ACLs", err.Error())
		return diags
	}

	var temporaryACLs map[string]ClientACLsModel = planAcls
	planAccessACLs, planTransactionalACLs, planIdempotentACLs, xdiags := convertTerraformACLsToEntities(ctx, temporaryACLs)
	diags.Append(xdiags...)
	if diags.HasError() {
		return diags
	}

	// apply access acls
	var accessToCreate, accessToRemove []entities.KafkaACL
	var existsAccessACLs []bool = make([]bool, len(orderAccessACLs))

	for _, pa := range planAccessACLs {
		exists := false
		for i, oa := range orderAccessACLs {
			if pa.ClientCN == oa.ClientCN && pa.ClientRole == oa.ClientRole && pa.Type == oa.Type {
				if pa.Equal(oa) { // if all topic_names exists
					existsAccessACLs[i] = true
					exists = true
					break
				} else if pa.Type == "by_name" { // if topic_names is difference
					var toAddedTopics, toRemoveTopics []string
					exists = true
					existsAccessACLs[i] = true
					existsOrder := make([]bool, len(oa.TopicNames))
					for _, paTopics := range pa.TopicNames {
						topicExists := false
						for index, oaTopics := range oa.TopicNames {
							if paTopics == oaTopics {
								topicExists = true
								existsOrder[index] = true
								break
							}
						}
						if !topicExists {
							toAddedTopics = append(toAddedTopics, paTopics)
						}
					}
					for index, exists := range existsOrder {
						if !exists {
							toRemoveTopics = append(toRemoveTopics, oa.TopicNames[index])
						}
					}
					if len(toAddedTopics) > 0 {
						accessToCreate = append(accessToCreate, entities.KafkaACL{
							ClientCN:   pa.ClientCN,
							ClientRole: pa.ClientRole,
							Type:       pa.Type,
							TopicNames: toAddedTopics,
						})
					}
					if len(toRemoveTopics) > 0 {
						accessToRemove = append(accessToRemove, entities.KafkaACL{
							ClientCN:   pa.ClientCN,
							ClientRole: pa.ClientRole,
							Type:       pa.Type,
							TopicNames: toRemoveTopics,
						})
					}
				}
			}
		}
		if !exists {
			accessToCreate = append(accessToCreate, pa)
		}
	}

	for i, exists := range existsAccessACLs {
		if !exists {
			accessToRemove = append(accessToRemove, orderAccessACLs[i])
		}
	}

	// apply transactional acls
	var transactionalToCreate []entities.KafkaTransactionalACL
	var transactionalToRemove []entities.KafkaTransactionalACL
	var existsTransactionalACLs []bool = make([]bool, len(orderTransactionalACLs))

	for _, pa := range planTransactionalACLs {
		exists := false
		for i, oa := range orderTransactionalACLs {
			if pa.Equal(oa) {
				existsTransactionalACLs[i] = true
				exists = true
				break
			}
		}
		if !exists {
			transactionalToCreate = append(transactionalToCreate, pa)
		}
	}

	for i, exists := range existsTransactionalACLs {
		if !exists {
			transactionalToRemove = append(transactionalToRemove, orderTransactionalACLs[i])
		}
	}

	// apply idempotent acls
	var idempotentToCreate []entities.KafkaIdempotentACL
	var idempotentToRemove []entities.KafkaIdempotentACL
	var existsIdempotentACLs []bool = make([]bool, len(orderIdempotentACLs))

	for _, pa := range planIdempotentACLs {
		exists := false
		for i, oa := range orderIdempotentACLs {
			if pa.Equal(oa) {
				existsIdempotentACLs[i] = true
				exists = true
				break
			}
		}
		if !exists {
			idempotentToCreate = append(idempotentToCreate, pa)
		}
	}

	for i, exists := range existsIdempotentACLs {
		if !exists {
			idempotentToRemove = append(idempotentToRemove, orderIdempotentACLs[i])
		}
	}

	if len(accessToCreate) > 0 {
		attempts, err := utils.RetryWithExponentialBackoff(
			CREATE_ACLS_ATTEMPTS,
			CREATE_ACLS_DELAY*time.Second,
			func() error {
				return order.CreateACLs(accessToCreate, false)
			},
		)
		if err != nil {
			diags.AddWarning(
				"Create access acls",
				fmt.Sprintf(
					"Kafka don't created access ACLs or created parted.\nError: %s,\nAttempts to create acls: %v",
					err.Error(), attempts,
				),
			)
			return diags
		}
	}

	if len(transactionalToCreate) > 0 {
		attempts, err := utils.RetryWithExponentialBackoff(
			CREATE_ACLS_ATTEMPTS,
			CREATE_ACLS_DELAY*time.Second,
			func() error {
				return order.CreateTransactionalACLs(transactionalToCreate, false)
			},
		)

		if err != nil {
			diags.AddWarning(
				"Create transactional acls",
				fmt.Sprintf(
					"Kafka don't created transactional ACLs or created parted.\nError: %s.\nAttempts to create acls: %v",
					err.Error(), attempts,
				),
			)
			return diags
		}
	}

	if len(idempotentToCreate) > 0 {
		attempts, err := utils.RetryWithExponentialBackoff(
			CREATE_ACLS_ATTEMPTS,
			CREATE_ACLS_DELAY*time.Second,
			func() error {
				return order.CreateIdempotentACLs(idempotentToCreate, false)
			},
		)

		if err != nil {
			diags.AddWarning(
				"Create idempotent acls",
				fmt.Sprintf(
					"Kafka don't created idempotent ACLs or created parted.\nError: %s.\nAttempts to create acls: %v",
					err.Error(), attempts,
				),
			)
			return diags
		}
	}

	if len(accessToRemove) > 0 {
		err = order.DeleteACLs(accessToRemove, false)
		if err != nil {
			diags.AddWarning(
				"Delete access acls",
				fmt.Sprintf(
					"Kafka don't deleted access ACLs or deleted parted.\nError: %s",
					err.Error(),
				),
			)
			return diags
		}
	}

	if len(transactionalToRemove) > 0 {
		err = order.DeleteTransactionalACLs(transactionalToRemove, false)
		if err != nil {
			diags.AddWarning(
				"Delete transactional acls",
				fmt.Sprintf(
					"Kafka don't deleted transactional ACLs or deleted parted.\nError: %s",
					err.Error(),
				),
			)
			return diags
		}
	}

	if len(idempotentToRemove) > 0 {
		err = order.DeleteIdempotentACLs(idempotentToRemove, false)
		if err != nil {
			diags.AddWarning(
				"Delete idempotent acls",
				fmt.Sprintf(
					"Kafka don't deleted idempotent ACLs or deleted parted.\nError: %s",
					err.Error(),
				),
			)
			return diags
		}
	}
	return diags
}

func updateACLs(
	ctx context.Context,
	order *orders.Kafka,
	planAcls map[string]ClientACLsModel,
	resp *resource.UpdateResponse,
) {
	var tmpClientACLs map[string]ClientACLsModel = make(map[string]ClientACLsModel)
	tmpClientACLs = planAcls

	accessACLs, transactionalACLs, idempotentACLs, diags := convertTerraformACLsToEntities(ctx, tmpClientACLs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	orderAccessACLs, err := order.GetACLs()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Get ACLs ended with error.\nError: %s", err.Error()),
		)
		return
	}

	orderTransactionalACLs, err := order.GetTransactionalACLs()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Get transactional ACLs ended error.\nError: %s", err.Error()),
		)
		return
	}

	orderIdempotentACLs, err := order.GetIdempotentACLs()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Get idempotent ACLs ended with error.\nError: %s", err.Error()),
		)
		return
	}

	if isAccessACLsChagned(accessACLs, orderAccessACLs) ||
		isTransactionalACLsChanged(transactionalACLs, orderTransactionalACLs) ||
		isIdempotentACLsChanged(idempotentACLs, orderIdempotentACLs) {
		diags = applyACLs(ctx, planAcls, order)
		resp.Diagnostics.Append(diags...)
	}
}

func convertEntitiesACLsToTerraform(
	ctx context.Context,
	accessACLs []entities.KafkaACL,
	transactionalACLs []entities.KafkaTransactionalACL,
	idempotentACLs []entities.KafkaIdempotentACL,
) (
	clientACLs map[string]ClientACLsModel,
	diags diag.Diagnostics,
) {
	var pointerACLs map[string]*ClientACLsModel = make(map[string]*ClientACLsModel)
	for _, acl := range accessACLs {
		var err diag.Diagnostics
		cACL := getOrCreateACL(pointerACLs, acl.ClientCN)
		switch acl.Type {
		case "by_name":
			switch acl.ClientRole {
			case "producer":
				cACL.ProducerByName, err = utils.AppendToSet(ctx, cACL.ProducerByName, acl.TopicNames...)
				diags.Append(err...)
			case "consumer":
				cACL.ConsumerByName, err = utils.AppendToSet(ctx, cACL.ConsumerByName, acl.TopicNames...)
				diags.Append(err...)
			default:
				diags.AddWarning(
					"Sync Kafka consumer/producer by_name ACLs",
					fmt.Sprintf(
						"Client with CN = '%s' has unknown role of ACLs (role equal: %s)",
						acl.ClientCN, acl.ClientRole,
					),
				)
			}
		case "by_mask":
			switch acl.ClientRole {
			case "producer":
				cACL.ProducerByMask, err = utils.AppendToSet(ctx, cACL.ProducerByMask, acl.Name)
				diags.Append(err...)
			case "consumer":
				cACL.ConsumerByMask, err = utils.AppendToSet(ctx, cACL.ConsumerByMask, acl.Name)
				diags.Append(err...)
			default:
				diags.AddWarning(
					"Sync Kafka consumer/producer by_mask ACLs",
					fmt.Sprintf(
						"Client with CN = '%s' has unknown role of ACLs (type equal: %s)",
						acl.ClientCN, acl.ClientRole,
					),
				)
			}
		default:
			diags.AddWarning(
				"Sync Kafka consumer/producer ACLs",
				fmt.Sprintf(
					"Client with CN = '%s' has unknown type of ACLs (type equal: %s)",
					acl.ClientCN, acl.Type,
				),
			)
		}
	}

	for _, acl := range transactionalACLs {
		var err diag.Diagnostics
		cACL := getOrCreateACL(pointerACLs, acl.ClientCN)
		switch acl.Type {
		case "by_name":
			cACL.TransactionalByName, err = utils.AppendToSet(ctx, cACL.TransactionalByName, acl.Value)
			diags.Append(err...)
		case "by_mask":
			cACL.TransactionalByMask, err = utils.AppendToSet(ctx, cACL.TransactionalByMask, acl.Value)
			diags.Append(err...)
		default:
			diags.AddWarning(
				"Sync Kafka transactional ACLs",
				fmt.Sprintf(
					"Client with CN = '%s' has unknown type of ACL (type equal: %s)",
					acl.ClientCN, acl.Type,
				),
			)
		}
	}

	for _, acl := range idempotentACLs {
		cACL := getOrCreateACL(pointerACLs, acl.ClientCN)
		cACL.Idempotent = types.BoolValue(true)
	}

	clientACLs = make(map[string]ClientACLsModel)
	emptySet, _ := basetypes.NewSetValueFrom(ctx, types.StringType, []types.String{})
	for clientCN, pointerACL := range pointerACLs {
		if len(pointerACL.ConsumerByName.Elements()) == 0 {
			pointerACL.ConsumerByName = emptySet
		}
		if len(pointerACL.ConsumerByMask.Elements()) == 0 {
			pointerACL.ConsumerByMask = emptySet
		}
		if len(pointerACL.ProducerByName.Elements()) == 0 {
			pointerACL.ProducerByName = emptySet
		}
		if len(pointerACL.ProducerByMask.Elements()) == 0 {
			pointerACL.ProducerByMask = emptySet
		}
		if len(pointerACL.TransactionalByName.Elements()) == 0 {
			pointerACL.TransactionalByName = emptySet
		}
		if len(pointerACL.TransactionalByMask.Elements()) == 0 {
			pointerACL.TransactionalByMask = emptySet
		}
		clientACLs[clientCN] = *pointerACL
	}
	return clientACLs, diags
}

func convertTerraformACLsToEntities(ctx context.Context, ACLs map[string]ClientACLsModel) (
	accessACLs []entities.KafkaACL,
	transactionalACLs []entities.KafkaTransactionalACL,
	idempotentACLs []entities.KafkaIdempotentACL,
	diags diag.Diagnostics,
) {
	for clientCN, clientACL := range ACLs {
		// prepare consumer acls
		var consumerByName []string
		diag := clientACL.ConsumerByName.ElementsAs(ctx, &consumerByName, false)
		diags.Append(diag...)

		if len(consumerByName) > 0 {
			accessACLs = append(accessACLs, entities.KafkaACL{
				ClientCN:   clientCN,
				ClientRole: "consumer",
				Type:       "by_name",
				TopicNames: consumerByName,
			})
		}

		var consumerByMask = []string{}
		diag = clientACL.ConsumerByMask.ElementsAs(ctx, &consumerByMask, false)
		diags.Append(diag...)

		for _, mask := range consumerByMask {
			accessACLs = append(accessACLs, entities.KafkaACL{
				ClientCN:   clientCN,
				ClientRole: "consumer",
				Type:       "by_mask",
				Name:       mask,
			})
		}

		// prepare producer acls
		var producerByName []string
		diag = clientACL.ProducerByName.ElementsAs(ctx, &producerByName, false)
		diags.Append(diag...)

		if len(producerByName) > 0 {
			accessACLs = append(accessACLs, entities.KafkaACL{
				ClientCN:   clientCN,
				ClientRole: "producer",
				Type:       "by_name",
				TopicNames: producerByName,
			})
		}
		var producerByMask = []string{}
		diag = clientACL.ProducerByMask.ElementsAs(ctx, &producerByMask, false)
		diags.Append(diag...)

		for _, mask := range producerByMask {
			accessACLs = append(accessACLs, entities.KafkaACL{
				ClientCN:   clientCN,
				ClientRole: "producer",
				Type:       "by_mask",
				Name:       mask,
			})
		}
		// prepare transactional acls
		var transactionalByName = []string{}
		diag = clientACL.TransactionalByName.ElementsAs(ctx, &transactionalByName, false)
		diags.Append(diag...)

		for _, name := range transactionalByName {
			transactionalACLs = append(transactionalACLs, entities.KafkaTransactionalACL{
				ClientCN: clientCN,
				Type:     "by_name",
				Value:    name,
			})
		}

		var transactionalByMask = []string{}
		diag = clientACL.TransactionalByMask.ElementsAs(ctx, &transactionalByMask, false)
		diags.Append(diag...)

		for _, mask := range transactionalByMask {
			transactionalACLs = append(transactionalACLs, entities.KafkaTransactionalACL{
				ClientCN: clientCN,
				Type:     "by_mask",
				Value:    mask,
			})
		}

		if clientACL.Idempotent.ValueBool() {
			idempotentACLs = append(idempotentACLs, entities.KafkaIdempotentACL{
				ClientCN: clientCN,
			})
		}
	}
	return
}

// quotas
func applyQuotas(
	order *orders.Kafka,
	planQuotas,
	stateQuotas []QuotaModel,
	diags *diag.Diagnostics,
) {
	// map для cравения квот с учитыванием default типа
	stateQuotasMap := make(map[string]QuotaModel)
	planQuotasMap := make(map[string]QuotaModel)

	for _, q := range stateQuotas {
		if q.Type.ValueString() == "default" {
			stateQuotasMap["default"] = q
		} else {
			stateQuotasMap[q.ClientCN.ValueString()] = q
		}
	}

	for _, q := range planQuotas {
		if q.Type.ValueString() == "default" {
			planQuotasMap["default"] = q
		} else {
			planQuotasMap[q.ClientCN.ValueString()] = q
		}
	}

	toApplyQuotas := []orders.KafkaQuotasBulkAction{}

	for key, planQuota := range planQuotasMap {
		stateQuota, exists := stateQuotasMap[key]

		needsUpdate := !exists || !reflect.DeepEqual(planQuota.ProducerByteRate, stateQuota.ProducerByteRate)

		if needsUpdate {
			toApplyQuota := orders.KafkaQuotasBulkAction{
				QuotaType:        planQuota.Type.ValueString(),
				ProducerByteRate: planQuota.ProducerByteRate.ValueInt64(),
			}

			if planQuota.Type.ValueString() != "default" {
				toApplyQuota.ClientCNs = []string{planQuota.ClientCN.ValueString()}
			}

			toApplyQuotas = append(toApplyQuotas, toApplyQuota)
		}
	}

	if len(toApplyQuotas) > 0 {
		err := order.CreateOrUpdateQuotas(toApplyQuotas, false)
		if err != nil {
			diags.AddError(
				consts.UPDATE_RES_FAIL,
				fmt.Sprintf("Apply quotas ended with error.\nError: %s", err.Error()),
			)
		}
	}
}

func deleteQuotas(
	order *orders.Kafka,
	planQuotas,
	stateQuotas []QuotaModel,
	diags *diag.Diagnostics,
) {
	// map для быстрого поиска квот в плане
	planQuotasMap := make(map[string]QuotaModel)

	var defaultExistsInPlan bool
	for _, quota := range planQuotas {
		if quota.Type.ValueString() == "default" {
			defaultExistsInPlan = true
			planQuotasMap[quota.ClientCN.ValueString()] = quota
		} else {
			planQuotasMap[quota.ClientCN.ValueString()] = quota
		}
	}

	var toDelete []orders.KafkaQuotasBulkAction
	for _, stateQuota := range stateQuotas {

		if stateQuota.Type.ValueString() == "default" && !defaultExistsInPlan {
			toDelete = append(toDelete, orders.KafkaQuotasBulkAction{
				ClientCNs: []string{"<default>"},
			})
			continue
		}

		_, exists := planQuotasMap[stateQuota.ClientCN.ValueString()]
		if !exists {
			toDelete = append(toDelete, orders.KafkaQuotasBulkAction{
				ClientCNs: []string{stateQuota.ClientCN.ValueString()},
			})
		}
	}

	if len(toDelete) > 0 {
		err := order.DeleteQuotas(toDelete, false)
		if err != nil {
			diags.AddError(
				consts.UPDATE_RES_FAIL,
				fmt.Sprintf("Delete quotas ended with error.\nError: %s", err.Error()),
			)
		}
	}
}

func readQuotas(ctx context.Context, kafkaConfig entities.KafkaItemConfig) basetypes.SetValue {

	var actualQuotas []QuotaModel
	for _, quota := range kafkaConfig.Quotas {

		q := QuotaModel{ProducerByteRate: types.Int64Value(quota.ProducerByteRate)}

		if quota.ClientCN == "<default>" {
			q.Type = types.StringValue("default")
		} else {
			q.Type = types.StringValue("personal")
			q.ClientCN = types.StringValue(quota.ClientCN)
		}
		actualQuotas = append(actualQuotas, q)
	}

	quotas, _ := types.SetValueFrom(
		ctx,
		types.ObjectType{AttrTypes: QuotaModel{}.AttributeTypes()},
		actualQuotas,
	)
	return quotas
}

func readACLS(
	kafkaConfig entities.KafkaItemConfig,
	ctx context.Context,
	resp *resource.ReadResponse,
) basetypes.MapValue {

	accessACLs := kafkaConfig.ACLs
	transactionlACLs := kafkaConfig.TransactionalACLs
	idempotentACLs := kafkaConfig.IdempotentACLs

	portalACLs, diags := convertEntitiesACLsToTerraform(ctx, accessACLs, transactionlACLs, idempotentACLs)
	resp.Diagnostics.Append(diags...)
	var tfAcls basetypes.MapValue
	if len(portalACLs) > 0 {
		tfAcls, _ = types.MapValueFrom(
			ctx,
			types.ObjectType{AttrTypes: ClientACLsModel{}.AttributeTypes()},
			portalACLs,
		)
	} else {
		tfAcls = types.MapNull(
			types.ObjectType{AttrTypes: ClientACLsModel{}.AttributeTypes()},
		)
	}

	return tfAcls
}

// Обновление кластера для новой версии Kafka
func upgradeKafkaVersion(
	order *orders.Kafka,
	plan, state *KafkaClusterResourceModel,
) diag.Diagnostics {
	var diag diag.Diagnostics

	versionIsGreather, err := checkKafkaVersionIsGreather(state.KafkaVersion.ValueString(), plan.KafkaVersion.ValueString())
	if err != nil {
		diag.AddAttributeError(
			path.Root("kafka_version"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("error while compare kafka_version: %s", err.Error()),
		)
		return diag
	}
	if !versionIsGreather {
		diag.AddAttributeError(
			path.Root("kafka_version"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("new kafka version: %v smaller current kafka version: %v", plan.KafkaVersion.ValueString(), state.KafkaVersion.ValueString()),
		)
		return diag
	}
	attrs := orders.UpgradeKafkaVersionAttrs{
		Accept:              true,
		CurrentKafkaVersion: state.KafkaVersion.ValueString(),
		NewKafkaVersion:     plan.KafkaVersion.ValueString(),
	}

	err = order.UpgradeKafkaVersion(attrs)
	if err != nil {
		diag.AddAttributeError(
			path.Root("kafka_version"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("error while upgrade_kafka_version: %s", err.Error()),
		)
		return diag
	}

	return diag
}

// Горизонтальное масштабирование
func (r KafkaResource) horizontalScaling(
	order *orders.Kafka,
	plan, state *KafkaClusterResourceModel,
	resp *resource.UpdateResponse,
) {
	err := validateKafkaCount(order)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "error for validate kafka count"),
			err.Error(),
		)
		return
	}

	newLayoutName, err := references.GetGeoDistributionLayoutNameByID(
		r.client.Creds,
		plan.LayoutID.ValueString(),
		"kafka",
		r.client.Organization,
		plan.Core.NetSegmentCode.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "error for fetch layout_name"),
			err.Error(),
		)
		return
	}

	currentLayoutName, err := references.GetGeoDistributionLayoutNameByID(
		r.client.Creds,
		state.LayoutID.ValueString(),
		"kafka",
		r.client.Organization,
		state.Core.NetSegmentCode.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "error for fetch layout_name"),
			err.Error(),
		)
		return
	}

	kafkaQuantity, err := r.compareKafkaCount(currentLayoutName, newLayoutName, plan.Core.NetSegmentCode.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "error for comparing kafka quantity."),
			err.Error(),
		)
		return
	}

	err = order.HorizontalScaling(orders.HorizontalScalingAttrs{
		NewBrokersCount: int64(kafkaQuantity),
		Accept:          true,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "horizontal scaling ended with error"),
			err.Error(),
		)
		return
	}
}

// Обновление версии дистрибутива ВТБ Кафка
func (r KafkaResource) updateKafkaDistribVersion(
	order *orders.Kafka,
	currentVersion string,
	latestVersion string,
	resp *resource.UpdateResponse,
) {
	changedVersion, err := utils.IsVersionOlder(currentVersion, latestVersion)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(
				consts.UPDATE_RES_FAIL,
				"error while comparing versions",
			), err.Error(),
		)
		return
	}

	if !changedVersion {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("build_version"),
			"Current build version is latest, update kafka distrib not supported",
			fmt.Sprintf("Current build version: %v, Latest build version: %v", currentVersion, latestVersion),
		)
		return
	}

	err = order.UpgradeKafkaDistribVersion()
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(
				consts.UPDATE_RES_FAIL, "update kafka distrib version ended with error"), err.Error(),
		)
		return
	}
}

// Изменение имени кластера
func (r KafkaResource) editClusterName(
	order *orders.Kafka,
	plan *KafkaClusterResourceModel,
	resp *resource.UpdateResponse,
) {
	err := order.EditClusterName(plan.ClusterName.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("cluster_name"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("error while edit cluster name: %s", err.Error()),
		)
		return
	}
}

func (r KafkaResource) changeFinancialSource(
	order *orders.Kafka,
	finProjId string,
	resp *resource.UpdateResponse,
) {
	err := order.ChangeFinancialProject(finProjId)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("financial_project"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Change financial source for order ended with error.\nError: %s", err.Error()),
		)
		return
	}
}

// Misc
func (r KafkaResource) prepareAttrs(
	plan KafkaClusterResourceModel,
	layoutName string,
) orders.KafkaAttrs {

	attrs := orders.KafkaAttrs{
		KafkaBasicAttrs: prepareKafkaBasicAttrs(
			plan,
			layoutName,
		),
		ClusterName:     plan.ClusterName.ValueString(),
		Layout:          plan.LayoutID.ValueString(),
		LayoutName:      layoutName,
		KafkaVersion:    plan.KafkaVersion.ValueString(),
		LogRetention:    plan.RetentionMinutes.ValueInt64(),
		GeoDistribution: plan.Image.GeoDistribution.ValueBool(),
		LayoutObj: orders.LayoutObj{
			Name: layoutName,
			UUID: plan.LayoutID.ValueString(),
		},
		ClusterType: "personal",
	}
	if strings.EqualFold(r.client.EnvironmentName, "lt") {
		defaultLtLayout := "default-lt-layout"
		attrs.Performance = &defaultLtLayout
	} else {
		attrs.Performance = nil
	}
	return attrs
}

func (r KafkaResource) compareKafkaCount(currentLayoutName, newLayoutName, netSegment string) (int, error) {
	prefix := r.getKafkaLayoutsPrefix(netSegment)

	kafkaLayouts := map[string]int{
		fmt.Sprintf("%s:kafka-4:zookeeper-3", prefix):  4,
		fmt.Sprintf("%s:kafka-6:zookeeper-3", prefix):  6,
		fmt.Sprintf("%s:kafka-8:zookeeper-3", prefix):  8,
		fmt.Sprintf("%s:kafka-16:zookeeper-3", prefix): 16,
	}
	currentKafkaCount, ok := kafkaLayouts[currentLayoutName]
	if !ok {
		return 0, fmt.Errorf("`current_layout` not found: %s", currentLayoutName)
	}

	newVMCount, ok := kafkaLayouts[newLayoutName]
	if !ok {
		return 0, fmt.Errorf("`new_layout` not found: %s", newLayoutName)
	}

	if currentKafkaCount > newVMCount {
		return 0, fmt.Errorf("`new_layout` must be greater than `current_layout`: `current_layout`=%s (%d), `new_layout`=%s (%d)",
			currentLayoutName, currentKafkaCount, newLayoutName, newVMCount)
	}

	return newVMCount - currentKafkaCount, nil

}

func (r KafkaResource) getKafkaLayoutsPrefix(netSegment string) string {
	prefix := "one_dc"
	switch {
	case strings.EqualFold(r.client.Environment, "prod"):
		prefix = "geo"
	case strings.EqualFold(netSegment, "b2b-hce-ts-dev-srv-app"):
		prefix = "ts"
	}
	return prefix
}

func prepareKafkaBasicAttrs(
	plan KafkaClusterResourceModel,
	layoutName string,
) orders.KafkaBasicAttrs {
	layoutParts := strings.Split(layoutName, ":")

	ba := orders.KafkaBasicAttrs{
		ADIntegration:    plan.Image.ADIntegration.ValueBool(),
		ADLogonGrants:    utils.PrepareADLogonGrants(plan.Access),
		ExtraMounts:      utils.PrepareExtraMountsAttrs(plan.ExtraMounts),
		OnSupport:        plan.Image.OnSupport.ValueBool(),
		OsVersion:        plan.Image.OsVersion.ValueString(),
		AvailabilityZone: plan.Core.Zone.ValueString(),
		Domain:           plan.Core.Domain.ValueString(),
		Platform:         plan.Core.Platform.ValueString(),
		DefaultNic: entities.DefaultNic{
			NetSegment: plan.Core.NetSegmentCode.ValueString(),
		},
		CreatedWithOpenTofu: true,
	}

	if strings.Join(layoutParts[1:3], ":") == "kafka:zookeeper" {
		ba.OneNodeFlavor = &entities.Flavor{
			Cores:  plan.Flavor.Cores.ValueInt64(),
			Memory: plan.Flavor.Memory.ValueInt64(),
			Name:   plan.Flavor.Name.ValueString(),
			UUID:   plan.Flavor.UUID.ValueString(),
		}
	} else {
		ba.Flavor = &entities.Flavor{
			Cores:  plan.Flavor.Cores.ValueInt64(),
			Memory: plan.Flavor.Memory.ValueInt64(),
			Name:   plan.Flavor.Name.ValueString(),
			UUID:   plan.Flavor.UUID.ValueString(),
		}
	}

	return ba
}

func warnDeprecatedACLsFunction(planAcls map[string]ClientACLsModel, diags *diag.Diagnostics, planKafkaVersion string) {
	if planKafkaVersion != "2.13-2.4.1" {
		for cn, acl := range planAcls {
			if acl.Idempotent.ValueBool() {
				diags.AddError(
					"Deprecated function (Idempotent acl)",
					fmt.Sprintln(
						cn, "can't have idepontent acl, "+
							"because all writes already idempotent in kafka version '2.13-2.8.2' or '2.13-3.6.2'",
					),
				)
			}
		}
	}
}

func isTopicsChanged(planTopics []entities.KafkaTopic, orderTopics []entities.KafkaTopic) bool {

	if len(planTopics) != len(orderTopics) {
		return true
	}

	for _, orderTopic := range orderTopics {
		exists := false
		for _, planTopic := range planTopics {
			if orderTopic.Equal(planTopic) {
				exists = true
			}
		}
		if !exists {
			return true
		}
	}
	return false
}

func isTransactionalACLsChanged(planACLs, orderACLs []entities.KafkaTransactionalACL) bool {

	if len(planACLs) != len(orderACLs) {
		return true
	}

	for _, orderACL := range orderACLs {
		exists := false
		for _, planACL := range planACLs {
			if orderACL.Equal(planACL) {
				exists = true
			}
		}
		if !exists {
			return true
		}
	}
	return false
}

func isAccessACLsChagned(planACLs, orderACLs []entities.KafkaACL) bool {

	if len(planACLs) != len(orderACLs) {
		return true
	}

	for _, orderACL := range orderACLs {
		exists := false
		for _, planACL := range planACLs {
			if orderACL.Equal(planACL) {
				exists = true
			}
		}
		if !exists {
			return true
		}
	}
	return false
}

func isIdempotentACLsChanged(planACLs, orderACLs []entities.KafkaIdempotentACL) bool {

	if len(planACLs) != len(orderACLs) {
		return true
	}

	for _, orderACL := range orderACLs {
		exists := false
		for _, planACL := range planACLs {
			if orderACL.Equal(planACL) {
				exists = true
			}
		}
		if !exists {
			return true
		}
	}
	return false
}

func mbToBytes(mb int64) int64 {
	return mb * 1024 * 1024
}

func bytesToMb(bytes int64) int64 {
	return bytes / 1024 / 1024
}

func validateKafkaCount(order *orders.Kafka) error {
	orderItem, err := order.GetParentItem()
	if err != nil {
		return fmt.Errorf(
			"can't get cluster item from portal with order_id '%s': %w",
			order.ID, err,
		)
	}

	kafkaConfig, ok := orderItem.Data.Config.(entities.KafkaItemConfig)
	if !ok {
		return fmt.Errorf("invalid config type for Kafka item")
	}

	if len(strings.Split(kafkaConfig.ConnectionURL, ",")) < 4 {
		return fmt.Errorf("horizontal scale not supported for this layout configuration")
	}

	return nil
}

func checkKafkaVersionIsGreather(currentVersion, newVersion string) (bool, error) {
	currentCleanVersion := strings.Split(currentVersion, "-")[1]
	newCleanVersion := strings.Split(newVersion, "-")[1]

	currentV, err := semver.NewVersion(currentCleanVersion)
	if err != nil {
		return false, fmt.Errorf("error for parsing `current_kafka_version`: %v", err)
	}

	newV, err := semver.NewVersion(newCleanVersion)
	if err != nil {
		return false, fmt.Errorf("error for parsing `new_kafka_version`: %v", err)
	}

	return newV.GreaterThan(currentV), nil

}

func getOrCreateACL(acls map[string]*ClientACLsModel, clientCN string) *ClientACLsModel {
	if acl, exists := acls[clientCN]; exists {
		return acl
	}
	acl := &ClientACLsModel{
		Idempotent: types.BoolValue(false),
	}
	acls[clientCN] = acl
	return acl
}
