package tarantool

import (
	"context"
	"fmt"
	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/common"
	"terraform-provider-vtb/internal/consts"
	"terraform-provider-vtb/internal/customvalidators"
	"terraform-provider-vtb/internal/services/core"
	"terraform-provider-vtb/internal/utils"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"
	"terraform-provider-vtb/pkg/client/sources"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

var (
	_ resource.Resource                = TarantoolClusterResource{}
	_ resource.ResourceWithImportState = TarantoolClusterResource{}
	_ resource.ResourceWithModifyPlan  = TarantoolClusterResource{}
)

type TarantoolClusterResource struct {
	client *client.CloudClient
}

func NewTarantoolClusterResource() resource.Resource {
	return &TarantoolClusterResource{}
}

func (r TarantoolClusterResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_tarantool_cluster"
}

func (r *TarantoolClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type TarantoolClusterResourceModel struct {
	Lifetime              types.Int64                `tfsdk:"lifetime"`
	Label                 types.String               `tfsdk:"label"`
	OrderID               types.String               `tfsdk:"order_id"`
	ItemID                types.String               `tfsdk:"item_id"`
	Core                  core.CoreModel             `tfsdk:"core"`
	Image                 TarantoolClusterImageModel `tfsdk:"image"`
	Layout                types.String               `tfsdk:"layout"`
	TarantoolAccessGroup  types.Set                  `tfsdk:"tarantool_access"`
	ActiveDirectoryAccess map[string][]string        `tfsdk:"access"`
	TarantoolVersion      types.String               `tfsdk:"tarantool_version"`
	FinancialProject      types.String               `tfsdk:"financial_project"`
	ClusterName           types.String               `tfsdk:"cluster_name"`
	TarantoolType         types.String               `tfsdk:"tarantool_type"`
	Zones                 types.Map                  `tfsdk:"zones"`
}

// Модель для отображения данных о зонах и их инстансах
type ZoneConfigModel struct {
	RunningInstances types.Set `tfsdk:"running_instances"`
	StoppedInstances types.Set `tfsdk:"stopped_instances"`
}

func (m ZoneConfigModel) AttributeTypes(ctx context.Context) map[string]attr.Type {
	return map[string]attr.Type{
		"running_instances": types.SetType{
			ElemType: types.StringType,
		},
		"stopped_instances": types.SetType{
			ElemType: types.StringType,
		},
	}
}

var tarantoolClusterImageSchema = map[string]schema.Attribute{
	"distribution": schema.StringAttribute{
		Required:            true,
		Description:         "Название дистрибутива (RHEL или Astra)",
		MarkdownDescription: "Название дистрибутива (RHEL или Astra)",
		Validators: []validator.String{
			stringvalidator.OneOf("rhel", "astra"),
		},
	},
	"os_version": schema.StringAttribute{
		Required:            true,
		Description:         "Версия дистибутива образа",
		MarkdownDescription: "Версия дистрибутива образа",
	},
	"product_id": schema.StringAttribute{
		Required:            true,
		Description:         "Product id для продукта Tarantool Data Grid/Tarantool Enterprise",
		MarkdownDescription: "Product id для продукта Tarantool Data Grid/Tarantool Enterprise",
	},
	"on_support": schema.BoolAttribute{
		Required:            true,
		MarkdownDescription: "Флаг означает, что продукт будет поддерживаться группой поддержки",
	},
	"ad_integration": schema.BoolAttribute{
		Required:            true,
		Description:         "Флаг интеграции ВМ с Active Directory",
		MarkdownDescription: "Флаг интеграции ВМ с Active Directory",
	},
	"default_tarantool_version": schema.StringAttribute{
		Required:            true,
		Description:         "Версия Tarantool по умолчанию",
		MarkdownDescription: "Версия Tarantool по умолчанию",
	},
	"geo_distribution": schema.BoolAttribute{
		Required:            true,
		Description:         "Флаг использования георезервирования",
		MarkdownDescription: "Флаг использования георезервирования",
	},
}

func (r TarantoolClusterResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Управление облачным продуктом Tarantool Data Grid/Tarantool Enterprise",
		Attributes: map[string]schema.Attribute{
			// basic schema
			"lifetime": schema.Int64Attribute{
				Optional:            true,
				Description:         "Время жизни заказа в днях (2, 7, 30)",
				MarkdownDescription: "Время жизни заказа в днях (2, 7, 30)",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
			},
			"label": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Метка заказа",
				Description:         "Метка заказа",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"order_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Идентификатор заказа. Становится известен после создания заказа.",
				Description:         "Идентификатор заказа. Становится известен после создания заказа.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"item_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Идентификатор сущностей ВМ, известен после создания ВМ.",
				Description:         "Идентификатор сущностей ВМ, известен после создания ВМ.",
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
			"image": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          tarantoolClusterImageSchema,
				MarkdownDescription: "Образ вычислительно экземпляра",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"layout": schema.StringAttribute{
				Required:            true,
				Description:         "Layout ID из справочника geo_distribution.",
				MarkdownDescription: "Layout ID из справочника geo_distribution.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"access": schema.MapAttribute{
				Required: true,
				Description: "Карта, где ключом является роль, а значением - список групп, " +
					"которые предоставляют доступ для входа в Active Directory",
				MarkdownDescription: "Карта, где ключом является роль, а значением - список групп, " +
					"которые предоставляют доступ для входа в Active Directory",
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
				Validators: []validator.Map{
					customvalidators.EmptyAccessGroupListValidator{},
				},
			},
			"tarantool_access": schema.SetAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "Группы управления Tarantool",
				Description:         "Группы управления Tarantool",
			},
			"tarantool_version": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Версия Tarantool Data Grid/Tarantool Enterprise",
				Description:         "Версия Tarantool Data Grid/Tarantool Enterprise",
				Validators: []validator.String{
					stringvalidator.OneOf("2.12.1", "0.1.0.0", "2.8.0.0"),
				},
			},
			"tarantool_type": schema.StringAttribute{
				Computed:            true,
				Description:         "Тип приложения Tarantool",
				MarkdownDescription: "Тип приложения Tarantool",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Полное имя кластера Tarantool Data Grid/Tarantool Enterprise",
				Description:         "Полное имя кластера Tarantool Data Grid/Tarantool Enterprise",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"financial_project": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Источник финансирования",
				Description:         "Источник финансирования",
			},
			"zones": schema.MapNestedAttribute{
				Computed:            true,
				Optional:            true,
				Description:         "Зоны кластера",
				MarkdownDescription: "Зоны кластера",
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"running_instances": schema.SetAttribute{
							ElementType:         types.StringType,
							Required:            true,
							Description:         "Включенные инстансы внутри зоны кластера",
							MarkdownDescription: "Включенные инстансы внутри зоны кластера",
						},
						"stopped_instances": schema.SetAttribute{
							ElementType:         types.StringType,
							Required:            true,
							Description:         "Выключенные инстансы внутри зоны кластера",
							MarkdownDescription: "Выключенные инстансы внутри зоны кластера",
						},
					},
				},
			},
		},
	}
}

func (r TarantoolClusterResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r TarantoolClusterResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan TarantoolClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	checkOrderIsDeleted := utils.CheckOrderIsDeleted(
		r.client,
		plan.OrderID,
		"cluster",
		"tarantool_v2",
	)

	utils.HandleExternalDeletionRecreate(
		ctx,
		resp,
		&plan,
		checkOrderIsDeleted.IsDeleted,
		checkOrderIsDeleted.Diagnostics,
	)

	validateAccessRolesDiags := utils.ValidateAccessRolesV1(
		r.client,
		"cluster:tarantool",
		plan.ActiveDirectoryAccess,
	)
	resp.Diagnostics.Append(validateAccessRolesDiags...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r TarantoolClusterResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan TarantoolClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	attrs := r.prepareAtts(&plan)
	resp.Diagnostics.Append(plan.TarantoolAccessGroup.ElementsAs(ctx, &attrs.AccessGroup, false)...)

	finProj, err := sources.GetFinancialProjectByName(
		r.client.Creds,
		r.client.ProjectName,
		plan.FinancialProject.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("financial_project"), consts.CREATE_RES_FAIL, err.Error())
		return
	}

	order := orders.NewTarantoolCluster(
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
		resp.Diagnostics.AddError("Error while getting item: ", err.Error())
		return
	}
	clusterConfig, ok := orderItem.Data.Config.(entities.TarantoolClusterItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			"Invalid type for clusterConfig",
		)
		return
	}

	zones, diags := r.convertZonesToTF(ctx, order, clusterConfig)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	plan.Zones, diags = types.MapValueFrom(
		ctx,
		types.ObjectType{AttrTypes: ZoneConfigModel{}.AttributeTypes(ctx)},
		zones,
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(orderItem.ID)
	plan.ClusterName = types.StringValue(clusterConfig.ClusterName)
	plan.TarantoolType = types.StringValue(clusterConfig.TarantoolType)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r TarantoolClusterResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var orderID types.String
	req.State.GetAttribute(ctx, path.Root("order_id"), &orderID)
	if orderID.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("order_id"),
			consts.READ_RES_FAIL,
			"Resource hasn't order_id. Without order_id can't get order from portal",
		)
		return
	}
	order, err := orders.GetTarantoolClusterOrder(
		r.client.Creds,
		r.client.ProjectName,
		orderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("order_id"),
			fmt.Sprintln(consts.READ_RES_FAIL, "fetch order from portal API ended with error"),
			err.Error(),
		)
		return
	}

	orderItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("order_id"),
			fmt.Sprintln(consts.READ_RES_FAIL, "can't get cluster item"),
			err.Error(),
		)
		return
	}
	clusterConfig, ok := orderItem.Data.Config.(entities.TarantoolClusterItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Invalid type for clusterConfig",
		)
		return
	}

	vmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Can't get tarantool cluster items.\nError: %s", err.Error()),
		)
		return
	}

	vmConfig, ok := vmItems[0].Data.Config.(entities.VMItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Invalid type for vmConfig",
		)
		return
	}

	tarantoolAccess, diags := types.SetValueFrom(ctx, types.StringType, clusterConfig.DomainRoles[0].Members)
	if diags.HasError() {
		return
	}

	zones, diags := r.convertZonesToTF(ctx, order, clusterConfig)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := TarantoolClusterResourceModel{
		Core: core.CoreModel{
			Platform:       types.StringValue(utils.SelectPlatform(order.Attrs.Platform)),
			Domain:         types.StringValue(order.Attrs.Domain),
			NetSegmentCode: types.StringValue(vmConfig.DefaultNic.NetSegment),
			Zone:           types.StringValue(order.Attrs.AvailabilityZone),
		},
		ActiveDirectoryAccess: utils.ReadAccessMapVV1(vmItems[0].Data.ACLs),
		ItemID:                types.StringValue(orderItem.ID),
		Label:                 types.StringValue(order.Label),
		FinancialProject:      types.StringValue(order.FinancialSource.Name),
		Layout:                types.StringValue(order.Attrs.Layout),
		TarantoolAccessGroup:  tarantoolAccess,
		TarantoolVersion:      types.StringValue(clusterConfig.TarantoolVersion),
		OrderID:               orderID,
		Image: TarantoolClusterImageModel{
			ProductID:               types.StringValue(order.ProductID),
			OnSupport:               types.BoolValue(order.Attrs.OnSupport),
			ADIntegration:           types.BoolValue(order.Attrs.ADIntegration),
			Distribution:            types.StringValue(vmConfig.Image.Os.Distribution),
			OsVersion:               types.StringValue(vmConfig.Image.Os.Version),
			GeoDistribution:         types.BoolValue(order.Attrs.GeoDistribution),
			DefaultTarantoolVersion: types.StringValue(clusterConfig.TarantoolVersion),
		},
		ClusterName:   types.StringValue(clusterConfig.ClusterName),
		TarantoolType: types.StringValue(clusterConfig.TarantoolType),
	}
	state.Zones, diags = types.MapValueFrom(
		ctx,
		types.ObjectType{AttrTypes: ZoneConfigModel{}.AttributeTypes(ctx)},
		zones,
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	var lifetime types.Int64
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("lifetime"), &lifetime)...)
	if !lifetime.IsNull() {
		state.Lifetime = lifetime
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}

func (r TarantoolClusterResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state TarantoolClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetTarantoolClusterOrder(
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

	if !plan.Label.Equal(state.Label) {
		utils.ChangeOrderLabel(order, plan.Label.ValueString(), resp)
		resp.State.SetAttribute(ctx, path.Root("label"), plan.Label.ValueString())
	}

	if !plan.FinancialProject.Equal(state.FinancialProject) {
		err = order.ChangeFinancialProject(finProj.ID)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("financial_project"), consts.UPDATE_RES_FAIL, err.Error())
			return
		}
	}
	if !plan.Zones.IsNull() && !plan.Zones.Equal(state.Zones) {
		r.applyZoneConfig(ctx, plan, state, order, resp)
	}

	if !plan.TarantoolVersion.Equal(state.TarantoolVersion) {
		r.updateTarantoolVersion(&plan, order, resp)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

}

func (r TarantoolClusterResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state TarantoolClusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetTarantoolClusterOrder(
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
			fmt.Sprintf("Can't get state from portal.\nError: %s", err.Error()),
		)
		return
	}

	if orderState == "deleted" || orderState == "deprovisioned" {
		resp.Diagnostics.AddWarning(
			"Delete resource",
			fmt.Sprintf(
				"Order with id '%s' was deleted by another way. Not from terraform. "+
					"Resource was recreated.",
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
	if err := order.DeleteTwoLayer(false); err != nil {
		resp.Diagnostics.AddError("While process delete order: ", err.Error())
		return
	}
}

// custom logic
func (r TarantoolClusterResource) prepareAtts(plan *TarantoolClusterResourceModel) (attrs orders.TarantoolClusterAttrs) {

	var ADLogonGrants []entities.ADLogonGrants
	for role, groups := range plan.ActiveDirectoryAccess {
		var groupsNames []string
		groupsNames = append(groupsNames, groups...)
		ADLogonGrants = append(ADLogonGrants, entities.ADLogonGrants{
			Role:   role,
			Groups: groupsNames,
		})
	}

	attrs = orders.TarantoolClusterAttrs{
		ADLogonGrants:    ADLogonGrants,
		ADIntegration:    plan.Image.ADIntegration.ValueBool(),
		AvailabilityZone: plan.Core.Zone.ValueString(),
		Domain:           plan.Core.Domain.ValueString(),
		Platform:         plan.Core.Platform.ValueString(),
		OnSupport:        plan.Image.OnSupport.ValueBool(),
		OsVersion:        plan.Image.OsVersion.ValueString(),
		DefaultNic: entities.DefaultNic{
			NetSegment: plan.Core.NetSegmentCode.ValueString(),
		},
		TarantoolVersion:    plan.TarantoolVersion.ValueString(),
		Layout:              plan.Layout.ValueString(),
		GeoDistribution:     plan.Image.GeoDistribution.ValueBool(),
		ExtraNics:           []interface{}{},
		CreatedWithOpenTofu: true,
	}
	return attrs
}

func (r TarantoolClusterResource) convertZoneToTF(
	ctx context.Context,
	zoneName string,
	order *orders.TarantoolCluster,
	cluster map[string][]entities.ZoneInstance,
) (ZoneConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	running, err := order.GetZoneInstancesByStateFilter(zoneName, "on", cluster)
	if err != nil {
		diags.AddError(
			fmt.Sprintf("Failed to get running instances for zone %s", zoneName),
			err.Error(),
		)
	}

	stopped, err := order.GetZoneInstancesByStateFilter(zoneName, "off", cluster)
	if err != nil {
		diags.AddError(
			fmt.Sprintf("Failed to get stopped instances for zone %s", zoneName),
			err.Error(),
		)
	}

	if len(running) == 0 {
		running = []string{}
	}

	if len(stopped) == 0 {
		stopped = []string{}
	}

	runningSet, d := types.SetValueFrom(ctx, types.StringType, running)
	diags.Append(d...)

	stoppedSet, d := types.SetValueFrom(ctx, types.StringType, stopped)
	diags.Append(d...)

	model := ZoneConfigModel{
		RunningInstances: runningSet,
		StoppedInstances: stoppedSet,
	}

	return model, diags

}

func (r TarantoolClusterResource) convertZonesToTF(
	ctx context.Context,
	order *orders.TarantoolCluster,
	config entities.TarantoolClusterItemConfig,
) (map[string]ZoneConfigModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	zones, err := order.GetZones()
	if err != nil {
		diags.AddError(
			fmt.Sprintln("Error to get cluster zones"),
			err.Error(),
		)
	}

	result := make(map[string]ZoneConfigModel)
	for _, zoneName := range zones {
		zoneModel, zoneDiags := r.convertZoneToTF(ctx, zoneName, order, config.Cluster)
		if zoneDiags.HasError() {
			diags.Append(zoneDiags...)
			continue
		}
		result[zoneName] = zoneModel
	}

	if len(result) == 0 {
		diags.AddError("No zones processed", "All zone conversions failed")
	}

	if diags.HasError() {
		return nil, diags
	}

	return result, diags
}

func (r TarantoolClusterResource) applyZoneConfig(
	ctx context.Context,
	plan, state TarantoolClusterResourceModel,
	order *orders.TarantoolCluster,
	resp *resource.UpdateResponse,
) {
	planZones := make(map[string]ZoneConfigModel)
	if !plan.Zones.IsNull() {
		resp.Diagnostics.Append(plan.Zones.ElementsAs(ctx, &planZones, false)...)
	}

	stateZones := make(map[string]ZoneConfigModel)
	if !state.Zones.IsNull() {
		resp.Diagnostics.Append(state.Zones.ElementsAs(ctx, &stateZones, false)...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	for zoneName, planZone := range planZones {
		stateZone, exists := stateZones[zoneName]
		if !exists {
			resp.Diagnostics.AddError(
				consts.UPDATE_RES_FAIL,
				"Add new zone is unvailable",
			)
			return
		}

		toEnable, toDisable := r.compareZoneChanges(ctx, planZone, stateZone, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}

		zoneNum, err := utils.ExtractZoneNumber(zoneName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Invalid zone name",
				fmt.Sprintf("Failed to parse zone number from '%s': %v", zoneName, err),
			)
			return
		}

		if len(toEnable) > 0 {
			attrs := orders.SwitchInstancesStateAttrs{
				Instances: toEnable,
				Type:      "Instance",
			}
			if err := order.EnableZoneInstances(attrs, zoneNum); err != nil {
				resp.Diagnostics.AddError(
					consts.UPDATE_RES_FAIL,
					fmt.Sprintf("Failed to enable instances in zone %s: %v", zoneName, err),
				)
				return
			}
		}

		if len(toDisable) > 0 {
			attrs := orders.SwitchInstancesStateAttrs{
				Instances: toDisable,
				Type:      "Instance",
			}
			if err := order.DisableZoneInstances(attrs, zoneNum); err != nil {
				resp.Diagnostics.AddError(
					consts.UPDATE_RES_FAIL,
					fmt.Sprintf("Failed to disable instances in zone %s: %v", zoneName, err),
				)
				return
			}
		}
	}
}

func (r TarantoolClusterResource) compareZoneChanges(
	ctx context.Context,
	planZone, stateZone ZoneConfigModel,
	diags *diag.Diagnostics,
) (toRunning, toStopped []string) {

	oldRunning, d := utils.ConvertSetToList(ctx, stateZone.RunningInstances)
	diags.Append(d...)

	newRunning, d := utils.ConvertSetToList(ctx, planZone.RunningInstances)
	diags.Append(d...)

	oldStopped, d := utils.ConvertSetToList(ctx, stateZone.StoppedInstances)
	diags.Append(d...)

	newStopped, d := utils.ConvertSetToList(ctx, planZone.StoppedInstances)
	diags.Append(d...)

	totalOld := len(oldRunning) + len(oldStopped)
	totalNew := len(newRunning) + len(newStopped)

	if totalOld != totalNew {
		diags.AddError(
			"Instance count mismatch",
			fmt.Sprintf(
				"Total instance count must remain constant. Old: %d, New: %d",
				totalOld,
				totalNew,
			),
		)
		return nil, nil
	}

	stateInstances := make(map[string]bool)
	for _, inst := range append(oldRunning, oldStopped...) {
		stateInstances[inst] = true
	}

	for _, inst := range append(newRunning, newStopped...) {
		if !stateInstances[inst] {
			diags.AddError(
				"Invalid instance reference",
				fmt.Sprintf("Instance '%s' not found in current state", inst),
			)
			return nil, nil
		}
	}

	if diags.HasError() {
		return nil, nil
	}

	toRunning = utils.SliceDifference(newRunning, oldRunning)
	toStopped = utils.SliceDifference(newStopped, oldStopped)

	return toRunning, toStopped
}

func (r TarantoolClusterResource) updateTarantoolVersion(
	plan *TarantoolClusterResourceModel,
	order *orders.TarantoolCluster,
	resp *resource.UpdateResponse,
) {
	attrs := orders.UpdateTarantoolVersionAttrs{
		TarantoolType:       plan.TarantoolType.ValueString(),
		NewTarantoolVersion: plan.TarantoolVersion.ValueString(),
	}

	err := order.UpdateTarantoolVersion(attrs)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "upgrade tarantool version ended with error"),
			err.Error(),
		)
	}
}
