package nginx

import (
	"context"
	"fmt"
	"reflect"

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &NginxResource{}
	_ resource.ResourceWithImportState = &NginxResource{}
)

type NginxResource struct {
	client *client.CloudClient
}

func NewNginxResource() resource.Resource {
	return &NginxResource{}
}

func (r NginxResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_nginx_instance"
}

func (r *NginxResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type NginxResourceModel struct {
	Lifetime         types.Int64                       `tfsdk:"lifetime"`
	Label            types.String                      `tfsdk:"label"`
	OrderID          types.String                      `tfsdk:"order_id"`
	ItemID           types.String                      `tfsdk:"item_id"`
	Core             core.CoreModel                    `tfsdk:"core"`
	Flavor           flavor.FlavorModel                `tfsdk:"flavor"`
	Image            common.ImageStandardModel         `tfsdk:"image"`
	Access           map[string][]string               `tfsdk:"access"`
	ExtraMounts      map[string]common.ExtraMountModel `tfsdk:"extra_mounts"`
	Hostname         types.String                      `tfsdk:"hostname"`
	NginxVersion     types.String                      `tfsdk:"nginx_version"`
	FinancialProject types.String                      `tfsdk:"financial_project"`
	BuildVersion     types.String                      `tfsdk:"build_version"`
}

func (r NginxResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
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
				MarkdownDescription: "Идентификатор заказа. Становиться известен после создания заказа.",
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

			"core": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.CoreSchema,
				MarkdownDescription: "Основные параметры для виртуальных машин в заказе.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},

			"flavor": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.FlavorSchema,
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
			"access": schema.MapAttribute{
				Required: true,
				Description: "Карта, где ключом является роль, а значением - список групп, " +
					"которой предоставлен доступ для входа в Active Directory.",
				MarkdownDescription: "Карта, где ключом является роль, а значением - список групп, " +
					"который предоставит доступ для входа в Active Directory",
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
				Validators: []validator.Map{
					customvalidators.EmptyAccessGroupListValidator{},
				},
			},
			"extra_mounts": schema.MapNestedAttribute{
				Required:            true,
				MarkdownDescription: "Дополнительные точки монтирования для ВМ.",
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.OneOf("/app")),
					mapvalidator.SizeBetween(1, 1),
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

			"hostname": schema.StringAttribute{
				Computed:            true,
				Description:         "Имя хоста виртуальной машины.",
				MarkdownDescription: "Имя хоста виртуальной машины.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"nginx_version": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf("1.20.2", "1.22.0", "1.26.1"),
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
			"build_version": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Версия продукта",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r NginxResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r NginxResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	var plan NginxResourceModel

	if req.Plan.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	validateRolesDiags := utils.ValidateAccessRolesV1(
		r.client,
		"app:nginx",
		plan.Access,
	)
	resp.Diagnostics.Append(validateRolesDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	checkOrderIsDeleted := utils.CheckOrderIsDeleted(r.client, plan.OrderID, "app", "nginx")
	resp.Diagnostics.Append(checkOrderIsDeleted.Diagnostics...)
	if resp.Diagnostics.HasError() {
		return
	}

	utils.HandleExternalDeletionRecreate(
		ctx,
		resp,
		&plan,
		checkOrderIsDeleted.IsDeleted,
		checkOrderIsDeleted.Diagnostics,
	)
}

func (r NginxResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan NginxResourceModel
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

	attrs := r.prepareOrderAttrs(plan)

	order := orders.NewNginx(
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
		resp.Diagnostics.AddError("Can't create virtual machine: ", err.Error())
		return
	}

	err = order.Sync()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprint(
				"Can't get order's information from order service: ", err.Error(),
			),
		)
		return
	}

	nginxItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Get nginx item ended with error.\nError: %v", err.Error()),
		)
		return
	}

	vmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Get vm items ended with error.\nError: %v", err.Error()),
		)
		return
	}
	itemData := vmItems[0].Data

	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(nginxItem.ID)
	plan.Hostname = types.StringValue(itemData.Config.(entities.VMItemConfig).Hostname)
	plan.BuildVersion = types.StringValue(nginxItem.Data.Build.SetupVersion)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r NginxResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var orderID types.String
	req.State.GetAttribute(ctx, path.Root("order_id"), &orderID)

	order, err := orders.GetNginxOrder(
		r.client.Creds,
		r.client.ProjectName,
		orderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("order_id"), consts.READ_RES_FAIL, err.Error())
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
			"Invalid type for VM Config",
		)
		return
	}

	appExtraMount, err := order.GetExtraMount("/app")
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get vm extra mount '/app' from portal with order_id '%s'.\nError: %s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}

	nginxItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Get nginx item ended with error.\nError: %v", err.Error()),
		)
		return
	}

	state := NginxResourceModel{
		OrderID:  orderID,
		Access:   utils.ReadAccessMapVV1(vmItem.Data.ACLs),
		Label:    types.StringValue(order.Label),
		ItemID:   types.StringValue(vmItem.ID),
		Hostname: types.StringValue(vmConfig.Hostname),
		Flavor: flavor.FlavorModel{
			Cores:  types.Int64Value(vmConfig.Flavor.Cores),
			Memory: types.Int64Value(vmConfig.Flavor.Memory),
			UUID:   types.StringValue(vmConfig.Flavor.UUID),
			Name:   types.StringValue(vmConfig.Flavor.Name),
		},
		ExtraMounts: map[string]common.ExtraMountModel{
			"/app": {
				Size:       types.Int64Value(appExtraMount.Size),
				FileSystem: types.StringValue(appExtraMount.FileSystem),
			},
		},
		Image: common.ImageStandardModel{
			Distribution:  types.StringValue(vmConfig.Image.Os.Distribution),
			OsVersion:     types.StringValue(vmConfig.Image.Os.Version),
			OnSupport:     types.BoolValue(order.Attrs.OnSupport),
			ProductID:     types.StringValue(order.ProductID),
			ADIntegration: types.BoolValue(order.Attrs.ADIntegration),
		},
		Core: core.CoreModel{
			Platform:       types.StringValue(utils.SelectPlatform(order.Attrs.Platform)),
			Domain:         types.StringValue(order.Attrs.Domain),
			NetSegmentCode: types.StringValue(vmConfig.DefaultNic.NetSegment),
			Zone:           types.StringValue(order.Attrs.AvailabilityZone),
		},
		FinancialProject: types.StringValue(order.FinancialSource.Name),
		NginxVersion:     types.StringValue(order.Attrs.NginxVersion),
		BuildVersion:     types.StringValue(nginxItem.Data.Build.SetupVersion),
	}

	var lifetime types.Int64
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("lifetime"), &lifetime)...)
	if !lifetime.IsNull() {
		state.Lifetime = lifetime
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r NginxResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state NginxResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nginx, err := orders.GetNginxOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Update product",
			fmt.Sprint("Can't get order from portal: ", err.Error()),
		)
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
	financialProjectChanged := !plan.FinancialProject.Equal(state.FinancialProject)

	// change label
	if labelChanged {
		utils.ChangeOrderLabel(nginx, plan.Label.ValueString(), resp)
	}

	if financialProjectChanged {
		r.changeFinancialSource(nginx, finProj.ID, resp)
	}

	if mountChanged {
		r.changeExtraMountsNginx(nginx, &plan, resp)
	}

	if flavorChanged {
		r.changeFlavorNginx(nginx, &plan, resp)
	}

	if accessChanged {
		r.changeAccess(nginx, &state, &plan, resp)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r NginxResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state NginxResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetNginxOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)

	if err != nil {
		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf("Get nginx order from portal ended with error.\nError: %v", err.Error()),
		)
		return
	}

	orderState, err := order.GetState()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf("Get order state from portal ended with error.\nError: %v", err.Error()),
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

	err = order.DeleteTwoLayer(false)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf("Delete order from portal ended with error.\nError: %v", err.Error()),
		)
		return
	}
}

// custom logic
func (r NginxResource) changeFlavorNginx(
	order *orders.Nginx,
	plan *NginxResourceModel,
	resp *resource.UpdateResponse,
) {

	flavor := entities.Flavor{
		Cores:  plan.Flavor.Cores.ValueInt64(),
		Memory: plan.Flavor.Memory.ValueInt64(),
		UUID:   string(plan.Flavor.UUID.ValueString()),
		Name:   string(plan.Flavor.Name.ValueString()),
	}

	err := order.ChangeFlavor(flavor)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Change machine state (on changing machine state) ended with error.\nError: %v", err.Error()),
		)
		return
	}
}

func (r NginxResource) changeExtraMountsNginx(
	order *orders.Nginx,
	plan *NginxResourceModel,
	resp *resource.UpdateResponse,
) {

	for path, planExtraMount := range plan.ExtraMounts {
		err := order.ExpandMountPoint(entities.ExtraMount{
			Path:       path,
			Size:       planExtraMount.Size.ValueInt64(),
			FileSystem: planExtraMount.FileSystem.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError(
				consts.UPDATE_RES_FAIL,
				fmt.Sprintf("Change extra_mounts ended with error.\nError: %v", err.Error()),
			)
		}
		return
	}
}

func (r NginxResource) changeAccess(
	order *orders.Nginx,
	state,
	plan *NginxResourceModel,
	resp *resource.UpdateResponse,
) {

	deletedRoles, changedRoles, addedRoles := r.compareADRoles(*plan, *state)

	if len(addedRoles) > 0 {
		rolesFromCloud, err := references.GetRoles(
			r.client.Creds,
			r.client.Environment,
			"app:nginx",
		)
		if err != nil {
			resp.Diagnostics.AddError(
				consts.UPDATE_RES_FAIL,
				fmt.Sprintf("Get availiable group roles ended with error.\nError: %v", err.Error()))
			return
		}

		permissionsMap := make(map[string][]string)
		for _, value := range rolesFromCloud {
			permissionsMap[value.Data.GroupName] = value.Data.Permissions
		}
		for _, role := range addedRoles {
			err := order.AddAccessGroup(role, permissionsMap[role.Role])
			if err != nil {
				resp.Diagnostics.AddError(
					consts.UPDATE_RES_FAIL,
					fmt.Sprintf("Add access group ended with error.\nError: %v", err.Error()),
				)
				return
			}
		}
	}

	if len(changedRoles) > 0 {
		for _, role := range changedRoles {
			err := order.ChangeAccessGroup(role)
			if err != nil {
				resp.Diagnostics.AddError(
					consts.UPDATE_RES_FAIL,
					fmt.Sprintf("Change access group ended with error.\nError: %v", err.Error()),
				)
				return
			}
		}
	}

	if len(deletedRoles) > 0 {
		for _, role := range deletedRoles {
			err := order.DeleteAccessGroup(role)
			if err != nil {
				resp.Diagnostics.AddError(
					consts.UPDATE_RES_FAIL,
					fmt.Sprintf("Delete access group ended with error.\nError: %v", err.Error()),
				)
				return
			}
		}
	}
}

func (r *NginxResource) compareADRoles(
	plan, state NginxResourceModel) (deletedRoles, changedRoles, addedRoles []entities.ADLogonGrants) {

	for role, groups := range state.Access {
		if _, ok := plan.Access[role]; !ok {
			deletedRoles = append(
				deletedRoles,
				entities.ADLogonGrants{Role: role, Groups: groups})
		} else {
			if len(plan.Access[role]) == len(state.Access[role]) {
				if utils.IsDifferent(plan.Access[role], state.Access[role]) {
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
		if state.Access[role] == nil {
			addedRoles = append(
				addedRoles,
				entities.ADLogonGrants{Role: role, Groups: groups},
			)
		}
	}
	return deletedRoles, changedRoles, addedRoles
}

func (r NginxResource) prepareOrderAttrs(
	plan NginxResourceModel,
) (attrs orders.NginxAttrs) {
	var ADLogonGrants []entities.ADLogonGrants
	for role, groups := range plan.Access {
		var groupsNames []string
		groupsNames = append(groupsNames, groups...)
		ADLogonGrants = append(ADLogonGrants, entities.ADLogonGrants{
			Role:   role,
			Groups: groupsNames,
		})
	}

	basicAttrs := orders.BasicAttrs{
		ADIntegration:    plan.Image.ADIntegration.ValueBool(),
		ADLogonGrants:    ADLogonGrants,
		Platform:         plan.Core.Platform.ValueString(),
		AvailabilityZone: plan.Core.Zone.ValueString(),
		Domain:           plan.Core.Domain.ValueString(),
		OnSupport:        plan.Image.OnSupport.ValueBool(),
		OsVersion:        plan.Image.OsVersion.ValueString(),
		ExtraMounts:      utils.PrepareExtraMountsAttrs(plan.ExtraMounts),
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
	}

	devOpsGroupName := fmt.Sprintf("cloud-%s-ngc-devops-%s", r.client.RisShortName, r.client.EnvPrefix)

	attrs = orders.NginxAttrs{
		NginxVersion: plan.NginxVersion.ValueString(),
		ExtraNics:    []interface{}{},
		BasicAttrs:   basicAttrs,
		DevOpsGroup:  devOpsGroupName,
		InfoDevOps:   map[string]interface{}{},
	}

	return attrs
}

func (r NginxResource) changeFinancialSource(
	order *orders.Nginx,
	financialSourceId string,
	resp *resource.UpdateResponse,
) {
	err := order.ChangeFinancialProject(financialSourceId)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Change financial source ended with error.\nError: %v", err.Error()),
		)
		return
	}
}
