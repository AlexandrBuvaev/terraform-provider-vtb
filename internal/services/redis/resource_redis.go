package redis

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &RedisResource{}
	_ resource.ResourceWithImportState = &RedisResource{}
	_ resource.ResourceWithModifyPlan  = &RedisResource{}
)

type RedisResource struct {
	client *client.CloudClient
}

func NewRedisResource() resource.Resource {
	return &RedisResource{}
}

func (r RedisResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_redis_instance"
}

func (r *RedisResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type RedisResourceModel struct {
	Core        core.CoreModel                    `tfsdk:"core"`
	Flavor      flavor.FlavorModel                `tfsdk:"flavor"`
	Image       common.ImageStandardModel         `tfsdk:"image"`
	Lifetime    types.Int64                       `tfsdk:"lifetime"`
	Label       types.String                      `tfsdk:"label"`
	OrderID     types.String                      `tfsdk:"order_id"`
	ItemID      types.String                      `tfsdk:"item_id"`
	Access      map[string][]types.String         `tfsdk:"access"`
	ExtraMounts map[string]common.ExtraMountModel `tfsdk:"extra_mounts"`

	Hostnames            types.Set    `tfsdk:"hostnames"`
	RedisVersion         types.String `tfsdk:"redis_version"`
	User                 types.String `tfsdk:"user"`
	UserPassword         types.String `tfsdk:"user_password"`
	NotifyKeyspaceEvents types.String `tfsdk:"notify_keyspace_events"`
	FinancialProject     types.String `tfsdk:"financial_project"`
}

func (r RedisResource) Schema(
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
				Description:         "Тип вычислительного ресурса",
				MarkdownDescription: "Тип вычислительного ресурса",
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
			},
			"extra_mounts": schema.MapNestedAttribute{
				Description:         "Дополнительные точки монтирования ВМ",
				MarkdownDescription: "Дополнительные точки монтирования ВМ",
				Required:            true,
				Validators: []validator.Map{
					mapvalidator.KeysAre(
						stringvalidator.OneOf(
							"/app/redis/data",
							"/app/redis/logs",
						),
					),
					mapvalidator.SizeBetween(2, 2),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"size": schema.Int64Attribute{
							Description:         "Размер точки монтированиия в ГБ",
							MarkdownDescription: "Размер точки монтированиия в ГБ",
							Required:            true,
							Validators: []validator.Int64{
								int64validator.OneOf(10),
							},
							PlanModifiers: []planmodifier.Int64{
								custommodifires.GrowOnly(),
								custommodifires.MininalGrow(10),
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

			// extra product schema
			"hostnames": schema.SetAttribute{
				MarkdownDescription: "Имя хоста виртуальной машины.",
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"redis_version": schema.StringAttribute{
				MarkdownDescription: "Версия Redis",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("7.2.4", "7.0.15", "6.2.14"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"user": schema.StringAttribute{
				MarkdownDescription: "Имя пользователя Redis",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-z0-9][a-z0-9-_]{0,23}[a-z0-9]$"),
						"Username must match pattern ^[a-z0-9][a-z0-9-_]{0,23}[a-z0-9]$"),
				},
			},
			"notify_keyspace_events": schema.StringAttribute{
				MarkdownDescription: "Значение параметра Notify-keyspace-events",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("AKE"),
				Validators: []validator.String{
					stringvalidator.OneOf("Ex", "KEA", "AKE"),
				},
			},
			"user_password": schema.StringAttribute{
				MarkdownDescription: "Пароля для пользователя Redis",
				Sensitive:           true,
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile("[A-Z]"),
						"Password must have 1 uppercase letter"),
					stringvalidator.RegexMatches(regexp.MustCompile("[a-z]"),
						"Password must have 1 lowercase letter"),
					stringvalidator.RegexMatches(regexp.MustCompile("[0-9]"),
						"Password must have 1 numeric character"),
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-zA-Z0-9]{64,80}$"),
						"Password must match pattern ^[a-zA-Z0-9]{64,80}$"),
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

func (r RedisResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r RedisResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan RedisResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	checkIsOrderDeleted := utils.CheckOrderIsDeleted(r.client, plan.OrderID, "app", "redis")
	resp.Diagnostics.Append(checkIsOrderDeleted.Diagnostics...)

	utils.HandleExternalDeletionRecreate(ctx, resp, &plan, checkIsOrderDeleted.IsDeleted, checkIsOrderDeleted.Diagnostics)

	validateFlavorDiags := utils.ValidateFlavor(r.client, plan.Flavor, "app:redis")
	resp.Diagnostics.Append(validateFlavorDiags...)

	validateAccessForEnvDiags := utils.ValidateActiveDirectoryAccessForEnvironmentsV1(r.client, plan.Access, "DEV", "TEST")
	resp.Diagnostics.Append(validateAccessForEnvDiags...)

	validateAccessRolesDiags := utils.ValidateAccessRolesV2(r.client, "app:redis", plan.Access)
	resp.Diagnostics.Append(validateAccessRolesDiags...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r RedisResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan RedisResourceModel
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

	attrs, diags := r.prepareAttrs(plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	order := orders.NewRedis(
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
			fmt.Sprintf("Error while get item of order.\nError: %v", err.Error()))
		return
	}

	vmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Error while update state.\nError: %v", err.Error()),
		)
		return
	}

	if (orderItem.Type == "app") && (len(vmItems) != 1) {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			"Error while update state: number of VM is not equal 1",
		)
		return

	} else if (orderItem.Type == "cluster") && (len(vmItems) != 2) {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			"Error while update state: number of VM is not equal 2",
		)
		return
	}

	hostnames, err := getVMHostnames(vmItems)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Error while get hostnames.\nError: %v", err.Error()),
		)
		return
	}
	tfhostnames, diags := types.SetValueFrom(ctx, types.StringType, hostnames)
	if diags.HasError() {
		return
	}

	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(orderItem.ID)
	plan.Hostnames = tfhostnames
	if diags.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r RedisResource) Read(
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

	order, err := orders.GetRedisOrder(
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

	appItem, err := order.GetParentItem()
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

	appExtraMountRedisData, err := order.GetExtraMount("/app/redis/data")
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get Redis extra mount '/app/redis/data' "+
					"from portal with order_id '%s'.\nError:%s",
				orderID, err.Error(),
			),
		)
		return
	}

	appExtraMountRedisLogs, err := order.GetExtraMount("/app/redis/logs")
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get Redis extra mount '/app/redis/logs' "+
					"from portal with order_id '%s'.\nError:%s",
				orderID, err.Error(),
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

	if appItem.Type == "app" {
		if len(vmItems) != 1 {
			resp.Diagnostics.AddError(
				consts.READ_RES_FAIL,
				"Error while read state: number of VM is not equal 1",
			)
			return
		}
	} else {
		if len(vmItems) != 2 {
			resp.Diagnostics.AddError(
				consts.READ_RES_FAIL,
				"Error while read state: number of VM is not equal 2",
			)
			return
		}
	}

	vmItem := vmItems[0]

	hostnames, err := getVMHostnames(vmItems)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Error while get hostnames.\nError: %v", err.Error()),
		)
		return
	}

	redisUser, err := order.GetDefaultRedisUser()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Get user from order.\nError: %v", err.Error()),
		)
		return
	}

	var UserPassword types.String
	req.State.GetAttribute(ctx, path.Root("user_password"), &UserPassword)

	redisConfig, ok := appItem.Data.Config.(entities.RedisItemConfig)
	if !ok {

		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Invalid type for redisConfig",
		)
		return
	}

	vmConfig, ok := vmItem.Data.Config.(entities.VMItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Invalid type for vmConfig",
		)
		return
	}

	var notifyKeyspaceEvents basetypes.StringValue
	if redisConfig.NotifyKeyspaceEvents == "" {
		notifyKeyspaceEvents = types.StringNull()
	} else {
		notifyKeyspaceEvents = types.StringValue(redisConfig.NotifyKeyspaceEvents)
	}

	tfhostnames, diags := types.SetValueFrom(ctx, types.StringType, hostnames)
	if diags.HasError() {
		return
	}

	state := RedisResourceModel{
		OrderID:              orderID,
		Access:               utils.ReadAccessMapV2(vmItem.Data.ACLs),
		Label:                types.StringValue(order.Label),
		NotifyKeyspaceEvents: notifyKeyspaceEvents,
		RedisVersion:         types.StringValue(redisConfig.Version),
		User:                 types.StringValue(redisUser.UserName),
		UserPassword:         UserPassword,
		ItemID:               types.StringValue(appItem.ID),
		Hostnames:            tfhostnames,
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
			"/app/redis/logs": {
				Size:       types.Int64Value(appExtraMountRedisLogs.Size),
				FileSystem: types.StringValue(appExtraMountRedisLogs.FileSystem),
			},
			"/app/redis/data": {
				Size:       types.Int64Value(appExtraMountRedisData.Size),
				FileSystem: types.StringValue(appExtraMountRedisData.FileSystem),
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

func (r RedisResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state RedisResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetRedisOrder(
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
	mountChanged := utils.IsExtraMountChanged(state.ExtraMounts, plan.ExtraMounts)
	notifyKeyspaceEventsChanged := plan.NotifyKeyspaceEvents != state.NotifyKeyspaceEvents
	financialProjectChanged := !plan.FinancialProject.Equal(state.FinancialProject)
	userPasswordChanged := state.UserPassword != plan.UserPassword
	accessChanged := !reflect.DeepEqual(state.Access, plan.Access)

	if labelChanged {
		utils.ChangeOrderLabel(order, plan.Label.ValueString(), resp)
	}

	if financialProjectChanged {
		r.changeFinancialProject(order, finProj.ID, resp)
	}

	if notifyKeyspaceEventsChanged {
		r.changeNotifyKeyspaceEvents(order, &plan, resp)
	}

	if userPasswordChanged {
		r.changeUserPassword(order, &plan, &state, resp)
	}

	if mountChanged {
		r.changeRedisExtraMounts(order, &plan, resp)
	}

	if flavorChanged {
		r.changeRedisFlavor(ctx, order, &plan, resp)
	}

	if accessChanged {
		r.changeAccess(order, &plan, resp)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r RedisResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data RedisResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetRedisOrder(
		r.client.Creds,
		r.client.ProjectName,
		data.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("order_id"), consts.DELETE_RES_FAIL, err.Error())
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

	if err := order.DeleteTwoLayer(false); err != nil {
		resp.Diagnostics.AddError("While process delete redis: ", err.Error())
		return
	}
}

// * Custom logic *

func (r RedisResource) changeRedisExtraMounts(
	order *orders.Redis,
	plan *RedisResourceModel,
	resp *resource.UpdateResponse,
) {
	for path, planEM := range plan.ExtraMounts {
		err := order.ExpandMountPoint(entities.ExtraMount{
			Path:       path,
			Size:       planEM.Size.ValueInt64(),
			FileSystem: planEM.FileSystem.ValueString(),
		}, false)
		if err != nil {
			resp.Diagnostics.AddError("Incorrent changes in extra_mounts parameters", err.Error())
		}
		return
	}
}

func (r RedisResource) changeRedisFlavor(
	ctx context.Context,
	order *orders.Redis,
	plan *RedisResourceModel,
	resp *resource.UpdateResponse,
) {
	flavor := entities.Flavor{
		Cores:  plan.Flavor.Cores.ValueInt64(),
		Memory: plan.Flavor.Memory.ValueInt64(),
		UUID:   plan.Flavor.UUID.ValueString(),
		Name:   plan.Flavor.Name.ValueString(),
	}

	if strings.EqualFold(r.client.Environment, "prod") {
		var hostnames []string
		diag := plan.Hostnames.ElementsAs(ctx, &hostnames, false)
		if diag.HasError() {
			return
		}

		for _, hostname := range hostnames {
			err := order.ChangeFlavorProd(flavor, hostname)
			if err != nil {
				resp.Diagnostics.AddError(
					consts.UPDATE_RES_FAIL,
					fmt.Sprintf("Change app state (on changing machine state),\nError: %v", err.Error()))
				return
			}
		}
	} else {
		err := order.ChangeFlavor(flavor)
		if err != nil {
			resp.Diagnostics.AddError(
				consts.UPDATE_RES_FAIL,
				fmt.Sprintf("Change app state (on changing machine state),\nError: %v", err.Error()))
			return
		}
	}
}

func (r RedisResource) changeFinancialProject(
	order *orders.Redis,
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

func (r RedisResource) changeAccess(
	order *orders.Redis,
	plan *RedisResourceModel,
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
				"app:redis",
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

func (r RedisResource) compareADRoles(planAccess, stateAccess map[string][]string) (deletedRoles, changedRoles, addedRoles []entities.ADLogonGrants) {
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

func (r RedisResource) changeUserPassword(
	order *orders.Redis,
	plan, state *RedisResourceModel,
	resp *resource.UpdateResponse,
) {
	err := order.ChangeUserPassword(state.User.ValueString(), plan.UserPassword.ValueString(), false)
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

func (r RedisResource) prepareAttrs(plan RedisResourceModel) (attrs orders.RedisAttrs, diags diag.Diagnostics) {

	basicAttrs := utils.PrepareBasicAttrs(
		&plan.Flavor,
		&plan.Core,
		plan.Access,
		plan.ExtraMounts,
		plan.Image.OsVersion.ValueString(),
		plan.Image.ADIntegration.ValueBool(),
		plan.Image.OnSupport.ValueBool(),
	)

	if strings.EqualFold(r.client.Environment, "prod") {
		layout, err := references.GetGeoPage(
			r.client.Creds,
			"redis",
			plan.Core.NetSegmentCode.ValueString(),
		)
		if err != nil {
			diags.AddError(
				"Can't geo distribution informantion", err.Error(),
			)
			return
		}

		attrs = orders.RedisAttrs{
			BasicAttrs:           basicAttrs,
			ExtraNics:            []interface{}{},
			GeoDistribution:      true,
			LayoutID:             layout.ID,
			RedisVersion:         plan.RedisVersion.ValueString(),
			User:                 plan.User.ValueString(),
			UserPassword:         plan.UserPassword.ValueString(),
			NotifyKeyspaceEvents: plan.NotifyKeyspaceEvents.ValueString(),
			AcceptDocumentation:  true,
			Accept:               true,
			Auth:                 false,
		}
	} else {
		attrs = orders.RedisAttrs{
			BasicAttrs:           basicAttrs,
			ExtraNics:            []interface{}{},
			RedisVersion:         plan.RedisVersion.ValueString(),
			User:                 plan.User.ValueString(),
			UserPassword:         plan.UserPassword.ValueString(),
			NotifyKeyspaceEvents: plan.NotifyKeyspaceEvents.ValueString(),
			AcceptDocumentation:  true,
			Accept:               true,
			Auth:                 false,
		}
	}
	return attrs, diags
}

func (r RedisResource) changeNotifyKeyspaceEvents(
	order *orders.Redis,
	plan *RedisResourceModel,
	resp *resource.UpdateResponse,
) {
	err := order.ChangeRedisParamNotify(plan.NotifyKeyspaceEvents.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Change notify-keyspace-event value",
			fmt.Sprintf(
				"Changing notify-keyspace-event value ended with error.\nError message: %s",
				err.Error(),
			),
		)
		return
	}
}

func getVMHostnames(items []entities.Item) ([]string, error) {
	var hostnames []string
	for _, vmItem := range items {
		vmItem, ok := vmItem.Data.Config.(entities.VMItemConfig)
		if !ok {
			return nil, fmt.Errorf("invalid type for vmItem")
		}
		hostnames = append(hostnames, vmItem.Hostname)
	}
	return hostnames, nil
}
