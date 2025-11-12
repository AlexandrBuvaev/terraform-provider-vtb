package airflow

import (
	"context"
	"fmt"
	"reflect"
	"regexp"

	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/common"
	"terraform-provider-vtb/internal/consts"
	"terraform-provider-vtb/internal/custommodifires"
	"terraform-provider-vtb/internal/services/core"
	"terraform-provider-vtb/internal/services/flavor"
	"terraform-provider-vtb/internal/utils"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"
	"terraform-provider-vtb/pkg/client/references"
	"terraform-provider-vtb/pkg/client/sources"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	"golang.org/x/exp/slices"
)

var (
	_ resource.Resource                = AirflowStandaloneResource{}
	_ resource.ResourceWithImportState = AirflowStandaloneResource{}
	_ resource.ResourceWithModifyPlan  = AirflowStandaloneResource{}
)

type AirflowStandaloneResource struct {
	client *client.CloudClient
}

func NewAirflowStandaloneResource() resource.Resource {
	return &AirflowStandaloneResource{}
}

func (r AirflowStandaloneResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_airflow_standalone"
}

func (r *AirflowStandaloneResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type AirflowPostgresConfig struct {
	DBUser     types.String `tfsdk:"db_user"`
	DBDatabase types.String `tfsdk:"db_database"`
	DBPassword types.String `tfsdk:"db_password"`
	DBOrderID  types.String `tfsdk:"db_order_id"`
}

type AirflowClientCert struct {
	Postfix  types.String `tfsdk:"cert_name_postfix"`
	Password types.String `tfsdk:"password"`
}

type AirflowStandaloneResourceModel struct {
	Core        core.CoreModel                    `tfsdk:"core"`
	Flavor      flavor.FlavorModel                `tfsdk:"flavor"`
	Image       AirflowImageDataSourceData        `tfsdk:"image"`
	Lifetime    types.Int64                       `tfsdk:"lifetime"`
	Label       types.String                      `tfsdk:"label"`
	OrderID     types.String                      `tfsdk:"order_id"`
	ItemID      types.String                      `tfsdk:"item_id"`
	Access      map[string][]types.String         `tfsdk:"access"`
	ExtraMounts map[string]common.ExtraMountModel `tfsdk:"extra_mounts"`

	BuildVersion     types.String                       `tfsdk:"build_version"`
	Executor         types.String                       `tfsdk:"executor"`
	ClusterName      types.String                       `tfsdk:"cluster_name"`
	DeployGrants     *DeployGrantsModel                 `tfsdk:"deploy_grants"`
	WebConsoleGrants map[string][]basetypes.StringValue `tfsdk:"web_console_grants"`
	PostgreSQLConfig AirflowPostgresConfig              `tfsdk:"postgresql_config"`

	UpdateMode       types.String `tfsdk:"update_product_mode"`
	FinancialProject types.String `tfsdk:"financial_project"`
}

type DeployGrantsModel struct {
	AirflowDeploy types.Set `tfsdk:"airflow_deploy"`
}

func (r AirflowStandaloneResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"core": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.CoreSchema,
				MarkdownDescription: "Основные параметры для вирутальных машин в заказе.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"flavor": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.FlavorSchema,
				MarkdownDescription: "Кол-во CPU/RAM для виртуальных машин.",
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
				MarkdownDescription: "Имя приложения",
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
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Разрешения для входа в Active Directory.",
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
				},
				PlanModifiers: []planmodifier.Map{
					custommodifires.EmptyAccessGroup(),
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
								int64validator.AtMost(consts.CLOUD_EXTRA_MOUNT_MAX_SIZE),
								int64validator.AtLeast(30),
							},
							PlanModifiers: []planmodifier.Int64{
								custommodifires.GrowOnly(),
								custommodifires.MininalGrow(10),
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
				},
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
			},
			"postgresql_config": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Configuration of PostgreSQL component",
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
				Computed: true,
				Default:  stringdefault.StaticString("none"),
				Optional: true,
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

func (r AirflowStandaloneResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r AirflowStandaloneResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan, state AirflowStandaloneResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	}

	checkIsOrderDeleted := utils.CheckOrderIsDeleted(
		r.client,
		plan.OrderID,
		"app",
		"airflow",
	)
	resp.Diagnostics.Append(checkIsOrderDeleted.Diagnostics...)

	utils.HandleExternalDeletionRecreate(
		ctx,
		resp,
		&plan,
		checkIsOrderDeleted.IsDeleted,
		checkIsOrderDeleted.Diagnostics,
	)

	validateAccessForEnvDiags := utils.ValidateActiveDirectoryAccessForEnvironmentsV1(
		r.client,
		plan.Access,
		"DEV",
	)
	resp.Diagnostics.Append(validateAccessForEnvDiags...)

	validateAccessRolesDiags := utils.ValidateAccessRolesV2(
		r.client,
		"app:airflow",
		plan.Access,
	)
	resp.Diagnostics.Append(validateAccessRolesDiags...)

	validateAirflowDeployGrantsDiags := utils.ValidateGroupTypeSet(
		ctx,
		r.client,
		"deploy_grants",
		"service-accounts",
		plan.DeployGrants.AirflowDeploy,
	)
	resp.Diagnostics.Append(validateAirflowDeployGrantsDiags...)

	validateAirflowWebConsoleGrantsDiags := utils.ValidateGroupTypeMap(
		ctx,
		r.client,
		"web_console_grants",
		"personal",
		plan.WebConsoleGrants,
	)

	resp.Diagnostics.Append(validateAirflowWebConsoleGrantsDiags...)

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
						consts.MODIFY_PLAN_FAIL,
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

func (r AirflowStandaloneResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan AirflowStandaloneResourceModel
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
		resp.Diagnostics.AddAttributeError(path.Root("financial_project"), consts.CREATE_RES_FAIL, err.Error())
		return
	}

	plan.Image.ProductType = types.StringValue("stand-alone")
	plan.Executor = types.StringValue("localexecutor")

	deployGroups := []string{}
	resp.Diagnostics.Append(plan.DeployGrants.AirflowDeploy.ElementsAs(ctx, &deployGroups, false)...)
	deployGrants := entities.ADLogonGrants{
		Role:   "airflow_deploy",
		Groups: deployGroups,
	}

	attrs := orders.AirflowStandaloneAttrs{
		BasicAttrs: utils.PrepareBasicAttrs(
			&plan.Flavor,
			&plan.Core,
			plan.Access,
			plan.ExtraMounts,
			plan.Image.OsVersion.ValueString(),
			plan.Image.ADIntegration.ValueBool(),
			plan.Image.OnSupport.ValueBool(),
		),
		OnBackup:        plan.Image.OnBackup.ValueBool(),
		GeoDistribution: plan.Image.GeoDistribution.ValueBool(),
		ExtraNics:       []interface{}{},
		AirflowStandaloneConfig: orders.AirflowStandaloneConfig{
			Executor:         plan.Executor.ValueString(),
			ClusterName:      plan.ClusterName.ValueString(),
			AirflowVersion:   plan.Image.AirflowVersion.ValueString(),
			DeployGrants:     []entities.ADLogonGrants{deployGrants},
			WebConsoleGrants: utils.PrepareADLogonGrants(plan.WebConsoleGrants),
			PostgresqlConfig: orders.AirflowPostgresConfig{
				DBUser:       plan.PostgreSQLConfig.DBUser.ValueString(),
				DBDatabase:   plan.PostgreSQLConfig.DBDatabase.ValueString(),
				DBPassword:   plan.PostgreSQLConfig.DBPassword.ValueString(),
				Notification: true,
				DBOrderID:    plan.PostgreSQLConfig.DBOrderID.ValueString(),
			},
		},
	}

	order := orders.NewAirflowStandalone(
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
		resp.Diagnostics.AddError("Error while getting item", err.Error())
		return
	}

	plan.OrderID = types.StringValue(order.GetOrder().ID)
	plan.ItemID = types.StringValue(orderItem.ID)
	plan.BuildVersion = types.StringValue(orderItem.Data.Build.SetupVersion)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r AirflowStandaloneResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var orderID types.String
	req.State.GetAttribute(ctx, path.Root("order_id"), &orderID)
	if orderID.IsNull() {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
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
	order, err := orders.GetAirflowStandaloneOrder(
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

	if order.Attrs.Executor != "localexecutor" {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Wrong resource type (vtb_airflow_standalone) for airflow of type %s with order_id equal '%s'.",
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
				"Can't get app item from portal with order_id '%s'.\nError:%s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}
	airflowConfig := airflowParentItem.Data.Config.(entities.AirflowStandaloneItemConfig)

	airflowVM, err := order.GetAirflowVMItem()
	if err != nil {
		resp.Diagnostics.AddError(
			"Read order from portal",
			fmt.Sprintf("Can't get airflow VMs items.\nError: %s", err.Error()),
		)
		return
	}
	vmConfig := airflowVM.Data.Config.(entities.VMItemConfig)

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

	state := AirflowStandaloneResourceModel{
		Core: core.CoreModel{
			Platform:       types.StringValue(utils.SelectPlatform(order.Attrs.Platform)),
			Domain:         types.StringValue(order.Attrs.Domain),
			NetSegmentCode: types.StringValue(vmConfig.DefaultNic.NetSegment),
			Zone:           types.StringValue(order.Attrs.AvailabilityZone),
		},
		Flavor: flavor.FlavorModel{
			Cores:  types.Int64Value(vmConfig.Flavor.Cores),
			Memory: types.Int64Value(vmConfig.Flavor.Memory),
			Name:   types.StringValue(vmConfig.Flavor.Name),
			UUID:   types.StringValue(vmConfig.Flavor.UUID),
		},
		Image: AirflowImageDataSourceData{
			Distribution:    types.StringValue(vmConfig.Image.Os.Distribution),
			OsVersion:       types.StringValue(vmConfig.Image.Os.Version),
			OnSupport:       types.BoolValue(order.Attrs.OnSupport),
			OnBackup:        types.BoolValue(order.Attrs.OnBackup),
			ProductID:       types.StringValue(order.ProductID),
			ADIntegration:   types.BoolValue(order.Attrs.ADIntegration),
			ProductType:     types.StringValue("stand-alone"),
			AirflowVersion:  types.StringValue(airflowConfig.CoreVersion),
			GeoDistribution: types.BoolValue(order.Attrs.GeoDistribution),
		},
		Label:   types.StringValue(order.Label),
		OrderID: orderID,
		ItemID:  types.StringValue(airflowParentItem.ID),
		Access:  utils.ReadAccessMapV2(airflowVM.Data.ACLs),
		ExtraMounts: map[string]common.ExtraMountModel{
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
		FinancialProject: types.StringValue(order.FinancialSource.Name),
	}

	var lifetime types.Int64
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("lifetime"), &lifetime)...)
	if !lifetime.IsNull() {
		state.Lifetime = lifetime
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r AirflowStandaloneResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state AirflowStandaloneResourceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	order, err := orders.GetAirflowStandaloneOrder(
		r.client.Creds,
		r.client.ProjectName,
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
			r.client.Organization,
		)

		resp.Diagnostics.AddError(
			"Delete error",
			fmt.Sprintf(
				"Order with order_id '%s' isn't deletable. "+
					"Please, make your order deletable by the portal VTB.Cloud, and repeat again."+
					"\nURL reference: %s",
				order.GetOrder().ID, orderURL,
			),
		)
		return
	}

	err = orders.AirflowDeleteTwoLayer(order, false)
	if err != nil {
		resp.Diagnostics.AddError(consts.DELETE_RES_FAIL, err.Error())
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r AirflowStandaloneResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state AirflowStandaloneResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	order, err := orders.GetAirflowStandaloneOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("order_id"),
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "can't get order from portal"),
			err.Error(),
		)
		return
	}

	airflowImageData, err := references.GetAirflowImageData(
		r.client.Creds,
		state.Image.Distribution.ValueString(),
		"stand-alone",
		r.client.Organization,
		r.client.Environment,
		state.Image.OsVersion.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(consts.READ_RES_FAIL, err.Error())
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

	accessChanged := !reflect.DeepEqual(plan.Access, state.Access)
	if accessChanged {
		resp.Diagnostics.Append(r.changeAccess(order, &state, &plan)...)
	}

	// change deploy groups
	if !plan.DeployGrants.AirflowDeploy.Equal(state.DeployGrants.AirflowDeploy) {
		var deployGrantsPlan = []string{}
		resp.Diagnostics.Append(plan.DeployGrants.AirflowDeploy.ElementsAs(ctx, &deployGrantsPlan, false)...)

		err := orders.AirflowChangeDeployGroups(order, deployGrantsPlan)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("deploy_grants"),
				fmt.Sprintln(consts.UPDATE_RES_FAIL, "deploy groups updating ended with error"),
				err.Error(),
			)
		}
	}

	// change web console groups
	if !utils.IsADLogonGrantsEqual(plan.WebConsoleGrants, state.WebConsoleGrants) {
		consoleGrants := utils.PrepareADLogonGrants(plan.WebConsoleGrants)

		err := orders.AirflowChangeWebConsoleGroups(order, consoleGrants)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("web_console_grants"),
				fmt.Sprintln(consts.UPDATE_RES_FAIL, "console groups updating ended with error"),
				err.Error(),
			)
		}
	}

	// expand mount point
	mountChanged := utils.IsExtraMountChanged(state.ExtraMounts, plan.ExtraMounts)
	if mountChanged {
		r.expandMountPoint(order, &plan, &state, resp)
	}

	// vertical scaling
	if plan.Flavor != state.Flavor {
		flavor := entities.Flavor{
			Cores:  plan.Flavor.Cores.ValueInt64(),
			Memory: plan.Flavor.Memory.ValueInt64(),
			UUID:   plan.Flavor.UUID.ValueString(),
			Name:   plan.Flavor.Name.ValueString(),
		}
		err := order.AirflowVerticalScaling(flavor)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("flavor"),
				fmt.Sprintln(consts.UPDATE_RES_FAIL, "vertical scaling ended with error"),
				err.Error(),
			)
		}
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

func (r *AirflowStandaloneResource) changeAccess(
	order *orders.AirflowStandalone,
	state,
	plan *AirflowStandaloneResourceModel,
) (diags diag.Diagnostics) {

	deletedRoles, changedRoles, addedRoles := r.compareADRoles(*plan, *state)

	if len(addedRoles) > 0 {
		rolesFromCloud, err := references.GetRoles(
			r.client.Creds,
			r.client.Environment,
			"app:airflow",
		)
		if err != nil {
			diags.AddError(consts.UPDATE_RES_FAIL, err.Error())
			return
		}

		permissionsMap := make(map[string][]string)
		for _, value := range rolesFromCloud {
			permissionsMap[value.Data.GroupName] = value.Data.Permissions
		}
		for _, role := range addedRoles {
			err := order.AddAccessGroup(role, permissionsMap[role.Role])
			if err != nil {
				diags.AddError(consts.UPDATE_RES_FAIL, err.Error())
				return
			}
		}
	}

	if len(changedRoles) > 0 {
		for _, role := range changedRoles {
			err := order.ChangeAccessGroup(role)
			if err != nil {
				diags.AddError(consts.UPDATE_RES_FAIL, err.Error())
			}
		}
	}

	if len(deletedRoles) > 0 {
		for _, role := range deletedRoles {
			err := order.DeleteAccessGroup(role)
			if err != nil {
				diags.AddError(consts.UPDATE_RES_FAIL, err.Error())
			}
		}
	}
	return
}

func prepareAccessAttr(schemaAccess map[string][]basetypes.StringValue) map[string][]string {
	access := make(map[string][]string)
	for role, groups := range schemaAccess {
		groupSl := []string{}
		for _, gr := range groups {
			groupSl = append(groupSl, gr.ValueString())
		}
		access[role] = groupSl
	}
	return access
}

func (r *AirflowStandaloneResource) compareADRoles(
	plan, state AirflowStandaloneResourceModel,
) (deletedRoles, changedRoles, addedRoles []entities.ADLogonGrants) {
	accessPlan := prepareAccessAttr(plan.Access)
	accessState := prepareAccessAttr(state.Access)

	for role, groups := range accessState {
		if _, ok := accessPlan[role]; !ok {
			deletedRoles = append(
				deletedRoles,
				entities.ADLogonGrants{Role: role, Groups: groups})
		} else {
			if len(accessPlan[role]) == len(accessState[role]) {
				if utils.IsDifferent(accessPlan[role], accessState[role]) {
					changedRoles = append(
						changedRoles,
						entities.ADLogonGrants{Role: role, Groups: accessPlan[role]},
					)
				}
			} else {
				changedRoles = append(
					changedRoles,
					entities.ADLogonGrants{Role: role, Groups: accessPlan[role]},
				)
			}
		}
	}

	for role, groups := range accessPlan {
		if accessState[role] == nil {
			addedRoles = append(
				addedRoles,
				entities.ADLogonGrants{Role: role, Groups: groups},
			)
		}
	}
	return deletedRoles, changedRoles, addedRoles
}

func (r AirflowStandaloneResource) expandMountPoint(
	order *orders.AirflowStandalone,
	plan *AirflowStandaloneResourceModel,
	state *AirflowStandaloneResourceModel,
	resp *resource.UpdateResponse,
) {
	for mount, data := range plan.ExtraMounts {
		for mountSt, dataSt := range state.ExtraMounts {
			if mountSt == mount {
				delta := data.Size.ValueInt64() - dataSt.Size.ValueInt64()
				if delta >= 10 {
					err := order.ExpandMountPoint(mountSt, delta)
					if err != nil {
						resp.Diagnostics.AddError(
							fmt.Sprintln(consts.UPDATE_RES_FAIL, "error expand mount point"),
							err.Error(),
						)
					}
				}

			}
		}
	}
}

func (r AirflowStandaloneResource) updateReleaseVersion(
	currentVersion string,
	latestVersion string,
	resp *resource.UpdateResponse,
	order *orders.AirflowStandalone,
) {
	changedVersion, err := utils.IsVersionOlder(currentVersion, latestVersion)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "error while comparing release versions."),
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
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "product updating ended with error"),
			err.Error(),
		)
		return
	}
}

func (r AirflowStandaloneResource) changeDbPassword(
	order *orders.AirflowStandalone,
	plan *AirflowStandaloneResourceModel,
	resp *resource.UpdateResponse,
) {
	err := orders.AirflowChangeDBPassword(order, plan.PostgreSQLConfig.DBPassword.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintln("change db password ended with error", err.Error()),
		)
		return
	}
}
