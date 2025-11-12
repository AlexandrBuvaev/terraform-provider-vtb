package astra

import (
	"context"
	"fmt"
	"reflect"

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &ComputeResource{}
	_ resource.ResourceWithImportState = &ComputeResource{}
	_ resource.ResourceWithModifyPlan  = &ComputeResource{}
)

type ComputeResource struct {
	client *client.CloudClient
}

func NewComputeResource() resource.Resource {
	return &ComputeResource{}
}

func (r *ComputeResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_compute_instance"
}

func (r *ComputeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type ComputeResourceModel struct {
	Core             core.CoreModel                    `tfsdk:"core"`
	Flavor           flavor.FlavorModel                `tfsdk:"flavor"`
	Image            common.ImageStandardModel         `tfsdk:"image"`
	Lifetime         types.Int64                       `tfsdk:"lifetime"`
	Label            types.String                      `tfsdk:"label"`
	OrderID          types.String                      `tfsdk:"order_id"`
	ItemID           types.String                      `tfsdk:"item_id"`
	Access           map[string][]string               `tfsdk:"access"`
	ExtraMounts      map[string]common.ExtraMountModel `tfsdk:"extra_mounts"`
	Hostname         types.String                      `tfsdk:"hostname"`
	FixedIP          types.String                      `tfsdk:"fixed_ip"`
	FinancialProject types.String                      `tfsdk:"financial_project"`
}

func (r *ComputeResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"core": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.CoreSchema,
				Description:         "Ключевые параметры для VM и заказа.",
				MarkdownDescription: "Ключевые параметры для VM и заказа.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"flavor": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.FlavorSchema,
				Description:         "CPU/RAM для вычислительного экземпляра.",
				MarkdownDescription: "CPU/RAM для вычислительного экземпляра.",
			},
			"image": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.ImageStandardSchema,
				Description:         "Тип вычилительного экземпляра.",
				MarkdownDescription: "Тип вычилительного экземпляра.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
					objectplanmodifier.UseStateForUnknown(),
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
				Description:         "Название заказа.",
				MarkdownDescription: "Название заказа.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"order_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Идентификатор заказа, известен после того, как будет создан заказ.",
				MarkdownDescription: "Идентификатор заказа, известен после того, как будет создан заказ.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"item_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Идентификатор сущности VM в заказе. Известен после того, как будет создана VM.",
				MarkdownDescription: "Идентификатор сущности VM в заказе. Известен после того, как будет создана VM.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"access": schema.MapAttribute{
				Required: true,
				Description: "Словарь,где ключом является роль, а значением список групп, " +
					"которые предоставляют доступ для входа в Active Directory",
				MarkdownDescription: "Словарь,где ключом является роль, а значением список групп, " +
					"которые предоставляют доступ для входа в Active Directory",
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
				},
			},
			"extra_mounts": schema.MapNestedAttribute{
				Required:            true,
				Description:         "Дополнительные точки монтирования для вычслительной машины.",
				MarkdownDescription: "Дополнительные точки монтирования для вычслительной машины.",
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.OneOf("/app")),
					mapvalidator.SizeBetween(1, 1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"size": schema.Int64Attribute{
							Required:            true,
							MarkdownDescription: "Размер точки монтирования (размер в ГБ)",
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
			"hostname": schema.StringAttribute{
				Computed:            true,
				Description:         "Имя хоста заказаной вычислительной машины.",
				MarkdownDescription: "Имя хоста заказаной вычислительной машины.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"fixed_ip": schema.StringAttribute{
				Computed:            true,
				Description:         "Фиксированый IPv4 в сети по умолчанию.",
				MarkdownDescription: "Фиксированый IPv4 в сети по умолчанию.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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

func (r *ComputeResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	var plan ComputeResourceModel

	if req.Plan.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	validateRolesDiags := utils.ValidateAccessRolesV1(
		r.client,
		"vm:linux",
		plan.Access,
	)
	resp.Diagnostics.Append(validateRolesDiags...)

	checkOrderIsDeleted := utils.CheckOrderIsDeleted(r.client, plan.OrderID, "app", "nginx")
	resp.Diagnostics.Append(checkOrderIsDeleted.Diagnostics...)

	utils.HandleExternalDeletionRecreate(
		ctx,
		resp,
		&plan,
		checkOrderIsDeleted.IsDeleted,
		checkOrderIsDeleted.Diagnostics,
	)

	if resp.Diagnostics.HasError() {
		return
	}

}

func (r *ComputeResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r *ComputeResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan ComputeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ADLogonGrants := []entities.ADLogonGrants{}
	for role, groups := range plan.Access {
		ADLogonGrants = append(ADLogonGrants, entities.ADLogonGrants{
			Role:   role,
			Groups: groups,
		})
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

	attrs := orders.ComputeAttrs{
		ExtraNics: []interface{}{},
		BasicAttrs: orders.BasicAttrs{
			ExtraMounts:      utils.PrepareExtraMountsAttrs(plan.ExtraMounts),
			ADIntegration:    plan.Image.ADIntegration.ValueBool(),
			ADLogonGrants:    ADLogonGrants,
			AvailabilityZone: plan.Core.Zone.ValueString(),
			OnSupport:        plan.Image.OnSupport.ValueBool(),
			Platform:         plan.Core.Platform.ValueString(),
			Domain:           plan.Core.Domain.ValueString(),
			DefaultNic: entities.DefaultNic{
				NetSegment: plan.Core.NetSegmentCode.ValueString(),
			},
			Flavor: entities.Flavor{
				Cores:  plan.Flavor.Cores.ValueInt64(),
				Memory: plan.Flavor.Memory.ValueInt64(),
				Name:   string(plan.Flavor.Name.ValueString()),
				UUID:   string(plan.Flavor.UUID.ValueString()),
			},
			CreatedWithOpenTofu: true,
		},
	}

	// Получения image_id для продукта Linux preprom (Тест для УСИ Линукс)
	// Только для blue!
	if plan.Label.ValueString() == "linux_preprom" {
		imageID, err := references.GetImageID(
			*r.client.Creds,
			plan.Image.OsVersion.ValueString(),
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Can't fetch image data for distribution linux_preprom",
				err.Error(),
			)
		}
		attrs.ImageID = imageID
	} else {
		attrs.OsVersion = plan.Image.OsVersion.ValueString()
	}

	order := orders.NewCompute(
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
		resp.Diagnostics.AddError("Can't create virtual machine: ", err.Error())
		return
	}

	parentItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError("Error while update state: ", err.Error())
		return
	}

	config := parentItem.Data.Config.(entities.VMItemConfig)

	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(parentItem.ID)
	plan.Hostname = types.StringValue(config.Hostname)
	plan.FixedIP = types.StringValue(config.DefaultNic.Addresses[0].Address)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ComputeResource) Read(
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

	order, err := orders.GetComputeOrder(
		r.client.Creds,
		r.client.ProjectName,
		orderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("order_id"), consts.READ_RES_FAIL, err.Error())
		return
	}

	item, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get vm item from portal with order_id='%s'.\nError:%s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}
	config := item.Data.Config.(entities.VMItemConfig)

	appExtraMount, err := order.GetExtraMount("/app")
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get vm extra mount '/app' from portal with order_id '%s'.\nError:%s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}

	state := ComputeResourceModel{
		OrderID:  orderID,
		Access:   utils.ReadAccessMapVV1(item.Data.ACLs),
		ItemID:   types.StringValue(item.ID),
		Label:    types.StringValue(order.Label),
		Hostname: types.StringValue(config.Hostname),
		FixedIP:  types.StringValue(config.DefaultNic.Addresses[0].Address),
		Flavor: flavor.FlavorModel{
			Cores:  types.Int64Value(config.Flavor.Cores),
			Memory: types.Int64Value(config.Flavor.Memory),
			UUID:   types.StringValue(config.Flavor.UUID),
			Name:   types.StringValue(config.Flavor.Name),
		},
		ExtraMounts: map[string]common.ExtraMountModel{
			"/app": {
				Size:       types.Int64Value(appExtraMount.Size),
				FileSystem: types.StringValue(appExtraMount.FileSystem),
			},
		},
		Core: core.CoreModel{
			Platform:       types.StringValue(utils.SelectPlatform(order.Attrs.Platform)),
			Domain:         types.StringValue(order.Attrs.Domain),
			NetSegmentCode: types.StringValue(config.DefaultNic.NetSegment),
			Zone:           types.StringValue(order.Attrs.AvailabilityZone),
		},
		Image: common.ImageStandardModel{
			Distribution:  types.StringValue(config.Image.Os.Distribution),
			OsVersion:     types.StringValue(config.Image.Os.Version),
			OnSupport:     types.BoolValue(order.Attrs.OnSupport),
			ProductID:     types.StringValue(order.ProductID),
			ADIntegration: types.BoolValue(order.Attrs.ADIntegration),
		},
		FinancialProject: types.StringValue(order.FinancialSource.Name),
	}

	var lifetime types.Int64
	req.State.GetAttribute(ctx, path.Root("lifetime"), &lifetime)
	if !lifetime.IsNull() {
		state.Lifetime = lifetime
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ComputeResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state ComputeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetComputeOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Can't get order from portal: ", err.Error())
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
	mountChanged := utils.IsExtraMountChanged(state.ExtraMounts, plan.ExtraMounts)
	flavorChanged := plan.Flavor != state.Flavor
	accessChanged := !reflect.DeepEqual(plan.Access, state.Access)

	if labelChanged {
		utils.ChangeOrderLabel(order, plan.Label.ValueString(), resp)
	}

	if mountChanged {
		changeExtraMounts(order, &plan, &resp.Diagnostics)
	}

	if flavorChanged {
		changeFlavor(order, &plan, &resp.Diagnostics)
	}

	if !plan.FinancialProject.Equal(state.FinancialProject) {
		order.ChangeFinancialProject(finProj.ID)
	}

	if accessChanged {
		resp.Diagnostics.Append(r.changeAccess(order, &state, &plan)...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ComputeResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state ComputeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetComputeOrder(
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
			"Delete resource",
			fmt.Sprintf("Can't get state from portal.\nError:%s", err.Error()),
		)
		return
	}

	if orderState == "deleted" || orderState == "deprovisioned" {
		resp.Diagnostics.AddWarning(
			"Delete resource",
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
			"Deletable error",
			fmt.Sprintf(
				"Order with order_id '%s' isn't deletable. "+
					"Please, make your order is deletable by the portal VTB.Cloud, and repeat again."+
					"\nURL reference: %s",
				order.ID, orderURL,
			),
		)
		return
	}

	err = order.Delete(false)
	if err != nil {
		resp.Diagnostics.AddError(consts.DELETE_RES_FAIL, err.Error())
		return
	}
}

// * Custom logic *

func (r ComputeResource) changeAccess(
	order *orders.Compute,
	state,
	plan *ComputeResourceModel,
) (diags diag.Diagnostics) {

	var deletedRoles []entities.ADLogonGrants
	var changedRoles []entities.ADLogonGrants
	var addedRoles []entities.ADLogonGrants

	for role, groups := range state.Access {
		if _, ok := plan.Access[role]; !ok {
			deletedRoles = append(
				deletedRoles,
				entities.ADLogonGrants{Role: role, Groups: groups},
			)
		} else {
			if len(plan.Access[role]) == len(state.Access[role]) {
				if utils.DifferenceLen(plan.Access[role], state.Access[role]) {
					changedRoles = append(
						changedRoles,
						entities.ADLogonGrants{Role: role, Groups: plan.Access[role]},
					)
				}
			} else {
				changedRoles = append(
					changedRoles,
					entities.ADLogonGrants{Role: role, Groups: plan.Access[role]},
				)
			}
		}
	}

	for role, groups := range plan.Access {
		if _, ok := state.Access[role]; !ok {
			addedRoles = append(
				addedRoles,
				entities.ADLogonGrants{Role: role, Groups: groups},
			)
		}
	}

	if len(addedRoles) > 0 {
		rolesFromCloud, err := references.GetRoles(
			r.client.Creds,
			r.client.Environment,
			"vm:linux",
		)
		if err != nil {
			diags.AddError("Receiving roles from Cloud", err.Error())
			return
		}

		permissionsMap := make(map[string][]string)
		for _, value := range rolesFromCloud {
			permissionsMap[value.Data.GroupName] = value.Data.Permissions
		}

		for _, roles := range addedRoles {
			err := order.AddAccessGroup(roles, permissionsMap[roles.Role])
			if err != nil {
				diags.AddError("Adding new VM roles", err.Error())
			}
		}
	}

	if len(changedRoles) > 0 {
		for _, roles := range changedRoles {
			err := order.ChangeAccessGroup(roles)
			if err != nil {
				diags.AddError("Changing groups in VM roles", err.Error())
				return
			}
		}
	}

	if len(deletedRoles) > 0 {
		for _, roles := range deletedRoles {
			err := order.DeleteAccessGroup(roles)
			if err != nil {
				diags.AddError("Deleting VM roles", err.Error())
			}
		}
	}
	return
}

func changeFlavor(
	order *orders.Compute,
	plan *ComputeResourceModel,
	diags *diag.Diagnostics,
) {
	flavor := entities.Flavor{
		Cores:  plan.Flavor.Cores.ValueInt64(),
		Memory: plan.Flavor.Memory.ValueInt64(),
		UUID:   string(plan.Flavor.UUID.ValueString()),
		Name:   string(plan.Flavor.Name.ValueString()),
	}

	err := order.ChangeFlavor(flavor)
	if err != nil {
		diags.AddError("Change machine state (on changing machine state)", err.Error())
	}
}

func changeExtraMounts(
	order *orders.Compute,
	plan *ComputeResourceModel,
	diags *diag.Diagnostics,
) {
	for path, planExtraMount := range plan.ExtraMounts {
		err := order.ExpandMountPoint(entities.ExtraMount{
			Path:       path,
			Size:       planExtraMount.Size.ValueInt64(),
			FileSystem: planExtraMount.FileSystem.ValueString(),
		})
		if err != nil {
			diags.AddError("Incorrent changes in extra_mounts parameters", err.Error())
		}
	}
}
