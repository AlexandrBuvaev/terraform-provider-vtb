package grafana

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &GrafanaResource{}
	_ resource.ResourceWithImportState = &GrafanaResource{}
	_ resource.ResourceWithModifyPlan  = &GrafanaResource{}
)

type GrafanaResource struct {
	client *client.CloudClient
}

func NewGrafanaResource() resource.Resource {
	return &GrafanaResource{}
}

func (r GrafanaResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_grafana_instance"
}

func (r *GrafanaResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type GrafanaResourceModel struct {
	Core             core.CoreModel                    `tfsdk:"core"`
	Flavor           flavor.FlavorModel                `tfsdk:"flavor"`
	Lifetime         types.Int64                       `tfsdk:"lifetime"`
	Label            types.String                      `tfsdk:"label"`
	ItemID           types.String                      `tfsdk:"item_id"`
	OrderID          types.String                      `tfsdk:"order_id"`
	Access           map[string][]string               `tfsdk:"access"`
	ExtraMounts      map[string]common.ExtraMountModel `tfsdk:"extra_mounts"`
	FinancialProject types.String                      `tfsdk:"financial_project"`

	Image               common.ImageStandardModel `tfsdk:"image"`
	GrafanaUserName     types.String              `tfsdk:"grafana_user_name"`
	GrafanaUserPassword types.String              `tfsdk:"grafana_user_password"`
	GrafanaVersion      types.String              `tfsdk:"grafana_version"`
}

func (r GrafanaResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Grafana Resource",
		Attributes: map[string]schema.Attribute{
			// basic schema
			"core": schema.SingleNestedAttribute{
				Required:            true,
				Description:         "Основные параметры для ВМ в заказе.",
				MarkdownDescription: "Основные параметры для ВМ в заказе.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: common.CoreSchema,
			},
			"flavor": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.FlavorSchema,
				Description:         "Кол-во CPU/RAM для ВМ.",
				MarkdownDescription: "Кол-во CPU/RAM для ВМ.",
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
				Description:         "Метка заказа",
				MarkdownDescription: "Метка заказа",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"order_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Идентификатор заказа. Становиться известным после создания заказа.",
				MarkdownDescription: "Идентификатор заказа. Становиться известным после создания заказа.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"item_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Идентификатор родительского итема заказа",
				MarkdownDescription: "Идентификатор родительского итема заказа",
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
							"/app",
						),
					),
					mapvalidator.SizeBetween(1, 1),
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
							}, PlanModifiers: []planmodifier.Int64{
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
			"access": schema.MapAttribute{
				Optional: true,
				Computed: true,
				Description: "Словарь,где ключом является роль, а занчением список групп, " +
					"которые предоставляют доступ для входа в Active Directory",
				MarkdownDescription: "Словарь,где ключом является роль, а занчением список групп, " +
					"которые предоставляют доступ для входа в Active Directory",
				PlanModifiers: []planmodifier.Map{
					custommodifires.EmptyAccessGroup(),
				},
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
			},
			"financial_project": schema.StringAttribute{
				Required:            true,
				Description:         "Источник финансорования заказа",
				MarkdownDescription: "Источник финансорования заказа",
			},

			// image schema
			"image": schema.SingleNestedAttribute{
				Required:            true,
				Description:         "Тип вычислительного ресурса",
				MarkdownDescription: "Тип вычислительного ресурса",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"distribution": schema.StringAttribute{
						Required:            true,
						Description:         "Имя дистрибутива (RHEL или Astra)",
						MarkdownDescription: "Имя дистрибутива (RHEL или Astra)",
						Validators: []validator.String{
							stringvalidator.OneOf("rhel", "astra"),
						},
					},
					"os_version": schema.StringAttribute{
						Required:            true,
						Description:         "Версия образа",
						MarkdownDescription: "Версия образа",
					},
					"on_support": schema.BoolAttribute{
						Required:            true,
						Description:         "Флаг обозначающий, осуществляется ли поддержка группой сопровождения",
						MarkdownDescription: "Флаг обозначающий, осуществляется ли поддержка группой сопровождения",
					},
					"product_id": schema.StringAttribute{
						Required:            true,
						Description:         "Идентификатор продукта в продуктовом каталоге",
						MarkdownDescription: "Идентификатор продукта в продуктовом каталоге",
					},
					"ad_integration": schema.BoolAttribute{
						Required:            true,
						Description:         "Флаг обозначающий, включена ли интеграция Active Directory.",
						MarkdownDescription: "Флаг обозначающий, включена ли интеграция Active Directory.",
					},
				},
			},
			"grafana_version": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Версия Grafana",
				MarkdownDescription: "Версия Grafana",
				Default:             stringdefault.StaticString("9.1.6"),
				Validators: []validator.String{
					stringvalidator.OneOf("9.1.6", "8.1.2", "7.3.7"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"grafana_user_name": schema.StringAttribute{
				Required:            true,
				Description:         "Имя пользователя Grafana",
				MarkdownDescription: "Имя пользователя Grafana",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 23),
				},
			},
			"grafana_user_password": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				Description:         "Пароль пользователя Grafana",
				MarkdownDescription: "Пароль пользователя Grafana",
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile("[A-Z]"),
						"Password must have 1 uppercase letter"),
					stringvalidator.RegexMatches(regexp.MustCompile("[a-z]"),
						"Password must have 1 lowercase letter"),
					stringvalidator.RegexMatches(regexp.MustCompile("[0-9]"),
						"Password must have 1 numeric character"),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-zA-Z0-9\\._-]{20,64}$`),
						"Password must match pattern ^[a-zA-Z0-9\\._-]{20,64}$"),
				},
			},
		},
	}
}
func (r GrafanaResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r GrafanaResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan GrafanaResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	checkOrderIsDeleted := utils.CheckOrderIsDeleted(r.client, plan.OrderID, "app", "grafana")
	resp.Diagnostics.Append(checkOrderIsDeleted.Diagnostics...)

	utils.HandleExternalDeletionRecreate(ctx, resp, &plan, checkOrderIsDeleted.IsDeleted, checkOrderIsDeleted.Diagnostics)

	validateAccessForEnvDiags := utils.ValidateActiveDirectoryAccessForEnvironmentsV2(r.client, plan.Access, "DEV", "TEST")
	resp.Diagnostics.Append(validateAccessForEnvDiags...)

	validateAccessRolesDiags := utils.ValidateAccessRolesV1(r.client, "app:grafana", plan.Access)
	resp.Diagnostics.Append(validateAccessRolesDiags...)

	if resp.Diagnostics.HasError() {
		return
	}

}

func (r GrafanaResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan GrafanaResourceModel
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

	basicAttrs := orders.BasicAttrs{
		ADIntegration:    plan.Image.ADIntegration.ValueBool(),
		ADLogonGrants:    ADLogonGrants,
		ExtraMounts:      utils.PrepareExtraMountsAttrs(plan.ExtraMounts),
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
	attrs := orders.GrafanaAttrs{
		BasicAttrs:          basicAttrs,
		GrafanaUser:         plan.GrafanaUserName.ValueString(),
		GrafanaUserPassword: plan.GrafanaUserPassword.ValueString(),
		GrafanaVersion:      plan.GrafanaVersion.ValueString(),
	}

	order := orders.NewGrafanaOrder(
		r.client.Creds,
		r.client.ProjectName,
		plan.Image.ProductID.ValueString(),
		attrs,
	)

	finProj, err := sources.GetFinancialProjectByName(
		r.client.Creds,
		r.client.ProjectName,
		plan.FinancialProject.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("financial_project"), consts.CREATE_RES_FAIL, err.Error())
		return
	}

	err = order.Create(orders.CreateOrderPayload{
		Label:        plan.Label.ValueString(),
		Lifetime:     int(plan.Lifetime.ValueInt64()),
		FinProjectID: finProj.ID,
	})
	if err != nil {
		resp.Diagnostics.AddError(consts.CREATE_RES_FAIL, err.Error())
		return
	}
	err = order.Sync()
	if err != nil {
		resp.Diagnostics.AddError(consts.CREATE_RES_FAIL, err.Error())
		return
	}

	orderItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(consts.CREATE_RES_FAIL, err.Error())
		return
	}
	err = order.Sync()
	if err != nil {
		resp.Diagnostics.AddError(consts.CREATE_RES_FAIL, err.Error())
		return
	}

	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(orderItem.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r GrafanaResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data GrafanaResourceModel

	var vmItem entities.Item
	var orderID types.String
	var grafanaUserPassword types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("order_id"), &orderID)...)
	req.State.GetAttribute(ctx, path.Root("grafana_user_password"), &grafanaUserPassword)

	if orderID.IsNull() {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Resource hasn't order_id. Without order_id can't get order from portal",
		)
	}

	data.OrderID = orderID
	order, err := orders.GetGrafanaOrder(
		r.client.Creds,
		r.client.ProjectName,
		data.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't read order with order_id equal '%s'.\nError:%s",
				data.OrderID.ValueString(), err.Error(),
			),
		)
		return
	}

	item, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get parent item from portal with order_id '%s'. \nError:%s",
				data.OrderID.ValueString(), err.Error(),
			),
		)
		return
	}

	config := item.Data.Config.(entities.GrafanaItemConfig)
	grafanaUser, err := order.GetDefaultUser()

	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Can't get grafana default user.\nError:%s", err.Error()),
		)
		return
	}

	vmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Error:%s", err.Error()),
		)
		return
	}

	vmItem = vmItems[0]
	vmConfig := vmItem.Data.Config.(entities.VMItemConfig)

	data.Label = types.StringValue(order.Label)
	data.ItemID = types.StringValue(item.ID)
	data.GrafanaUserName = types.StringValue(grafanaUser.Username)
	data.GrafanaUserPassword = grafanaUserPassword
	data.Access = utils.ReadAccessMapVV1(vmItem.Data.ACLs)
	data.FinancialProject = types.StringValue(order.FinancialSource.Name)
	data.GrafanaVersion = types.StringValue(config.GrafanaVersion)
	data.Flavor = flavor.FlavorModel{
		Cores:  types.Int64Value(vmConfig.Flavor.Cores),
		Memory: types.Int64Value(vmConfig.Flavor.Memory),
		Name:   types.StringValue(vmConfig.Flavor.Name),
		UUID:   types.StringValue(vmConfig.Flavor.UUID),
	}

	data.ExtraMounts = make(map[string]common.ExtraMountModel)
	orderExtraMounts, err := order.GetExtraMounts()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Can't get grafana extra mounts.\nError:%s", err.Error()),
		)
		return
	}
	for _, extraMount := range orderExtraMounts {
		data.ExtraMounts[extraMount.Path] = common.ExtraMountModel{
			Size:       types.Int64Value(extraMount.Size),
			FileSystem: types.StringValue(extraMount.FileSystem),
		}
	}

	data.Image = common.ImageStandardModel{
		Distribution:  types.StringValue(vmConfig.Image.Os.Distribution),
		OsVersion:     types.StringValue(vmConfig.Image.Os.Version),
		OnSupport:     types.BoolValue(order.Attrs.OnSupport),
		ProductID:     types.StringValue(order.ProductID),
		ADIntegration: types.BoolValue(order.Attrs.ADIntegration),
	}

	data.Core = core.CoreModel{
		Platform:       types.StringValue(utils.SelectPlatform(order.Attrs.Platform)),
		Domain:         types.StringValue(order.Attrs.Domain),
		NetSegmentCode: types.StringValue(vmConfig.DefaultNic.NetSegment),
		Zone:           types.StringValue(order.Attrs.AvailabilityZone),
	}

	var lifetime types.Int64
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("lifetime"), &lifetime)...)
	if !lifetime.IsNull() {
		data.Lifetime = lifetime
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
func (r GrafanaResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan GrafanaResourceModel
	var state GrafanaResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	order, err := orders.GetGrafanaOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)

	if err != nil {
		diags.AddError(consts.UPDATE_RES_FAIL, err.Error())
		resp.Diagnostics.Append(diags...)
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

	if !plan.FinancialProject.Equal(state.FinancialProject) {
		order.ChangeFinancialProject(finProj.ID)
	}

	labelChanged := plan.Label != state.Label
	mountChanged := utils.IsExtraMountChanged(state.ExtraMounts, plan.ExtraMounts)
	passwordChanged := state.GrafanaUserPassword != plan.GrafanaUserPassword
	flavorChanged := state.Flavor != plan.Flavor
	accessChanged := !reflect.DeepEqual(plan.Access, state.Access)

	if accessChanged {
		resp.Diagnostics.Append(r.changeAccess(order, &state, &plan)...)
	}

	if flavorChanged {
		changeGrafanaFlavor(order, &plan, &resp.Diagnostics)
	}

	if labelChanged {
		utils.ChangeOrderLabel(order, plan.Label.ValueString(), resp)
	}
	if passwordChanged {
		err = order.ChangeUserPassword(state.GrafanaUserName.ValueString(), plan.GrafanaUserPassword.ValueString())
		if err != nil {
			diags.AddError(consts.UPDATE_RES_FAIL, fmt.Sprintf("Change password failed: %v", err.Error()))
			resp.Diagnostics.Append(diags...)
		}
	}
	if mountChanged {
		diags = changeGrafanaExtraMounts(order, &plan)
		resp.Diagnostics.Append(diags...)
	}
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)

}
func (r GrafanaResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state GrafanaResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetGrafanaOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(consts.DELETE_RES_FAIL, err.Error())
		return
	}

	orderState, err := order.GetState()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf("Can't get state from portal.\nError: %s", err.Error()),
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
					"Please, make your order is deletable by the portal VTB.Cloud, "+
					"and repeat again.\nURL reference: %s",
				order.ID, orderURL,
			),
		)
		return
	}

	err = order.Delete()
	if err != nil {
		resp.Diagnostics.AddError(consts.DELETE_RES_FAIL, err.Error())
		return
	}

}

func changeGrafanaExtraMounts(
	order *orders.GrafanaOrder,
	planResource *GrafanaResourceModel,
) (diags diag.Diagnostics) {
	vmItems, _ := order.GetVMItems()
	for _, vmItem := range vmItems {
		config := vmItem.Data.Config.(entities.VMItemConfig)
		for _, vmExtraMount := range config.ExtraMounts {
			planExtraMountSize := planResource.ExtraMounts[vmExtraMount.Mount].Size.ValueInt64()
			if planExtraMountSize != int64(vmExtraMount.Size) && planExtraMountSize > int64(vmExtraMount.Size) {
				err := order.ExpandMountPointForVm(vmItem.ID, vmExtraMount.Mount, planExtraMountSize-int64(vmExtraMount.Size))

				if err != nil {
					diags.AddError("Expand mount point error", err.Error())
					return
				}
			}
		}
	}
	return
}

func changeGrafanaFlavor(
	order *orders.GrafanaOrder,
	plan *GrafanaResourceModel,
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

func (r GrafanaResource) changeAccess(
	order *orders.GrafanaOrder,
	state,
	plan *GrafanaResourceModel,
) (diags diag.Diagnostics) {

	deletedRoles, changedRoles, addedRoles := r.compareADRoles(*plan, *state)

	if len(addedRoles) > 0 {
		rolesFromCloud, err := references.GetRoles(
			r.client.Creds,
			r.client.Environment,
			"app:grafana",
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

func (r *GrafanaResource) compareADRoles(
	plan, state GrafanaResourceModel) (deletedRoles, changedRoles, addedRoles []entities.ADLogonGrants) {

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
