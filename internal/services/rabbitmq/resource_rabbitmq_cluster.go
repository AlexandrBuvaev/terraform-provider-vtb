package rabbitmq

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                = &RabbitMQClusterResource{}
	_ resource.ResourceWithModifyPlan  = &RabbitMQClusterResource{}
	_ resource.ResourceWithImportState = &RabbitMQClusterResource{}
)

type RabbitMQClusterResource struct {
	client *client.CloudClient
}

func NewRabbitMQResource() resource.Resource {
	return &RabbitMQClusterResource{}
}

func (r RabbitMQClusterResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_rabbitmq_cluster"
}

func (r *RabbitMQClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type RabbitMQClusterModel struct {
	Lifetime              types.Int64                               `tfsdk:"lifetime"`
	Label                 types.String                              `tfsdk:"label"`
	OrderID               types.String                              `tfsdk:"order_id"`
	ItemID                types.String                              `tfsdk:"item_id"`
	Core                  core.CoreModel                            `tfsdk:"core"`
	Flavor                flavor.FlavorModel                        `tfsdk:"flavor"`
	Image                 common.ClusterProductImageDataSourceModel `tfsdk:"image"`
	Access                map[string][]types.String                 `tfsdk:"access"`
	ExtraMounts           map[string]common.ExtraMountModel         `tfsdk:"extra_mounts"`
	LayoutID              types.String                              `tfsdk:"layout_id"`
	BuildVersion          types.String                              `tfsdk:"build_version"`
	ClusterName           types.String                              `tfsdk:"cluster_name"`
	RabbitMQVersion       types.String                              `tfsdk:"rabbitmq_version"`
	WebAccess             *WebAccessModel                           `tfsdk:"web_access"`
	FinancialProject      types.String                              `tfsdk:"financial_project"`
	CertificateCn         types.String                              `tfsdk:"certificate_cn"`
	CertificateExpiration types.String                              `tfsdk:"certificate_expiration"`
	CertificateValidFrom  types.String                              `tfsdk:"certificate_valid_from"`
	UpdateMode            types.String                              `tfsdk:"update_product_mode"`
}

type WebAccessModel struct {
	Admins   types.Set `tfsdk:"admins"`
	Managers types.Set `tfsdk:"managers"`
}

func (r RabbitMQClusterResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			// basic schema
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
				MarkdownDescription: "Название заказа",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"order_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Идентификатор заказа. Становится известен после создания заказа",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"item_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Идентификатор элемента виртуальной машины по заказе. Становиться известен после создания виртуальной машины",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
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
				MarkdownDescription: "Количество ЦПУ и оперативной памяти, которые необходимы на вирутальных машинах",
			},
			"image": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.ClusterProductImageSchema,
				MarkdownDescription: "Образ вычислительно экземпляра",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
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
					mapplanmodifier.RequiresReplace(),
					custommodifires.EmptyAccessGroup(),
				},
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
			},
			"extra_mounts": schema.MapNestedAttribute{
				MarkdownDescription: "Дополнительные точки монтирования",
				Required:            true,
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.OneOf("/app")),
					mapvalidator.SizeBetween(1, 1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"size": schema.Int64Attribute{
							MarkdownDescription: "Размер точки монтирования (указвается в ГБ)",
							Required:            true,
							Validators: []validator.Int64{
								int64validator.AtMost(consts.CLOUD_EXTRA_MOUNT_MAX_SIZE),
								int64validator.AtLeast(30),
							},
							PlanModifiers: []planmodifier.Int64{
								custommodifires.GrowOnly(),
								// MininalGrow(10),
							},
						},
						"file_system": schema.StringAttribute{
							Computed:            true,
							Default:             stringdefault.StaticString("xfs"),
							Description:         "Название типа файловой системы",
							MarkdownDescription: "Название типа файловой системы",
						},
					},
				},
			},

			// extra product schema
			"layout_id": schema.StringAttribute{
				Required:            true,
				Description:         "Layout ID из справочника geo_distribution.",
				MarkdownDescription: "Layout ID из справочника geo_distribution.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Клиентская часть имени кластера",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[A-Za-z0-9]+([-][A-Za-z0-9]+)*$`),
						"may contain uppercase/lowercase Latin letters, numbers, "+
							"punctuation marks ['-']. (cannot start or end with punctuation)",
					),
				},
			},
			"rabbitmq_version": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Версия RabbitMQ",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("3.8.14", "3.11.26"),
				},
			},
			"build_version": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Версия релиза продукта",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"web_access": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Управление пользователями по ролям(администраторы, менеджеры) для доступа к веб-интерфейсу кластера",
				Attributes: map[string]schema.Attribute{
					"admins": schema.SetAttribute{
						Optional:    true,
						ElementType: types.StringType,
						PlanModifiers: []planmodifier.Set{
							setplanmodifier.UseStateForUnknown(),
						},
					},
					"managers": schema.SetAttribute{
						Required:    true,
						ElementType: types.StringType,
						PlanModifiers: []planmodifier.Set{
							setplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},

			"financial_project": schema.StringAttribute{
				Required:            true,
				Description:         "Источник финансирования для заказа.",
				MarkdownDescription: "Источник финансирования для заказа.",
			},
			"certificate_cn": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description:         "Название сертификата",
				MarkdownDescription: "Название сертификата",
			},
			"certificate_expiration": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description:         "Дата начала действия сертификата",
				MarkdownDescription: "Дата начала действия сертификата",
			},
			"certificate_valid_from": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description:         "Дата окончания действия сертификата",
				MarkdownDescription: "Дата окончания действия сертификата",
			},
			"update_product_mode": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("none"),
				MarkdownDescription: `Мод для регулировки политики обновлений("latest"/"none"),если флаг проставлен в "latest", 
								      то будет активировано действие "Обновить версию релиза продукта"`,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("latest", "none"),
				},
			},
		},
	}
}

func (r RabbitMQClusterResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r RabbitMQClusterResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan RabbitMQClusterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	var state RabbitMQClusterModel
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	}

	checkOrderIsDeleted := utils.CheckOrderIsDeleted(r.client, plan.OrderID, "cluster", "rabbitmq")
	resp.Diagnostics.Append(checkOrderIsDeleted.Diagnostics...)

	utils.HandleExternalDeletionRecreate(ctx, resp, &plan, checkOrderIsDeleted.IsDeleted, checkOrderIsDeleted.Diagnostics)

	valiadteAccessForEnvDiags := utils.ValidateActiveDirectoryAccessForEnvironmentsV1(
		r.client,
		plan.Access,
		"DEV",
	)
	resp.Diagnostics.Append(valiadteAccessForEnvDiags...)

	validateAccessRolesDiags := utils.ValidateAccessRolesV2(r.client, "cluster:rabbitmq", plan.Access)
	resp.Diagnostics.Append(validateAccessRolesDiags...)

	r.validateWebAccessAdminsForSpecificEnv(plan, resp)

	r.minMoutSizeWithFlavorModifyPlan(&plan, resp)

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r RabbitMQClusterResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan RabbitMQClusterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !atLeastOneManager(ctx, &plan) {
		resp.Diagnostics.AddAttributeError(
			path.Root("web_access"),
			consts.CREATE_RES_FAIL,
			"at least one group must specified for `web_access = { managers = ['your-group-for-web-access'] }` "+
				"while initial cluster creation",
		)
		return
	}

	attrs := r.prepareAttrs(ctx, &plan)
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

	order := orders.NewRabbitMQ(
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
		resp.Diagnostics.AddError("Can't create rabbitmq order: ", err.Error())
		return
	}

	orderItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError("Error while getting item: ", err.Error())
		return
	}

	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(orderItem.ID)
	plan.BuildVersion = types.StringValue(orderItem.Data.Build.SetupVersion)
	plan.CertificateCn = types.StringValue(orderItem.Data.Config.(entities.RabbitMQItemConfig).CertificateCn)
	plan.CertificateExpiration = types.StringValue(orderItem.Data.Config.(entities.RabbitMQItemConfig).CertificateExpiration)
	plan.CertificateValidFrom = types.StringValue(orderItem.Data.Config.(entities.RabbitMQItemConfig).CertificateValidFrom)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r RabbitMQClusterResource) Read(
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

	order, err := orders.GetRabbitMQOrder(
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

	cluster, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("order_id"),
			fmt.Sprintln(consts.READ_RES_FAIL, "can't get cluster item"),
			err.Error(),
		)
		return
	}

	VMs, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Can't get rabbitmq VMs items.\nError: %s", err.Error()),
		)
		return
	}
	vmConfig, ok := VMs[0].Data.Config.(entities.VMItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Invalid type for vm config.",
		)
		return
	}

	clusterConfig := cluster.Data.Config.(entities.RabbitMQItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Invalid type for cluster config",
		)
		return
	}
	admins, managers := readWebAccessGroups(ctx, &clusterConfig, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	rmqNode, err := order.GetRabbitMQNode()
	if err != nil {
		resp.Diagnostics.AddError(consts.READ_RES_FAIL, err.Error())
		return
	}
	rmqNodeConfig, ok := rmqNode.Data.Config.(entities.VMItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Invalid type for RabbitMQ Node Config",
		)
	}

	prefix := r.getRabbitMQLayoutsPrefix(vmConfig.DefaultNic.NetSegment)

	rabbitmqLayouts := map[int]string{
		2: fmt.Sprintf("%s:rabbitmq-2:quorum-1", prefix),
		4: fmt.Sprintf("%s:rabbitmq-4:quorum-1", prefix),
		6: fmt.Sprintf("%s:rabbitmq-6:quorum-1", prefix),
	}

	rabbimqQuantity := clusterConfig.HostsInfo.QuantityRabbitMQ
	layoutId, err := references.GetGeoDistributionLayoutID(
		r.client.Creds,
		rabbitmqLayouts[int(rabbimqQuantity)],
		"rabbitmq",
		r.client.Organization,
		vmConfig.DefaultNic.NetSegment,
	)
	if err != nil {
		resp.Diagnostics.AddError(consts.READ_RES_FAIL, err.Error())
		return
	}
	clusterName := strings.SplitN(clusterConfig.ClusterName, "-", 2)

	state := RabbitMQClusterModel{
		Access:  utils.ReadAccessMapV2(VMs[0].Data.ACLs),
		ItemID:  types.StringValue(cluster.ID),
		Label:   types.StringValue(order.Label),
		OrderID: orderID,
		Core: core.CoreModel{
			Platform:       types.StringValue(utils.SelectPlatform(order.Attrs.Platform)),
			Domain:         types.StringValue(order.Attrs.Domain),
			NetSegmentCode: types.StringValue(vmConfig.DefaultNic.NetSegment),
			Zone:           types.StringValue(order.Attrs.AvailabilityZone),
		},
		ExtraMounts: map[string]common.ExtraMountModel{
			"/app": {
				Size:       types.Int64Value(clusterConfig.ExtraMounts.Size),
				FileSystem: types.StringValue(clusterConfig.ExtraMounts.FSType),
			},
		},
		Flavor: flavor.FlavorModel{
			Cores:  types.Int64Value(rmqNodeConfig.Flavor.Cores),
			Memory: types.Int64Value(rmqNodeConfig.Flavor.Memory),
			Name:   types.StringValue(rmqNodeConfig.Flavor.Name),
			UUID:   types.StringValue(rmqNodeConfig.Flavor.UUID),
		},
		ClusterName:     types.StringValue(clusterName[1]),
		RabbitMQVersion: types.StringValue(*cluster.Data.Build.RabbitMQVersion),
		BuildVersion:    types.StringValue(cluster.Data.Build.SetupVersion),
		LayoutID:        types.StringValue(layoutId),
		WebAccess: &WebAccessModel{
			Admins:   admins,
			Managers: managers,
		},
		Image: common.ClusterProductImageDataSourceModel{
			ProductID:       types.StringValue(order.ProductID),
			OnSupport:       types.BoolValue(order.Attrs.OnSupport),
			ADIntegration:   types.BoolValue(order.Attrs.ADIntegration),
			Distribution:    types.StringValue(vmConfig.Image.Os.Distribution),
			OsVersion:       types.StringValue(vmConfig.Image.Os.Version),
			GeoDistribution: types.BoolValue(order.Attrs.GeoDistribution),
		},
		FinancialProject:      types.StringValue(order.FinancialSource.Name),
		CertificateCn:         types.StringValue(clusterConfig.CertificateCn),
		CertificateExpiration: types.StringValue(clusterConfig.CertificateExpiration),
		CertificateValidFrom:  types.StringValue(clusterConfig.CertificateValidFrom),
		UpdateMode:            types.StringValue("none"),
	}

	var lifetime types.Int64
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("lifetime"), &lifetime)...)
	if !lifetime.IsNull() {
		state.Lifetime = lifetime
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r RabbitMQClusterResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state RabbitMQClusterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetRabbitMQOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Can't get order from portal: ", err.Error())
		return
	}
	rabbitmqImageData, err := references.GetRabbitMQImageData(
		r.client.Creds,
		state.Image.Distribution.ValueString(),
		r.client.Organization,
		r.client.Environment,
		state.Image.OsVersion.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Can't get rabbitmq_image_data from refernces",
			err.Error(),
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

	if !plan.FinancialProject.Equal(state.FinancialProject) {
		order.ChangeFinancialProject(finProj.ID)
	}

	labelChanged := plan.Label != state.Label
	if labelChanged {
		utils.ChangeOrderLabel(order, plan.Label.ValueString(), resp)
	}

	if isWebAccessChanged(&plan, &state) {
		r.updateWebAcces(ctx, order, &plan, resp)
	}

	if isVerticalScalingNeeded(&state, &plan) {
		r.verticalScaling(order, &plan, resp)
	}

	if !plan.LayoutID.Equal(state.LayoutID) {
		r.horizontalScaling(order, &plan, &state, resp)
	}

	if !plan.RabbitMQVersion.Equal(state.RabbitMQVersion) {
		r.upgradeRabbitMQVersion(order, &plan, &state, resp)
	}

	currentReleaseVersion := state.BuildVersion.ValueString()
	latestReleaseVersion := rabbitmqImageData.LatestReleaseVersion

	if !plan.UpdateMode.IsNull() && plan.UpdateMode.ValueString() == "latest" {
		r.updateReleaseVersion(order, currentReleaseVersion, latestReleaseVersion, resp)
	}

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r RabbitMQClusterResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state RabbitMQClusterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetRabbitMQOrder(
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
		resp.Diagnostics.AddWarning(
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
					"Please, make your order is deletable by the portal VTB.Cloud, and repeat again.\nURL reference: %s",
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

// * Custom logic*

func (r RabbitMQClusterResource) minMoutSizeWithFlavorModifyPlan(
	plan *RabbitMQClusterModel,
	resp *resource.ModifyPlanResponse,
) {
	type ClusterConfig struct {
		Memory       int64
		Cores        int64
		MinMountSize int64
	}

	portalConfigs := []ClusterConfig{
		{Cores: 2, Memory: 4, MinMountSize: 30},
		{Cores: 2, Memory: 8, MinMountSize: 50},
		{Cores: 4, Memory: 8, MinMountSize: 50},
		{Cores: 4, Memory: 16, MinMountSize: 70},
		{Cores: 8, Memory: 16, MinMountSize: 70},
		{Cores: 8, Memory: 32, MinMountSize: 100},
	}

	memory := plan.Flavor.Memory.ValueInt64()
	cores := plan.Flavor.Cores.ValueInt64()

	extraMounts := utils.PrepareExtraMountsAttrs(plan.ExtraMounts)
	for _, config := range portalConfigs {
		if memory == config.Memory && cores == config.Cores {

			if extraMounts[0].Size < config.MinMountSize {
				resp.Diagnostics.AddAttributeError(
					path.Root("extra_mounts"),
					consts.MODIFY_PLAN_FAIL,
					fmt.Sprintf(
						"for specified `flavor = {memory = %v, cores = %v}` "+
							"size of `/app` mount should be at minimum %v Gb.",
						memory, cores, config.MinMountSize,
					),
				)
				return
			}

		}
	}
}

func (r RabbitMQClusterResource) prepareAttrs(
	ctx context.Context,
	plan *RabbitMQClusterModel,
) orders.RabbitMQAttrs {

	admins, managers := prepareWebAccessGroups(ctx, plan)
	extraMount := plan.ExtraMounts["/app"]

	return orders.RabbitMQAttrs{
		Flavor: entities.Flavor{
			Cores:  plan.Flavor.Cores.ValueInt64(),
			Memory: plan.Flavor.Memory.ValueInt64(),
			Name:   plan.Flavor.Name.ValueString(),
			UUID:   plan.Flavor.UUID.ValueString(),
		},
		ADIntegration:    plan.Image.ADIntegration.ValueBool(),
		ADLogonGrants:    utils.PrepareADLogonGrants(plan.Access),
		Domain:           plan.Core.Domain.ValueString(),
		AvailabilityZone: plan.Core.Zone.ValueString(),
		Platform:         plan.Core.Platform.ValueString(),
		DefaultNic: entities.DefaultNic{
			NetSegment: plan.Core.NetSegmentCode.ValueString(),
		},
		OnSupport:   plan.Image.OnSupport.ValueBool(),
		OsVersion:   plan.Image.OsVersion.ValueString(),
		Path:        "/app",
		FileSystem:  "xfs",
		Size:        extraMount.Size.ValueInt64(),
		SizingCheck: true,

		ExtraNics:           []interface{}{},
		ClusterName:         plan.ClusterName.ValueString(),
		LayoutID:            plan.LayoutID.ValueString(),
		RabbitMQVersion:     plan.RabbitMQVersion.ValueString(),
		GeoDistribution:     plan.Image.GeoDistribution.ValueBool(),
		WebAdmingGroups:     admins,
		WebManagerGroups:    managers,
		CreatedWithOpenTofu: true,
	}
}

func (r RabbitMQClusterResource) updateWebAcces(
	ctx context.Context,
	order *orders.RabbitMQ,
	plan *RabbitMQClusterModel,
	resp *resource.UpdateResponse,
) {
	admins, managers := prepareWebAccessGroups(ctx, plan)
	attrs := orders.UpdateWebAccessGroupsAttrs{
		Accept:   true,
		Admins:   admins,
		Managers: managers,
	}
	if strings.EqualFold(r.client.Environment, "prod") {
		attrs.ChangeRequest = true
	}

	err := order.UpdateWebAccessGroups(attrs)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("web_access"),
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "web access updating ended with error"),
			err.Error(),
		)
		return
	}
}

func (r RabbitMQClusterResource) verticalScaling(
	order *orders.RabbitMQ,
	plan *RabbitMQClusterModel,
	resp *resource.UpdateResponse,
) {
	var extraMounts []entities.RabbitMQExtraMounts
	for path, extraM := range plan.ExtraMounts {
		extraMounts = append(extraMounts, entities.RabbitMQExtraMounts{
			Mount:      path,
			Size:       extraM.Size.ValueInt64(),
			FileSystem: extraM.FileSystem.ValueString(),
		})
	}
	qourumFlavor, err := references.GetFlavorByFilter(
		r.client.Creds,
		"flavor:vm:linux:"+strings.ToLower(r.client.Environment),
		4,
		2,
	)

	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "getQourumFlavor ended with error"),
			err.Error(),
		)
		return
	}

	attrs := orders.RabbitMQVerticalScalingAttrs{
		CheckAgree:    true,
		SelectMode:    "one_vm",
		Mount:         extraMounts[0].Mount,
		Fstype:        extraMounts[0].FileSystem,
		Size:          extraMounts[0].Size,
		CurrentFlavor: strconv.Itoa(int(order.Attrs.Size)),
		Flavor: entities.Flavor{
			Cores:  plan.Flavor.Cores.ValueInt64(),
			Memory: plan.Flavor.Memory.ValueInt64(),
			Name:   plan.Flavor.Name.ValueString(),
			UUID:   plan.Flavor.UUID.ValueString(),
		},
		QuorumExtraMounts: entities.QuorumExtraMounts{
			Mount:  "/app",
			Size:   10,
			FStype: "xfs",
		},
		QuorumFlavor: entities.Flavor{
			Cores:  qourumFlavor.Data.Cores,
			Memory: qourumFlavor.Data.Memory,
			Name:   qourumFlavor.Name,
			UUID:   qourumFlavor.ID,
		},
	}

	err = order.VerticalScaling(attrs)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "vertical scaling ended with error"),
			err.Error(),
		)
		return
	}
}

// Функция для горизонтального масштабирования
func (r RabbitMQClusterResource) horizontalScaling(
	order *orders.RabbitMQ,
	plan *RabbitMQClusterModel,
	state *RabbitMQClusterModel,
	resp *resource.UpdateResponse,
) {
	netSegment := plan.Core.NetSegmentCode.ValueString()
	layoutName, err := references.GetGeoDistributionLayoutNameByID(
		r.client.Creds,
		plan.LayoutID.ValueString(),
		"rabbitmq",
		r.client.Organization,
		plan.Core.NetSegmentCode.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "error for fetch layout"),
			err.Error(),
		)
		return
	}
	currentLayoutName, err := references.GetGeoDistributionLayoutNameByID(
		r.client.Creds,
		state.LayoutID.ValueString(),
		"rabbitmq",
		r.client.Organization,
		state.Core.NetSegmentCode.ValueString(),
	)

	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "error for fetch layout"),
			err.Error(),
		)
		return
	}

	err = utils.ValidateRabbitMQCount(currentLayoutName, layoutName)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "validation failed"),
			err.Error(),
		)
		return
	}

	prefix := r.getRabbitMQLayoutsPrefix(netSegment)

	attrs := orders.RabbitMQGorizontalScalingAttrs{
		CurrentLayout: strings.Replace(currentLayoutName, prefix+":", "", 1),
		NetSegment:    netSegment,
		Layout:        layoutName,
	}

	err = order.HorizontalScaling(attrs)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "gorizontal scaling ended with error"),
			err.Error(),
		)
		return
	}

}

func (r RabbitMQClusterResource) upgradeRabbitMQVersion(
	order *orders.RabbitMQ,
	plan *RabbitMQClusterModel,
	state *RabbitMQClusterModel,
	resp *resource.UpdateResponse,
) {
	availableRabbitMQVersions := []string{"3.8.14", "3.11.26"}
	currentRabbitMQVersion := state.RabbitMQVersion.ValueString()
	TargetRabbitMQVersion := plan.RabbitMQVersion.ValueString()

	attrs := orders.RabbitMQUpgradeVersionAttrs{
		Maintance:             true,
		TargetRabbitMQVersion: TargetRabbitMQVersion,
	}
	if currentRabbitMQVersion == "3.11.26" {
		resp.Diagnostics.AddAttributeError(
			path.Root("rabbitmq_version"),
			consts.MODIFY_PLAN_FAIL,
			fmt.Sprintf("current rabbitmq version is latest, avaliable versions: %v. Downgrade version is not available.", availableRabbitMQVersions),
		)
		return
	}

	err := order.UpgradeRabbitMQVersion(attrs)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "upgrade rabbitmq version ended with error"),
			err.Error(),
		)
		return
	}
}

func (r RabbitMQClusterResource) updateReleaseVersion(
	order *orders.RabbitMQ,
	current string,
	latest string,
	resp *resource.UpdateResponse,

) {
	changedVersion, err := utils.IsVersionOlder(current, latest)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "error when comparing versions"),
			err.Error(),
		)
		return
	}

	if !changedVersion {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("build_version"),
			"Current version is latest",
			fmt.Sprintf("Current version: %v, Latest version: %v", current, latest),
		)
		return
	}

	accept := true
	err = order.UpdateReleaseVersion(accept)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "update release ended with error"),
			err.Error(),
		)
		return
	}

}

// atLeastOneManager проверяет что при первоначальном заказе кластера в конфиге указана
// хотя бы одна группа доступа в роли manager для веб UI
func atLeastOneManager(ctx context.Context, plan *RabbitMQClusterModel) bool {
	managers := []string{}
	if (plan.WebAccess != nil) && !plan.WebAccess.Managers.IsNull() {
		plan.WebAccess.Managers.ElementsAs(ctx, &managers, false)
	}
	return len(managers) >= 1
}

func isVerticalScalingNeeded(state, plan *RabbitMQClusterModel) bool {
	mountChanged := false
	for statePath, stateEM := range state.ExtraMounts {
		for planPath, planEM := range plan.ExtraMounts {
			if statePath == planPath && stateEM.Size != planEM.Size {
				mountChanged = true
			}
		}
	}
	flavorChanged := plan.Flavor != state.Flavor

	return mountChanged || flavorChanged
}

func isWebAccessChanged(plan, state *RabbitMQClusterModel) bool {

	adminsChanged := !plan.WebAccess.Admins.Equal(state.WebAccess.Admins)
	managersChanged := !plan.WebAccess.Managers.Equal(state.WebAccess.Managers)

	return adminsChanged || managersChanged
}

func prepareWebAccessGroups(ctx context.Context, plan *RabbitMQClusterModel) ([]string, []string) {

	admins := []string{}
	if !plan.WebAccess.Admins.IsNull() {
		plan.WebAccess.Admins.ElementsAs(ctx, &admins, false)
	}

	managers := []string{}
	if !plan.WebAccess.Managers.IsNull() {
		plan.WebAccess.Managers.ElementsAs(ctx, &managers, false)
	}

	return admins, managers
}

func readWebAccessGroups(
	ctx context.Context,
	clusterConfig *entities.RabbitMQItemConfig,
	diags *diag.Diagnostics,
) (basetypes.SetValue, basetypes.SetValue) {

	admins := []string{}
	managers := []string{}
	if !clusterConfig.WebAccessGroups.IsEmpty() {
		admins = clusterConfig.WebAccessGroups.Admins
		managers = clusterConfig.WebAccessGroups.Managers
	}

	adminsSetValues, diag := types.SetValueFrom(ctx, types.StringType, admins)
	diags.Append(diag...)

	if len(adminsSetValues.Elements()) == 0 {
		adminsSetValues = types.SetNull(types.StringType)
	}

	managersSetValues, diag := types.SetValueFrom(ctx, types.StringType, managers)
	diags.Append(diag...)

	return adminsSetValues, managersSetValues
}

func (r RabbitMQClusterResource) getRabbitMQLayoutsPrefix(netSegment string) string {
	prefix := "one_dc"
	switch {
	case strings.EqualFold(r.client.Environment, "prod"):
		prefix = "geo"
	case strings.EqualFold(netSegment, "b2b-hce-ts-dev-srv-app"):
		prefix = "ts"
	}
	return prefix
}

func (r RabbitMQClusterResource) validateWebAccessAdminsForSpecificEnv(
	plan RabbitMQClusterModel,
	resp *resource.ModifyPlanResponse,
) {
	if !plan.WebAccess.Admins.IsNull() && !strings.EqualFold(r.client.Environment, "dev") {
		resp.Diagnostics.AddError(
			consts.MODIFY_PLAN_FAIL,
			fmt.Sprintf("`admins` avaliable only in dev environment, current environment: %s", r.client.Environment),
		)
		return
	}
}
