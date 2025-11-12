package balancerv3

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
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
	_ resource.Resource                = BalancerV3Resource{}
	_ resource.ResourceWithImportState = BalancerV3Resource{}
	_ resource.ResourceWithModifyPlan  = BalancerV3Resource{}
)

type BalancerV3Resource struct {
	client *client.CloudClient
}

func NewBalancerV3Resource() resource.Resource {
	return &BalancerV3Resource{}
}

func (r BalancerV3Resource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_balancer_v3_cluster"
}

func (r *BalancerV3Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type BalancerV3ResourceModel struct {
	FinancialProject      types.String                              `tfsdk:"financial_project"`
	Lifetime              types.Int64                               `tfsdk:"lifetime"`
	Label                 types.String                              `tfsdk:"label"`
	OrderID               types.String                              `tfsdk:"order_id"`
	ItemID                types.String                              `tfsdk:"item_id"`
	Core                  core.CoreModel                            `tfsdk:"core"`
	Flavor                flavor.FlavorModel                        `tfsdk:"flavor"`
	Image                 common.ClusterProductImageDataSourceModel `tfsdk:"image"`
	ActiveDirectoryAccess map[string][]string                       `tfsdk:"access"`
	ExtraMounts           map[string]common.ExtraMountModel         `tfsdk:"extra_mounts"`

	Config types.Object `tfsdk:"config"`

	LayoutID     types.String `tfsdk:"layout_id"`
	SetupVersion types.String `tfsdk:"setup_version"`
	ClusterName  types.String `tfsdk:"cluster_name"`
	DNSZone      types.String `tfsdk:"dns_zone"`
}

func (r BalancerV3Resource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Управление облачным продуктом Balancer V3",
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
				MarkdownDescription: "Метка заказа",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"order_id": schema.StringAttribute{
				MarkdownDescription: "Идентификатор заказа. Становится известен после создания заказа.",
				Computed:            true,
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
				MarkdownDescription: "CPU/RAM виртуальных машин",
			},
			"image": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.ClusterProductImageSchema,
				MarkdownDescription: "Тип вычислительного экземпляра",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"access": schema.MapAttribute{
				Optional: true,
				Computed: true,
				Description: "Карта, где ключом является роль, а значением - список групп, " +
					"которые предоставляют доступ для входа в Active Directory",
				MarkdownDescription: "Карта, где ключом является роль, а значением - список групп, " +
					"которые предоставляют доступ для входа в Active Directory",
				PlanModifiers: []planmodifier.Map{
					custommodifires.EmptyAccessGroup(),
				},
				Validators: []validator.Map{
					customvalidators.EmptyAccessGroupListValidator{},
				},
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
			},
			"extra_mounts": schema.MapNestedAttribute{
				MarkdownDescription: "Дополнительные точки монтирования",
				Required:            true,
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
							MarkdownDescription: "Размер точки монтирования (в ГБ)",
							Required:            true,
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
							Description:         "Тип файловой системы",
							MarkdownDescription: "Тип файловой системы",
						},
					},
				},
			},

			"financial_project": schema.StringAttribute{
				Required:            true,
				Description:         "Источник финансирования заказа",
				MarkdownDescription: "Источник финансирования заказа",
			},

			// extra product schema
			"layout_id": schema.StringAttribute{
				Required:            true,
				Description:         "Layout ID from geo_distribution.",
				MarkdownDescription: "Layout ID from geo_distribution.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"setup_version": schema.StringAttribute{
				MarkdownDescription: "Версия balancer v3",
				Required:            true,
				PlanModifiers:       []planmodifier.String{},
			},

			"cluster_name": schema.StringAttribute{
				MarkdownDescription: "Пользовательское имя кластера",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z0-9]{6}$`),
						"may contain lowercase Latin letters, numbers. length must be equel 6",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"dns_zone": schema.StringAttribute{
				MarkdownDescription: "DNS зона",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"config": schema.SingleNestedAttribute{
				MarkdownDescription: "Схема конфигурации",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: BalancerV3ConfigScheme,
			},
		},
	}
}

func (r BalancerV3Resource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan BalancerV3ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	attrs := r.prepareAtts(&plan)

	finProj, err := sources.GetFinancialProjectByName(
		r.client.Creds,
		r.client.ProjectName,
		plan.FinancialProject.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("financial_project"), consts.CREATE_RES_FAIL, err.Error())
		return
	}

	version := utils.GetMinorVersion(plan.SetupVersion.ValueString())

	if version != "4" {
		bootDisk := entities.BootDisk{
			Size: 30,
		}
		attrs.BootDisk = &bootDisk
	}

	product_name := fmt.Sprintf(
		"%s_%s",
		orders.BALANCER_V3_PREFIX, version,
	)
	image, err := references.GetBalancerV3ImageData(
		r.client.Creds,
		plan.Image.Distribution.ValueString(),
		r.client.Organization,
		r.client.Environment,
		plan.Image.OsVersion.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("get balancer v3 image data"), consts.CREATE_RES_FAIL, err.Error())
		return
	}

	value, exists := image.Products[product_name]

	if !exists {
		resp.Diagnostics.AddAttributeError(
			path.Root("get balancer v3 image data"),
			consts.CREATE_RES_FAIL,
			fmt.Errorf("not found %s in allowed to order balancer versions", product_name).Error(),
		)
		return
	}
	order := orders.NewBalancerV3(
		r.client.Creds,
		attrs,
		value.Id,
		r.client.ProjectName,
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

	orderItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError("Error while getting item: ", err.Error())
		return
	}

	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(orderItem.ID)

	if !plan.Config.IsNull() && !plan.Config.IsUnknown() {
		resp.Diagnostics.Append(r.applyNewConfigurationBalancerV3(ctx, order, plan.Config)...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		bI, ok := orderItem.Data.Config.(entities.BalancerV3ItemData)
		if !ok {
			resp.Diagnostics.AddError(
				consts.CREATE_RES_FAIL,
				"invalid type for balancer item config",
			)
			return
		}

		tfConfig, diag := types.ObjectValueFrom(ctx, BalancerV3ConfigDataSourceModel{}.AttrTypes(), bI.Config)
		if diag.HasError() {
			return
		}
		plan.Config = tfConfig
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r BalancerV3Resource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r BalancerV3Resource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	var plan, state BalancerV3ResourceModel

	if req.Plan.Raw.IsNull() {
		return
	}

	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	}
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	validateRolesDiags := utils.ValidateAccessRolesV1(
		r.client,
		"cluster:balancer",
		plan.ActiveDirectoryAccess,
	)
	resp.Diagnostics.Append(validateRolesDiags...)

	checkOrderIsDeleted := utils.CheckOrderIsDeleted(r.client, plan.OrderID, "cluster", "balancer_v3")
	resp.Diagnostics.Append(checkOrderIsDeleted.Diagnostics...)

	utils.HandleExternalDeletionRecreate(
		ctx,
		resp,
		&plan,
		checkOrderIsDeleted.IsDeleted,
		checkOrderIsDeleted.Diagnostics,
	)

	r.validateSetupVersion(&plan, &state, resp)
	r.validateDnsZone(&plan, resp)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r BalancerV3Resource) validateDnsZone(
	plan *BalancerV3ResourceModel,
	resp *resource.ModifyPlanResponse,
) {
	dnsZone := plan.DNSZone.ValueString()
	netSegment := plan.Core.NetSegmentCode.ValueString()
	dnsZones, err := references.GetBalancerDnsZones(r.client.Creds, netSegment)
	if err != nil {
		resp.Diagnostics.AddError(
			"get balancer_v3 dns zones error", err.Error(),
		)
		return
	}
	dnsZoneIndex := utils.FindIndexInSlice(dnsZone, dnsZones)
	if dnsZoneIndex == -1 {
		resp.Diagnostics.AddError(
			"dns zone is not valid",
			fmt.Sprintf("order in net_segment:[%s] not support dns_zone:[%s]. allowed zones: %s", netSegment, dnsZone, dnsZones),
		)
		return
	}
}

func (r BalancerV3Resource) validateSetupVersion(
	plan, state *BalancerV3ResourceModel,
	resp *resource.ModifyPlanResponse,
) {
	stateVer, planVer := state.SetupVersion.ValueString(), plan.SetupVersion.ValueString()
	if stateVer != "" {
		isOlder, err := utils.IsVersionOlder(stateVer, planVer)
		if err != nil {
			resp.Diagnostics.AddError(
				"Wrong balancer format versions", err.Error(),
			)
			return
		}
		if stateVer != planVer && !isOlder {
			resp.Diagnostics.AddError(
				"setup version should be newer then exist",
				fmt.Sprintf("%s is older then %s", planVer, stateVer),
			)
			return
		}
	}

	image, err := references.GetBalancerV3ImageData(
		r.client.Creds,
		plan.Image.Distribution.ValueString(),
		r.client.Organization,
		r.client.Environment,
		plan.Image.OsVersion.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("get balancer v3 image data error", err.Error())
		return
	}

	planIndex := utils.FindIndexInSlice(planVer, image.AllowedVersions)
	if planIndex == -1 {
		resp.Diagnostics.AddError(
			"setup version not in allowed versions",
			fmt.Sprintf("plan setup version %s not in allowed:%s", planVer, image.AllowedVersions),
		)
		return
	}

	if stateVer != "" {
		stateIndex := utils.FindIndexInSlice(stateVer, image.AllowedVersions)
		if stateIndex == -1 {
			resp.Diagnostics.AddError(
				"setup version not in allowed versions",
				fmt.Sprintf("state setup version %s not in allowed:[%s]", stateVer, image.AllowedVersions),
			)
			return
		}
	}
}

func (r BalancerV3Resource) Read(
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

	order, err := orders.GetBalancerV3Order(
		r.client.Creds,
		r.client.ProjectName,
		orderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Read order from portal",
			fmt.Sprintf(
				"Can't read order with order_id equal '%s'.\nError:%s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}

	balancerItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			"Read order from portal",
			fmt.Sprintf("Can't get balancer_v3 item.\nError: %s", err.Error()),
		)
		return
	}

	balancerVmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			"Read order from portal",
			fmt.Sprintf("Can't get balancer VM item.\nError: %s", err.Error()),
		)
		return
	}

	appMount, err := order.GetExtraMount("/app")
	if err != nil {
		resp.Diagnostics.AddError(
			"Read order from portal",
			fmt.Sprintf(
				"Can't get vm extra mount '/app' from portal with order_id '%s'.\nError:%s",
				orderID, err.Error(),
			),
		)
		return
	}

	vmItem := balancerVmItems[0]
	vmConfig := vmItem.Data.Config.(entities.VMItemConfig)
	bI := balancerItem.Data.Config.(entities.BalancerV3ItemData)

	prefix := ""
	if strings.EqualFold(vmConfig.DefaultNic.NetSegment, "b2b-hce-ts-dev-srv-app") {
		prefix = "ts:"
	}

	if strings.EqualFold(r.client.Environment, "prod") {
		prefix = "geo:"
	}

	balancerLayouts := map[int]string{
		1:  fmt.Sprintf("%sbalancer-1", prefix),
		2:  fmt.Sprintf("%sbalancer-2", prefix),
		4:  fmt.Sprintf("%sbalancer-4", prefix),
		6:  fmt.Sprintf("%sbalancer-6", prefix),
		8:  fmt.Sprintf("%sbalancer-8", prefix),
		10: fmt.Sprintf("%sbalancer-8", prefix),
	}

	layoutId, err := references.GetGeoDistributionLayoutID(
		r.client.Creds,
		balancerLayouts[len(bI.ClusterMembers)],
		"balancer",
		r.client.Organization,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintln("error for fetch current layout_id", err.Error()),
		)
	}

	state := BalancerV3ResourceModel{
		OrderID:               orderID,
		ActiveDirectoryAccess: utils.ReadAccessMapVV1(vmItem.Data.ACLs),
		Label:                 types.StringValue(order.Label),
		ItemID:                types.StringValue(balancerItem.ID),
		Flavor: flavor.FlavorModel{
			Cores:  types.Int64Value(vmConfig.Flavor.Cores),
			Memory: types.Int64Value(vmConfig.Flavor.Memory),
			UUID:   types.StringValue(vmConfig.Flavor.UUID),
			Name:   types.StringValue(vmConfig.Flavor.Name),
		},
		ExtraMounts: map[string]common.ExtraMountModel{
			"/app": {
				Size:       types.Int64Value(appMount.Size),
				FileSystem: types.StringValue(appMount.FileSystem),
			},
		},
		Core: core.CoreModel{
			Platform:       types.StringValue(utils.SelectPlatform(order.Attrs.Platform)),
			Domain:         types.StringValue(vmConfig.Domain),
			NetSegmentCode: types.StringValue(vmConfig.DefaultNic.NetSegment),
			Zone:           types.StringValue(order.Attrs.AvailabilityZone),
		},
		Image: common.ClusterProductImageDataSourceModel{
			Distribution:    types.StringValue(vmConfig.Image.Os.Distribution),
			OsVersion:       types.StringValue(vmConfig.Image.Os.Version),
			OnSupport:       types.BoolValue(vmConfig.OnSupport),
			ProductID:       types.StringValue("mock"),
			ADIntegration:   types.BoolValue(vmConfig.AdIntegration),
			GeoDistribution: types.BoolValue(order.Attrs.GeoDistribution),
		},
		FinancialProject: types.StringValue(order.FinancialSource.Name),
		LayoutID:         types.StringValue(layoutId),

		SetupVersion: types.StringValue(bI.Build.SetupVersion),
		ClusterName:  types.StringValue(bI.ClusterConfig.ClusterName),
		DNSZone:      types.StringValue(bI.ClusterConfig.DNSZone),
	}

	tfConfig, diag := types.ObjectValueFrom(ctx, BalancerV3ConfigDataSourceModel{}.AttrTypes(), bI.Config)
	resp.Diagnostics.Append(diag...)
	state.Config = tfConfig

	var lifetime types.Int64
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("lifetime"), &lifetime)...)
	if !lifetime.IsNull() {
		state.Lifetime = lifetime
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r BalancerV3Resource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state BalancerV3ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetBalancerV3Order(
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
	accessChanged := !reflect.DeepEqual(plan.ActiveDirectoryAccess, state.ActiveDirectoryAccess)
	layoutChanged := plan.LayoutID != state.LayoutID
	configChanged := !plan.Config.Equal(state.Config)
	versionChanged := plan.SetupVersion != state.SetupVersion

	if versionChanged || configChanged || accessChanged || layoutChanged || flavorChanged || mountChanged {
		r.validateClusterMemebersMainStatus(order, resp)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if labelChanged {
		utils.ChangeOrderLabel(order, plan.Label.ValueString(), resp)
		resp.State.SetAttribute(ctx, path.Root("label"), plan.Label.ValueString())
	}

	if !plan.FinancialProject.Equal(state.FinancialProject) {
		order.ChangeFinancialProject(finProj.ID)
	}

	if configChanged {
		resp.Diagnostics.Append(r.applyNewConfigurationBalancerV3(ctx, order, plan.Config)...)
	}

	if mountChanged {
		resp.Diagnostics.Append(r.changeExtraMountsBalancerV3(order, &plan)...)
	}

	if accessChanged {
		resp.Diagnostics.Append(r.changeAccess(order, &state, &plan)...)
	}

	if flavorChanged {
		resp.Diagnostics.Append(r.verticalScalingBalancerV3(order, &plan)...)
	}

	// TODO BALANCER change cluster

	if versionChanged {
		resp.Diagnostics.Append(r.migrateToNewVersion(order, &plan, &state)...)
	}

	if layoutChanged {
		resp.Diagnostics.Append(r.horizontalScalingBalancerV3(order, &plan, r.client.Organization)...)
	}

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r BalancerV3Resource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state BalancerV3ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetBalancerV3Order(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Can't get order from portal: ", err.Error())
	}

	orderState, err := order.GetState()
	if err != nil {
		resp.Diagnostics.AddError(
			"Delete resource",
			fmt.Sprintf("Can't get state from portal.\nError:%s", err.Error()),
		)
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
	if err := order.DeleteTwoLayer(false); err != nil {
		resp.Diagnostics.AddError("While process delete VM: ", err.Error())
		return
	}
}

func (r BalancerV3Resource) prepareAtts(plan *BalancerV3ResourceModel) orders.BalancerV3Attrs {

	ADLogonGrants := []entities.ADLogonGrants{}
	for role, groups := range plan.ActiveDirectoryAccess {
		var groupsNames []string
		groupsNames = append(groupsNames, groups...)
		ADLogonGrants = append(ADLogonGrants, entities.ADLogonGrants{
			Role:   role,
			Groups: groupsNames,
		})
	}

	attrs := orders.BalancerV3Attrs{
		BasicAttrs: orders.BasicAttrs{
			ADLogonGrants:    ADLogonGrants,
			ADIntegration:    plan.Image.ADIntegration.ValueBool(),
			ExtraMounts:      utils.PrepareExtraMountsAttrs(plan.ExtraMounts),
			AvailabilityZone: plan.Core.Zone.ValueString(),
			Domain:           plan.Core.Domain.ValueString(),
			Flavor: entities.Flavor{
				Cores:  plan.Flavor.Cores.ValueInt64(),
				Memory: plan.Flavor.Memory.ValueInt64(),
				Name:   string(plan.Flavor.Name.ValueString()),
				UUID:   string(plan.Flavor.UUID.ValueString()),
			},
			Platform:  plan.Core.Platform.ValueString(),
			OnSupport: plan.Image.OnSupport.ValueBool(),
			OsVersion: plan.Image.OsVersion.ValueString(),
			DefaultNic: entities.DefaultNic{
				NetSegment: plan.Core.NetSegmentCode.ValueString(),
			},
			CreatedWithOpenTofu: true,
		},

		Layout:          plan.LayoutID.ValueString(),
		ClusterName:     plan.ClusterName.ValueString(),
		DNSZone:         plan.DNSZone.ValueString(),
		Password:        utils.GenerateSecurePassword(16), // secure lenght 16
		SetupVersion:    plan.SetupVersion.ValueString(),
		GeoDistribution: true,
	}
	return attrs
}

func (r BalancerV3Resource) migrateToNewVersion(order *orders.BalancerV3, plan, state *BalancerV3ResourceModel) (diags diag.Diagnostics) {
	image, err := references.GetBalancerV3ImageData(
		order.Creds,
		plan.Image.Distribution.ValueString(),
		r.client.Organization,
		r.client.Environment,
		plan.Image.OsVersion.ValueString(),
	)
	if err != nil {
		diags.AddError("get balancer v3 image data error", err.Error())
		return
	}

	stateV := state.SetupVersion.ValueString()
	planV := plan.SetupVersion.ValueString()

	stateIndex := utils.FindIndexInSlice(stateV, image.AllowedVersions)
	planIndex := utils.FindIndexInSlice(planV, image.AllowedVersions)

	for {
		stateIndex++
		if stateIndex > planIndex {
			break
		}
		versionTo := image.AllowedVersions[stateIndex]
		err = order.Migrate(versionTo)
		if err != nil {
			diags.AddError("Error while migrating to new version", err.Error())
			return
		}
	}
	return

}

func (r BalancerV3Resource) applyNewConfigurationBalancerV3(ctx context.Context, order *orders.BalancerV3, config types.Object) (diags diag.Diagnostics) {
	jsonData, diags := ConvertModelToConfig(ctx, config)
	err := order.ComplexApply(*jsonData)
	if err != nil {
		diags.AddWarning("Error while changing config", err.Error())
	}
	return diags
}

func (r BalancerV3Resource) changeExtraMountsBalancerV3(order *orders.BalancerV3, plan *BalancerV3ResourceModel) (diags diag.Diagnostics) {
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
	return
}

func (r BalancerV3Resource) changeAccess(
	order *orders.BalancerV3,
	state,
	plan *BalancerV3ResourceModel,
) (diags diag.Diagnostics) {

	var deletedRoles []entities.ADLogonGrants
	var changedRoles []entities.ADLogonGrants
	var addedRoles []entities.ADLogonGrants

	for role, groups := range state.ActiveDirectoryAccess {
		if _, ok := plan.ActiveDirectoryAccess[role]; !ok {
			deletedRoles = append(
				deletedRoles,
				entities.ADLogonGrants{Role: role, Groups: groups},
			)
		} else {
			if len(plan.ActiveDirectoryAccess[role]) == len(state.ActiveDirectoryAccess[role]) {
				if utils.DifferenceLen(plan.ActiveDirectoryAccess[role], state.ActiveDirectoryAccess[role]) {
					changedRoles = append(
						changedRoles,
						entities.ADLogonGrants{Role: role, Groups: plan.ActiveDirectoryAccess[role]},
					)
				}
			} else {
				changedRoles = append(
					changedRoles,
					entities.ADLogonGrants{Role: role, Groups: plan.ActiveDirectoryAccess[role]},
				)
			}
		}
	}

	for role, groups := range plan.ActiveDirectoryAccess {
		if _, ok := state.ActiveDirectoryAccess[role]; !ok {
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
			"cluster:balancer",
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

func (r BalancerV3Resource) verticalScalingBalancerV3(order *orders.BalancerV3, plan *BalancerV3ResourceModel) (diags diag.Diagnostics) {
	flavor := entities.Flavor{
		Cores:  plan.Flavor.Cores.ValueInt64(),
		Memory: plan.Flavor.Memory.ValueInt64(),
		UUID:   plan.Flavor.UUID.ValueString(),
		Name:   plan.Flavor.Name.ValueString(),
	}
	err := order.VerticalScaling(flavor)
	if err != nil {
		diags.AddError("Changing VM flavor", err.Error())
	}
	return
}

func (r BalancerV3Resource) horizontalScalingBalancerV3(order *orders.BalancerV3, plan *BalancerV3ResourceModel, org string) (diags diag.Diagnostics) {

	layoutName, err := references.GetGeoDistributionLayoutNameByID(
		order.Creds,
		plan.LayoutID.ValueString(),
		"balancer",
		org,
		plan.Core.NetSegmentCode.ValueString(),
	)

	if err != nil {
		diags.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "error for fetch layout"),
			err.Error(),
		)
		return
	}

	newCountNodes := strings.Split(layoutName, "-")
	if len(newCountNodes) < 2 {
		diags.AddError(
			consts.UPDATE_RES_FAIL,
			"new nodes count fetch error",
		)
		return
	}
	new_haproxy_nodes, err := strconv.ParseInt(newCountNodes[1], 10, 64)
	if err != nil {
		diags.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "error parsing the string representation of the number of new nodes into an integer"),
			err.Error(),
		)
		return
	}

	err = order.HorizontalScaling(int(new_haproxy_nodes))
	if err != nil {
		diags.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "horizontal scaling ended with error"),
			err.Error(),
		)
	}
	return
}

func (r BalancerV3Resource) validateClusterMemebersMainStatus(order *orders.BalancerV3, resp *resource.UpdateResponse) {
	mainStatus, err := order.CheckClusterItemStatusOn()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintln("CheckClusterItemStatusOn ended with error", err.Error()),
		)
		return
	}

	if !mainStatus {
		resp.Diagnostics.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintln("All cluster member main status must be on `on` status. Check order on portal."),
		)
		return
	}
}
