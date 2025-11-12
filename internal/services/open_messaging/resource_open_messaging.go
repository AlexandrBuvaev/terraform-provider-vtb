package openmessaging

import (
	"context"
	"fmt"
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
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
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
)

const (
	ADMIN_ROLE     = "artemis_admin"
	USER_ROLE      = "user"
	SUPERUSER_ROLE = "superuser"
)

var (
	_ resource.Resource                = &OpenMessagingResource{}
	_ resource.ResourceWithImportState = &OpenMessagingResource{}

	defaultClientDn = "CN=TEST.artemisusr, OU=ART, O=VTB Bank, C=RU"
)

type OpenMessagingResource struct {
	client *client.CloudClient
}

func NewOpenMessagingResource() resource.Resource {
	return &OpenMessagingResource{}
}

func (r OpenMessagingResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_open_messaging_instance"
}

func (r *OpenMessagingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type OpenMessagingResourceModel struct {
	Core             core.CoreModel                    `tfsdk:"core"`
	Flavor           flavor.FlavorModel                `tfsdk:"flavor"`
	Image            common.ImageStandardModel         `tfsdk:"image"`
	Lifetime         types.Int64                       `tfsdk:"lifetime"`
	Label            types.String                      `tfsdk:"label"`
	OrderID          types.String                      `tfsdk:"order_id"`
	ItemID           types.String                      `tfsdk:"item_id"`
	ExtraMounts      map[string]common.ExtraMountModel `tfsdk:"extra_mounts"`
	FinancialProject types.String                      `tfsdk:"financial_project"`

	AdminGroups     types.List `tfsdk:"admin_groups"`
	UserGroups      types.List `tfsdk:"user_groups"`
	SuperuserGroups types.List `tfsdk:"superuser_groups"`
}

func (r OpenMessagingResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"core": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.CoreSchema,
				MarkdownDescription: "Основные параметры для ВМ в заказе",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"flavor": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.FlavorSchema,
				MarkdownDescription: "Core/memory вычислительного экземпляра",
			},
			"image": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.ImageStandardSchema,
				MarkdownDescription: "Тип вычислительного экземпляра",
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
				Required:            true,
				MarkdownDescription: "Метка заказа",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"order_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Идентификатор заказа. Известен после создания заказа",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"item_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Идентификатор сущности ВМ в заказе. Известен после создания заказа",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"extra_mounts": schema.MapNestedAttribute{
				Required:            true,
				MarkdownDescription: "Дополнительные точки монтирования в вычислительном экземпляре",
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.OneOf("/app")),
					mapvalidator.SizeBetween(1, 1),
				},
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"size": schema.Int64Attribute{
							Required:            true,
							MarkdownDescription: "Размер точки монтирования (в ГБ)",
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
							Description:         "Тип файловой системы",
							MarkdownDescription: "Тип файловой системы",
						},
					},
				},
			},

			"admin_groups": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Список групп доступа с ролью `artemis_admin`",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
			"user_groups": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Список групп доступа с ролью `user`",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},
			"superuser_groups": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Список групп доступа с ролью `superuser`",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
				},
			},

			"financial_project": schema.StringAttribute{
				Required:            true,
				Description:         "Источник финансирования заказа",
				MarkdownDescription: "Источник финансирования заказа",
			},
		},
	}
}

func (r OpenMessagingResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r OpenMessagingResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan OpenMessagingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	checkOrderIsDeleted := utils.CheckOrderIsDeleted(r.client, plan.OrderID, "vm", "openstack")
	resp.Diagnostics.Append(checkOrderIsDeleted.Diagnostics...)

	utils.HandleExternalDeletionRecreate(ctx, resp, &plan, checkOrderIsDeleted.IsDeleted, checkOrderIsDeleted.Diagnostics)

	r.validateFieldsForSpecificEnvironments(plan, resp)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r OpenMessagingResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan OpenMessagingResourceModel
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

	var superUsers, users, admins []string
	if !plan.SuperuserGroups.IsNull() && !plan.SuperuserGroups.IsUnknown() {
		plan.SuperuserGroups.ElementsAs(ctx, &superUsers, false)
	} else {
		superUsers = []string{}
	}

	if !plan.UserGroups.IsNull() && !plan.UserGroups.IsUnknown() {
		plan.UserGroups.ElementsAs(ctx, &users, false)
	} else {
		users = []string{}
	}

	plan.AdminGroups.ElementsAs(ctx, &admins, false)

	extraMounts := utils.PrepareExtraMountsAttrs(plan.ExtraMounts)

	attrs := orders.OpenMessagingAttrs{
		Path:             "/app",
		FileSystem:       "xfs",
		Size:             int(extraMounts[0].Size),
		Domain:           plan.Core.Domain.ValueString(),
		Platform:         plan.Core.Platform.ValueString(),
		OnSupport:        plan.Image.OnSupport.ValueBool(),
		OsVersion:        plan.Image.OsVersion.ValueString(),
		AdIntegration:    plan.Image.ADIntegration.ValueBool(),
		AvailabilityZone: plan.Core.Zone.ValueString(),
		DefaultNic: entities.DefaultNic{
			NetSegment: plan.Core.NetSegmentCode.ValueString(),
		},
		Flavor: entities.Flavor{
			Cores:  plan.Flavor.Cores.ValueInt64(),
			Memory: plan.Flavor.Memory.ValueInt64(),
			Name:   string(plan.Flavor.Name.ValueString()),
			UUID:   string(plan.Flavor.UUID.ValueString()),
		},
		SuperuserGroups:     superUsers,
		UserGroups:          users,
		AdminGroups:         admins,
		CreatedWithOpenTofu: true,
	}

	if strings.EqualFold(r.client.EnvironmentName, "lt") {
		attrs.ClientDn = &defaultClientDn
	}

	order := orders.NewOpenMessaging(
		r.client.Creds,
		r.client.ProjectName,
		plan.Image.ProductID.ValueString(),
		attrs,
	)

	err = order.Create(
		orders.CreateOrderPayload{
			Label:        plan.Label.ValueString(),
			Lifetime:     int(plan.Lifetime.ValueInt64()),
			FinProjectID: finProj.ID,
		},
	)
	if err != nil {
		resp.Diagnostics.AddError(consts.CREATE_RES_FAIL, err.Error())
		return
	}

	orderItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError("Error while getting item: ", err.Error())
		return
	}

	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(orderItem.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r OpenMessagingResource) Read(
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

	order, err := orders.GetOpenMessagingOrder(
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

	openMessagingItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get vm item from portal with order_id '%s'.\nError:%s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}

	vmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Can't get OM `vm` items.\nError: %s", err.Error()),
		)
		return
	}

	vmConfig := vmItems[0].Data.Config.(entities.VMItemConfig)

	users, err := order.GetRoleGroups("user")
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("user_groups"),
			consts.READ_RES_FAIL,
			fmt.Sprintf("Error for fetch groups for `user` role, error: %s", err.Error()),
		)
		return
	}

	superUsers, err := order.GetRoleGroups("superuser")
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("superuser_groups"),
			consts.READ_RES_FAIL,
			fmt.Sprintf("Error for fetch groups for `superuser` role, error: %s", err.Error()),
		)
		return
	}

	admins, err := order.GetRoleGroups("artemis_admin")
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("admin_groups"),
			consts.READ_RES_FAIL,
			fmt.Sprintf("Error for fetch groups for `artemis_admin` role, error: %s", err.Error()),
		)
		return
	}

	tfusers, _ := types.ListValueFrom(ctx, types.StringType, users)
	tfadmins, _ := types.ListValueFrom(ctx, types.StringType, admins)
	tfsuperUsers, _ := types.ListValueFrom(ctx, types.StringType, superUsers)

	state := OpenMessagingResourceModel{
		OrderID: orderID,
		ItemID:  types.StringValue(openMessagingItem.ID),
		Label:   types.StringValue(order.Label),
		Flavor: flavor.FlavorModel{
			Cores:  types.Int64Value(vmConfig.Flavor.Cores),
			Memory: types.Int64Value(vmConfig.Flavor.Memory),
			UUID:   types.StringValue(vmConfig.Flavor.UUID),
			Name:   types.StringValue(vmConfig.Flavor.Name),
		},
		ExtraMounts: map[string]common.ExtraMountModel{
			"/app": {
				Size:       types.Int64Value(int64(vmConfig.ExtraMounts[0].Size)),
				FileSystem: types.StringValue(vmConfig.ExtraMounts[0].Fstype),
			},
		},
		Core: core.CoreModel{
			Platform:       types.StringValue(utils.SelectPlatform(order.Attrs.Platform)),
			Zone:           types.StringValue(order.Attrs.AvailabilityZone),
			Domain:         types.StringValue(order.Attrs.Domain),
			NetSegmentCode: types.StringValue(order.Attrs.DefaultNic.NetSegment),
		},
		Image: common.ImageStandardModel{
			Distribution:  types.StringValue(vmConfig.Image.Os.Distribution),
			OsVersion:     types.StringValue(vmConfig.Image.Os.Version),
			OnSupport:     types.BoolValue(order.Attrs.OnSupport),
			ProductID:     types.StringValue(order.ProductID),
			ADIntegration: types.BoolValue(order.Attrs.AdIntegration),
		},
		FinancialProject: types.StringValue(order.FinancialSource.Name),
		AdminGroups:      tfadmins,
		UserGroups:       tfusers,
		SuperuserGroups:  tfsuperUsers,
	}

	var lifetime types.Int64
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("lifetime"), &lifetime)...)
	if !lifetime.IsNull() {
		state.Lifetime = lifetime
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r OpenMessagingResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state OpenMessagingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetOpenMessagingOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(consts.UPDATE_RES_FAIL, err.Error())
		return
	}

	finProj, err := sources.GetFinancialProjectByName(
		r.client.Creds,
		r.client.ProjectName,
		plan.FinancialProject.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("financial_project"), consts.UPDATE_RES_FAIL, err.Error())
		return
	}

	labelChanged := plan.Label != state.Label
	flavorChanged := plan.Flavor != state.Flavor

	if labelChanged {
		utils.ChangeOrderLabel(order, plan.Label.ValueString(), resp)
	}

	if !plan.FinancialProject.Equal(state.FinancialProject) {
		err := order.ChangeFinancialProject(finProj.ID)
		if err != nil {
			resp.Diagnostics.AddError(
				consts.UPDATE_RES_FAIL,
				fmt.Sprintf("Change financial source for project ended with error: %s", err.Error()),
			)
			return
		}
	}

	if !plan.AdminGroups.Equal(state.AdminGroups) {
		r.applyAccessGroups(ctx, ADMIN_ROLE, plan.AdminGroups, state.AdminGroups, order, resp)
	}

	if !plan.UserGroups.Equal(state.UserGroups) {
		r.applyAccessGroups(ctx, USER_ROLE, plan.UserGroups, state.UserGroups, order, resp)
	}

	if !plan.SuperuserGroups.Equal(state.SuperuserGroups) {
		r.applyAccessGroups(ctx, SUPERUSER_ROLE, plan.SuperuserGroups, state.SuperuserGroups, order, resp)
	}

	if flavorChanged {
		r.updateFlavor(order, &plan, &state, resp)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r OpenMessagingResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state OpenMessagingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetOpenMessagingOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Can't get order from portal: ", err.Error())
		return
	}

	orderState, err := order.GetState()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf("Can't get state from portal.\nError:%s", err.Error()),
		)
		return
	}

	if orderState == "deleted" || orderState == "deprovisioned" {
		resp.Diagnostics.AddWarning(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf(
				"Order with id '%s' was deleted by another way. Not from terraform. "+
					"Resource was recreated",
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
		resp.Diagnostics.AddError(consts.DELETE_RES_FAIL, err.Error())
		return
	}
}

func (r OpenMessagingResource) updateFlavor(
	order *orders.OpenMessagingOrder,
	plan *OpenMessagingResourceModel,
	state *OpenMessagingResourceModel,
	resp *resource.UpdateResponse,
) {
	attrs := orders.OpenMessagingVerticalScalingAttrs{
		ReadIT: true,
		Flavor: entities.Flavor{
			Cores:  plan.Flavor.Cores.ValueInt64(),
			Memory: plan.Flavor.Memory.ValueInt64(),
			Name:   plan.Flavor.Name.ValueString(),
			UUID:   plan.Flavor.UUID.ValueString(),
		},
		CurrentFlavor:          state.Flavor.Name.ValueString(),
		StateServiceRAM:        state.Flavor.Memory.ValueInt64(),
		StateServiceCPU:        state.Flavor.Cores.ValueInt64(),
		StateServiceFlavorName: state.Flavor.Name.ValueString(),
	}

	if strings.ToLower(r.client.Environment) == "prod" {
		attrs.ChangeRequest = true
	}

	err := order.VerticalScaling(attrs, r.client.EnvironmentName)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("flavor"),
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "vertical scaling ended with error"),
			err.Error(),
		)
	}
}

func (r OpenMessagingResource) applyAccessGroups(
	ctx context.Context,
	role string,
	planGroups, stateGroups types.List,
	order *orders.OpenMessagingOrder,
	resp *resource.UpdateResponse,
) {
	if planGroups.IsNull() || planGroups.IsUnknown() {
		var groups []string
		diags := stateGroups.ElementsAs(ctx, &groups, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		toDelte := entities.ADLogonGrants{
			Role:   role,
			Groups: groups,
		}

		if len(toDelte.Groups) == 0 {
			resp.Diagnostics.AddWarning(
				consts.UPDATE_RES_FAIL,
				fmt.Sprintf("There are no groups available to delete for this role: %s", toDelte.Role),
			)
			return
		}

		err := order.DeleteAccessGroup(toDelte, r.client.Environment)
		if err != nil {
			resp.Diagnostics.AddError(
				consts.UPDATE_RES_FAIL,
				fmt.Sprintf("Delete access group ended with error %s: ", err.Error()),
			)
			return
		}
	}

	if !planGroups.IsNull() && stateGroups.IsNull() {
		var groups []string
		diags := planGroups.ElementsAs(ctx, &groups, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		toAdd := entities.ADLogonGrants{
			Role:   role,
			Groups: groups,
		}

		rolesFromCloud, err := references.GetRoles(
			r.client.Creds,
			r.client.Environment,
			"app:artemis",
		)
		if err != nil {
			diags.AddError("Receiving roles from Cloud", err.Error())
			return
		}

		permissionsMap := make(map[string][]string)
		for _, value := range rolesFromCloud {
			permissionsMap[value.Data.GroupName] = value.Data.Permissions
		}
		err = order.AddAccessGroup(toAdd, permissionsMap[role], r.client.Environment)
		if err != nil {
			resp.Diagnostics.AddError(
				consts.UPDATE_RES_FAIL,
				fmt.Sprintf("Add access group ended with error %s: ", err.Error()),
			)
			return
		}
	}

	if !planGroups.IsNull() && !planGroups.IsUnknown() && !stateGroups.IsNull() {
		var groups []string
		diags := planGroups.ElementsAs(ctx, &groups, false)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		toUpdate := entities.ADLogonGrants{
			Role:   role,
			Groups: groups,
		}
		err := order.UpdateAccessGroup(toUpdate, r.client.Environment)
		if err != nil {
			resp.Diagnostics.AddError(
				consts.UPDATE_RES_FAIL,
				fmt.Sprintf("Update access group ended with error %s: ", err.Error()),
			)
			return
		}
	}
}

func (r OpenMessagingResource) validateFieldsForSpecificEnvironments(
	plan OpenMessagingResourceModel,
	resp *resource.ModifyPlanResponse,
) {
	if !strings.EqualFold(r.client.Environment, "dev") {
		if !plan.SuperuserGroups.IsNull() {
			resp.Diagnostics.AddError(
				consts.MODIFY_PLAN_FAIL,
				"Field: `superuser_groups`, are specific for DEV environment only",
			)
		}
		return
	}

	if plan.SuperuserGroups.IsNull() {
		resp.Diagnostics.AddError(
			consts.MODIFY_PLAN_FAIL,
			"Field: `superuser_groups` are required for DEV environment",
		)
		return
	}
}
