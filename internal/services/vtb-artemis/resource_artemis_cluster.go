package vtbartemis

import (
	"context"
	"fmt"
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
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &ArtemisClusterResource{}
	_ resource.ResourceWithImportState = &ArtemisClusterResource{}
	_ resource.ResourceWithModifyPlan  = &ArtemisClusterResource{}
)

type ArtemisClusterResource struct {
	client *client.CloudClient
}

func NewArtemisResource() resource.Resource {
	return &ArtemisClusterResource{}
}

func (r ArtemisClusterResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_artemis_cluster"
}

func (r *ArtemisClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type ArtemisClusterResourceModel struct {
	Core        core.CoreModel                            `tfsdk:"core"`
	Flavor      flavor.FlavorModel                        `tfsdk:"flavor"`
	Image       common.ClusterProductImageDataSourceModel `tfsdk:"image"`
	Lifetime    types.Int64                               `tfsdk:"lifetime"`
	Label       types.String                              `tfsdk:"label"`
	OrderID     types.String                              `tfsdk:"order_id"`
	ItemID      types.String                              `tfsdk:"item_id"`
	Access      map[string][]types.String                 `tfsdk:"access"`
	ExtraMounts map[string]common.ExtraMountModel         `tfsdk:"extra_mounts"`

	ClusterName      types.String `tfsdk:"cluster_name"`
	BuildVersion     types.String `tfsdk:"build_version"`
	LayoutID         types.String `tfsdk:"layout_id"`
	ProtocolCore     types.Bool   `tfsdk:"protocol_core"`
	ProtocolAMQP     types.Bool   `tfsdk:"protocol_amqp"`
	ArtemisVersion   types.String `tfsdk:"artemis_version"`
	FinancialProject types.String `tfsdk:"financial_project"`
	UpdateMode       types.String `tfsdk:"update_product_mode"`
	Plugins          PluginsModel `tfsdk:"plugins"`
}

type PluginsModel struct {
	UniqueID    types.Bool  `tfsdk:"unique_id"`
	Limits      types.Bool  `tfsdk:"limits"`
	PacketLimit types.Int64 `tfsdk:"packet_limit"`
	RateLimit   types.Int64 `tfsdk:"rate_limit"`
	SizeLimit   types.Int64 `tfsdk:"size_limit"`
}

func (r ArtemisClusterResource) Schema(
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
				MarkdownDescription: "Основные параметры для вирутальных машин в заказе.",
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
				Attributes:          common.ClusterProductImageSchema,
				MarkdownDescription: "Тип вычислительного экземпляра.",
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
				MarkdownDescription: "Имя заказа.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},

			"order_id": schema.StringAttribute{
				MarkdownDescription: "Идентификатор заказа, известен после создания заказа.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"item_id": schema.StringAttribute{
				MarkdownDescription: "Идентификатор сущностей ВМ, известен после создания ВМ.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"access": schema.MapAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Разрешения для входа в Active Directory.",
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
					mapplanmodifier.UseStateForUnknown(),
					custommodifires.EmptyAccessGroup(),
				},
				ElementType: types.SetType{
					ElemType: types.StringType,
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
								int64validator.AtMost(consts.CLOUD_EXTRA_MOUNT_MAX_SIZE),
								int64validator.AtLeast(30),
							},
							PlanModifiers: []planmodifier.Int64{
								custommodifires.GrowOnly(),
								custommodifires.MininalGrow(1),
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
			"build_version": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Версия релиза продукта",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			// extra product schema
			"artemis_version": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Версия ядра Artemis.",
				Description:         "Версия ядра Artemis.",
				Validators: []validator.String{
					stringvalidator.OneOf("2.19.1"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"layout_id": schema.StringAttribute{
				Required:            true,
				Description:         "Идентификатор layout из справочника geo_distribution.",
				MarkdownDescription: "Идентификатор layout из справочника geo_distribution.",
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

			"protocol_core": schema.BoolAttribute{
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				Description:         "Флаг для включения CORE протокола. Включен по умолчанию, отключение недоступно.",
				MarkdownDescription: "Флаг для включения CORE протокола. Включен по умолчанию, отключение недоступно.",
			},
			"protocol_amqp": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				Description:         "Флаг для вклюяения AMQP протокола. Выключен по умолчанию.",
				MarkdownDescription: "Флаг для вклюяения AMQP протокола. Выключен по умолчанию.",
			},
			"plugins": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"unique_id": schema.BoolAttribute{
						Computed:            true,
						Optional:            true,
						Default:             booldefault.StaticBool(false),
						Description:         "Флаг для включения UniqueID плагина. Выключен по умолчанию.",
						MarkdownDescription: "Флаг для включения UniqueID плагина. Выключен по умолчанию.",
					},
					"limits": schema.BoolAttribute{
						Computed:            true,
						Optional:            true,
						Default:             booldefault.StaticBool(false),
						Description:         "Флаг для включения протокола Limits. Выключен по умолчанию.",
						MarkdownDescription: "Флаг для включения протокола Limits. Выключен по умолчанию.",
					},
					"packet_limit": schema.Int64Attribute{
						Computed:    true,
						Optional:    true,
						Description: "Ограничение максимального пакета, ограничивает размер однопакетных сообщений.",
						Default:     int64default.StaticInt64(102400),
						Validators: []validator.Int64{
							int64validator.AtMost(1048576),
						},
					},
					"rate_limit": schema.Int64Attribute{
						Computed:    true,
						Optional:    true,
						Description: "Ограничение количества получаемых блоков за секунду",
						Default:     int64default.StaticInt64(2147483647),
						Validators: []validator.Int64{
							int64validator.AtMost(2147483647),
						},
					},
					"size_limit": schema.Int64Attribute{
						Computed:    true,
						Optional:    true,
						Description: "Ограничение размера максимального сообщения.",
						Default:     int64default.StaticInt64(209715200),
						Validators: []validator.Int64{
							int64validator.AtMost(209715200),
						},
					},
				},
				Optional: true,
				Computed: true,
				Default: objectdefault.StaticValue(
					types.ObjectValueMust(
						map[string]attr.Type{
							"unique_id":    types.BoolType,
							"limits":       types.BoolType,
							"packet_limit": types.Int64Type,
							"rate_limit":   types.Int64Type,
							"size_limit":   types.Int64Type,
						},
						map[string]attr.Value{
							"unique_id":    types.BoolValue(false),
							"limits":       types.BoolValue(false),
							"packet_limit": types.Int64Value(102400),
							"rate_limit":   types.Int64Value(2147483647),
							"size_limit":   types.Int64Value(209715200),
						},
					),
				),
			},

			"financial_project": schema.StringAttribute{
				Required:            true,
				Description:         "Источник финансирования для заказа.",
				MarkdownDescription: "Источник финансирования для заказа.",
			},
			"update_product_mode": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("none"),
				MarkdownDescription: `Мод для регулировки политики обновлений продукта("latest/none"), если флаг проставлен в "latest"
				                     то будет активировано действие "Обвновить версию релиза продукта"`,
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

func (r ArtemisClusterResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r ArtemisClusterResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan, state ArtemisClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	}

	checkIsOrderDeleted := utils.CheckOrderIsDeleted(
		r.client,
		plan.OrderID,
		"cluster",
		"vtb-artemis",
	)
	resp.Diagnostics.Append(checkIsOrderDeleted.Diagnostics...)

	utils.HandleExternalDeletionRecreate(
		ctx,
		resp,
		&plan,
		checkIsOrderDeleted.IsDeleted,
		checkIsOrderDeleted.Diagnostics,
	)

	valiadteAccessForEnvDiags := utils.ValidateActiveDirectoryAccessForEnvironmentsV1(
		r.client,
		plan.Access,
		"DEV",
	)
	resp.Diagnostics.Append(valiadteAccessForEnvDiags...)

	if plan.Plugins.Limits.Equal(types.BoolValue(false)) && state.Plugins.Limits.Equal(types.BoolValue(false)) {
		plan.Plugins.PacketLimit = types.Int64Value(0)
		plan.Plugins.RateLimit = types.Int64Value(0)
		plan.Plugins.SizeLimit = types.Int64Value(0)
	}
	if plan.Plugins.Limits.Equal(types.BoolValue(true)) {
		if plan.Plugins.RateLimit.ValueInt64() < 1 {
			resp.Diagnostics.AddError(
				consts.MODIFY_PLAN_FAIL,
				fmt.Sprintf("Validation error, rate_limit must be at least 1, current rate_limit: %v",
					plan.Plugins.RateLimit.ValueInt64()),
			)
		}

		if plan.Plugins.SizeLimit.ValueInt64() < 1 {
			resp.Diagnostics.AddError(
				consts.MODIFY_PLAN_FAIL,
				fmt.Sprintf("Validation error, size_limit must be at least 1, current size_limit: %v",
					plan.Plugins.SizeLimit.ValueInt64()),
			)
		}

		if plan.Plugins.PacketLimit.ValueInt64() < 102400 {
			resp.Diagnostics.AddError(
				consts.MODIFY_PLAN_FAIL,
				fmt.Sprintf("Validation error, packet_limit must be at least 102400, current packet_limit: %v",
					plan.Plugins.PacketLimit.ValueInt64()),
			)
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)

}

func (r ArtemisClusterResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan ArtemisClusterResourceModel
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

	extraMount := plan.ExtraMounts["/app"]
	attrs := orders.ArtemisClusterAttrs{
		Path:       "/app",
		FileSystem: "xfs",
		Size:       extraMount.Size.ValueInt64(),
		Domain:     plan.Core.Domain.ValueString(),
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
		ADIntegration:       plan.Image.ADIntegration.ValueBool(),
		ADLogonGrants:       utils.PrepareADLogonGrants(plan.Access),
		AMQP:                plan.ProtocolAMQP.ValueBool(),
		CORE:                plan.ProtocolCore.ValueBool(),
		AvailabilityZone:    plan.Core.Zone.ValueString(),
		ArtemisVersion:      plan.ArtemisVersion.ValueString(),
		GeoDistribution:     plan.Image.GeoDistribution.ValueBool(),
		LayoutID:            plan.LayoutID.ValueString(),
		ClusterName:         plan.ClusterName.ValueString(),
		TypeLoadBalancer:    "ON_DEMAND",
		TitleProtocols:      map[string]interface{}{},
		TitleForMountPoint:  map[string]interface{}{},
		CreatedWithOpenTofu: true,
	}

	order := orders.NewArtemisOrder(
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

	artemisItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.CREATE_RES_FAIL, "can't get paren item ID"),
			err.Error(),
		)
		return
	}

	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(artemisItem.ID)
	plan.BuildVersion = types.StringValue(artemisItem.Data.Build.SetupVersion)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r ArtemisClusterResource) Read(
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

	order, err := orders.GetArtemisOrder(
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

	clusterItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			"Read order from portal",
			fmt.Sprintf(
				"Can't get cluster item from portal with order_id '%s'.\nError:%s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}
	clusterConfig := clusterItem.Data.Config.(entities.VTBArtemisItemConfig)

	vmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			"Read order from portal",
			fmt.Sprintf("Can't get vtb-artemis `vm` items.\nError: %s", err.Error()),
		)
		return
	}
	vmACLs := vmItems[0].Data.ACLs
	vmConfig := vmItems[0].Data.Config.(entities.VMItemConfig)
	clusterName := strings.SplitN(clusterConfig.ClusterName, "-", 2)
	hostsCount := clusterConfig.HostsInfo.Count

	prefix := "one_dc"
	if strings.EqualFold(r.client.Environment, "prod") {
		prefix = "geo"
	}

	if strings.EqualFold(vmConfig.DefaultNic.NetSegment, "b2b-hce-ts-dev-srv-app") {
		prefix = "ts"
	}

	artemisLayouts := map[int]string{
		2:  fmt.Sprintf("%s:artemis-1:artemis-1", prefix),
		4:  fmt.Sprintf("%s:artemis-2:artemis-2", prefix),
		6:  fmt.Sprintf("%s:artemis-3:artemis-3", prefix),
		8:  fmt.Sprintf("%s:artemis-4:artemis-4", prefix),
		10: fmt.Sprintf("%s:artemis-5:artemis-5", prefix),
	}

	layoutId, err := references.GetGeoDistributionLayoutID(
		r.client.Creds,
		artemisLayouts[int(hostsCount)],
		"artemis",
		r.client.Organization,
		vmConfig.DefaultNic.NetSegment,
	)
	if err != nil {
		resp.Diagnostics.AddError(consts.READ_RES_FAIL, err.Error())
	}

	state := ArtemisClusterResourceModel{
		Access:           utils.ReadAccessMapV2(vmACLs),
		Label:            types.StringValue(order.Label),
		LayoutID:         types.StringValue(layoutId),
		ItemID:           types.StringValue(clusterItem.ID),
		ArtemisVersion:   types.StringValue(clusterItem.Data.Build.ArtemisVersion.(string)),
		BuildVersion:     types.StringValue(clusterItem.Data.Build.SetupVersion),
		ClusterName:      types.StringValue(clusterName[1]),
		FinancialProject: types.StringValue(order.FinancialSource.Name),
		OrderID:          orderID,
		Flavor: flavor.FlavorModel{
			Cores:  types.Int64Value(vmConfig.Flavor.Cores),
			Memory: types.Int64Value(vmConfig.Flavor.Memory),
			Name:   types.StringValue(vmConfig.Flavor.Name),
			UUID:   types.StringValue(vmConfig.Flavor.UUID),
		},
		Image: common.ClusterProductImageDataSourceModel{
			Distribution:    types.StringValue(vmConfig.Image.Os.Distribution),
			OsVersion:       types.StringValue(vmConfig.Image.Os.Version),
			OnSupport:       types.BoolValue(vmConfig.OnSupport),
			ProductID:       types.StringValue(order.ProductID),
			ADIntegration:   types.BoolValue(vmConfig.AdIntegration),
			GeoDistribution: types.BoolValue(order.Attrs.GeoDistribution),
		},
		Core: core.CoreModel{
			Platform:       types.StringValue(utils.SelectPlatform(order.Attrs.Platform)),
			Domain:         types.StringValue(vmConfig.Domain),
			NetSegmentCode: types.StringValue(vmConfig.DefaultNic.NetSegment),
			Zone:           types.StringValue(order.Attrs.AvailabilityZone),
		},
		ExtraMounts: map[string]common.ExtraMountModel{
			"/app": {
				Size:       types.Int64Value(int64(vmConfig.ExtraMounts[0].Size)),
				FileSystem: types.StringValue(vmConfig.ExtraMounts[0].Fstype),
			},
		},
		ProtocolCore: types.BoolValue(clusterConfig.Protocols.CORE),
		ProtocolAMQP: types.BoolValue(clusterConfig.Protocols.AMQP),
		UpdateMode:   types.StringValue("none"),
		Plugins: PluginsModel{
			Limits:      types.BoolValue(clusterConfig.Plugins.Limits.Status),
			RateLimit:   types.Int64Value(clusterConfig.Plugins.Limits.RateLimit),
			SizeLimit:   types.Int64Value(clusterConfig.Plugins.Limits.SizeLimit),
			PacketLimit: types.Int64Value(clusterConfig.Plugins.Limits.PacketLimit),
			UniqueID:    types.BoolValue(clusterConfig.Plugins.UniqueID.Status),
		},
	}

	var lifetime types.Int64
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("lifetime"), &lifetime)...)
	if !lifetime.IsNull() {
		state.Lifetime = lifetime
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r ArtemisClusterResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state ArtemisClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	artemis, err := orders.GetArtemisOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(consts.READ_RES_FAIL, err.Error())
		return
	}

	artemisImageData, err := references.GetArtemisImageData(
		r.client.Creds,
		state.Image.Distribution.ValueString(),
		r.client.Organization,
		r.client.Environment,
		state.Image.OsVersion.ValueString(),
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
	amqpFlagChanged := plan.ProtocolAMQP != state.ProtocolAMQP
	flavorChanged := plan.Flavor != state.Flavor
	mountChanged := utils.IsExtraMountChanged(state.ExtraMounts, plan.ExtraMounts)
	layoutChanged := plan.LayoutID != state.LayoutID
	currentReleaseVersion := state.BuildVersion.ValueString()
	latestReleaseVersion := artemisImageData.LatestReleaseVersion
	pluginsChanged := plan.Plugins != state.Plugins

	if labelChanged {
		utils.ChangeOrderLabel(artemis, plan.Label.ValueString(), resp)
	}

	if amqpFlagChanged {
		r.updateProtocols(artemis, &plan, resp)
	}

	if flavorChanged {
		r.updateFlavor(artemis, &plan, &state, resp)
	}

	if mountChanged {
		r.updateExtraMount(artemis, &plan, resp)
	}

	if layoutChanged {
		r.horizontalScaling(artemis, &plan, &state, resp)
	}

	if !plan.FinancialProject.Equal(state.FinancialProject) {
		artemis.ChangeFinancialProject(finProj.ID)
	}

	if !plan.UpdateMode.IsNull() && plan.UpdateMode.ValueString() == "latest" {
		r.updateReleaseVersion(artemis, currentReleaseVersion, latestReleaseVersion, resp)
	}

	if pluginsChanged {
		r.updatePlugins(artemis, &plan, resp)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r ArtemisClusterResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state ArtemisClusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetArtemisOrder(
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

	if err := order.Delete(); err != nil {
		resp.Diagnostics.AddError(consts.DELETE_RES_FAIL, err.Error())
		return
	}
}

// * Custom logic *
func (r ArtemisClusterResource) updateFlavor(
	order *orders.ArtemisOrder,
	plan *ArtemisClusterResourceModel,
	state *ArtemisClusterResourceModel,
	resp *resource.UpdateResponse,
) {
	attrs := orders.VTBArtemisVerticalScalingAttrs{
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

	err := order.VerticalScaling(attrs)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("flavor"),
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "vertical scaling ended with error"),
			err.Error(),
		)
	}
}

func (r ArtemisClusterResource) updateExtraMount(
	artemis *orders.ArtemisOrder,
	plan *ArtemisClusterResourceModel,
	resp *resource.UpdateResponse,
) {
	for path, planExtraMount := range plan.ExtraMounts {
		toExpand := entities.ExtraMount{
			Path:       path,
			Size:       planExtraMount.Size.ValueInt64(),
			FileSystem: planExtraMount.FileSystem.ValueString(),
		}

		isProd := false
		if strings.ToLower(r.client.Environment) == "prod" {
			isProd = true
		}

		err := artemis.ExpandMountPoint(isProd, toExpand, false)
		if err != nil {
			resp.Diagnostics.AddError("Incorrent changes in extra_mounts parameters", err.Error())
		}
	}
}

func (r ArtemisClusterResource) updateProtocols(
	artemis *orders.ArtemisOrder,
	plan *ArtemisClusterResourceModel,
	resp *resource.UpdateResponse,
) {
	err := artemis.SwitchProtocol(
		plan.ProtocolAMQP.ValueBool(),
		plan.ProtocolCore.ValueBool(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("protocol_amqp"), consts.UPDATE_RES_FAIL, err.Error())
	}
}

func (r ArtemisClusterResource) updatePlugins(
	artemis *orders.ArtemisOrder,
	plan *ArtemisClusterResourceModel,
	resp *resource.UpdateResponse,
) {
	attrs := orders.SwitchPluginAttrs{
		Limits:      plan.Plugins.Limits.ValueBool(),
		UniqueID:    plan.Plugins.UniqueID.ValueBool(),
		SizeLimit:   plan.Plugins.SizeLimit.ValueInt64(),
		RateLimit:   plan.Plugins.RateLimit.ValueInt64(),
		PacketLimit: plan.Plugins.PacketLimit.ValueInt64(),
		ReadIt:      true,
		BlockSize:   4096,
	}
	err := artemis.SwitchPlugins(attrs)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "error for fetch layout_name"),
			err.Error(),
		)
	}
}

func (r ArtemisClusterResource) horizontalScaling(
	artemis *orders.ArtemisOrder,
	plan, state *ArtemisClusterResourceModel,
	resp *resource.UpdateResponse,
) {
	newLayoutName, err := references.GetGeoDistributionLayoutNameByID(
		r.client.Creds,
		plan.LayoutID.ValueString(),
		"artemis",
		r.client.Organization,
		plan.Core.NetSegmentCode.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "error for fetch layout_name"),
			err.Error(),
		)
	}
	currentLayoutName, err := references.GetGeoDistributionLayoutNameByID(
		r.client.Creds,
		state.LayoutID.ValueString(),
		"artemis",
		r.client.Organization,
		state.Core.NetSegmentCode.ValueString(),
	)

	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "error for fetch layout_name"),
			err.Error(),
		)
	}
	quantity, err := compareVmCount(currentLayoutName, newLayoutName, r.client.Environment, plan.Core.NetSegmentCode.ValueString())

	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "error for comparing vm quantity"),
			err.Error(),
		)
		return
	}
	err = artemis.GorizontalScaling(int64(quantity))
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "horizontal_scaling ended with error"),
			err.Error(),
		)
	}

}

func (r ArtemisClusterResource) updateReleaseVersion(
	order *orders.ArtemisOrder,
	currentVersion string,
	latestVersion string,
	resp *resource.UpdateResponse,
) {
	changedVersion, err := utils.IsVersionOlder(currentVersion, latestVersion)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "error while comparing release versions"),
			err.Error(),
		)
	}

	if !changedVersion {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("build_version"),
			"Current version is latest",
			fmt.Sprintf("Current release version: %v, Latest release version: %v", currentVersion, latestVersion),
		)
		return
	}

	read_it := true
	err = order.UpdateReleaseVersion(read_it)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "update release version ended with error"),
			err.Error(),
		)
	}
}

func (r ArtemisClusterResource) updateArtemisVersion(
	order *orders.ArtemisOrder,
	plan *ArtemisClusterResourceModel,
	resp *resource.UpdateResponse,
) {

	err := order.SwitchArtemisVersion(plan.ArtemisVersion.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "upgrade artemis version ended with error"),
			err.Error(),
		)
	}
}

func compareVmCount(currentLayoutName, newLayoutName, env, netSegment string) (int, error) {
	prefix := "one_dc"
	if strings.EqualFold(env, "prod") {
		prefix = "geo"
	}

	if strings.EqualFold(netSegment, "b2b-hce-ts-dev-srv-app") {
		prefix = "ts"
	}
	artemisLayouts := map[string]int{
		fmt.Sprintf("%s:artemis-1:artemis-1", prefix): 2,
		fmt.Sprintf("%s:artemis-2:artemis-2", prefix): 4,
		fmt.Sprintf("%s:artemis-3:artemis-3", prefix): 6,
		fmt.Sprintf("%s:artemis-4:artemis-4", prefix): 8,
		fmt.Sprintf("%s:artemis-5:artemis-5", prefix): 10,
	}
	currentVmCount := artemisLayouts[currentLayoutName]
	newVmCount := artemisLayouts[newLayoutName]

	if currentVmCount > newVmCount {
		return 0, fmt.Errorf("new_layout must be greather current_layout, current_layout: %s, new_layout: %s", currentLayoutName, newLayoutName)
	}
	quantity := newVmCount - currentVmCount
	return quantity, nil
}
