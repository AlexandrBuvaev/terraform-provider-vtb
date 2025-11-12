package scylladb

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"

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

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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

var (
	_ resource.Resource                = &ScyllaDbClusterResource{}
	_ resource.ResourceWithImportState = &ScyllaDbClusterResource{}
	_ resource.ResourceWithModifyPlan  = &ScyllaDbClusterResource{}
)

type ScyllaDbClusterResource struct {
	client *client.CloudClient
}

func NewScyllaDbClusterResource() resource.Resource {
	return &ScyllaDbClusterResource{}
}

func (r ScyllaDbClusterResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_scylla_db_cluster_instance"
}

func (r *ScyllaDbClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type ScyllaDbClusterResourceModel struct {
	Core                       core.CoreModel                    `tfsdk:"core"`
	Flavor                     flavor.FlavorModel                `tfsdk:"flavor"`
	Image                      common.ImageStandardModel         `tfsdk:"image"`
	Lifetime                   types.Int64                       `tfsdk:"lifetime"`
	Label                      types.String                      `tfsdk:"label"`
	OrderID                    types.String                      `tfsdk:"order_id"`
	ItemID                     types.String                      `tfsdk:"item_id"`
	Access                     map[string][]types.String         `tfsdk:"access"`
	FinancialProject           types.String                      `tfsdk:"financial_project"`
	ScyllaDbVersion            types.String                      `tfsdk:"scylladb_version"`
	ExtraMountsLog             map[string]common.ExtraMountModel `tfsdk:"extra_mounts_log"`
	ExtraMountsData            map[string]common.ExtraMountModel `tfsdk:"extra_mounts_data"`
	Databases                  types.Set                         `tfsdk:"db_names"`
	DbUsers                    map[string]ScyllaDbUsersModel     `tfsdk:"db_users"`
	DbPermissions              types.Set                         `tfsdk:"db_permissions"`
	ScyllaClusterConfiguration ScyllaClusterConfigurationModel   `tfsdk:"scylla_cluster_configuration"`
}

type ScyllaClusterConfigurationModel struct {
	DC1 types.Int64 `tfsdk:"dc1"`
	DC2 types.Int64 `tfsdk:"dc2"`
	DC3 types.Int64 `tfsdk:"dc3"`
}

type ScyllaDbUsersModel struct {
	DbmsRole     types.String `tfsdk:"dbms_role"`
	UserPassword types.String `tfsdk:"user_password"`
}

func (r ScyllaDbClusterResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"core": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.CoreSchema,
				Description:         "Основные параметры для виртуальных машин в заказе.",
				MarkdownDescription: "Основные параметры для виртуальных машин в заказе.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"flavor": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.FlavorSchema,
				Description:         "Кол-во CPU/RAM для виртуальных машин.",
				MarkdownDescription: "Кол-во CPU/RAM для виртуальных машин.",
			},
			"image": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.ImageStandardSchema,
				MarkdownDescription: "Тип вычислительного экземпляра.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
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
				Description:         "Метка заказа.",
				MarkdownDescription: "Метка заказа.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"order_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Идентификатор заказа. Становиться известен после создания заказа.",
				MarkdownDescription: "Идентификатор заказа. Становиться известен после создания заказа.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"item_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Идентификатор сущностей ВМ, известен после создания ВМ.",
				MarkdownDescription: "Идентификатор сущностей ВМ, известен после создания ВМ.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"access": schema.MapAttribute{
				Optional: true,
				Computed: true,
				Description: "Словарь,где ключом является роль, а значением список групп, " +
					"которые предоставляют доступ для входа в Active Directory",
				MarkdownDescription: "Словарь,где ключом является роль, а значением список групп, " +
					"которые предоставляют доступ для входа в Active Directory",
				PlanModifiers: []planmodifier.Map{
					custommodifires.EmptyAccessGroup(),
				},
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
				Validators: []validator.Map{
					customvalidators.EmptyAccessGroupListValidator{},
				},
			},
			"scylladb_version": schema.StringAttribute{
				MarkdownDescription: "Версия ScyllaDB",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("5.4.4"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"financial_project": schema.StringAttribute{
				Required:            true,
				Description:         "Источник финансирования.",
				MarkdownDescription: "Источник финансирования.",
			},
			"extra_mounts_data": schema.MapNestedAttribute{
				Required:            true,
				MarkdownDescription: "Точка монтирования /app/scylla/data",
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.OneOf("/app/scylla/data")),
					mapvalidator.SizeBetween(1, 1),
				},
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"size": schema.Int64Attribute{
							Required:            true,
							MarkdownDescription: "Размер точки монтирования (в ГБ.)",
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
							Description:         "Тип файловой системы.",
							MarkdownDescription: "Тип файловой системы.",
						},
					},
				},
			},
			"extra_mounts_log": schema.MapNestedAttribute{
				Required:            true,
				MarkdownDescription: "Точка монтирования /app/scylla/logs",
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.OneOf("/app/scylla/logs")),
					mapvalidator.SizeBetween(1, 1),
				},
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"size": schema.Int64Attribute{
							Required:            true,
							MarkdownDescription: "Размер точки монтирования (в ГБ.)",
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
							Description:         "Тип файловой системы.",
							MarkdownDescription: "Тип файловой системы.",
						},
					},
				},
			},
			"db_users": schema.MapNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Пользователи",
				Validators: []validator.Map{
					mapvalidator.KeysAre(
						stringvalidator.LengthBetween(3, 16),
						stringvalidator.NoneOf("user", "admin"),
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^[a-z0-9_]{0,15}[a-z0-9]$`),
							"Ошибка проверки Regexp: '^(?!(?:user|admin)$)[a-z0-9_]{0,15}[a-z0-9]$'. Запрещено использовать ключевые слова 'user' и 'admin'. Допускается использование только латинских строчных букв и цифр. От 3 до 16 символов",
						),
					),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"dbms_role": schema.StringAttribute{
							Required:            true,
							Description:         "Роль пользователя БД.  Изменять права доступа в дальнейшем не представляется возможным",
							MarkdownDescription: "Роль пользователя БД.",
							Validators: []validator.String{
								stringvalidator.OneOf("admin", "user", "reader"),
							},
						},
						"user_password": schema.StringAttribute{
							Required:            true,
							Sensitive:           true,
							Description:         "Пароль пользователя БД",
							MarkdownDescription: "Пароль пользователя БД",
							Validators: []validator.String{
								stringvalidator.LengthBetween(16, 63),
								stringvalidator.RegexMatches(regexp.MustCompile("[A-Z]"),
									"Password must have 1 uppercase letter"),
								stringvalidator.RegexMatches(regexp.MustCompile("[a-z]"),
									"Password must have 1 lowercase letter"),
								stringvalidator.RegexMatches(regexp.MustCompile("[0-9]"),
									"Password must have 1 numeric character"),
								stringvalidator.RegexMatches(regexp.MustCompile(`^[a-zA-Z0-9\\._-]{16,63}$`),
									"Password must match pattern ^[a-zA-Z0-9\\._-]{16,63}$"),
							},
						},
					},
				},
			},
			"db_permissions": schema.SetAttribute{
				Optional:            true,
				MarkdownDescription: "Права доступа пользователю БД",
				ElementType:         types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^\w+:\w+$`),
							"Права доступа указываются в формате user:db",
						),
					),
				},
			},
			"db_names": schema.SetAttribute{
				Optional:            true,
				MarkdownDescription: "Базы данных",
				ElementType:         types.StringType,
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.LengthBetween(3, 32),
						stringvalidator.NoneOf("user$", "reader$", "admin$", "role", "system", "default", "auditor", "maintenance", "user_backup", "user_repaire", "dba", "cassandra", "scylla", "root"),
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^[a-z][a-z0-9_]{1,30}[a-z0-9]$.*`),
							"Ошибка проверки Regexp: '^(?!user$|reader$|admin$)(?!role|system|default)(?!.*(auditor|maintenance|user_backup|user_repaire|dba|cassandra|scylla|root))(?=[a-z][a-z0-9_]{1,30}[a-z0-9]$).*'. Может содержать строчные латинские буквы от «a» до «z», числовые значения и символ «_». Длина имени от 3 до 32 символов",
						),
					),
				},
			},
			"scylla_cluster_configuration": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Конфигурация кластера",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"dc1": schema.Int64Attribute{
						Required:            true,
						MarkdownDescription: "Количество нод в виртуальном ЦОД 1",
					},
					"dc2": schema.Int64Attribute{
						Required:            true,
						MarkdownDescription: "Количество нод в виртуальном ЦОД 2",
					},
					"dc3": schema.Int64Attribute{
						Required:            true,
						MarkdownDescription: "Количество нод в виртуальном ЦОД 3",
					},
				},
			},
		},
	}
}

func (r ScyllaDbClusterResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r ScyllaDbClusterResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan ScyllaDbClusterResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	checkIsDeletedOrder := utils.CheckOrderIsDeleted(
		r.client,
		plan.OrderID,
		"cluster",
		"scylladb",
	)

	utils.HandleExternalDeletionRecreate(
		ctx,
		resp,
		&plan,
		checkIsDeletedOrder.IsDeleted,
		checkIsDeletedOrder.Diagnostics,
	)

	validateAvaliableAccessForEnvDiags := utils.ValidateActiveDirectoryAccessForEnvironmentsV1(
		r.client,
		plan.Access,
		"DEV",
	)
	resp.Diagnostics.Append(validateAvaliableAccessForEnvDiags...)

	validateAccessRolesDiags := utils.ValidateAccessRolesV2(
		r.client,
		"cluster:scylladb",
		plan.Access,
	)
	resp.Diagnostics.Append(validateAccessRolesDiags...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r ScyllaDbClusterResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan ScyllaDbClusterResourceModel
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
			fmt.Sprintf("Get financial source from portal ended with error.\nError: %s", err.Error()))
		return
	}

	attrs, diags := r.prepareAttrs(plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	order := orders.NewScyllaDbCluster(
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
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Order creating ended with error: %s", err.Error()),
		)
		return
	}

	err = order.Sync()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Create order ended with error.\nError: %s", err.Error()))
		return
	}

	orderItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Get order item ended with error.\nError: %v", err.Error()),
		)
		return
	}

	vmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Get VM Items ended with error.\nError: %s", err.Error()),
		)
		return
	}

	if len(vmItems) < 1 {
		resp.Diagnostics.AddError(
			"Error while updating state:",
			"Length vmItems >= then 1",
		)
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(orderItem.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r ScyllaDbClusterResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var orderID types.String
	var dbUsers map[string]ScyllaDbUsersModel
	var dbUsersPortal []entities.ScyllaDbusers

	req.State.GetAttribute(ctx, path.Root("order_id"), &orderID)

	if orderID.IsNull() {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Read order from portal. Resource hasn't order_id. Without order_id can't get order from portal",
		)
		return
	}

	order, err := orders.GetScyllaDbClusterOrder(
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

	parentItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get app item from portal with order_id '%s'.\nError:%s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}

	vmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Can't get VMs items.\nError: %s", err.Error()),
		)
		return
	}

	vmItem := vmItems[0]
	vmConfig, ok := vmItem.Data.Config.(entities.VMItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Invalid type for vmConfig",
		)
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	parentConfig, ok := parentItem.Data.Config.(entities.ScyllaDbClusterItemConfig)
	if !ok {

		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Invalid type for parentConfig",
		)
		return
	}

	ExtraMountsLog, err := order.GetExtraMount("/app/scylla/logs")
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get vm extra mount '/app/scylla/logs' from portal with order_id '%s'.\nError:%s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}

	ExtraMountsData, err := order.GetExtraMount("/app/scylla/data")
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get vm extra mount '/app/scylla/data' from portal with order_id '%s'.\nError:%s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}

	DbPermissions1, err := order.GetDbPermissions()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get db permissions from portal with order_id '%s'.\nError:%s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}

	DbPermissions, diag := types.SetValueFrom(ctx, types.StringType, DbPermissions1)
	resp.Diagnostics.Append(diag...)

	state := ScyllaDbClusterResourceModel{
		OrderID:         orderID,
		Access:          utils.ReadAccessMapV2(vmItem.Data.ACLs),
		Label:           types.StringValue(order.Label),
		ScyllaDbVersion: types.StringValue(parentConfig.Version),
		ItemID:          types.StringValue(parentItem.ID),
		Flavor: flavor.FlavorModel{
			Cores:  types.Int64Value(vmConfig.Flavor.Cores),
			Memory: types.Int64Value(vmConfig.Flavor.Memory),
			Name:   types.StringValue(vmConfig.Flavor.Name),
			UUID:   types.StringValue(vmConfig.Flavor.UUID),
		},
		Image: common.ImageStandardModel{
			Distribution:  types.StringValue(vmConfig.Image.Os.Distribution),
			OsVersion:     types.StringValue(vmConfig.Image.Os.Version),
			OnSupport:     types.BoolValue(order.Attrs.OnSupport),
			ProductID:     types.StringValue(order.ProductID),
			ADIntegration: types.BoolValue(order.Attrs.AdIntegration),
		},
		Core: core.CoreModel{
			Platform:       types.StringValue(order.Attrs.Platform),
			Domain:         types.StringValue(order.Attrs.Domain),
			NetSegmentCode: types.StringValue(vmConfig.DefaultNic.NetSegment),
			Zone:           types.StringValue(order.Attrs.AvailabilityZone),
		},
		ExtraMountsLog: map[string]common.ExtraMountModel{
			"/app/scylla/logs": {
				Size:       types.Int64Value(ExtraMountsLog.Size),
				FileSystem: types.StringValue(ExtraMountsLog.FileSystem),
			},
		},
		ExtraMountsData: map[string]common.ExtraMountModel{
			"/app/scylla/data": {
				Size:       types.Int64Value(ExtraMountsData.Size),
				FileSystem: types.StringValue(ExtraMountsData.FileSystem),
			},
		},
		Databases:        scyllaReadDbs(ctx, parentConfig),
		DbPermissions:    DbPermissions,
		FinancialProject: types.StringValue(order.FinancialSource.Name),
		ScyllaClusterConfiguration: ScyllaClusterConfigurationModel{
			DC1: types.Int64Value(int64(order.Attrs.ScyllaClusterConfiguration.DC1)),
			DC2: types.Int64Value(int64(order.Attrs.ScyllaClusterConfiguration.DC2)),
			DC3: types.Int64Value(int64(order.Attrs.ScyllaClusterConfiguration.DC3)),
		},
	}

	var lifetime types.Int64
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("lifetime"), &lifetime)...)
	if !lifetime.IsNull() {
		state.Lifetime = lifetime
	}

	req.State.GetAttribute(ctx, path.Root("db_users"), &dbUsers)
	dbUsersPortal, err = order.GetUsers()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Can't get scylla db users.\nError: %s", err.Error()),
		)
		return
	}
	state.DbUsers = entitiesScyllaDBUsersToTerraform(dbUsers, dbUsersPortal)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r ScyllaDbClusterResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state ScyllaDbClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetScyllaDbClusterOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("order_id"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Get order data from portal ended with error.\nError: %v", err.Error()),
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
			fmt.Sprintf("Get financial source for project ended with error.\nError: %v", err.Error()))
		return
	}

	labelChanged := plan.Label != state.Label
	flavorChanged := plan.Flavor != state.Flavor
	financialProjectChanged := !plan.FinancialProject.Equal(state.FinancialProject)
	accessChanged := !reflect.DeepEqual(state.Access, plan.Access)
	dbChanged := !reflect.DeepEqual(plan.Databases, state.Databases)
	dbUsersChanged := !reflect.DeepEqual(plan.DbUsers, state.DbUsers)
	dbPermissionsChanged := !reflect.DeepEqual(plan.DbPermissions, state.DbPermissions)

	if labelChanged {
		utils.ChangeOrderLabel(order, plan.Label.ValueString(), resp)
	}

	if financialProjectChanged {
		r.changeFinancialProject(order, finProj.ID, resp)
	}

	if flavorChanged {
		r.changeFlavorScyllaDbCluster(order, &plan, resp)
	}

	if accessChanged {
		r.changeAccess(order, &plan, resp)
	}

	if dbChanged {
		diags := changeScyllaDbClusterDbs(order, &state, &plan)
		resp.Diagnostics.Append(diags...)
	}
	if dbUsersChanged {
		diags := configureScyllaDBUsers(&state, &plan, order)
		resp.Diagnostics.Append(diags...)
	}
	if dbPermissionsChanged {
		diags := changeScyllaDbClusterDbPermissions(order, &state, &plan)
		resp.Diagnostics.Append(diags...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r ScyllaDbClusterResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data ScyllaDbClusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetScyllaDbClusterOrder(
		r.client.Creds,
		r.client.ProjectName,
		data.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("order_id"),
			consts.DELETE_RES_FAIL,
			fmt.Sprintf("Get order data from portal ended with error.\nError:%v", err.Error()),
		)
		return
	}

	orderState, err := order.GetState()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf("Can't get state from portal.\nError:%s", err.Error()),
		)
	}

	if orderState == "deleted" || orderState == "deprovisioned" {
		resp.Diagnostics.AddWarning(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf(
				"Order with id '%s' was deleted by another way. Not from terraform. "+
					"Resource was recreated",
				data.OrderID.ValueString(),
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

	if err := order.Delete(); err != nil {
		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprint("While process delete ScyllaDB: ", err.Error()),
		)
		return
	}
}

func (r ScyllaDbClusterResource) prepareAttrs(plan ScyllaDbClusterResourceModel) (attrs orders.ScyllaDbClusterAttrs, diags diag.Diagnostics) {

	attrs = orders.ScyllaDbClusterAttrs{
		Domain: plan.Core.Domain.ValueString(),
		Flavor: entities.Flavor{
			Cores:  plan.Flavor.Cores.ValueInt64(),
			Memory: plan.Flavor.Memory.ValueInt64(),
			Name:   plan.Flavor.Name.ValueString(),
			UUID:   plan.Flavor.UUID.ValueString(),
		},
		Platform:  plan.Core.Platform.ValueString(),
		OnSupport: plan.Image.OnSupport.ValueBool(),
		OsVersion: plan.Image.OsVersion.ValueString(),
		DefaultNic: entities.DefaultNic{
			NetSegment: plan.Core.NetSegmentCode.ValueString(),
		},
		AvailabilityZone: plan.Core.Zone.ValueString(),
		AdIntegration:    plan.Image.ADIntegration.ValueBool(),
		ADLogonGrants:    utils.PrepareADLogonGrants(plan.Access),
		ScylladbVersion:  plan.ScyllaDbVersion.ValueString(),
		ExtraMountsLog:   utils.PrepareExtraMountsAttrs(plan.ExtraMountsLog),
		ExtraMountsData:  utils.PrepareExtraMountsAttrs(plan.ExtraMountsData),
		ScyllaClusterConfiguration: orders.ScyllaClusterConfiguration{
			DC1: int(plan.ScyllaClusterConfiguration.DC1.ValueInt64()),
			DC2: int(plan.ScyllaClusterConfiguration.DC2.ValueInt64()),
			DC3: int(plan.ScyllaClusterConfiguration.DC3.ValueInt64()),
		},
		CreatedWithOpenTofu: true,
	}
	return attrs, diags
}

func (r ScyllaDbClusterResource) changeFlavorScyllaDbCluster(
	order *orders.ScyllaDbCluster,
	plan *ScyllaDbClusterResourceModel,
	resp *resource.UpdateResponse) {

	flavor := entities.Flavor{
		Cores:  plan.Flavor.Cores.ValueInt64(),
		Memory: plan.Flavor.Memory.ValueInt64(),
		UUID:   plan.Flavor.UUID.ValueString(),
		Name:   plan.Flavor.Name.ValueString(),
	}
	err := order.ChangeFlavor(flavor, false)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Change app state (on changing machine state).\nError: %v", err.Error()),
		)
		return
	}
}

func (r ScyllaDbClusterResource) changeFinancialProject(
	order *orders.ScyllaDbCluster,
	finProjectId string,
	resp *resource.UpdateResponse,
) {
	err := order.ChangeFinancialProject(finProjectId)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("financial_project"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Change financial source for project ended wiht error: %v", err.Error()),
		)
		return
	}
}

func (r ScyllaDbClusterResource) changeAccess(
	order *orders.ScyllaDbCluster,
	plan *ScyllaDbClusterResourceModel,
	resp *resource.UpdateResponse,
) {
	vmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Get VM items ended with error.\nError: %v", err.Error()))
		return
	}

	for _, vmItem := range vmItems {
		stateAccess := utils.ReadAccessMapVV1(vmItem.Data.ACLs)
		planAccess := utils.ConvertAccessMap(plan.Access)
		deletedRoles, changedRoles, addedRoles := r.compareADRoles(planAccess, stateAccess)

		if len(addedRoles) > 0 {
			rolesFromCloud, err := references.GetRoles(
				r.client.Creds,
				r.client.Environment,
				"cluster:scylladb",
			)
			if err != nil {
				resp.Diagnostics.AddError(consts.UPDATE_RES_FAIL, err.Error())
				return
			}

			permissionsMap := make(map[string][]string)
			for _, value := range rolesFromCloud {
				permissionsMap[value.Data.GroupName] = value.Data.Permissions
			}

			for _, roles := range addedRoles {
				err := order.AddAccessGroupForVm(vmItem.ID, roles, permissionsMap[roles.Role])
				if err != nil {
					resp.Diagnostics.AddError(consts.UPDATE_RES_FAIL, err.Error())
					return
				}
			}
		}

		if len(changedRoles) > 0 {
			for _, roles := range changedRoles {
				err := order.ChangeAccessGroupForVm(vmItem.ID, roles)
				if err != nil {
					resp.Diagnostics.AddError(consts.UPDATE_RES_FAIL, err.Error())
					return
				}
			}
		}

		if len(deletedRoles) > 0 {
			for _, roles := range deletedRoles {
				err := order.DeleteAccessGroupForVm(vmItem.ID, roles)
				if err != nil {
					resp.Diagnostics.AddError(consts.UPDATE_RES_FAIL, err.Error())
					return
				}
			}
		}
	}
}

func (r ScyllaDbClusterResource) compareADRoles(planAccess, stateAccess map[string][]string) (deletedRoles, changedRoles, addedRoles []entities.ADLogonGrants) {
	for role, groups := range stateAccess {
		if _, ok := planAccess[role]; !ok {
			deletedRoles = append(
				deletedRoles,
				entities.ADLogonGrants{Role: role, Groups: groups},
			)
		} else {
			if len(planAccess[role]) == len(stateAccess[role]) {
				if utils.DifferenceLen(planAccess[role], stateAccess[role]) {
					changedRoles = append(
						changedRoles,
						entities.ADLogonGrants{Role: role, Groups: planAccess[role]},
					)
				}
			} else {
				changedRoles = append(
					changedRoles,
					entities.ADLogonGrants{Role: role, Groups: planAccess[role]},
				)
			}
		}
	}

	for role, groups := range planAccess {
		if _, ok := stateAccess[role]; !ok {
			addedRoles = append(
				addedRoles,
				entities.ADLogonGrants{Role: role, Groups: groups},
			)
		}
	}
	return deletedRoles, changedRoles, addedRoles
}

func changeScyllaDbClusterDbs(
	order *orders.ScyllaDbCluster,
	state,
	plan *ScyllaDbClusterResourceModel,
) (diags diag.Diagnostics) {

	deletedDbs, addedDbs := compareScyllaDbClusterDbs(state, plan)

	if len(deletedDbs) > 0 {
		for _, db := range deletedDbs {
			err := order.DeleteDb(db.ValueString())
			if err != nil {
				diags.AddError(
					consts.UPDATE_RES_FAIL,
					fmt.Sprint("Deleting db: ", err.Error()),
				)
				return diags
			}
		}
	}

	if len(addedDbs) > 0 {
		for _, db := range addedDbs {
			err := order.CreateDb(db.ValueString())
			if err != nil {
				diags.AddError(
					consts.UPDATE_RES_FAIL,
					fmt.Sprint("Adding db: ", err.Error()),
				)
				return diags
			}
		}
	}
	return
}

func compareScyllaDbClusterDbs(
	state, plan *ScyllaDbClusterResourceModel,
) (deletedDbs, addedDbs []types.String) {

	stateDbs := state.Databases.Elements()
	planDbs := plan.Databases.Elements()
	for _, statedb := range stateDbs {
		if !slices.Contains(planDbs, statedb) {
			deletedDbs = append(deletedDbs, statedb.(types.String))
		}
	}
	for _, plandb := range planDbs {
		if !slices.Contains(stateDbs, plandb) {
			addedDbs = append(addedDbs, plandb.(types.String))
		}
	}

	return deletedDbs, addedDbs
}

func scyllaReadDbs(ctx context.Context, parentConfig entities.ScyllaDbClusterItemConfig) basetypes.SetValue {

	var actualDbs []string
	for _, db := range parentConfig.Dbs {
		actualDbs = append(actualDbs, db.DbName)
	}

	dbs, _ := types.SetValueFrom(
		ctx,
		types.StringType,
		actualDbs,
	)
	return dbs
}

func entitiesScyllaDBUsersToTerraform(
	dbUsersState map[string]ScyllaDbUsersModel,
	dbUsersPortal []entities.ScyllaDbusers,
) map[string]ScyllaDbUsersModel {

	users := make(map[string]ScyllaDbUsersModel)

	for _, user := range dbUsersPortal {
		if _, exist := dbUsersState[user.UserName]; exist {
			users[user.UserName] = dbUsersState[user.UserName]
		} else {
			userPortal := ScyllaDbUsersModel{
				DbmsRole:     types.StringValue(user.DbmsRole),
				UserPassword: types.StringNull(),
			}
			users[user.UserName] = userPortal
		}
	}
	if len(users) == 0 {
		return nil
	}
	return users
}

func changeScyllaDbClusterDbPermissions(
	order *orders.ScyllaDbCluster,
	state, plan *ScyllaDbClusterResourceModel,
) (diags diag.Diagnostics) {

	parentItem, err := order.GetParentItem()
	if err != nil {
		diags.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Get order item ended with error.\nError: %v", err.Error()),
		)
		return
	}
	parentConfig, ok := parentItem.Data.Config.(entities.ScyllaDbClusterItemConfig)
	if !ok {

		diags.AddError(
			consts.READ_RES_FAIL,
			"Invalid type for parentConfig",
		)
		return
	}

	deletedDbPermissions, addedDbPermissions := compareScyllaDbClusterDbPermissions(state, plan)

	var portalDbUsers []string
	var portalDbs []string
	for _, user := range parentConfig.DbUsers {
		portalDbUsers = append(portalDbUsers, user.UserName)
	}

	for _, db := range parentConfig.Dbs {
		portalDbs = append(portalDbs, db.DbName)
	}

	if len(deletedDbPermissions) > 0 {
		for _, id := range deletedDbPermissions {
			id2 := strings.Replace(id, "\"", "", -1)
			dbpermission := entities.ScyllaDbPermissions{
				UserName: strings.Split(id2, ":")[0],
				DbName:   strings.Split(id2, ":")[1],
				ID:       id2,
			}
			err := order.DeleteDbPermission(dbpermission.DbName, dbpermission.UserName, dbpermission.ID)
			if err != nil {
				diags.AddError(
					consts.UPDATE_RES_FAIL,
					fmt.Sprint("Deleting dbpermission: ", err.Error()),
				)
				return diags
			}
		}
	}

	if len(addedDbPermissions) > 0 {
		for _, id := range addedDbPermissions {
			id2 := strings.Replace(id, "\"", "", -1)
			dbpermission := entities.ScyllaDbPermissions{
				UserName: strings.Split(id2, ":")[0],
				DbName:   strings.Split(id2, ":")[1],
			}
			if slices.Contains(portalDbUsers, dbpermission.UserName) && slices.Contains(portalDbs, dbpermission.DbName) {
				err := order.CreateDbPermission(dbpermission.DbName, dbpermission.UserName)
				if err != nil {
					diags.AddError(
						consts.UPDATE_RES_FAIL,
						fmt.Sprint("Adding dbpermission: ", err.Error()),
					)
					return diags
				}
			} else {
				diags.AddError(
					consts.UPDATE_RES_FAIL,
					fmt.Sprintf("Database user '%s' or database name '%s' not found. Available users: %s. Available databases: %s",
						dbpermission.UserName,
						dbpermission.DbName,
						portalDbUsers,
						portalDbs),
				)
				return diags
			}
		}
	}
	return
}

func compareScyllaDbClusterDbPermissions(
	state, plan *ScyllaDbClusterResourceModel,
) (deletedDbPermissions, addedDbPermissions []string) {

	stateDbPermissions := state.DbPermissions.Elements()
	planDbPermissions := plan.DbPermissions.Elements()
	for _, statedbid := range stateDbPermissions {
		if !slices.Contains(planDbPermissions, statedbid) {
			deletedDbPermissions = append(deletedDbPermissions, statedbid.String())
		}
	}
	for _, plandb := range planDbPermissions {
		if !slices.Contains(stateDbPermissions, plandb) {
			addedDbPermissions = append(addedDbPermissions, plandb.String())
		}
	}

	return deletedDbPermissions, addedDbPermissions
}

func configureScyllaDBUsers(
	state, plan *ScyllaDbClusterResourceModel,
	order *orders.ScyllaDbCluster,
) (diags diag.Diagnostics) {

	err := order.Sync()
	if err != nil {
		diags.AddWarning(
			"Sync ScyllaDB Cluster",
			fmt.Sprintf("Can't sync scyllaDB.\nError: %s", err.Error()),
		)
	}

	dbUsersToCreate := make(map[string]ScyllaDbUsersModel)
	dbUsersToChangePass := make(map[string]ScyllaDbUsersModel)
	dbUsersToDelete := make(map[string]ScyllaDbUsersModel)

	if err != nil {
		diags.AddError("Get scylladb users", err.Error())
		return diags
	}

	for userName, user := range plan.DbUsers {
		userCompare, exist := state.DbUsers[userName]
		if !exist {
			dbUsersToCreate[userName] = user
			continue
		}

		isDbmsRoleSame := user.DbmsRole.ValueString() == userCompare.DbmsRole.ValueString()
		isPasswordChanged := userCompare.UserPassword.ValueString() != user.UserPassword.ValueString()

		if isDbmsRoleSame && isPasswordChanged {
			dbUsersToChangePass[userName] = user
		}
	}

	for userName, user := range state.DbUsers {
		if _, exist := plan.DbUsers[userName]; !exist {
			dbUsersToDelete[userName] = user
		}
	}

	if len(dbUsersToDelete) > 0 {
		for user := range dbUsersToDelete {
			err := order.DeleteDbUser(user)
			if err != nil {
				diags.AddError(
					consts.UPDATE_RES_FAIL,
					fmt.Sprint("Deleting dbuser: ", err.Error()),
				)
			}
		}
	}

	if len(dbUsersToCreate) > 0 {
		for user := range dbUsersToCreate {
			scyllaDbUser := entities.ScyllaDbusers{
				DbmsRole:     plan.DbUsers[user].DbmsRole.ValueString(),
				UserName:     user,
				UserPassword: plan.DbUsers[user].UserPassword.ValueString(),
			}
			err := order.CreateDbUser(scyllaDbUser.UserName, scyllaDbUser.DbmsRole, scyllaDbUser.UserPassword)
			if err != nil {
				diags.AddError(
					consts.UPDATE_RES_FAIL,
					fmt.Sprint("Adding dbuser: ", err.Error()),
				)
			}
		}
	}

	if len(dbUsersToChangePass) > 0 {
		for userName, user := range dbUsersToChangePass {
			err = order.ChangeDbUserPassword(
				userName,
				user.UserPassword.ValueString(),
				false,
			)
			if err != nil {
				diags.AddError(
					consts.UPDATE_RES_FAIL,
					fmt.Sprint("Update user password: ", err.Error()),
				)
				return diags
			}
		}

	}

	return diags
}
