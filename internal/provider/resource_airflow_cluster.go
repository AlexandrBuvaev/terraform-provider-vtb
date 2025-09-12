package provider

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"
	"terraform-provider-vtb/pkg/client/references"
	"terraform-provider-vtb/pkg/client/sources"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                = AirflowClusterResource{}
	_ resource.ResourceWithImportState = AirflowClusterResource{}
	_ resource.ResourceWithModifyPlan  = AirflowClusterResource{}
)

type AirflowClusterResource struct {
	provider *VTBCloudProvider
}

func NewAirflowClusterResource(p *VTBCloudProvider) AirflowClusterResource {
	return AirflowClusterResource{
		provider: p,
	}
}

func (r AirflowClusterResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = "vtb_airflow_cluster"
}

type AirflowRabbitConfig struct {
	BrokerOrderID  types.String `tfsdk:"broker_order_id"`
	BrokerVHost    types.String `tfsdk:"broker_vhost"`
	RabbitMQCertCN types.String `tfsdk:"rabbitmq_cert_cn"`
}

type AirflowClusterResourceModel struct {
	Core            CoreModel                  `tfsdk:"core"`
	FlavorScheduler FlavorModel                `tfsdk:"flavor_scheduler"`
	FlavorWorker    FlavorModel                `tfsdk:"flavor_worker"`
	FlavorWebserver FlavorModel                `tfsdk:"flavor_webserver"`
	Image           AirflowImageDataSourceData `tfsdk:"image"`
	Lifetime        types.Int64                `tfsdk:"lifetime"`
	Label           types.String               `tfsdk:"label"`
	OrderID         types.String               `tfsdk:"order_id"`
	ItemID          types.String               `tfsdk:"item_id"`
	Access          map[string][]types.String  `tfsdk:"access"`
	ExtraMounts     map[string]ExtraMountModel `tfsdk:"extra_mounts"`

	BuildVersion     types.String              `tfsdk:"build_version"`
	Executor         types.String              `tfsdk:"executor"`
	ClusterName      types.String              `tfsdk:"cluster_name"`
	LayoutID         types.String              `tfsdk:"layout_id"`
	DeployGrants     *DeployGrantsModel        `tfsdk:"deploy_grants"`
	WebConsoleGrants map[string][]types.String `tfsdk:"web_console_grants"`
	RabbitMQConfig   AirflowRabbitConfig       `tfsdk:"rabbitmq_config"`
	PostgreSQLConfig AirflowPostgresConfig     `tfsdk:"postgresql_config"`

	UpdateMode       types.String `tfsdk:"update_product_mode"`
	FinancialProject types.String `tfsdk:"financial_project"`
}

func (r AirflowClusterResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"core": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          coreSchema,
				MarkdownDescription: "Основные параметры для вирутальных машин в заказе.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"flavor_worker": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          flavorSchema,
				MarkdownDescription: "Кол-во CPU/RAM для виртуальных машин с ролью worker.",
			},
			"flavor_scheduler": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          flavorSchema,
				MarkdownDescription: "Кол-во CPU/RAM для виртуальных машин с ролью scheduler.",
			},
			"flavor_webserver": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          flavorSchema,
				MarkdownDescription: "Кол-во CPU/RAM для виртуальных машин с ролью webserver.",
			},
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
				MarkdownDescription: "Имя заказа.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"order_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Идентификатор заказа, известен после создания заказа.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					CheckIsDeleted(r.provider, "cluster", "airflow"),
				},
			},
			"item_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Идентификатор сущностей ВМ, известен после создания ВМ.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Имя кластера",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[A-Za-z0-9]+([-][A-Za-z0-9]+)*$`),
						"may contain uppercase/lowercase Latin letters, numbers, punctuation marks ['-']. (cannot start or end with punctuation)",
					),
				},
			},
			"access": schema.MapAttribute{
				Required:            true,
				MarkdownDescription: "Разрешения для входа в Active Directory.",
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
				},
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
					AvailableAccessRole(r.provider, "app:airflow"),
				},
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
			},
			"extra_mounts": schema.MapNestedAttribute{
				Required:            true,
				MarkdownDescription: "Дополнительные точки монтирования для ВМ.",
				Validators: []validator.Map{
					mapvalidator.KeysAre(
						stringvalidator.OneOf(
							"/app",
							"/app_data",
						),
					),
					mapvalidator.SizeBetween(2, 2),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"size": schema.Int64Attribute{
							Required:            true,
							MarkdownDescription: "Размер точки монтирования (в Гб)",
							Validators: []validator.Int64{
								int64validator.AtMost(CLOUD_EXTRA_MOUNT_MAX_SIZE),
								int64validator.AtLeast(30),
							},
							PlanModifiers: []planmodifier.Int64{
								GrowOnly(),
								MininalGrow(10),
							},
						},

						"file_system": schema.StringAttribute{
							Computed:            true,
							Default:             stringdefault.StaticString("xfs"),
							MarkdownDescription: "Тип файловой системы.",
						},
					},
				},
			},
			"layout_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Идентификатор layout из справочника geo_distribution.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
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
					"on_support": schema.BoolAttribute{
						Required:            true,
						MarkdownDescription: "Флаг регулирущий режим работы группы сопровождения.",
					},
					"on_backup": schema.BoolAttribute{
						Required:            true,
						MarkdownDescription: "Флаг постановки на СРК RuBackup",
					},
					"product_id": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Идентификатор продукта.",
					},
					"ad_integration": schema.BoolAttribute{
						Required:            true,
						MarkdownDescription: "Флаг для интеграции виртуальной машины в Active Directory",
					},
					"geo_distribution": schema.BoolAttribute{
						Required:            true,
						MarkdownDescription: "Включение/выключение механизма геораспределения.",
					},

					// extra image data attrs
					"product_type": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Тип Airflow установки: stand-alone или cluster.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
							stringplanmodifier.RequiresReplace(),
						},
					},
					"airflow_version": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Версия ядра Airflow",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"build_version": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Версия релиза продукта",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"executor": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Тип Airflow executor (localexecutor или celeryexecutor).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"deploy_grants": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Группы доступа для добавления DAG-файлов",
				Attributes: map[string]schema.Attribute{
					"airflow_deploy": schema.SetAttribute{
						Required:    true,
						ElementType: types.StringType,
						PlanModifiers: []planmodifier.Set{
							setplanmodifier.UseStateForUnknown(),
							AvailableGroupsType(r.provider, "service-accounts"),
						},
					},
				},
			},
			"web_console_grants": schema.MapAttribute{
				Required:            true,
				MarkdownDescription: "Группы доступа к консоли управления.",
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
					mapvalidator.SizeAtMost(2),
					mapvalidator.KeysAre(
						stringvalidator.OneOf(
							"Operator",
							"Viewer",
						),
					),
				},
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
					AvailableGroupsTypeMap(r.provider, "personal"),
				},
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
			},
			"rabbitmq_config": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Конфигурация компонента RabbitMQ",
				Attributes: map[string]schema.Attribute{
					"broker_order_id": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "OrderID брокера RabbitMQ",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"broker_vhost": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Имя виртуального хоста RabbitMQ",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
							stringplanmodifier.RequiresReplace(),
						},
					},
					"rabbitmq_cert_cn": schema.StringAttribute{
						Computed:            true,
						MarkdownDescription: "CN сертификата RabbitMQ",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
							stringplanmodifier.RequiresReplace(),
						},
					},
				},
			},
			"postgresql_config": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Конфигурация компонента PostgreSQL",
				Attributes: map[string]schema.Attribute{
					"db_user": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Владелец БД",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
							stringplanmodifier.RequiresReplace(),
						},
					},
					"db_order_id": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Order_id заказа PostgreSQL",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"db_database": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Имя БД",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
							stringplanmodifier.RequiresReplace(),
						},
					},
					"db_password": schema.StringAttribute{
						Sensitive:           true,
						Required:            true,
						MarkdownDescription: "Пароль владельца БД",
						Validators: []validator.String{
							stringvalidator.LengthAtLeast(64),
							stringvalidator.LengthAtMost(128),
							stringvalidator.RegexMatches(regexp.MustCompile("[A-Z]"),
								"Password must have 1 uppercase letter"),
							stringvalidator.RegexMatches(regexp.MustCompile("[a-z]"),
								"Password must have 1 lowercase letter"),
							stringvalidator.RegexMatches(regexp.MustCompile("[0-9]"),
								"Password must have 1 numeric character"),
							stringvalidator.RegexMatches(regexp.MustCompile(`^[a-zA-Z0-9\\._-]{64,128}$`),
								"Password must match pattern ^[a-zA-Z0-9\\._-]{64,128}$"),
						},
					},
				},
			},

			"update_product_mode": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("none"),
				MarkdownDescription: `Мод для регулировки политики обновлений продукта("latest/none"), если флаг проставлен в "latest"
				                     то будет активировано действие "Обвновить версию релиза продукта"`,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("latest", "none"),
				},
			},
			"financial_project": schema.StringAttribute{
				Required:            true,
				Description:         "Источник финансирования для заказа.",
				MarkdownDescription: "Источник финансирования для заказа.",
			},
		},
	}
}

func (r AirflowClusterResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r AirflowClusterResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan, state AirflowClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	}

	if op, ok := plan.WebConsoleGrants["Operator"]; ok {
		if vw, ok := plan.WebConsoleGrants["Viewer"]; ok {
			vwSlice := make([]string, 0, len(vw))
			for _, el := range vw {
				vwSlice = append(vwSlice, el.ValueString())
			}
			for _, i := range op {
				if slices.Contains(vwSlice, i.ValueString()) {
					resp.Diagnostics.AddAttributeError(
						path.Root("web_console_grants"),
						MODIFY_PLAN_FAIL,
						fmt.Sprintln("you can`t set the same group for different roles."),
					)
				}
			}
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r AirflowClusterResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan AirflowClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	finProj, err := sources.GetFinancialProjectByName(
		r.provider.Creds,
		r.provider.ProjectName,
		plan.FinancialProject.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("financial_project"), CREATE_RES_FAIL, err.Error())
		return
	}

	// plan.ClusterName = types.StringValue(generateClusterName("airflow-clu")[:16])
	plan.Image.ProductType = types.StringValue("cluster")
	plan.Executor = types.StringValue("celeryexecutor")
	plan.RabbitMQConfig.RabbitMQCertCN = types.StringValue(r.provider.RisID + "-airflow-" + r.provider.EnvPrefix + "-cluster-" + plan.ClusterName.ValueString())

	deployGroups := []string{}
	resp.Diagnostics.Append(plan.DeployGrants.AirflowDeploy.ElementsAs(ctx, &deployGroups, false)...)
	deployGrants := entities.ADLogonGrants{
		Role:   "airflow_deploy",
		Groups: deployGroups,
	}

	attrs := orders.AirflowClusterAttrs{
		BasicAttrs: prepareBasicAttrs(
			&FlavorModel{}, //empty object
			&plan.Core,
			plan.Access,
			plan.ExtraMounts,
			plan.Image.OsVersion.ValueString(),
			plan.Image.ADIntegration.ValueBool(),
			plan.Image.OnSupport.ValueBool(),
		),
		OnBackup: plan.Image.OnBackup.ValueBool(),
		FlavorWorker: entities.Flavor{
			Cores:  plan.FlavorWorker.Cores.ValueInt64(),
			Memory: plan.FlavorWorker.Memory.ValueInt64(),
			Name:   plan.FlavorWorker.Name.ValueString(),
			UUID:   plan.FlavorWorker.UUID.ValueString(),
		},
		FlavorScheduler: entities.Flavor{
			Cores:  plan.FlavorScheduler.Cores.ValueInt64(),
			Memory: plan.FlavorScheduler.Memory.ValueInt64(),
			Name:   plan.FlavorScheduler.Name.ValueString(),
			UUID:   plan.FlavorScheduler.UUID.ValueString(),
		},
		FlavorWebserver: entities.Flavor{
			Cores:  plan.FlavorWebserver.Cores.ValueInt64(),
			Memory: plan.FlavorWebserver.Memory.ValueInt64(),
			Name:   plan.FlavorWebserver.Name.ValueString(),
			UUID:   plan.FlavorWebserver.UUID.ValueString(),
		},
		Layout:          plan.LayoutID.ValueString(),
		GeoDistribution: plan.Image.GeoDistribution.ValueBool(),
		ExtraNics:       []interface{}{},
		AirflowClusterConfig: orders.AirflowClusterConfig{
			Executor:         plan.Executor.ValueString(),
			ClusterName:      plan.ClusterName.ValueString(),
			AirflowVersion:   plan.Image.AirflowVersion.ValueString(),
			DeployGrants:     []entities.ADLogonGrants{deployGrants},
			WebConsoleGrants: prepareADLogonGrants(plan.WebConsoleGrants),
			RabbitMQConfig: orders.AirflowRabbitConfig{
				BrokerOrderID:  plan.RabbitMQConfig.BrokerOrderID.ValueString(),
				BrokerVHost:    plan.RabbitMQConfig.BrokerVHost.ValueString(),
				RabbitMQCertCN: plan.RabbitMQConfig.RabbitMQCertCN.ValueString(),
			},
			PostgresqlConfig: orders.AirflowPostgresConfig{
				DBUser:       plan.PostgreSQLConfig.DBUser.ValueString(),
				DBDatabase:   plan.PostgreSQLConfig.DBDatabase.ValueString(),
				DBPassword:   plan.PostgreSQLConfig.DBPassword.ValueString(),
				Notification: true,
				DBOrderID:    plan.PostgreSQLConfig.DBOrderID.ValueString(),
			},
		},
	}

	order := orders.NewAirflowCluster(
		r.provider.Creds,
		r.provider.ProjectName,
		plan.Image.ProductID.ValueString(),
		attrs,
	)

	err = order.Create(orders.CreateOrderPayload{
		Label:        plan.Label.ValueString(),
		Lifetime:     int(plan.Lifetime.ValueInt64()),
		FinProjectID: finProj.ID,
	})
	if err != nil {
		resp.Diagnostics.AddError(CREATE_RES_FAIL, err.Error())
		return
	}

	orderItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError("Error while getting item", err.Error())
		return
	}

	plan.OrderID = types.StringValue(order.GetOrder().ID)
	plan.ItemID = types.StringValue(orderItem.ID)
	plan.BuildVersion = types.StringValue(orderItem.Data.Build.SetupVersion)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r AirflowClusterResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var orderID types.String
	req.State.GetAttribute(ctx, path.Root("order_id"), &orderID)
	if orderID.IsNull() {
		resp.Diagnostics.AddError(
			READ_RES_FAIL,
			"Resource hasn't order_id. You can't get order from portal without order_id",
		)
		return
	}

	var dbpas types.String
	req.State.GetAttribute(ctx, path.Root("postgresql_config").AtName("db_password"), &dbpas)
	if orderID.IsNull() {
		dbpas = types.StringValue("")
	}

	// Get order data
	order, err := orders.GetAirflowClusterOrder(
		r.provider.Creds,
		r.provider.ProjectName,
		orderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			READ_RES_FAIL,
			fmt.Sprintf(
				"Can't read order with order_id equal '%s'.\nError:%s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}

	if order.Attrs.Executor != "celeryexecutor" {
		resp.Diagnostics.AddError(
			READ_RES_FAIL,
			fmt.Sprintf(
				"Wrong resource type (vtb_airflow_cluster) for airflow of type %s with order_id equal '%s'.",
				order.Attrs.Executor, orderID.ValueString(),
			),
		)
		return
	}

	airflowParentItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			"Read order from portal",
			fmt.Sprintf(
				"Can't get cluster item from portal with order_id '%s'.\nError:%s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}
	airflowConfig := airflowParentItem.Data.Config.(entities.AirflowClusterItemConfig)

	airflowVMs, err := order.GetAirflowVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			"Read order from portal",
			fmt.Sprintf("Can't get airflow VMs items.\nError: %s", err.Error()),
		)
		return
	}
	vmConfig := airflowVMs[0].Data.Config.(entities.VMItemConfig)

	schedulerItems, err := order.GetAirflowItemsByType("scheduler")
	if err != nil {
		resp.Diagnostics.AddError(
			"Read order from portal",
			fmt.Sprintf("Can't get airflow VMs items with role scheduler.\nError: %s", err.Error()),
		)
		return
	}
	schedulerFlavor := schedulerItems[0].Data.Config.(entities.VMItemConfig).Flavor
	schedulerCount := len(schedulerItems)

	workerItems, err := order.GetAirflowItemsByType("worker")
	if err != nil {
		resp.Diagnostics.AddError(
			"Read order from portal",
			fmt.Sprintf("Can't get airflow VMs items with role scheduler.\nError: %s", err.Error()),
		)
		return
	}
	workerFlavor := workerItems[0].Data.Config.(entities.VMItemConfig).Flavor
	workerCount := len(workerItems)

	webserverItems, err := order.GetAirflowItemsByType("webserver")
	if err != nil {
		resp.Diagnostics.AddError(
			"Read order from portal",
			fmt.Sprintf("Can't get airflow VMs items with role scheduler.\nError: %s", err.Error()),
		)
		return
	}
	webserverFlavor := webserverItems[0].Data.Config.(entities.VMItemConfig).Flavor
	webserverCount := len(webserverItems)

	prefix := "one_dc"
	if strings.EqualFold(r.provider.Environment, "prod") {
		prefix = "geo"
	}
	layoutName := fmt.Sprintf("%s:webserver-%d:scheduler-%d:worker-%d", prefix, webserverCount, schedulerCount, workerCount)

	layoutId, err := references.GetGeoDistributionLayoutID(
		r.provider.Creds,
		layoutName,
		"airflow",
		r.provider.Organization,
		vmConfig.DefaultNic.NetSegment,
	)
	if err != nil {
		resp.Diagnostics.AddError(READ_RES_FAIL, err.Error())
	}

	appMount, err := order.GetAirflowExtraMount("/app")
	if err != nil {
		resp.Diagnostics.AddError(
			"Read order from portal",
			fmt.Sprintf("Can't get Airflow extra mounts.\nError: %s", err.Error()),
		)
		return
	}

	appDataMount, err := order.GetAirflowExtraMount("/app_data")
	if err != nil {
		resp.Diagnostics.AddError(
			"Read order from portal",
			fmt.Sprintf("Can't get Airflow extra mounts.\nError: %s", err.Error()),
		)
		return
	}

	deployGroups, diag := types.SetValueFrom(ctx, types.StringType, airflowConfig.DeployGroups.Groups)
	resp.Diagnostics.Append(diag...)

	consoleGroups := make(map[string][]basetypes.StringValue, len(airflowConfig.LDAPGroups))
	for _, item := range airflowConfig.LDAPGroups {
		for _, group := range item.Groups {
			consoleGroups[item.Role] = append(consoleGroups[item.Role], types.StringValue(group))
		}
	}

	state := AirflowClusterResourceModel{
		Core: CoreModel{
			Platform:       types.StringValue(selectPlatform(order.Attrs.Platform)),
			Domain:         types.StringValue(order.Attrs.Domain),
			NetSegmentCode: types.StringValue(vmConfig.DefaultNic.NetSegment),
			Zone:           types.StringValue(order.Attrs.AvailabilityZone),
		},
		FlavorScheduler: FlavorModel{
			Cores:  types.Int64Value(schedulerFlavor.Cores),
			Memory: types.Int64Value(schedulerFlavor.Memory),
			Name:   types.StringValue(schedulerFlavor.Name),
			UUID:   types.StringValue(schedulerFlavor.UUID),
		},
		FlavorWorker: FlavorModel{
			Cores:  types.Int64Value(workerFlavor.Cores),
			Memory: types.Int64Value(workerFlavor.Memory),
			Name:   types.StringValue(workerFlavor.Name),
			UUID:   types.StringValue(workerFlavor.UUID),
		},
		FlavorWebserver: FlavorModel{
			Cores:  types.Int64Value(webserverFlavor.Cores),
			Memory: types.Int64Value(webserverFlavor.Memory),
			Name:   types.StringValue(webserverFlavor.Name),
			UUID:   types.StringValue(webserverFlavor.UUID),
		},
		Image: AirflowImageDataSourceData{
			Distribution:    types.StringValue(vmConfig.Image.Os.Distribution),
			OsVersion:       types.StringValue(vmConfig.Image.Os.Version),
			OnSupport:       types.BoolValue(order.Attrs.OnSupport),
			OnBackup:        types.BoolValue(order.Attrs.OnBackup),
			ProductID:       types.StringValue(order.ProductID),
			ADIntegration:   types.BoolValue(order.Attrs.ADIntegration),
			ProductType:     types.StringValue("cluster"),
			AirflowVersion:  types.StringValue(airflowConfig.CoreVersion),
			GeoDistribution: types.BoolValue(order.Attrs.GeoDistribution),
		},
		Label:    types.StringValue(order.Label),
		OrderID:  orderID,
		ItemID:   types.StringValue(airflowParentItem.ID),
		Access:   readAccessMapV2(airflowVMs[0].Data.ACLs),
		LayoutID: types.StringValue(layoutId),
		ExtraMounts: map[string]ExtraMountModel{
			"/app": {
				Size:       types.Int64Value(appMount.Size),
				FileSystem: types.StringValue(appMount.FileSystem),
			},
			"/app_data": {
				Size:       types.Int64Value(appDataMount.Size),
				FileSystem: types.StringValue(appDataMount.FileSystem),
			},
		},
		UpdateMode:   types.StringValue("none"),
		BuildVersion: types.StringValue(airflowParentItem.Data.Build.SetupVersion),
		Executor:     types.StringValue(order.Attrs.Executor),
		ClusterName:  types.StringValue(airflowConfig.ClusterName),
		DeployGrants: &DeployGrantsModel{
			AirflowDeploy: deployGroups,
		},
		WebConsoleGrants: consoleGroups,
		PostgreSQLConfig: AirflowPostgresConfig{
			DBUser:     types.StringValue(airflowConfig.PostgreSQLConfig.User),
			DBDatabase: types.StringValue(airflowConfig.PostgreSQLConfig.Database),
			DBPassword: dbpas,
			DBOrderID:  types.StringValue(order.Attrs.PostgresqlConfig.DBOrderID),
		},
		RabbitMQConfig: AirflowRabbitConfig{
			BrokerOrderID:  types.StringValue(order.Attrs.RabbitMQConfig.BrokerOrderID),
			BrokerVHost:    types.StringValue(airflowConfig.RabbitMQConfig.VHost),
			RabbitMQCertCN: types.StringValue(airflowConfig.RabbitMQConfig.RabbitMQCertCN),
		},
		FinancialProject: types.StringValue(order.FinancialSource.Name),
	}

	var lifetime types.Int64
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("lifetime"), &lifetime)...)
	if !lifetime.IsNull() {
		state.Lifetime = lifetime
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r AirflowClusterResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state AirflowClusterResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	order, err := orders.GetAirflowClusterOrder(
		r.provider.Creds,
		r.provider.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		diags.AddError("Can't get order from portal: ", err.Error())
		resp.Diagnostics.Append(diags...)
	}

	orderState, err := order.GetState()
	if err != nil {
		resp.Diagnostics.AddWarning(
			"Delete resource",
			fmt.Sprintf("Can't get state from portal.\nError:%s", err.Error()),
		)
		return
	}

	if orderState == "deleted" || orderState == "deprovisioned" {
		resp.Diagnostics.AddWarning(
			"Delete resource",
			fmt.Sprintf(
				"Order with id '%s' was deleted in a way other than terraform. Resource was recreated.",
				state.OrderID.ValueString(),
			),
		)
		return
	}

	if !order.GetOrder().Deletable {
		orderURL := fmt.Sprintf(
			"https://cloud.vtb.ru/all/orders/%s/main?context=%s&type=project&org=%s",
			order.GetOrder().ID,
			order.GetOrder().ProjectName,
			r.provider.Organization,
		)

		resp.Diagnostics.AddError(
			"Delete error",
			fmt.Sprintf(
				"Order with order_id '%s' isn't deletable. "+
					"Please, make your order is deletable by the portal VTB.Cloud, and repeat again."+
					"\nURL reference: %s",
				order.GetOrder().ID, orderURL,
			),
		)
		return
	}

	err = orders.AirflowDeleteTwoLayer(order, false)
	if err != nil {
		resp.Diagnostics.AddError(DELETE_RES_FAIL, err.Error())
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r AirflowClusterResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state AirflowClusterResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	order, err := orders.GetAirflowClusterOrder(
		r.provider.Creds,
		r.provider.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("order_id"),
			fmt.Sprintln(UPDATE_RES_FAIL, "can't get order from portal"),
			err.Error(),
		)
		return
	}

	airflowImageData, err := references.GetAirflowImageData(
		r.provider.Creds,
		state.Image.Distribution.ValueString(),
		"cluster",
		r.provider.Organization,
		r.provider.Environment,
		state.Image.OsVersion.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(READ_RES_FAIL, err.Error())
		return
	}
	currentReleaseVersion := state.BuildVersion.ValueString()
	latestReleaseVersion := airflowImageData.LatestReleaseVersion

	// change order label
	labelChanged := plan.Label != state.Label
	if labelChanged {
		err = order.ChangeLabel(plan.Label.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Change order label",
				fmt.Sprintf(
					"Changing order label ended with error.\nError message: %s",
					err.Error(),
				),
			)
			return
		}
		resp.State.SetAttribute(ctx, path.Root("label"), plan.Label)
	}

	// change deploy groups
	if !plan.DeployGrants.AirflowDeploy.Equal(state.DeployGrants.AirflowDeploy) {
		var deployGrantsPlan = []string{}
		resp.Diagnostics.Append(plan.DeployGrants.AirflowDeploy.ElementsAs(ctx, &deployGrantsPlan, false)...)

		err := orders.AirflowChangeDeployGroups(order, deployGrantsPlan)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("deploy_grants"),
				fmt.Sprintln(UPDATE_RES_FAIL, "deploy groups updating ended with error"),
				err.Error(),
			)
		}
	}

	// change web console groups
	if !isADLogonGrantsEqual(plan.WebConsoleGrants, state.WebConsoleGrants) {
		consoleGrants := prepareADLogonGrants(plan.WebConsoleGrants)

		err := orders.AirflowChangeWebConsoleGroups(order, consoleGrants)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("web_console_grants"),
				fmt.Sprintln(UPDATE_RES_FAIL, "console groups updating ended with error"),
				err.Error(),
			)
		}
	}

	// expand mount point
	mountChanged := isExtraMountChanged(state.ExtraMounts, plan.ExtraMounts)
	if mountChanged {
		r.expandMountPoint(order, &plan, &state, resp)
	}

	// vertical scaling
	flavorChanged := plan.FlavorScheduler != state.FlavorScheduler || plan.FlavorWorker != state.FlavorWorker || plan.FlavorWebserver != state.FlavorWebserver
	if flavorChanged {
		flavorScheduler := entities.Flavor{
			Cores:  plan.FlavorScheduler.Cores.ValueInt64(),
			Memory: plan.FlavorScheduler.Memory.ValueInt64(),
			UUID:   plan.FlavorScheduler.UUID.ValueString(),
			Name:   plan.FlavorScheduler.Name.ValueString(),
		}
		flavorWorker := entities.Flavor{
			Cores:  plan.FlavorWorker.Cores.ValueInt64(),
			Memory: plan.FlavorWorker.Memory.ValueInt64(),
			UUID:   plan.FlavorWorker.UUID.ValueString(),
			Name:   plan.FlavorWorker.Name.ValueString(),
		}
		flavorWebserver := entities.Flavor{
			Cores:  plan.FlavorWebserver.Cores.ValueInt64(),
			Memory: plan.FlavorWebserver.Memory.ValueInt64(),
			UUID:   plan.FlavorWebserver.UUID.ValueString(),
			Name:   plan.FlavorWebserver.Name.ValueString(),
		}
		attrs := orders.AirflowVerticalScalingAttrs{
			CheckAgree:      true,
			FlavorWorker:    flavorWorker,
			FlavorScheduler: flavorScheduler,
			FlavorWebserver: flavorWebserver,
			Executor:        "celeryexecutor",
		}
		err := order.AirflowVerticalScaling(attrs)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("flavor_scheduler"),
				fmt.Sprintln(UPDATE_RES_FAIL, "vertical scaling ended with error"),
				err.Error(),
			)
		}
	}

	// horizontal scaling
	if !plan.LayoutID.Equal(state.LayoutID) {
		r.horizontalScaling(order, &plan, &state, resp)
	}

	// update product
	if !plan.UpdateMode.IsNull() && plan.UpdateMode.ValueString() == "latest" {
		r.updateReleaseVersion(currentReleaseVersion, latestReleaseVersion, resp, order)
	}

	// change db password
	if !plan.PostgreSQLConfig.DBPassword.Equal(state.PostgreSQLConfig.DBPassword) {
		r.changeDbPassword(order, &plan, resp)
	}

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r AirflowClusterResource) horizontalScaling(
	order *orders.AirflowCluster,
	plan *AirflowClusterResourceModel,
	state *AirflowClusterResourceModel,
	resp *resource.UpdateResponse,
) {
	newlayoutName, err := references.GetGeoDistributionLayoutNameByID(
		r.provider.Creds,
		plan.LayoutID.ValueString(),
		"airflow",
		r.provider.Organization,
		plan.Core.NetSegmentCode.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(UPDATE_RES_FAIL, "error for fetch layout"),
			err.Error(),
		)
	}
	currentLayoutName, err := references.GetGeoDistributionLayoutNameByID(
		r.provider.Creds,
		state.LayoutID.ValueString(),
		"airflow",
		r.provider.Organization,
		state.Core.NetSegmentCode.ValueString(),
	)

	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(UPDATE_RES_FAIL, "error for fetch layout"),
			err.Error(),
		)
	}

	deltaW, deltaS, err := validateAirflowVMCount(currentLayoutName, newlayoutName)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(UPDATE_RES_FAIL, "validation failed"),
			err.Error(),
		)
		return
	}

	flavorWorker := entities.Flavor{
		Cores:  plan.FlavorWorker.Cores.ValueInt64(),
		Memory: plan.FlavorWorker.Memory.ValueInt64(),
		UUID:   plan.FlavorWorker.UUID.ValueString(),
		Name:   plan.FlavorWorker.Name.ValueString(),
	}

	flavorScheduler := entities.Flavor{
		Cores:  plan.FlavorScheduler.Cores.ValueInt64(),
		Memory: plan.FlavorScheduler.Memory.ValueInt64(),
		UUID:   plan.FlavorScheduler.UUID.ValueString(),
		Name:   plan.FlavorScheduler.Name.ValueString(),
	}

	attrs := orders.AirflowHorizontalScalingAttrs{
		FlavorWorker:      flavorWorker,
		NewWorkerCount:    deltaW,
		FlavorScheduler:   flavorScheduler,
		NewSchedulerCount: deltaS,
	}

	err = order.AirflowHorizontalScaling(attrs)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("gorizontal scaling ended with error, attrs=%v", attrs),
			err.Error(),
		)
	}
}

func (r AirflowClusterResource) expandMountPoint(
	order *orders.AirflowCluster,
	plan *AirflowClusterResourceModel,
	state *AirflowClusterResourceModel,
	resp *resource.UpdateResponse,
) {
	for mount, data := range plan.ExtraMounts {
		for mountSt, dataSt := range state.ExtraMounts {
			if mountSt == mount {
				delta := data.Size.ValueInt64() - dataSt.Size.ValueInt64()
				if delta >= 10 {
					err := orders.AirflowExpandMountPoint(order, mountSt, delta)
					if err != nil {
						resp.Diagnostics.AddError(
							fmt.Sprintln(UPDATE_RES_FAIL, "error expand mount point"),
							err.Error(),
						)
					}
				}
			}
		}
	}
}

func (r AirflowClusterResource) updateReleaseVersion(
	currentVersion string,
	latestVersion string,
	resp *resource.UpdateResponse,
	order *orders.AirflowCluster,
) {
	changedVersion, err := isVersionOlder(currentVersion, latestVersion)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(UPDATE_RES_FAIL, "error while comparing release versions."),
			err.Error(),
		)
		return
	}

	if !changedVersion {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("build_version"),
			"Current version is latest",
			fmt.Sprintf("Current release version: %v, Latest release version: %v", currentVersion, latestVersion),
		)
		return
	}

	err = orders.AirflowUpdateProduct(order, "localexecutor")
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("update_product_mode"),
			fmt.Sprintln(UPDATE_RES_FAIL, "product updating ended with error"),
			err.Error(),
		)
		return
	}
}

func (r AirflowClusterResource) changeDbPassword(
	order *orders.AirflowCluster,
	plan *AirflowClusterResourceModel,
	resp *resource.UpdateResponse,
) {
	err := orders.AirflowChangeDBPassword(order, plan.PostgreSQLConfig.DBPassword.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			UPDATE_RES_FAIL,
			fmt.Sprintln("change db password ended with error", err.Error()),
		)
		return
	}
}

func validateAirflowVMCount(currentLayout, newLayout string) (deltaW int, deltaS int, err error) {
	re_worker := regexp.MustCompile(`worker-(\d+)`)
	re_scheduler := regexp.MustCompile(`scheduler-(\d+)`)

	currentWorkerCount := re_worker.FindStringSubmatch(currentLayout)
	currentSchedulerCount := re_scheduler.FindStringSubmatch(currentLayout)

	newWorkerCount := re_worker.FindStringSubmatch(newLayout)
	newSchedulerCount := re_scheduler.FindStringSubmatch(newLayout)

	curWCount, err := strconv.Atoi(currentWorkerCount[1])
	if err != nil {
		return 0, 0, err
	}
	curSCount, err := strconv.Atoi(currentSchedulerCount[1])
	if err != nil {
		return 0, 0, err
	}
	newSCount, err := strconv.Atoi(newSchedulerCount[1])
	if err != nil {
		return 0, 0, err
	}
	newWCount, err := strconv.Atoi(newWorkerCount[1])
	if err != nil {
		return 0, 0, err
	}

	if curWCount > newWCount || curSCount > newSCount {
		return 0, 0, fmt.Errorf("scaling is not available: it is not possible to delete VM from cluster")
	}

	deltaW = newWCount - curWCount
	deltaS = newSCount - curSCount

	if deltaS+deltaW > 2 {
		return 0, 0, fmt.Errorf("scaling is not available: it is not possible to add more then 2 VMs")
	}

	return deltaW, deltaS, nil
}
