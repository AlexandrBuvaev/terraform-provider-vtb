package syncxpert

import (
	"context"
	"fmt"
	"reflect"
	"regexp"

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
	_ resource.Resource = &SyncXpertClusterResource{}
)

type SyncXpertClusterResource struct {
	client *client.CloudClient
}

func NewSyncXpertClusterResource() resource.Resource {
	return &SyncXpertClusterResource{}
}

func (r *SyncXpertClusterResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_sync_xpert_cluster"
}

func (r *SyncXpertClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type SyncXpertClusterResourceModel struct {
	Core        core.CoreModel                            `tfsdk:"core"`
	Flavor      flavor.FlavorModel                        `tfsdk:"flavor"`
	Image       common.ClusterProductImageDataSourceModel `tfsdk:"image"`
	Lifetime    types.Int64                               `tfsdk:"lifetime"`
	Label       types.String                              `tfsdk:"label"`
	OrderID     types.String                              `tfsdk:"order_id"`
	ItemID      types.String                              `tfsdk:"item_id"`
	Access      map[string][]string                       `tfsdk:"access"`
	ExtraMounts map[string]common.ExtraMountModel         `tfsdk:"extra_mounts"`
	LayoutID    types.String                              `tfsdk:"layout_id"`
	ClusterName types.String                              `tfsdk:"cluster_name"`

	APIUser         types.String `tfsdk:"api_user"`
	APIPassword     types.String `tfsdk:"api_password"`
	DebeziumVersion types.String `tfsdk:"debezium_version"`
	KafkaCertName   types.String `tfsdk:"kafka_cert_cname"`
	KafkaServer     types.String `tfsdk:"kafka_server"`
	ClusterGroupID  types.String `tfsdk:"cluster_group_id"`

	FinancialProject types.String `tfsdk:"financial_project"`
}

func (r *SyncXpertClusterResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Управление облачным продуктом SyncExpert Astra",
		Attributes: map[string]schema.Attribute{
			"core": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.CoreSchema,
				MarkdownDescription: "Основные параметры для виртуальных машин в заказе",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"flavor": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.FlavorSchema,
				MarkdownDescription: "Кол-во CPU/RAM для виртуальных машин",
			},
			"image": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.ClusterProductImageSchema,
				MarkdownDescription: "Тип вычислительного экземпляра",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
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
				MarkdownDescription: "Идетификаторо заказа, становится известен после создания заказа",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"item_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Идентификатор сущностей ВМ, известен после создания ВМ",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"access": schema.MapAttribute{
				Required: true,
				Description: "Карта, где ключом является роль, а значением - список групп, " +
					"который предоставит доступ для входа в Active Directory",
				MarkdownDescription: "Карта, где ключом является роль, а значением - список групп, " +
					"который предоставит доступ для входа в Active Directory",
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
					customvalidators.EmptyAccessGroupListValidator{},
				},
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
			},
			"extra_mounts": schema.MapNestedAttribute{
				Required:            true,
				MarkdownDescription: "Дополнительные точки монтирования для ВМ",
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
			"cluster_name": schema.StringAttribute{
				Required:            true,
				Description:         "Имя кластера",
				MarkdownDescription: "Имя кластера",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile("^[a-z0-9-]+$"),
						"must pattern ^[a-z0-9-]+$",
					),
				},
			},

			"layout_id": schema.StringAttribute{
				Required:            true,
				Description:         "Идентификатор конфигурации кластера",
				MarkdownDescription: "Идентификатор конфигурации кластера",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"debezium_version": schema.StringAttribute{
				Required:            true,
				Description:         "Версия Debezium",
				MarkdownDescription: "Версия Debezium",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("1.1.0"),
				},
			},
			"api_user": schema.StringAttribute{
				Required:            true,
				Description:         "Пользователь REST API",
				MarkdownDescription: "Пользователь REST API",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"api_password": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				Description:         "Пароль для REST_API",
				MarkdownDescription: "Пароль для REST_API",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},

			"kafka_cert_cname": schema.StringAttribute{
				Required: true,
				Description: "CommonName (CN) c которым будет выпущен клиентский сертификат." +
					"Пример: APD[код АПД]-[RIS код]-kafka-client-syncxpert-[префикс среды]-*",
				MarkdownDescription: "CommonName (CN) с которым будет выпущен клиентский сертификат." +
					"Пример: APD[код АПД]-[RIS код]-kafka-client-syncxpert-[префикс среды]-*",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"kafka_server": schema.StringAttribute{
				Required:            true,
				Description:         "Серверы в формате server:port, разделенные запятыми (bootstrap_servers)",
				MarkdownDescription: "Серверы в формате server:port, разделенные запятыми (bootstrap_servers)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cluster_group_id": schema.StringAttribute{
				Required: true,
				Description: "Группа для уникальной идентификации кластера Debezium, " +
					"используется при работе с системными разделами (топиками)",
				MarkdownDescription: "Группа для уникальной идентификации кластера Debezium, " +
					"используется при работе с ситемными разделами (топиками)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"financial_project": schema.StringAttribute{
				Required:            true,
				Description:         "Источник финансирования",
				MarkdownDescription: "Источник финансирования",
			},
		},
	}
}

func (r *SyncXpertClusterResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan SyncXpertClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	checkOrderIsDeleted := utils.CheckOrderIsDeleted(
		r.client,
		plan.OrderID,
		"cluster",
		"debezium",
	)
	resp.Diagnostics.Append(checkOrderIsDeleted.Diagnostics...)

	utils.HandleExternalDeletionRecreate(
		ctx,
		resp,
		&plan,
		checkOrderIsDeleted.IsDeleted,
		checkOrderIsDeleted.Diagnostics,
	)

	validateAccessRolesDiags := utils.ValidateAccessRolesV1(
		r.client,
		"cluster:debezium",
		plan.Access,
	)
	resp.Diagnostics.Append(validateAccessRolesDiags...)

	if resp.Diagnostics.HasError() {
		return
	}

}

func (r *SyncXpertClusterResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan SyncXpertClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterConfig := orders.SyncXpertClusterConfig{
		BasicAuthUser:     plan.APIUser.ValueString(),
		BasicAuthPassword: plan.APIPassword.ValueString(),
		DebeziumVersion:   plan.DebeziumVersion.ValueString(),
		KafkaCertCname:    plan.KafkaCertName.ValueString(),
		BootstrapServers:  plan.KafkaServer.ValueString(),
		ClusterGroupID:    plan.ClusterGroupID.ValueString(),
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
	var access []entities.ADLogonGrants
	for role, groups := range plan.Access {
		var groupsNames []string
		groupsNames = append(groupsNames, groups...)
		access = append(access, entities.ADLogonGrants{
			Role:   role,
			Groups: groupsNames,
		})
	}

	attrs := orders.SyncXpertAttrs{
		BasicAttrs: orders.BasicAttrs{
			ADLogonGrants:    access,
			ADIntegration:    plan.Image.ADIntegration.ValueBool(),
			ExtraMounts:      utils.PrepareExtraMountsAttrs(plan.ExtraMounts),
			AvailabilityZone: plan.Core.Zone.ValueString(),
			Domain:           plan.Core.Domain.ValueString(),
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
		},
		ExtraNics:       []interface{}{},
		Layout:          plan.LayoutID.ValueString(),
		GeoDistribution: plan.Image.GeoDistribution.ValueBool(),
		DebeziumConfig:  clusterConfig,
		AcceptVM:        true,
		NewInput1:       map[string]interface{}{},
		ClusterName:     plan.ClusterName.ValueString(),
	}

	order := orders.NewSyncXpertCluster(
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
			fmt.Sprintf("Error while getting item.\nError: %s", err.Error()),
		)
		return
	}

	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(orderItem.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SyncXpertClusterResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var orderID, apiPassword types.String
	req.State.GetAttribute(ctx, path.Root("api_password"), &apiPassword)
	req.State.GetAttribute(ctx, path.Root("order_id"), &orderID)
	if orderID.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("order_id"),
			consts.READ_RES_FAIL,
			"Resource hasn't order_id. Without order_id can't get order from portal",
		)
		return
	}

	order, err := orders.GetDebeziumOrder(
		r.client.Creds,
		r.client.ProjectName,
		orderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("order_id"),
			consts.READ_RES_FAIL,
			fmt.Sprintf("fetch order from portal API ended with error: %v", err.Error()),
		)
		return
	}

	orderItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("order_id"),
			consts.READ_RES_FAIL,
			fmt.Sprintf("error while getting cluster item, error: %v", err.Error()),
		)
		return
	}

	vmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Can't get VM items.\nError:%v", err.Error()),
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
	extraMount, err := order.GetExtraMount("/app")
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
	clusterConfig, ok := orderItem.Data.Config.(entities.SyncXpertItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Invalid type for SyncExpertCluster config",
		)
		return
	}

	state := SyncXpertClusterResourceModel{
		Core: core.CoreModel{
			Platform:       types.StringValue(utils.SelectPlatform(order.Attrs.Platform)),
			Domain:         types.StringValue(order.Attrs.Domain),
			NetSegmentCode: types.StringValue(vmConfig.DefaultNic.NetSegment),
			Zone:           types.StringValue(order.Attrs.AvailabilityZone),
		},
		Access:           utils.ReadAccessMapVV1(vmItems[0].Data.ACLs),
		ItemID:           types.StringValue(orderItem.ID),
		Label:            types.StringValue(order.Label),
		FinancialProject: types.StringValue(order.FinancialSource.Name),
		LayoutID:         types.StringValue(order.Attrs.Layout),
		OrderID:          orderID,
		Image: common.ClusterProductImageDataSourceModel{
			ProductID:       types.StringValue(order.ProductID),
			OnSupport:       types.BoolValue(order.Attrs.OnSupport),
			ADIntegration:   types.BoolValue(order.Attrs.ADIntegration),
			Distribution:    types.StringValue(vmConfig.Image.Os.Distribution),
			OsVersion:       types.StringValue(vmConfig.Image.Os.Version),
			GeoDistribution: types.BoolValue(order.Attrs.GeoDistribution),
		},
		Flavor: flavor.FlavorModel{
			Cores:  types.Int64Value(vmConfig.Flavor.Cores),
			Memory: types.Int64Value(vmConfig.Flavor.Memory),
			UUID:   types.StringValue(vmConfig.Flavor.UUID),
			Name:   types.StringValue(vmConfig.Flavor.Name),
		},
		ExtraMounts: map[string]common.ExtraMountModel{
			"/app": {
				Size:       types.Int64Value(extraMount.Size),
				FileSystem: types.StringValue(extraMount.FileSystem),
			},
		},
		DebeziumVersion: types.StringValue(order.Attrs.DebeziumConfig.DebeziumVersion),
		APIUser:         types.StringValue(clusterConfig.Certificates[0].RestUser),
		KafkaCertName:   types.StringValue(clusterConfig.Certificates[0].KafkaCn),
		KafkaServer:     types.StringValue(clusterConfig.BootstrapServers),
		ClusterName:     types.StringValue(clusterConfig.ClusterName),
		ClusterGroupID:  types.StringValue(order.Attrs.DebeziumConfig.ClusterGroupID),
		APIPassword:     apiPassword,
	}

	var lifetime types.Int64
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("lifetime"), &lifetime)...)
	if !lifetime.IsNull() {
		state.Lifetime = lifetime
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SyncXpertClusterResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state SyncXpertClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetDebeziumOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(consts.READ_RES_FAIL, err.Error())
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
	accessChanged := !reflect.DeepEqual(plan.Access, state.Access)
	mountChanged := utils.IsExtraMountChanged(state.ExtraMounts, plan.ExtraMounts)

	if labelChanged {
		utils.ChangeOrderLabel(order, plan.Label.ValueString(), resp)
	}

	if !plan.FinancialProject.Equal(state.FinancialProject) {
		err = order.ChangeFinancialProject(finProj.ID)
		if err != nil {
			resp.Diagnostics.AddError(
				consts.UPDATE_RES_FAIL,
				fmt.Sprintf("Change financial source ended with error.\nError: %v", err.Error()),
			)
			return
		}
	}

	if flavorChanged {
		r.verticalScaling(order, &plan, resp)
	}

	if accessChanged {
		r.changeAccess(order, &plan, &state, resp)
	}

	if mountChanged {
		r.changeExtraMountsSyncXpert(order, &plan, resp)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SyncXpertClusterResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state SyncXpertClusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetDebeziumOrder(
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

	err = order.DeleteTwoLayer(false)
	if err != nil {
		resp.Diagnostics.AddError(consts.DELETE_RES_FAIL, err.Error())
		return
	}
}

func (r *SyncXpertClusterResource) verticalScaling(
	order *orders.SyncXpertCluster,
	plan *SyncXpertClusterResourceModel,
	resp *resource.UpdateResponse,
) {
	flavor := entities.Flavor{
		Cores:  plan.Flavor.Cores.ValueInt64(),
		Memory: plan.Flavor.Memory.ValueInt64(),
		Name:   plan.Flavor.Name.ValueString(),
		UUID:   plan.Flavor.UUID.ValueString(),
	}
	err := order.VerticalScaling(flavor, false)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("flavor"),
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "debezium_vertical_scaling ended with error"),
			err.Error(),
		)
		return
	}
}

func (r *SyncXpertClusterResource) changeAccess(
	order *orders.SyncXpertCluster,
	plan, state *SyncXpertClusterResourceModel,
	resp *resource.UpdateResponse,
) {

	addedRoles, changedRoles, deletedRoles := r.compareADRoles(state, plan)

	if len(addedRoles) > 0 {
		rolesFromCloud, err := references.GetRoles(
			r.client.Creds,
			r.client.Environment,
			"app:wildfly",
		)
		if err != nil {
			resp.Diagnostics.AddError("Receiving roles from Cloud", err.Error())
			return
		}

		permissionsMap := make(map[string][]string)
		for _, value := range rolesFromCloud {
			permissionsMap[value.Data.GroupName] = value.Data.Permissions
		}

		for _, roles := range addedRoles {
			err := order.AddAccessGroup(roles, permissionsMap[roles.Role])
			if err != nil {
				resp.Diagnostics.AddError("Adding new VM roles", err.Error())
			}
		}
	}

	if len(changedRoles) > 0 {
		for _, roles := range changedRoles {
			err := order.ChangeAccessGroup(roles)
			if err != nil {
				resp.Diagnostics.AddError("Changing groups in VM roles", err.Error())
				return
			}
		}
	}

	if len(deletedRoles) > 0 {
		for _, roles := range deletedRoles {
			err := order.DeleteAccessGroup(roles)
			if err != nil {
				resp.Diagnostics.AddError("Deleting VM roles", err.Error())
			}
		}
	}
}

func (r *SyncXpertClusterResource) compareADRoles(
	state, plan *SyncXpertClusterResourceModel,
) (addedRoles, changedRoles, deletedRoles []entities.ADLogonGrants) {
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
	return addedRoles, changedRoles, deletedRoles
}

func (r *SyncXpertClusterResource) changeExtraMountsSyncXpert(
	order *orders.SyncXpertCluster,
	plan *SyncXpertClusterResourceModel,
	resp *resource.UpdateResponse) {
	for path, planExtraMount := range plan.ExtraMounts {
		err := order.ExpandMountPoint(entities.ExtraMount{
			Path:       path,
			Size:       planExtraMount.Size.ValueInt64(),
			FileSystem: planExtraMount.FileSystem.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("Incorrent changes in extra_mounts parameters", err.Error())
		}
	}
}
