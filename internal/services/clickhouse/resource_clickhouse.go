package clickhouse

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
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
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"

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

const (
	CLICKHOUSE_APP_ADMIN_ROLE_NAME = "clickhouse_app_admin_ad_groups"
	CLICKHOUSE_APP_USER_ROLE_NAME  = "clickhouse_user_ad_groups"
	CLICKHOUSE_SYSTEM_ADM_GROUP    = "system_adm_groups"
)

var (
	_ resource.Resource                = &ClickHouseResource{}
	_ resource.ResourceWithImportState = &ClickHouseResource{}
	_ resource.ResourceWithModifyPlan  = &ClickHouseResource{}
)

type ClickHouseResource struct {
	client *client.CloudClient
}

func NewClickHouseResource() resource.Resource {
	return &ClickHouseResource{}
}

func (r ClickHouseResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_clickhouse_instance"
}

func (r *ClickHouseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type ClickHouseResourceModel struct {
	Core        core.CoreModel                    `tfsdk:"core"`
	Flavor      flavor.FlavorModel                `tfsdk:"flavor"`
	Image       common.ImageStandardModel         `tfsdk:"image"`
	Lifetime    types.Int64                       `tfsdk:"lifetime"`
	Label       types.String                      `tfsdk:"label"`
	OrderID     types.String                      `tfsdk:"order_id"`
	ItemID      types.String                      `tfsdk:"item_id"`
	Access      map[string][]types.String         `tfsdk:"access"`
	ExtraMounts map[string]common.ExtraMountModel `tfsdk:"extra_mounts"`
	Hostname    types.String                      `tfsdk:"hostname"`

	ClickHouseVersion          types.String        `tfsdk:"ch_version"`
	ClickHouseUser             types.String        `tfsdk:"clickhouse_user"`
	ClickHousePassword         types.String        `tfsdk:"clickhouse_password"`
	ChCustomerPassword         types.String        `tfsdk:"ch_customer_password"`
	SystemAdmGroups            map[string][]string `tfsdk:"system_adm_groups"`
	ClickHouseAppAdminAdGroups map[string][]string `tfsdk:"clickhouse_app_admin_ad_groups"`
	ClickHouseUserAdGroups     map[string][]string `tfsdk:"clickhouse_user_ad_groups"`
	FinancialProject           types.String        `tfsdk:"financial_project"`
}

func (r ClickHouseResource) Schema(
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
				MarkdownDescription: "Type of compute instance",
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
			"extra_mounts": schema.MapNestedAttribute{
				Required:            true,
				Description:         "Дополнительные точки монтирования ВМ",
				MarkdownDescription: "Дополнительные точки монтирования ВМ",
				Validators: []validator.Map{
					mapvalidator.KeysAre(
						stringvalidator.OneOf(
							"/app/clickhouse",
						),
					),
					mapvalidator.SizeBetween(1, 4),
				},
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"size": schema.Int64Attribute{
							Required:            true,
							Description:         "Размер точки монтированиия в ГБ",
							MarkdownDescription: "Размер точки монтированиия в ГБ",
							Validators: []validator.Int64{
								int64validator.AtMost(2048),
								int64validator.AtLeast(10),
							},
						},
						"file_system": schema.StringAttribute{
							Computed:            true,
							Default:             stringdefault.StaticString("xfs"),
							Description:         "Тип файловой системы",
							MarkdownDescription: "Тип файловой системы",
						},
					},
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

			// extra product schema
			"hostname": schema.StringAttribute{
				MarkdownDescription: "Имя хоста виртуальной машины.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ch_version": schema.StringAttribute{
				MarkdownDescription: "Версия Clickhouse",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("25.3.2.39"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"clickhouse_user": schema.StringAttribute{
				MarkdownDescription: "Имя пользователя Clickhouse (доступно только для DEV среды)",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.NoneOf("ch_admin"),
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-z0-9][a-z0-9_]{0,23}[a-z0-9]$"),
						"Username must match pattern ^(?!^ch_admin$)[a-z0-9][a-z0-9_]{0,23}[a-z0-9]$"),
				},
			},
			"clickhouse_password": schema.StringAttribute{
				MarkdownDescription: "Пароля для пользователя Clickhouse (доступно только для DEV среды)",
				Sensitive:           true,
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile("[A-Z]"),
						"Password must have 1 uppercase letter"),
					stringvalidator.RegexMatches(regexp.MustCompile("[a-z]"),
						"Password must have 1 lowercase letter"),
					stringvalidator.RegexMatches(regexp.MustCompile("[0-9]"),
						"Password must have 1 numeric character"),
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-zA-Z0-9~!@#$%^*()_]{64,70}$"),
						"Password must match pattern ^[a-zA-Z0-9~!@#$%^*()_]{64,70}"),
				},
			},
			"ch_customer_password": schema.StringAttribute{
				MarkdownDescription: "Пароля для пользователя Clickhouse Customer",
				Sensitive:           true,
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile("[A-Z]"),
						"Password must have 1 uppercase letter"),
					stringvalidator.RegexMatches(regexp.MustCompile("[a-z]"),
						"Password must have 1 lowercase letter"),
					stringvalidator.RegexMatches(regexp.MustCompile("[0-9]"),
						"Password must have 1 numeric character"),
				},
			},
			"system_adm_groups": schema.MapAttribute{
				Optional:            true,
				Description:         "AD-группа с полными правами на кластер",
				MarkdownDescription: "AD-группа с полными правами на кластер",
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
				Validators: []validator.Map{
					customvalidators.EmptyAccessGroupListValidator{},
					mapvalidator.KeysAre(
						stringvalidator.OneOf(CLICKHOUSE_SYSTEM_ADM_GROUP),
					),
				},
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"clickhouse_user_ad_groups": schema.MapAttribute{
				Required:            true,
				Description:         "AD-группа с правами на кластер: Select, Insert, Alter, Create table, Remote on",
				MarkdownDescription: "AD-группа с правами на кластер: Select, Insert, Alter, Create table, Remote on",
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
				Validators: []validator.Map{
					customvalidators.EmptyAccessGroupListValidator{},
					mapvalidator.KeysAre(
						stringvalidator.OneOf("clickhouse_user_ad_groups"),
					),
				},
			},
			"clickhouse_app_admin_ad_groups": schema.MapAttribute{
				Required:            true,
				Description:         "AD-группа с правами чтение таблиц",
				MarkdownDescription: "AD-группа с правами чтение таблиц",
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
				Validators: []validator.Map{
					customvalidators.EmptyAccessGroupListValidator{},
					mapvalidator.KeysAre(
						stringvalidator.OneOf("clickhouse_app_admin_ad_groups"),
					),
				},
			},
			"financial_project": schema.StringAttribute{
				Required:            true,
				Description:         "Источник финансирования.",
				MarkdownDescription: "Источник финансирования.",
			},
		},
	}
}

func (r ClickHouseResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r ClickHouseResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	var plan ClickHouseResourceModel

	if req.Plan.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	checkOrderIsDeleted := utils.CheckOrderIsDeleted(r.client, plan.OrderID, "app", "clickhouse")
	resp.Diagnostics.Append(checkOrderIsDeleted.Diagnostics...)

	utils.HandleExternalDeletionRecreate(ctx, resp, &plan, checkOrderIsDeleted.IsDeleted, checkOrderIsDeleted.Diagnostics)

	validateFlavorDiags := utils.ValidateFlavor(r.client, plan.Flavor, "app:clickhouse")
	resp.Diagnostics.Append(validateFlavorDiags...)

	validateRolesDiags := utils.ValidateAccessRolesV2(r.client, "app:clickhouse", plan.Access)
	resp.Diagnostics.Append(validateRolesDiags...)

	validateAccessForEnvironmentsDiags := utils.ValidateActiveDirectoryAccessForEnvironmentsV1(
		r.client,
		plan.Access,
		"DEV",
	)
	resp.Diagnostics.Append(validateAccessForEnvironmentsDiags...)

	r.validateFieldsForSpecificEnvironments(plan, resp)

	if resp.Diagnostics.HasError() {
		return
	}

}

func (r ClickHouseResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan ClickHouseResourceModel
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
	attrs := r.prepareClickHouseOrderAttrs(plan)

	order := orders.NewClickHouse(
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
			fmt.Sprintf("Create order ended with error.\nError: %v", err.Error()),
		)
		return
	}

	orderItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Get order item from portal ended with error.\nError: %v", err.Error()),
		)
		return
	}

	vmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Get order vm items from portal ended with error.\nError: %v", err.Error()),
		)
		return
	}

	if (orderItem.Type == "app") && (len(vmItems) != 1) {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			"Error while update state: number of VM is more than 1",
		)
		return
	}

	itemData := vmItems[0].Data

	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(orderItem.ID)
	plan.Hostname = types.StringValue(itemData.Config.(entities.VMItemConfig).Hostname)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r ClickHouseResource) Read(
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

	order, err := orders.GetClickHouseOrder(
		r.client.Creds,
		r.client.ProjectName,
		orderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't read order with order_id equal '%s'.\nError: %s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}

	appItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get app item from portal with order_id '%s'.\nError: %s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}

	vmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Get order vm items ended with error.\nError: %v", err.Error()),
		)
		return
	}

	vmItem := vmItems[0]

	var ChCustomerPassword types.String
	req.State.GetAttribute(ctx, path.Root("ch_customer_password"), &ChCustomerPassword)

	var ClickHousePassword types.String
	req.State.GetAttribute(ctx, path.Root("clickhouse_password"), &ClickHousePassword)

	clickHouseUser, err := order.GetDefaultClickhouseUser()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Get user from order ended with error.\nError: %s", err.Error()),
		)
		return
	}

	var tfclickHouseUser basetypes.StringValue
	if clickHouseUser.UserName == "" {
		tfclickHouseUser = types.StringNull()
	} else {
		tfclickHouseUser = types.StringValue(clickHouseUser.UserName)
	}

	vmConfig, ok := vmItem.Data.Config.(entities.VMItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Invalid type for vmConfig",
		)
		return
	}

	appClick, err := order.GetExtraMount("/app/clickhouse")
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get vm extra mount '/app/clickhouse' from portal with order_id '%s'.\nError:%s",
				orderID, err.Error(),
			),
		)
		return
	}

	systemAdmGroups, err := order.GetAppAdGroups(CLICKHOUSE_SYSTEM_ADM_GROUP)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root(CLICKHOUSE_SYSTEM_ADM_GROUP),
			consts.READ_RES_FAIL,
			fmt.Sprintf("Get system_adm_groups from portal ended with error.\nError: %v", err.Error()),
		)
		return
	}

	if len(systemAdmGroups) < 1 {
		systemAdmGroups = nil
	}

	clickHouseUserAdGroups, err := order.GetAppAdGroups(CLICKHOUSE_APP_USER_ROLE_NAME)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("clickhouse_user_ad_groups"),
			consts.READ_RES_FAIL,
			fmt.Sprintf("Get clickhouse_user_ad_groups from portal ended with error.\nError: %v", err.Error()),
		)
		return
	}

	clickHouseAppAdminGroups, err := order.GetAppAdGroups(CLICKHOUSE_APP_ADMIN_ROLE_NAME)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("clickhouse_app_admin_ad_groups"),
			consts.READ_RES_FAIL,
			fmt.Sprintf("Get clickhouse_app_admin_ad_groups from portal ended with error.\nError: %v", err.Error()),
		)
		return
	}

	state := ClickHouseResourceModel{
		OrderID:                    orderID,
		Access:                     utils.ReadAccessMapV2(vmItem.Data.ACLs),
		Label:                      types.StringValue(order.Label),
		ClickHouseVersion:          types.StringValue(order.Attrs.ClickHouseVersion),
		ClickHouseUser:             tfclickHouseUser,
		ChCustomerPassword:         ChCustomerPassword,
		ClickHousePassword:         ClickHousePassword,
		SystemAdmGroups:            systemAdmGroups,
		ClickHouseAppAdminAdGroups: clickHouseAppAdminGroups,
		ClickHouseUserAdGroups:     clickHouseUserAdGroups,
		ItemID:                     types.StringValue(appItem.ID),
		Hostname:                   types.StringValue(vmConfig.Hostname),
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
			ADIntegration: types.BoolValue(order.Attrs.ADIntegration),
		},
		Core: core.CoreModel{
			Platform:       types.StringValue(order.Attrs.Platform),
			Domain:         types.StringValue(order.Attrs.Domain),
			NetSegmentCode: types.StringValue(vmConfig.DefaultNic.NetSegment),
			Zone:           types.StringValue(order.Attrs.AvailabilityZone),
		},
		ExtraMounts: map[string]common.ExtraMountModel{
			"/app/clickhouse": {
				Size:       types.Int64Value(appClick.Size),
				FileSystem: types.StringValue(appClick.FileSystem),
			},
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

func (r ClickHouseResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state ClickHouseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetClickHouseOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("order_id"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Get order from portal ended with error.\nError: %v", err.Error()),
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
			fmt.Sprintf("Get financial project ended with error.\nError: %v", err.Error()),
		)
		return
	}

	labelChanged := plan.Label != state.Label
	financialProjectChanged := !plan.FinancialProject.Equal(state.FinancialProject)
	clickhousePasswordChanged := state.ClickHousePassword != plan.ClickHousePassword
	chCustomerPasswordChanged := state.ChCustomerPassword != plan.ChCustomerPassword
	accessChanged := !reflect.DeepEqual(state.Access, plan.Access)
	adAdminGroupChanged := !reflect.DeepEqual(state.ClickHouseAppAdminAdGroups, plan.ClickHouseAppAdminAdGroups)
	adUserGroupChanged := !reflect.DeepEqual(state.ClickHouseUserAdGroups, plan.ClickHouseUserAdGroups)

	if financialProjectChanged {
		r.changeFinancialProject(order, finProj.ID, resp)
	}

	if labelChanged {
		utils.ChangeOrderLabel(order, plan.Label.ValueString(), resp)
	}

	if accessChanged {
		r.changeAccess(order, &plan, resp)
	}

	if clickhousePasswordChanged {
		r.changeUserPassword(order, &plan, &state, resp)
	}

	if chCustomerPasswordChanged {
		r.changeCHUserPassword(order, &plan, resp)
	}

	if adAdminGroupChanged {
		r.applyAdAdminGroups(order, plan, state, resp)
	}

	if adUserGroupChanged {
		r.applyAdUsersGroups(order, plan, state, resp)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r ClickHouseResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data ClickHouseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetClickHouseOrder(
		r.client.Creds,
		r.client.ProjectName,
		data.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("order_id"),
			consts.DELETE_RES_FAIL,
			fmt.Sprintf("Get order from portal ended with error.\nError: %v", err.Error()),
		)
		return
	}

	orderState, err := order.GetState()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf("Can't get state from portal.\nError: %v", err.Error()),
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

	if err := order.DeleteTwoLayer(false); err != nil {
		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf("Delete order from portal ended with error.\nError: %v", err.Error()),
		)
		return
	}
}

func (r ClickHouseResource) changeUserPassword(
	order *orders.ClickHouse,
	plan, state *ClickHouseResourceModel,
	resp *resource.UpdateResponse,
) {
	err := order.ChangeUserPassword(state.ClickHouseUser.ValueString(), plan.ClickHousePassword.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf(
				"Change user password enden with error.\nError: %v",
				err.Error(),
			),
		)
		return
	}
}

func (r ClickHouseResource) changeFinancialProject(
	order *orders.ClickHouse,
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

func (r ClickHouseResource) changeCHUserPassword(
	order *orders.ClickHouse,
	plan *ClickHouseResourceModel,
	resp *resource.UpdateResponse,
) {
	err := order.ChangeUserPassword("ch_customer", plan.ChCustomerPassword.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf(
				"Change user password enden with error.\nError: %v",
				err.Error(),
			),
		)
		return
	}
}

func (r ClickHouseResource) changeAccess(
	order *orders.ClickHouse,
	plan *ClickHouseResourceModel,
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
				"app:clickhouse",
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

func (r ClickHouseResource) compareADRoles(planAccess, stateAccess map[string][]string) (deletedRoles, changedRoles, addedRoles []entities.ADLogonGrants) {
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

func (r ClickHouseResource) prepareClickHouseOrderAttrs(
	plan ClickHouseResourceModel,
) (attrs orders.ClickHouseAttrs) {
	var SystemAdmGroupsAttrs []orders.SystemAdmGroupsModel
	for role, groups := range plan.SystemAdmGroups {
		var groupsNames []string
		groupsNames = append(groupsNames, groups...)
		SystemAdmGroupsAttrs = append(SystemAdmGroupsAttrs, orders.SystemAdmGroupsModel{
			Role:   role,
			Groups: groupsNames,
		})
	}

	var ClickHouseAppAdminAdGroupsAttrs []orders.ClickHouseAppAdminAdGroupsModel
	for role, groups := range plan.ClickHouseAppAdminAdGroups {
		var groupsNames []string
		groupsNames = append(groupsNames, groups...)
		ClickHouseAppAdminAdGroupsAttrs = append(ClickHouseAppAdminAdGroupsAttrs, orders.ClickHouseAppAdminAdGroupsModel{
			Role:   role,
			Groups: groupsNames,
		})
	}

	var ClickHouseUserAdGroupsAttrs []orders.ClickHouseUserAdGroupsModel
	for role, groups := range plan.ClickHouseUserAdGroups {
		var groupsNames []string
		groupsNames = append(groupsNames, groups...)
		ClickHouseUserAdGroupsAttrs = append(ClickHouseUserAdGroupsAttrs, orders.ClickHouseUserAdGroupsModel{
			Role:   role,
			Groups: groupsNames,
		})
	}

	basicAttrs := utils.PrepareBasicAttrs(
		&plan.Flavor,
		&plan.Core,
		plan.Access,
		plan.ExtraMounts,
		plan.Image.OsVersion.ValueString(),
		plan.Image.ADIntegration.ValueBool(),
		plan.Image.OnSupport.ValueBool(),
	)

	attrs = orders.ClickHouseAttrs{
		BasicAttrs:                      basicAttrs,
		Accept:                          true,
		ClickHouseVersion:               plan.ClickHouseVersion.ValueString(),
		ClickHouseUser:                  plan.ClickHouseUser.ValueStringPointer(),
		ClickHousePassword:              plan.ClickHousePassword.ValueStringPointer(),
		ChCustomerPassword:              plan.ChCustomerPassword.ValueString(),
		SystemAdmGroupsAttrs:            SystemAdmGroupsAttrs,
		ClickHouseAppAdminAdGroupsAttrs: ClickHouseAppAdminAdGroupsAttrs,
		ClickHouseUserAdGroupsAttrs:     ClickHouseUserAdGroupsAttrs,
	}

	return attrs
}

func (r ClickHouseResource) applyAdAdminGroups(
	order *orders.ClickHouse,
	plan, state ClickHouseResourceModel,
	resp *resource.UpdateResponse,
) {
	toAdd, toDelete, err := r.compareAppGroups(CLICKHOUSE_APP_ADMIN_ROLE_NAME, plan, state)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Compare App Admin Groups ended with error.\nError: %v", err.Error()),
		)
		return
	}

	if len(toAdd) > 0 {
		for _, groupName := range toAdd {
			err = order.CreateNewAppAdminGroupAd(groupName)
			if err != nil {
				resp.Diagnostics.AddError(
					consts.UPDATE_RES_FAIL,
					fmt.Sprintf("Create new app admin group ended with error.\nError: %v", err.Error()),
				)
				return
			}
		}
	}

	if len(toDelete) > 0 {
		for _, groupName := range toDelete {
			err = order.RemoveNewAppAdminGroupAd(groupName)
			if err != nil {
				resp.Diagnostics.AddError(
					consts.UPDATE_RES_FAIL,
					fmt.Sprintf("Remove new app admin group ended with error.\nError: %v", err.Error()),
				)
				return
			}
		}
	}
}

func (r ClickHouseResource) applyAdUsersGroups(
	order *orders.ClickHouse,
	plan, state ClickHouseResourceModel,
	resp *resource.UpdateResponse,
) {
	toAdd, toDelete, err := r.compareAppGroups(CLICKHOUSE_APP_USER_ROLE_NAME, plan, state)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Compare app user groups ended with error.\nError: %v", err.Error()),
		)
		return
	}

	if len(toAdd) > 0 {
		for _, groupName := range toAdd {
			err = order.CreateNewAppUserGroupAd(groupName)
			if err != nil {
				resp.Diagnostics.AddError(
					consts.UPDATE_RES_FAIL,
					fmt.Sprintf("Create new app user group ad ended with error.\nError: %v", err.Error()),
				)
				return
			}
		}
	}

	if len(toDelete) > 0 {
		for _, groupName := range toDelete {
			err = order.RemoveNewAppUserGroupAd(groupName)
			if err != nil {
				resp.Diagnostics.AddError(
					consts.UPDATE_RES_FAIL,
					fmt.Sprintf("Remove app user group ad ended with error.\nError: %v", err.Error()),
				)
				return
			}
		}
	}
}

func (r ClickHouseResource) compareAppGroups(
	roleName string,
	plan, state ClickHouseResourceModel,
) (groupsToAdd, groupsToDelete []string, err error) {

	switch roleName {
	case CLICKHOUSE_APP_ADMIN_ROLE_NAME:
		planAdminGroups, ok := plan.ClickHouseAppAdminAdGroups[CLICKHOUSE_APP_ADMIN_ROLE_NAME]
		if !ok {
			return nil, nil, fmt.Errorf("plan app admin groups not found")
		}

		stateAdminGroups, ok := state.ClickHouseAppAdminAdGroups[CLICKHOUSE_APP_ADMIN_ROLE_NAME]
		if !ok {
			return nil, nil, fmt.Errorf("state app admin groups not found")
		}

		groupsToAdd, groupsToDelete = utils.CompareSlices(stateAdminGroups, planAdminGroups)

	case CLICKHOUSE_APP_USER_ROLE_NAME:
		planAdminGroups, ok := plan.ClickHouseUserAdGroups[CLICKHOUSE_APP_USER_ROLE_NAME]
		if !ok {
			return nil, nil, fmt.Errorf("plan app user groups not found")
		}

		stateAdminGroups, ok := state.ClickHouseUserAdGroups[CLICKHOUSE_APP_USER_ROLE_NAME]
		if !ok {
			return nil, nil, fmt.Errorf("state app user groups not found")
		}

		groupsToAdd, groupsToDelete = utils.CompareSlices(stateAdminGroups, planAdminGroups)
	}

	return groupsToAdd, groupsToDelete, nil
}

func (r ClickHouseResource) validateFieldsForSpecificEnvironments(
	plan ClickHouseResourceModel,
	resp *resource.ModifyPlanResponse,
) {
	if !strings.EqualFold(r.client.Environment, "dev") {
		if !plan.ClickHouseUser.IsNull() || !plan.ClickHousePassword.IsNull() || plan.SystemAdmGroups != nil {
			resp.Diagnostics.AddError(
				consts.MODIFY_PLAN_FAIL,
				"Fields: `clickhouse_user`,`clickhouse_password` "+
					"and `system_adm_groups` are specific for DEV environment only",
			)
		}
		return
	}

	if plan.ClickHouseUser.IsNull() || plan.ClickHousePassword.IsNull() || plan.SystemAdmGroups == nil {
		resp.Diagnostics.AddError(
			consts.MODIFY_PLAN_FAIL,
			"Fields: `clickhouse_user`,`clickhouse_password` "+
				"and `system_adm_groups` are required for DEV environment",
		)
		return
	}
}
