package provider

import (
	"context"
	"fmt"
	"reflect"

	"regexp"

	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"
	"terraform-provider-vtb/pkg/client/references"
	"terraform-provider-vtb/pkg/client/sources"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                = &RedisSentinelResource{}
	_ resource.ResourceWithImportState = &RedisSentinelResource{}
)

type RedisSentinelResource struct {
	provider *VTBCloudProvider
}

func NewRedisSentinelResource(p *VTBCloudProvider) RedisSentinelResource {
	return RedisSentinelResource{
		provider: p,
	}
}

func (r RedisSentinelResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = "vtb_redis_sentinel_instance"
}

type RedisSentinelResourceModel struct {
	Core     CoreModel                 `tfsdk:"core"`
	Flavor   FlavorModel               `tfsdk:"flavor"`
	Image    ImageStandardModel        `tfsdk:"image"`
	Lifetime types.Int64               `tfsdk:"lifetime"`
	Label    types.String              `tfsdk:"label"`
	OrderID  types.String              `tfsdk:"order_id"`
	ItemID   types.String              `tfsdk:"item_id"`
	Access   map[string][]types.String `tfsdk:"access"`

	Hostname             types.String `tfsdk:"hostname"`
	RedisVersion         types.String `tfsdk:"redis_version"`
	User                 types.String `tfsdk:"user"`
	UserPassword         types.String `tfsdk:"user_password"`
	NotifyKeyspaceEvents types.String `tfsdk:"notify_keyspace_events"`
	FinancialProject     types.String `tfsdk:"financial_project"`
}

func (r RedisSentinelResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			// basic schema
			"core": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          coreSchema,
				Description:         "Основные параметры для виртуальных машин в заказе.",
				MarkdownDescription: "Основные параметры для виртуальных машин в заказе.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"flavor": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          flavorSchema,
				Description:         "Кол-во CPU/RAM для виртуальных машин.",
				MarkdownDescription: "Кол-во CPU/RAM для виртуальных машин.",
				PlanModifiers: []planmodifier.Object{
					AvailableFlavor(r.provider, "app:redis"),
				},
			},
			"image": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          imageStandardSchema,
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
					CheckIsDeleted(r.provider, "app", "redis_sentinel"),
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
					ActiveDirectoryAccesForSpecificEnvironment(r.provider, "DEV", "TEST"),
					AvailableAccessRole(r.provider, "app:redis"),
					EmptyAccessGroup(),
				},
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
				Validators: []validator.Map{
					EmptyAccessGroupListValidator{},
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

func (r RedisSentinelResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r RedisSentinelResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan RedisSentinelResourceModel
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

	attrs, diags := r.prepareAttrs(plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	order := orders.NewRedisSentinel(
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
		resp.Diagnostics.AddError(
			CREATE_RES_FAIL,
			fmt.Sprintf("Create order ended with error.\nError: %v", err.Error()),
		)
		return
	}

	orderItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			CREATE_RES_FAIL,
			fmt.Sprintf("Get order item ended with error.\nError: %v", err.Error()),
		)
		return
	}

	vmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			CREATE_RES_FAIL,
			fmt.Sprintf("Get order vm items ended with error.\nError: %v", err.Error()),
		)
		return
	}

	if (orderItem.Type == "app") && (len(vmItems) != 3) {
		resp.Diagnostics.AddError(
			CREATE_RES_FAIL,
			"Error while update state: number of VM is not equal 3",
		)
		return
	}

	itemData := vmItems[0].Data

	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(orderItem.ID)
	plan.Hostname = types.StringValue(itemData.Config.(entities.VMItemConfig).Hostname)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r RedisSentinelResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var orderID types.String
	req.State.GetAttribute(ctx, path.Root("order_id"), &orderID)
	if orderID.IsNull() {
		resp.Diagnostics.AddError(
			"Read order from portal",
			"Resource hasn't order_id. Without order_id can't get order from portal",
		)
		return
	}

	order, err := orders.GetRedisSentinelOrder(
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

	appItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			READ_RES_FAIL,
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
			READ_RES_FAIL,
			fmt.Sprintf("Get order vm items ended with error.\nError: %v", err.Error()),
		)
		return
	}

	var vmItem entities.Item
	if appItem.Type == "app" {
		if len(vmItems) == 3 {
			RedisSentinelVmItems, err := order.GetRedisSentinelVmItems()
			if err != nil {
				resp.Diagnostics.AddError(
					READ_RES_FAIL,
					fmt.Sprintf("Get Redis VM items ended with error.\nError: %v", err.Error()),
				)
			}
			vmItem = RedisSentinelVmItems[0]
		}
	}

	redisUser, err := order.GetDefaultRedisUser()
	if err != nil {
		resp.Diagnostics.AddError(
			"Get user from order",
			fmt.Sprintf("Error: %s", ctx),
		)
		return
	}

	var UserPassword types.String
	req.State.GetAttribute(ctx, path.Root("user_password"), &UserPassword)

	redisConfig, ok := appItem.Data.Config.(entities.RedisSentinelItemConfig)
	if !ok {

		resp.Diagnostics.AddError(
			READ_RES_FAIL,
			"Invalid type for redisConfig",
		)
		return
	}

	vmConfig, ok := vmItem.Data.Config.(entities.VMItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			READ_RES_FAIL,
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

	state := RedisSentinelResourceModel{
		OrderID:              orderID,
		Access:               readAccessMapV2(vmItem.Data.ACLs),
		Label:                types.StringValue(order.Label),
		NotifyKeyspaceEvents: notifyKeyspaceEvents,
		RedisVersion:         types.StringValue(redisConfig.Version),
		User:                 types.StringValue(redisUser.UserName),
		UserPassword:         UserPassword,
		ItemID:               types.StringValue(appItem.ID),
		Hostname:             types.StringValue(vmConfig.Hostname),
		Flavor: FlavorModel{
			Cores:  types.Int64Value(vmConfig.Flavor.Cores),
			Memory: types.Int64Value(vmConfig.Flavor.Memory),
			Name:   types.StringValue(vmConfig.Flavor.Name),
			UUID:   types.StringValue(vmConfig.Flavor.UUID),
		},
		Image: ImageStandardModel{
			Distribution:  types.StringValue(vmConfig.Image.Os.Distribution),
			OsVersion:     types.StringValue(vmConfig.Image.Os.Version),
			OnSupport:     types.BoolValue(order.Attrs.OnSupport),
			ProductID:     types.StringValue(order.ProductID),
			ADIntegration: types.BoolValue(order.Attrs.ADIntegration),
		},
		Core: CoreModel{
			Platform:       types.StringValue(order.Attrs.Platform),
			Domain:         types.StringValue(order.Attrs.Domain),
			NetSegmentCode: types.StringValue(vmConfig.DefaultNic.NetSegment),
			Zone:           types.StringValue(order.Attrs.AvailabilityZone),
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

func (r RedisSentinelResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state RedisSentinelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetRedisSentinelOrder(
		r.provider.Creds,
		r.provider.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("order_id"),
			UPDATE_RES_FAIL,
			fmt.Sprintf("Get order data from portal ended with error.\nError: %v", err.Error()),
		)
		return
	}

	finProj, err := sources.GetFinancialProjectByName(
		r.provider.Creds,
		r.provider.ProjectName,
		plan.FinancialProject.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("financial_project"),
			UPDATE_RES_FAIL,
			fmt.Sprintf("Get financial source for project ended with error.\nError: %v", err.Error()))
		return
	}

	labelChanged := plan.Label != state.Label
	passwordChanged := state.UserPassword != plan.UserPassword
	flavorChanged := plan.Flavor != state.Flavor
	notifyKeyspaceEventsChanged := plan.NotifyKeyspaceEvents != state.NotifyKeyspaceEvents
	financialProjectChanged := !plan.FinancialProject.Equal(state.FinancialProject)
	accessChanged := !reflect.DeepEqual(state.Access, plan.Access)

	if labelChanged {
		changeOrderLabel(order, plan.Label.ValueString(), resp)
	}

	if financialProjectChanged {
		r.changeFinancialProject(order, finProj.ID, resp)
	}

	if passwordChanged {
		r.changeUserPassword(order, &plan, &state, resp)
	}

	if notifyKeyspaceEventsChanged {
		r.changeNotifyKeyspaceEvents(order, &plan, resp)
	}

	if flavorChanged {
		r.changeFlavorRedisSentinel(order, &plan, resp)
	}

	if accessChanged {
		r.changeAccess(order, &plan, resp)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r RedisSentinelResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data RedisSentinelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetRedisSentinelOrder(
		r.provider.Creds,
		r.provider.ProjectName,
		data.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("order_id"),
			DELETE_RES_FAIL,
			fmt.Sprintf("Get order data from portla ended with error.\nError:%v", err.Error()),
		)
		return
	}

	orderState, err := order.GetState()
	if err != nil {
		resp.Diagnostics.AddError(
			DELETE_RES_FAIL,
			fmt.Sprintf("Can't get state from portal.\nError:%s", err.Error()),
		)
	}

	if orderState == "deleted" || orderState == "deprovisioned" {
		resp.Diagnostics.AddWarning(
			DELETE_RES_FAIL,
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
			r.provider.Organization,
		)

		resp.Diagnostics.AddError(
			DELETE_RES_FAIL,
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

// custom logic
func (r RedisSentinelResource) prepareAttrs(plan RedisSentinelResourceModel) (attrs orders.RedisSentinelAttrs, diags diag.Diagnostics) {

	basicAttrs := orders.BasicAttrs{
		ADIntegration:    plan.Image.ADIntegration.ValueBool(),
		ADLogonGrants:    prepareADLogonGrants(plan.Access),
		OnSupport:        plan.Image.OnSupport.ValueBool(),
		OsVersion:        plan.Image.OsVersion.ValueString(),
		AvailabilityZone: plan.Core.Zone.ValueString(),
		Domain:           plan.Core.Domain.ValueString(),
		Platform:         plan.Core.Platform.ValueString(),
		DefaultNic: entities.DefaultNic{
			NetSegment: plan.Core.NetSegmentCode.ValueString(),
		},
		Flavor: entities.Flavor{
			Cores:  plan.Flavor.Cores.ValueInt64(),
			Memory: plan.Flavor.Memory.ValueInt64(),
			Name:   plan.Flavor.Name.ValueString(),
			UUID:   plan.Flavor.UUID.ValueString(),
		},
		CreatedWithOpenTofu: true,
	}
	layout, err := references.GetGeoPage(
		r.provider.Creds,
		"redis_sentinel",
		plan.Core.NetSegmentCode.ValueString(),
	)
	if err != nil {
		diags.AddError(
			CREATE_RES_FAIL,
			fmt.Sprintf("Can't geo distribution informantion.\nError: %v", err.Error()),
		)
		return attrs, diags
	}

	attrs = orders.RedisSentinelAttrs{
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
	return attrs, diags
}

func (r RedisSentinelResource) changeFlavorRedisSentinel(
	order *orders.RedisSentinel,
	plan *RedisSentinelResourceModel,
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
			UPDATE_RES_FAIL,
			fmt.Sprintf("Change app state (on changing machine state).\nError: %v", err.Error()),
		)
		return
	}
}

func (r RedisSentinelResource) changeNotifyKeyspaceEvents(
	order *orders.RedisSentinel,
	plan *RedisSentinelResourceModel,
	resp *resource.UpdateResponse,
) {
	err := order.ChangeNotifyKeyspaceEvents(plan.NotifyKeyspaceEvents.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			UPDATE_RES_FAIL,
			fmt.Sprintf(
				"Chandge notify keyspace events ended with error.\nError: %s",
				err.Error(),
			),
		)
		return
	}
}

func (r RedisSentinelResource) changeUserPassword(
	order *orders.RedisSentinel,
	plan, state *RedisSentinelResourceModel,
	resp *resource.UpdateResponse,
) {
	err := order.ChangeUserPassword(state.User.ValueString(), plan.UserPassword.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			UPDATE_RES_FAIL,
			fmt.Sprintf(
				"Change user password enden with error.\nError: %v",
				err.Error(),
			),
		)
		return
	}
}

func (r RedisSentinelResource) changeFinancialProject(
	order *orders.RedisSentinel,
	finProjectId string,
	resp *resource.UpdateResponse,
) {
	err := order.ChangeFinancialProject(finProjectId)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("financial_project"),
			UPDATE_RES_FAIL,
			fmt.Sprintf("Change financial source for project ended wiht error: %v", err.Error()),
		)
		return
	}
}

func (r RedisSentinelResource) changeAccess(
	order *orders.RedisSentinel,
	plan *RedisSentinelResourceModel,
	resp *resource.UpdateResponse,
) {
	vmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			READ_RES_FAIL,
			fmt.Sprintf("Get VM items ended with error.\nError: %v", err.Error()))
		return
	}

	for _, vmItem := range vmItems {
		stateAccess := readAccessMapVV1(vmItem.Data.ACLs)
		planAccess := convertAccessMap(plan.Access)
		deletedRoles, changedRoles, addedRoles := r.compareADRoles(planAccess, stateAccess)

		if len(addedRoles) > 0 {
			rolesFromCloud, err := references.GetRoles(
				r.provider.Creds,
				r.provider.Environment,
				"app:redis",
			)
			if err != nil {
				resp.Diagnostics.AddError(UPDATE_RES_FAIL, err.Error())
				return
			}

			permissionsMap := make(map[string][]string)
			for _, value := range rolesFromCloud {
				permissionsMap[value.Data.GroupName] = value.Data.Permissions
			}

			for _, roles := range addedRoles {
				err := order.AddAccessGroupForVm(vmItem.ID, roles, permissionsMap[roles.Role])
				if err != nil {
					resp.Diagnostics.AddError(UPDATE_RES_FAIL, err.Error())
					return
				}
			}
		}

		if len(changedRoles) > 0 {
			for _, roles := range changedRoles {
				err := order.ChangeAccessGroupForVm(vmItem.ID, roles)
				if err != nil {
					resp.Diagnostics.AddError(UPDATE_RES_FAIL, err.Error())
					return
				}
			}
		}

		if len(deletedRoles) > 0 {
			for _, roles := range deletedRoles {
				err := order.DeleteAccessGroupForVm(vmItem.ID, roles)
				if err != nil {
					resp.Diagnostics.AddError(UPDATE_RES_FAIL, err.Error())
					return
				}
			}
		}
	}
}

func (r RedisSentinelResource) compareADRoles(planAccess, stateAccess map[string][]string) (deletedRoles, changedRoles, addedRoles []entities.ADLogonGrants) {
	for role, groups := range stateAccess {
		if _, ok := planAccess[role]; !ok {
			deletedRoles = append(
				deletedRoles,
				entities.ADLogonGrants{Role: role, Groups: groups},
			)
		} else {
			if len(planAccess[role]) == len(stateAccess[role]) {
				if differenceLen(planAccess[role], stateAccess[role]) {
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
