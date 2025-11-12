package k8scluster

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/common"
	"terraform-provider-vtb/internal/consts"
	"terraform-provider-vtb/internal/custommodifires"
	"terraform-provider-vtb/internal/customvalidators"
	"terraform-provider-vtb/internal/services/flavor"
	"terraform-provider-vtb/internal/utils"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"

	"terraform-provider-vtb/pkg/client/references"
	"terraform-provider-vtb/pkg/client/sources"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	// "github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                = &K8sClusterResource{}
	_ resource.ResourceWithModifyPlan  = &K8sClusterResource{}
	_ resource.ResourceWithImportState = &K8sClusterResource{}
)

type K8sClusterResource struct {
	client *client.CloudClient
}

func NewK8sClusterResource() resource.Resource {
	return &K8sClusterResource{}
}

func (r K8sClusterResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_k8s_cluster"
}

func (r *K8sClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type K8sClusterModel struct {
	OrderID          types.String `tfsdk:"order_id"`
	ItemID           types.String `tfsdk:"item_id"`
	FinancialProject types.String `tfsdk:"financial_project"`
	Lifetime         types.Int64  `tfsdk:"lifetime"`
	Label            types.String `tfsdk:"label"`

	Name             types.String           `tfsdk:"name"`
	Domain           types.String           `tfsdk:"domain"`
	Version          K8sClusterVersionModel `tfsdk:"version"`
	Platform         types.String           `tfsdk:"platform"`
	DataCenter       types.String           `tfsdk:"data_center"`
	NetSegment       types.String           `tfsdk:"net_segment"`
	ControlPanelSize types.String           `tfsdk:"control_panel_size"`
	BalancerDnsZone  types.String           `tfsdk:"balancer_dns_zone"`

	ContainerCPURatio    types.Int64 `tfsdk:"container_cpu_ratio"`
	ContainerMemoryRatio types.Int64 `tfsdk:"container_memory_ratio"`

	Regions      []K8sClusterRegionModel       `tfsdk:"regions"`
	Ingress      []K8sClusterIngressModel      `tfsdk:"ingress"`
	CniPlugin    K8sClusterCniPluginModel      `tfsdk:"cni_plugin"`
	Components   *K8sClusterComponentsModel    `tfsdk:"components"`
	ControlPlane []K8sClusterControlPlaneModel `tfsdk:"control_plane"`
	Products     []types.String                `tfsdk:"products"`
	Visibility   types.Bool                    `tfsdk:"visibility"`
	GslbOnly     types.Bool                    `tfsdk:"gslb_only"`
}

type K8sClusterVersionModel struct {
	K8sVersion     types.String `tfsdk:"k8s_version"`
	ProductVersion types.String `tfsdk:"product_version"`
}

type K8sClusterRegionModel struct {
	// VMs                  types.List                       `tfsdk:"vms"`
	Name                 types.String                     `tfsdk:"name"`
	Size                 types.Int64                      `tfsdk:"size"`
	Flavor               flavor.FlavorModel               `tfsdk:"flavor"`
	IsCodes              types.List                       `tfsdk:"iscodes"`
	ContainerCPURatio    types.Int64                      `tfsdk:"container_cpu_ratio"`
	ContainerMemoryRatio types.Int64                      `tfsdk:"container_memory_ratio"`
	PodCPUMax            types.Int64                      `tfsdk:"pod_cpu_max"`
	PodMemoryMax         types.Int64                      `tfsdk:"pod_memory_max"`
	Components           *K8sClusterRegionComponentsModel `tfsdk:"components"`
}

type K8sClusterRegionComponentsModel struct {
	Astrom       types.Bool `tfsdk:"astrom"`
	TsamOperator types.Bool `tfsdk:"tsam_operator"`
	ChaosMesh    types.Bool `tfsdk:"chaos_mesh"`
	TsdsOperator types.Bool `tfsdk:"tsds_operator"`
	TslgOperator types.Bool `tfsdk:"tslg_operator"`
	Tyk          types.Bool `tfsdk:"tyk"`
}

type K8sClusterIngressModel struct {
	Name           types.String                    `tfsdk:"name"`
	Size           types.Int64                     `tfsdk:"size"`
	Flavor         flavor.FlavorModel              `tfsdk:"flavor"`
	IsCodes        types.List                      `tfsdk:"iscodes"`
	Features       *K8sClusterIngressFeaturesModel `tfsdk:"features"`
	TcpUdpSettings []K8sClusterIngressTcpUdpModel  `tfsdk:"tcp_udp_settings"`
}

type K8sClusterIngressFeaturesModel struct {
	Http2Protocol types.Bool `tfsdk:"http2_protocol"`
	ProxyProtocol types.Bool `tfsdk:"proxy_protocol"`
}

type K8sClusterIngressTcpUdpModel struct {
	Mode    types.String `tfsdk:"mode"`
	Port    types.Int64  `tfsdk:"port"`
	Service types.String `tfsdk:"service"`
}

type K8sClusterCniPluginModel struct {
	Name types.String `tfsdk:"name"`
}

type K8sClusterControlPlaneModel struct {
	Size   types.Int64        `tfsdk:"size"`
	Flavor flavor.FlavorModel `tfsdk:"flavor"`
	Role   types.String       `tfsdk:"role"`
}

type RegionModel struct {
	// VMs                  types.List                       `tfsdk:"vms"`
	Name                 types.String                     `tfsdk:"name"`
	Size                 types.Int64                      `tfsdk:"size"`
	Flavor               types.Object                     `tfsdk:"flavor"`
	IsCodes              types.List                       `tfsdk:"iscodes"`
	ContainerCPURatio    types.Int64                      `tfsdk:"container_cpu_ratio"`
	ContainerMemoryRatio types.Int64                      `tfsdk:"container_memory_ratio"`
	PodCPUMax            types.Int64                      `tfsdk:"pod_cpu_max"`
	PodMemoryMax         types.Int64                      `tfsdk:"pod_memory_max"`
	Components           *K8sClusterRegionComponentsModel `tfsdk:"components"`
}

func (t K8sClusterResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "VTB.Cloud ресурс Kubernetes cluster",
		Attributes: map[string]schema.Attribute{
			"order_id": schema.StringAttribute{
				MarkdownDescription: "Идентификатор заказа. Становится известен после создания заказа",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"item_id": schema.StringAttribute{
				MarkdownDescription: "Идентификатор кластера в заказе. Становится известен после создания заказа",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"financial_project": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Источник финансирования заказа",
			},
			"lifetime": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Время жизни заказа",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
			},
			"label": schema.StringAttribute{
				MarkdownDescription: "Метка заказа",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 64),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Имя кластера Kubernetes. Становится известно после создания заказа",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "Домен",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.SingleNestedAttribute{
				MarkdownDescription: "Версия",
				Required:            true,
				Attributes:          K8sClusterVersionSchema,
				// PlanModifiers: []planmodifier.Object{
				// 	objectplanmodifier.RequiresReplace(),
				// },
			},
			"platform": schema.StringAttribute{
				MarkdownDescription: "Платформа",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					custommodifires.IgnoreChangesModifier{},
				},
			},
			"data_center": schema.StringAttribute{
				MarkdownDescription: "Дата-центр размещения заказа",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"net_segment": schema.StringAttribute{
				MarkdownDescription: "Сетевой сегмент размещения заказа",
				Required:            true,
			},
			"control_panel_size": schema.StringAttribute{
				MarkdownDescription: "Размер контрольной панели",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("tiny", "small", "medium", "large"),
				},
				PlanModifiers: []planmodifier.String{
					custommodifires.IgnoreChangesModifier{},
				},
			},
			"balancer_dns_zone": schema.StringAttribute{
				MarkdownDescription: "Доменная зона системного балансировщика",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					custommodifires.IgnoreChangesModifier{},
				},
			},
			"container_cpu_ratio": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Отношение лимитов к реквестам по цпу для контейнера",
			},
			"container_memory_ratio": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Отношение лимитов к реквестам по памяти для контейнера",
			},
			"regions": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "Регион рабочих узлов",
				NestedObject: schema.NestedAttributeObject{
					Attributes: K8sClusterRegionsSchema,
				},
			},
			"ingress": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "Ingress shard",
				NestedObject: schema.NestedAttributeObject{
					Attributes: K8sClusterIngressSchema,
				},
			},
			"cni_plugin": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Сетевой плагин",
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Название сетевого плагина",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							custommodifires.IgnoreChangesModifier{},
						},
					},
				},
			},
			"components": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Компоненты в кластере",
				Attributes:          K8sClusterComponentsSchema,
			},
			"control_plane": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "Контрольная панель",
				NestedObject: schema.NestedAttributeObject{
					Attributes: K8sClusterControlPlaneSchema,
				},
			},
			"products": schema.SetAttribute{
				Optional:            true,
				MarkdownDescription: "Список продуктов, для которых разрешено развертывание.",
				ElementType:         types.StringType,
			},
			"visibility": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Флаг для снятия ограничения видимости",
				PlanModifiers: []planmodifier.Bool{
					custommodifires.IgnoreChangesBoolModifier{},
				},
			},
			"gslb_only": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Балансировка gslb only",
			},
		},
	}
}

var K8sClusterVersionSchema = map[string]schema.Attribute{
	"k8s_version": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Версия k8s",
	},
	"product_version": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Версия продукта",
	},
}

var K8sClusterRegionsSchema = map[string]schema.Attribute{
	// "vms": schema.ListAttribute{
	// 	Optional:            true,
	// 	Computed:            true,
	// 	MarkdownDescription: "Список узлов региона",
	// 	ElementType:         types.StringType,
	// 	PlanModifiers: []planmodifier.List{
	// 		listplanmodifier.UseStateForUnknown(),
	// 	},
	// },
	"size": schema.Int64Attribute{
		Required:            true,
		MarkdownDescription: "Количество узлов",
	},
	"flavor": schema.SingleNestedAttribute{
		Required:            true,
		MarkdownDescription: "Конфигурация Core/RAM",
		Attributes:          common.FlavorSchema,
	},
	"iscodes": schema.ListAttribute{
		Required:            true,
		MarkdownDescription: "Коды ИС",
		ElementType:         types.StringType,
	},
	"name": schema.StringAttribute{
		MarkdownDescription: "Название. Становится известно после создания заказа",
		Computed:            true,
		Optional:            true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	},
	"container_cpu_ratio": schema.Int64Attribute{
		Optional:            true,
		Computed:            true,
		MarkdownDescription: "Отношение лимитов к реквестам по цпу для контейнера",
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.UseStateForUnknown(),
		},
	},
	"container_memory_ratio": schema.Int64Attribute{
		Optional:            true,
		Computed:            true,
		MarkdownDescription: "Отношение лимитов к реквестам по памяти для контейнера",
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.UseStateForUnknown(),
		},
	},
	"pod_cpu_max": schema.Int64Attribute{
		Optional:            true,
		Computed:            true,
		MarkdownDescription: "Максимальный размер пода по цпу",
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.UseStateForUnknown(),
		},
	},
	"pod_memory_max": schema.Int64Attribute{
		Optional:            true,
		Computed:            true,
		MarkdownDescription: "Максимальный размер пода по памяти",
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.UseStateForUnknown(),
		},
	},
	"components": schema.SingleNestedAttribute{
		Optional:            true,
		MarkdownDescription: "Компоненты в регионе",
		Attributes:          K8sClusterRegionComponentsSchema,
	},
}

var K8sClusterRegionComponentsSchema = map[string]schema.Attribute{
	"astrom": schema.BoolAttribute{
		Optional:            true,
		MarkdownDescription: "Флаг для включения Ключ-Астром",
	},
	"tsam_operator": schema.BoolAttribute{
		Optional:            true,
		MarkdownDescription: "Флаг для включения TSAM",
	},
	"chaos_mesh": schema.BoolAttribute{
		Optional:            true,
		MarkdownDescription: "Флаг для включения Chaos Mesh",
	},
	"tsds_operator": schema.BoolAttribute{
		Optional:            true,
		MarkdownDescription: "Флаг для включения TSDS",
	},
	"tslg_operator": schema.BoolAttribute{
		Optional:            true,
		MarkdownDescription: "Флаг для включения TSLG",
	},
	"tyk": schema.BoolAttribute{
		Optional:            true,
		MarkdownDescription: "Флаг для включения TYK",
	},
}

var K8sClusterIngressSchema = map[string]schema.Attribute{
	"size": schema.Int64Attribute{
		Required:            true,
		MarkdownDescription: "Количество узлов",
	},
	"flavor": schema.SingleNestedAttribute{
		Required:            true,
		MarkdownDescription: "Конфигурация Core/RAM",
		Attributes:          common.FlavorSchema,
	},
	"iscodes": schema.ListAttribute{
		Required:            true,
		MarkdownDescription: "Код ИС",
		ElementType:         types.StringType,
	},
	"name": schema.StringAttribute{
		MarkdownDescription: "Название. Становится известно после создания заказа",
		Computed:            true,
		Optional:            true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	},
	"tcp_udp_settings": schema.ListNestedAttribute{
		Optional:            true,
		MarkdownDescription: "Проброс tcp/udp портов",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"mode": schema.StringAttribute{
					MarkdownDescription: "Протокол",
					Optional:            true,
					Validators: []validator.String{
						stringvalidator.OneOf("tcp", "udp"),
					},
				},
				"port": schema.Int64Attribute{
					MarkdownDescription: "Порт",
					Optional:            true,
					Validators: []validator.Int64{
						int64validator.Between(30000, 30010),
					},
				},
				"service": schema.StringAttribute{
					MarkdownDescription: "Сервис",
					Optional:            true,
					Validators: []validator.String{
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^[a-z][a-z0-9-]*[a-z0-9]/[a-z0-9][a-z0-9-]*[a-z0-9]:((6553[0-5])|(655[0-2][0-9])|(65[0-4][0-9]{2})|(6[0-4][0-9]{3})|([1-5][0-9]{4})|([0-9]{1,4}))$`),
							"must match <namespace>/<service>:<port> format and not start with cluster-, kube- and default. Port range from 0 to 65535",
						),
						customvalidators.NamespaceNotReservedPrefixValidator{},
					},
				},
			},
		},
	},
	"features": schema.SingleNestedAttribute{
		Required:            true,
		MarkdownDescription: "Настройки Ingress контроллера",
		Attributes:          K8sClusterIngressFeaturesSchema,
		PlanModifiers: []planmodifier.Object{
			objectplanmodifier.UseStateForUnknown(),
		},
	},
}

var K8sClusterIngressFeaturesSchema = map[string]schema.Attribute{
	"http2_protocol": schema.BoolAttribute{
		Required: true,
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.UseStateForUnknown(),
		},
		MarkdownDescription: "Флаг для включения HTTP2 протокола",
	},
	"proxy_protocol": schema.BoolAttribute{
		Required: true,
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.UseStateForUnknown(),
		},
		MarkdownDescription: "Флаг для включения PROXY протокола",
	},
}

var K8sClusterControlPlaneSchema = map[string]schema.Attribute{
	"size": schema.Int64Attribute{
		Optional:            true,
		MarkdownDescription: "Количество ВМ",
		Validators: []validator.Int64{
			int64validator.Between(2, 10),
		},
	},
	"flavor": schema.SingleNestedAttribute{
		Required:            true,
		MarkdownDescription: "Конфигурация ВМ, Core/RAM",
		Attributes:          common.FlavorSchema,
	},
	"role": schema.StringAttribute{
		MarkdownDescription: "Название",
		Required:            true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	},
}

func (r K8sClusterResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {

	var plan K8sClusterModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

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

	version := utils.GetMinorVersion(plan.Version.ProductVersion.ValueString())

	product_name := fmt.Sprintf(
		"%s_%s_cluster",
		orders.KUBERNETES_V1_PREFIX, version,
	)

	image, err := references.GetK8sClusterImageData(
		r.client.Creds,
		r.client.Organization,
		r.client.Environment,
	)

	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("version"), consts.CREATE_RES_FAIL, err.Error())
		return
	}

	value, exists := image.Products[product_name]

	if !exists {
		resp.Diagnostics.AddAttributeError(
			path.Root("version"),
			consts.CREATE_RES_FAIL,
			fmt.Errorf("not found %s in allowed to order k8s cluster versions", product_name).Error(),
		)
		return
	}

	var ingressIsCodesList []string
	diags = plan.Ingress[0].IsCodes.ElementsAs(ctx, &ingressIsCodesList, false)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	var regionsIsCodesList []string
	diags = plan.Regions[0].IsCodes.ElementsAs(ctx, &regionsIsCodesList, false)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	attrsFeatures, featuresDiags := ConvertModelToFeatures(ctx, plan)
	resp.Diagnostics.Append(featuresDiags...)

	if resp.Diagnostics.HasError() {
		return
	}

	k8sClusterAttrs := orders.K8sClusterAttrs{
		DataCenter:       plan.DataCenter.ValueString(),
		Platform:         plan.Platform.ValueString(),
		Domain:           plan.Domain.ValueString(),
		BalancerDnsZone:  plan.BalancerDnsZone.ValueString(),
		ControlPanelSize: plan.ControlPanelSize.ValueString(),
		DefaultNic: entities.DefaultNic{
			NetSegment: plan.NetSegment.ValueString(),
		},
		Version: entities.VersionK8sCluster{
			K8sVersion:     plan.Version.K8sVersion.ValueString(),
			ProductVersion: plan.Version.ProductVersion.ValueString(),
		},
		Worker: entities.WorkerIngressK8sCluster{
			Size: image.DefaultRegionSize,
			Flavor: entities.K8sConfigNodeFlavor{
				Cpus:   plan.Regions[0].Flavor.Cores.ValueInt64(),
				Memory: plan.Regions[0].Flavor.Memory.ValueInt64(),
				Name:   plan.Regions[0].Flavor.Name.ValueString(),
				UUID:   plan.Regions[0].Flavor.UUID.ValueString(),
			},
			IsCodes: regionsIsCodesList,
		},
		Ingress: entities.WorkerIngressK8sCluster{
			Size: image.DefaultIngressSize,
			Flavor: entities.K8sConfigNodeFlavor{
				Cpus:   plan.Ingress[0].Flavor.Cores.ValueInt64(),
				Memory: plan.Ingress[0].Flavor.Memory.ValueInt64(),
				Name:   plan.Ingress[0].Flavor.Name.ValueString(),
				UUID:   plan.Ingress[0].Flavor.UUID.ValueString(),
			},
			IsCodes: ingressIsCodesList,
		},
		Features:            attrsFeatures,
		CreatedWithOpenTofu: true,
		GslbOnly:            plan.GslbOnly.ValueBool(),
	}

	order := orders.NewK8sClusterOrder(
		r.client.Creds,
		r.client.ProjectName,
		value.ProductID,
		k8sClusterAttrs,
	)

	err = order.Create(orders.CreateOrderPayload{
		Label:        plan.Label.ValueString(),
		Lifetime:     int(plan.Lifetime.ValueInt64()),
		FinProjectID: finProj.ID,
	})

	if err != nil {
		resp.Diagnostics.AddError("Can't create k8s cluster ", err.Error())
		return
	}

	err = order.Sync()
	if err != nil {
		resp.Diagnostics.AddError("Can't create k8s cluster: ", err.Error())
		return
	}

	orderItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError("Error while update state: ", err.Error())
		return
	}

	itemData := orderItem.Data
	itemDataConfig := itemData.Config.(entities.K8sClusterConfig)
	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(orderItem.ID)
	plan.Name = types.StringValue(itemDataConfig.Name)

	_, _, _, _, err = setComputedNames(&plan, order)
	if err != nil {
		resp.Diagnostics.AddError(consts.READ_RES_FAIL, fmt.Sprintf("Error while set computed fields data: %s", err.Error()))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	config := orders.CommonActionParams{
		Platform:            plan.Platform.ValueString(),
		NetSegment:          plan.NetSegment.ValueString(),
		DataCenter:          plan.DataCenter.ValueString(),
		RegionAddNodesMax:   image.RegionAddNodesMax,
		IngressAddNodesMax:  image.IngressAddNodesMax,
		RegionIngressAddMax: image.RegionIngressAddMax,
	}

	defaultCPConfig := image.ControlPlane[plan.ControlPanelSize.ValueString()]
	if isCPConfigChanged(defaultCPConfig, plan.ControlPlane) {
		resp.Diagnostics.Append(r.handleControlPlaneAction(plan.ControlPlane, ConvertDefaultCPToModel(defaultCPConfig), order, config)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if plan.Components.Istio != nil {
		if isIstioOptionsChangedFromDefault(plan.Components.Istio) {
			resp.Diagnostics.Append(r.configureIstioOptionsK8sCluster(ctx, order, &plan)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
		if plan.Components.Istio.ControlPlanes[0].Options.Eventrouter != types.BoolValue(true) {
			if err := order.K8sClusterConfigureIstioCP(ConvertModelToIstioCP(plan.Components.Istio.ControlPlanes[0]), "configure"); err != nil {
				resp.Diagnostics.AddError("Add istio control plane", err.Error())
				return
			}
		}
		if len(plan.Components.Istio.ControlPlanes) > 1 {
			resp.Diagnostics.Append(r.configureIstioCPsK8sCluster(order, &plan, "create")...)
			if resp.Diagnostics.HasError() {
				return
			}
			_, _, _, _, err = setComputedNames(&plan, order)
			if err != nil {
				resp.Diagnostics.AddError(consts.READ_RES_FAIL, fmt.Sprintf("Error while update state: %s", err.Error()))
			}
		}
	}

	componentsEnabled := getEnabledComponents(plan.Components)
	for _, component := range componentsEnabled {
		if component != "istio" {
			if err := order.K8sClusterManageComponent("add", component); err != nil {
				resp.Diagnostics.AddError(
					fmt.Sprintf("Add %s", component),
					fmt.Sprintf("Adding %s ended with error.\nError message: %s", component, err.Error()),
				)
				return
			}
		}
	}

	if !plan.ContainerCPURatio.IsUnknown() && !plan.ContainerCPURatio.IsNull() &&
		!plan.ContainerMemoryRatio.IsUnknown() && !plan.ContainerMemoryRatio.IsNull() {
		if plan.ContainerCPURatio.ValueInt64() != image.DefaultContainerCPURatio ||
			plan.ContainerMemoryRatio.ValueInt64() != image.DefaultContainerMemoryRatio {

			resp.Diagnostics.Append(r.configureRatioK8sCluster(order, plan)...)
			if resp.Diagnostics.HasError() {
				return
			}
		} else {
			plan.ContainerCPURatio = types.Int64Value(image.DefaultContainerCPURatio)
			plan.ContainerMemoryRatio = types.Int64Value(image.DefaultContainerMemoryRatio)
		}
	}

	if len(plan.Products) != 0 {
		resp.Diagnostics.Append(r.configureProductsK8sCluster(order, plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !plan.Visibility.IsNull() && !plan.Visibility.IsUnknown() {
		resp.Diagnostics.Append(r.configureVisibilityK8sCluster(order)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	firstRegionData := ConvertRegionToModel(orderItem.Data.Config.(entities.K8sClusterConfig).Regions[0], true)

	var planFirstRegion, stateFirstRegion []K8sClusterRegionModel
	planFirstRegion = append(planFirstRegion, plan.Regions[0])
	stateFirstRegion = append(stateFirstRegion, firstRegionData)

	resp.Diagnostics.Append(r.handleRegionsModifications(planFirstRegion, stateFirstRegion, order, config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(plan.Regions) > 1 {
		resp.Diagnostics.Append(r.addRegionsK8sCluster(order, plan, config)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	firstIngressData := ConvertModelToIngress(plan.Ingress[0])

	if plan.Ingress[0].Size.ValueInt64() > image.DefaultIngressSize {
		newNodesCount := plan.Ingress[0].Size.ValueInt64() - image.DefaultIngressSize
		if err := order.K8sClusterAddNodes(firstIngressData.Name, newNodesCount, config, "ingress"); err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Adding nodes to %s", plan.Ingress[0].Name),
				fmt.Sprintf("Adding nodes to %s ended with error.\nError message: %s", plan.Ingress[0].Name, err.Error()),
			)
			return
		}
	}

	if IsIngressFeaturesChangedFromDefault(plan.Ingress[0]) || len(plan.Ingress[0].TcpUdpSettings) > 0 {
		if err := order.K8sClusterIngressSettings(firstIngressData); err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Configure %s settings", plan.Ingress[0].Name),
				fmt.Sprintf("Configure %s settings ended with error.\nError message: %s", plan.Ingress[0].Name, err.Error()),
			)
			return
		}
	}

	if len(plan.Ingress) > 1 {
		resp.Diagnostics.Append(r.addIngressK8sCluster(order, plan, config)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r K8sClusterResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state K8sClusterModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	order, err := orders.GetK8sClusterOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		diags.AddError("Can't get order from portal: ", err.Error())
		resp.Diagnostics.Append(diags...)
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

	err = order.Delete()
	if err != nil {
		diags.AddError(consts.DELETE_RES_FAIL, err.Error())
		resp.Diagnostics.Append(diags...)
	}

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r K8sClusterResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var orderID types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("order_id"), &orderID)...)
	if orderID.IsNull() {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Resource hasn't order_id. Without order_id can't get order from portal",
		)
		return
	}

	order, err := orders.GetK8sClusterOrder(
		r.client.Creds,
		r.client.ProjectName,
		orderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't read order with order_id equal '%s'.\nError: %s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}

	item, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get k8s cluster item from portal with order_id '%s'.\nError: %s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}

	config, ok := item.Data.Config.(entities.K8sClusterConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Item data type conversion error",
		)
		return
	}

	state := K8sClusterModel{
		OrderID:          orderID,
		ItemID:           types.StringValue(item.ID),
		FinancialProject: types.StringValue(order.FinancialSource.Name),
		Label:            types.StringValue(order.Label),
		Name:             types.StringValue(config.Name),
		Domain:           types.StringValue(config.Domain),
		Version: K8sClusterVersionModel{
			K8sVersion:     types.StringValue(config.Version),
			ProductVersion: types.StringValue(config.ProductVersion),
		},
		Platform:             types.StringValue(config.Platform),
		DataCenter:           types.StringValue(order.Attrs.DataCenter),
		NetSegment:           types.StringValue(config.NetSegment),
		ControlPanelSize:     types.StringValue(order.Attrs.ControlPanelSize),
		ContainerCPURatio:    types.Int64Value(config.ContainerCPURatio),
		ContainerMemoryRatio: types.Int64Value(config.ContainerMemoryRatio),
		BalancerDnsZone:      types.StringValue(order.Attrs.BalancerDnsZone),
		Regions:              getListRegionsFromData(config.Regions),
		Ingress:              getListIngressFromData(config.Ingress),
		CniPlugin:            ConvertFeaturesToModel(config.Features),
		Components:           ConvertComponentsToModel(ctx, &config.Components),
		ControlPlane:         ConvertControlPlaneToModel(config.ControlPlane),
		Products:             getListProductsFromData(config.Products),
		GslbOnly:             ConvertGslbOnlyToModel(config.GslbOnly),
	}

	var visibility types.Bool
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("visibility"), &visibility)...)
	if !visibility.IsNull() {
		state.Visibility = visibility
	}

	var lifetime types.Int64
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("lifetime"), &lifetime)...)
	if !lifetime.IsNull() {
		state.Lifetime = lifetime
	}

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r K8sClusterResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r K8sClusterResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan K8sClusterModel
	var state K8sClusterModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetK8sClusterOrder(
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

	image, err := references.GetK8sClusterImageData(
		r.client.Creds,
		r.client.Organization,
		r.client.Environment,
	)

	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("get k8s cluster image data"), consts.UPDATE_RES_FAIL, err.Error())
		return
	}

	labelChanged := !plan.Label.Equal(state.Label)
	financialProjectChanged := !plan.FinancialProject.Equal(state.FinancialProject)
	controlPlaneChanged := !reflect.DeepEqual(plan.ControlPlane, state.ControlPlane)
	componentsChanged := isComponentsChanged(plan.Components, state.Components)
	ratioChanged := !plan.ContainerCPURatio.Equal(state.ContainerCPURatio) || !plan.ContainerMemoryRatio.Equal(state.ContainerMemoryRatio)
	regionsChanged := !reflect.DeepEqual(plan.Regions, state.Regions)
	ingressChanged := !reflect.DeepEqual(plan.Ingress, state.Ingress)
	istioChanged := plan.Components.Istio != nil && !reflect.DeepEqual(plan.Components.Istio, state.Components.Istio)
	productsChanged := len(plan.Products) != len(state.Products)
	versionChanged := !plan.Version.ProductVersion.Equal(state.Version.ProductVersion)

	if versionChanged {
		resp.Diagnostics.Append(r.updateK8sCluster(order, plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.State.SetAttribute(ctx, path.Root("version"), plan.Version)
	}

	if financialProjectChanged {
		err = order.ChangeFinancialProject(finProj.ID)
		if err != nil {
			resp.Diagnostics.AddError(
				consts.UPDATE_RES_FAIL,
				fmt.Sprintf(
					"Changing order financial project ended with error.\nError message: %s", err.Error(),
				),
			)
			return
		}
		resp.State.SetAttribute(ctx, path.Root("financial_project"), plan.FinancialProject)
	}

	if labelChanged {
		utils.ChangeOrderLabel(order, plan.Label.ValueString(), resp)
		resp.State.SetAttribute(ctx, path.Root("label"), plan.Label.ValueString())
	}

	config := orders.CommonActionParams{
		Platform:            plan.Platform.ValueString(),
		NetSegment:          plan.NetSegment.ValueString(),
		DataCenter:          plan.DataCenter.ValueString(),
		RegionAddNodesMax:   image.RegionAddNodesMax,
		IngressAddNodesMax:  image.IngressAddNodesMax,
		RegionIngressAddMax: image.RegionIngressAddMax,
	}

	if controlPlaneChanged {
		resp.Diagnostics.Append(r.handleControlPlaneAction(plan.ControlPlane, state.ControlPlane, order, config)...)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.State.SetAttribute(ctx, path.Root("control_plane"), plan.ControlPlane)
	}

	if istioChanged {
		switch {

		case plan.Components.Istio != nil && state.Components.Istio != nil:
			if isIstioOptionsChanged(plan, state) || isIstioFlavorChanged(plan, state) {
				resp.Diagnostics.Append(r.configureIstioOptionsK8sCluster(ctx, order, &plan)...)
				if resp.Diagnostics.HasError() {
					return
				}
			}
			resp.Diagnostics.Append(r.handleIstioCPAction(plan, state, order)...)
			if resp.Diagnostics.HasError() {
				return
			}
			setComputedNames(&plan, order)

		case plan.Components.Istio != nil && state.Components.Istio == nil:
			resp.Diagnostics.Append(r.addIstioK8sCluster(ctx, order, &plan)...)
			if resp.Diagnostics.HasError() {
				return
			}
			if len(plan.Components.Istio.ControlPlanes) > 1 {
				resp.Diagnostics.Append(r.configureIstioCPsK8sCluster(order, &plan, "create")...)
			}
			if resp.Diagnostics.HasError() {
				return
			}
			setComputedNames(&plan, order)
		}
		resp.State.SetAttribute(ctx, path.Root("components").AtName("istio"), plan.Components.Istio)
	}

	if componentsChanged {
		resp.Diagnostics.Append(r.handleClusterComponentAction(plan.Components, state.Components, order)...)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.State.SetAttribute(ctx, path.Root("components"), plan.Components)
	}

	if ratioChanged {
		resp.Diagnostics.Append(r.configureRatioK8sCluster(order, plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.State.SetAttribute(ctx, path.Root("container_cpu_ratio"), plan.ContainerCPURatio)
		resp.State.SetAttribute(ctx, path.Root("container_memory_ratio"), plan.ContainerMemoryRatio)
	}

	if productsChanged {
		resp.Diagnostics.Append(r.configureProductsK8sCluster(order, plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.State.SetAttribute(ctx, path.Root("products"), plan.Products)
	}

	visibilityNotSet := plan.Visibility.IsNull() || plan.Visibility.IsUnknown()

	if !visibilityNotSet && state.Visibility.IsNull() {
		resp.Diagnostics.Append(r.configureVisibilityK8sCluster(order)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	resp.State.SetAttribute(ctx, path.Root("visibility"), plan.Visibility)

	if regionsChanged {
		resp.Diagnostics.Append(r.handleRegionsAddRemove(plan, state, order, config)...)
		if resp.Diagnostics.HasError() {
			return
		}
		newRegionsPlan, newRegionsState, _, _, _ := setComputedNames(&plan, order)
		for _, region := range newRegionsPlan {
			if region.Size.ValueInt64() > config.RegionIngressAddMax {
				resp.Diagnostics.Append(r.resizeNewRegionK8sCluster(order, region, config)...)
				if resp.Diagnostics.HasError() {
					return
				}
			}
			for i := range newRegionsState {
				if newRegionsState[i].Name == region.Name {
					newRegionsState[i].Size = region.Size
				}
			}
		}
		state.Regions = append(state.Regions, newRegionsState...)

		resp.Diagnostics.Append(r.handleRegionsModifications(plan.Regions, state.Regions, order, config)...)
		resp.State.SetAttribute(ctx, path.Root("regions"), plan.Regions)
	}

	if ingressChanged {
		resp.Diagnostics.Append(r.handleIngressAddRemove(plan, state, order, config)...)
		if resp.Diagnostics.HasError() {
			return
		}
		_, _, newIngressPlan, newIngressState, _ := setComputedNames(&plan, order)
		for _, ingress := range newIngressPlan {
			if ingress.Size.ValueInt64() > config.RegionIngressAddMax {
				resp.Diagnostics.Append(r.resizeNewIngressK8sCluster(order, ingress, config)...)
				if resp.Diagnostics.HasError() {
					return
				}
			}
			for i := range newIngressState {
				if newIngressState[i].Name == ingress.Name {
					newIngressState[i].Size = ingress.Size
				}
			}
		}
		state.Ingress = append(state.Ingress, newIngressState...)

		resp.Diagnostics.Append(r.handleIngressModifications(plan.Ingress, state.Ingress, order, config)...)
		resp.State.SetAttribute(ctx, path.Root("ingress"), plan.Ingress)
	}
}

func (r K8sClusterResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {

	var plan, state K8sClusterModel

	if req.Plan.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	}

	checkOrderIsDeleted := utils.CheckOrderIsDeleted(r.client, plan.OrderID, "cluster", "kubernetes")
	resp.Diagnostics.Append(checkOrderIsDeleted.Diagnostics...)

	utils.HandleExternalDeletionRecreate(ctx, resp, &plan, checkOrderIsDeleted.IsDeleted, checkOrderIsDeleted.Diagnostics)

	r.NetSegmentModifyPlan(ctx, &plan, &state, resp)
	r.DataCenterModifyPlan(ctx, &plan, &state, resp)
	r.BalancerDnsZoneModifyPlan(ctx, &plan, resp)
	r.DomainModifyPlan(ctx, &plan, resp)
	r.ProductVersionModifyPlan(ctx, &plan, &state, resp)

	validateRegionsIsCodes(plan.Regions, resp)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r K8sClusterResource) BalancerDnsZoneModifyPlan(
	ctx context.Context,
	plan *K8sClusterModel,
	resp *resource.ModifyPlanResponse,
) {
	dnsZone := plan.BalancerDnsZone.ValueString()
	netSegment := plan.NetSegment.ValueString()
	dnsZones, err := references.GetBalancerDnsZones(r.client.Creds, netSegment)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.MODIFY_PLAN_FAIL,
			fmt.Sprintf("Getting balancer DNS zone ended with error: %s", err.Error()),
		)
		return
	}
	dnsZoneIndex := utils.FindIndexInSlice(dnsZone, dnsZones)
	if dnsZoneIndex == -1 {
		resp.Diagnostics.AddAttributeError(
			path.Root("balancer_dns_zone"),
			consts.MODIFY_PLAN_FAIL,
			fmt.Sprintf("Order in net_segment:[%s] not support dns_zone:[%s]. Allowed zones: %s",
				netSegment, dnsZone, dnsZones),
		)
		return
	}
}

func (r K8sClusterResource) DomainModifyPlan(
	ctx context.Context,
	plan *K8sClusterModel,
	resp *resource.ModifyPlanResponse,
) {

	availableDomains, err := sources.GetDomainsByProjectName(r.client.Creds, r.client.ProjectName)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.MODIFY_PLAN_FAIL,
			fmt.Sprintf("Getting domains ended with error: %s", err.Error()),
		)
		return
	}

	domainAvaiable := false
	var domains []string
	for _, domain := range availableDomains {
		if domain.Label == plan.Domain.ValueString() {
			domainAvaiable = true
			return
		}
		domains = append(domains, domain.Label)
	}

	if !domainAvaiable {
		resp.Diagnostics.AddAttributeError(
			path.Root("domain"),
			consts.MODIFY_PLAN_FAIL,
			fmt.Sprintf("Available domains [%s].",
				strings.Join(domains, ", ")),
		)
		return
	}
}

func (r K8sClusterResource) NetSegmentModifyPlan(
	ctx context.Context,
	plan,
	state *K8sClusterModel,
	resp *resource.ModifyPlanResponse,
) {
	netSegments, err := sources.GetNetSegments(
		r.client.Creds,
		r.client.ProjectName,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.MODIFY_PLAN_FAIL,
			fmt.Sprintf("Getting network segments ended with error: %s", err.Error()),
		)
		return
	}

	netSegmentAvaiable := false
	var segmentsCodes []string
	for _, segment := range netSegments {
		if segment.Code == plan.NetSegment.ValueString() {
			netSegmentAvaiable = true
			return
		}
		segmentsCodes = append(segmentsCodes, segment.Code)
	}

	if !netSegmentAvaiable {
		resp.Diagnostics.AddAttributeError(
			path.Root("net_segment"),
			consts.MODIFY_PLAN_FAIL,
			fmt.Sprintf(
				"`net_segment=%s` not found.\nAvailable network segments is [%s]",
				plan.NetSegment.ValueString(),
				strings.Join(segmentsCodes, ", "),
			),
		)
	}
}

func (r K8sClusterResource) DataCenterModifyPlan(
	ctx context.Context,
	plan,
	state *K8sClusterModel,
	resp *resource.ModifyPlanResponse,
) {
	dataCenters, err := sources.GetDataCenters(
		r.client.Creds,
		r.client.ProjectName,
		r.client.Organization,
		r.client.Environment,
		plan.NetSegment.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.MODIFY_PLAN_FAIL,
			fmt.Sprintf("Getting data centers ended with error: %s", err.Error()),
		)
		return
	}

	dcAvailable := false
	var dcCodes []string
	for _, dc := range dataCenters {
		if dc.Code == plan.DataCenter.ValueString() {
			dcAvailable = true
			break
		}
		dcCodes = append(dcCodes, fmt.Sprintf(`"%s" - %s`, dc.Code, dc.Site))
	}

	if !dcAvailable {
		resp.Diagnostics.AddAttributeError(
			path.Root("data_center"),
			consts.MODIFY_PLAN_FAIL,
			fmt.Sprintf(
				"You can't choose data center=`%s`"+
					"\nAvailable data centers: %v",
				plan.DataCenter.ValueString(),
				strings.Join(dcCodes, ", "),
			),
		)
	}
}

func (r K8sClusterResource) ProductVersionModifyPlan(
	ctx context.Context,
	plan, state *K8sClusterModel,
	resp *resource.ModifyPlanResponse,
) {
	stateVer, planVer := state.Version.ProductVersion.ValueString(), plan.Version.ProductVersion.ValueString()
	if stateVer != "" {
		isOlder, err := utils.IsVersionOlder(stateVer, planVer)
		if err != nil {
			resp.Diagnostics.AddError(
				"Wrong versions format", err.Error(),
			)
			return
		}
		if stateVer != planVer && !isOlder {
			resp.Diagnostics.AddError(
				"Product version should be newer then exist",
				fmt.Sprintf("%s is older then %s", planVer, stateVer),
			)
			return
		}
	}

	// image, err := references.GetK8sClusterImageData(
	// 	r.provider.Creds,
	// 	r.provider.Organization,
	// 	r.provider.Environment,
	// )

	// if err != nil {
	// 	resp.Diagnostics.AddAttributeError(path.Root("get k8s cluster image data"), CREATE_RES_FAIL, err.Error())
	// 	return
	// }

	// planIndex := findIndexInSlice(planVer, image.AllowedVersions)
	// if planIndex == -1 {
	// 	resp.Diagnostics.AddError(
	// 		"Product version not in allowed versions",
	// 		fmt.Sprintf("Product version %s is not allowed. Allowed versions: %s", planVer, image.AllowedVersions),
	// 	)
	// 	return
	// }

	// if stateVer != "" {
	// 	stateIndex := findIndexInSlice(stateVer, image.AllowedVersions)
	// 	if stateIndex == -1 {
	// 		resp.Diagnostics.AddError(
	// 			"Product version not in allowed versions",
	// 			fmt.Sprintf("Current product version %s is not allowed. Allowed versions: %s", stateVer, image.AllowedVersions),
	// 		)
	// 		return
	// 	}
	// }
}

func (r K8sClusterResource) ValidateConfig(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {

	var clusterComponents K8sClusterComponentsModel
	diags := req.Config.GetAttribute(ctx, path.Root("components"), &clusterComponents)
	resp.Diagnostics.Append(diags...)

	var rawRegions basetypes.ListValue
	diags = req.Config.GetAttribute(ctx, path.Root("regions"), &rawRegions)
	resp.Diagnostics.Append(diags...)

	var regions []RegionModel
	for i, field := range rawRegions.Elements() {
		var region RegionModel
		err := tfsdk.ValueAs(ctx, field, &region)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("regions").AtListIndex(i),
				"Invalid region config",
				fmt.Sprintf("Сannot use region config at index %d: %s", i, err),
			)
			return
		}
		regions = append(regions, region)
	}

	clusterPlanComponents := getClusterEnabledComponents(clusterComponents)
	regionsPlanComponents := getAllRegionEnabledComponents(regions)

	for _, name := range regionsPlanComponents {
		if _, ok := clusterPlanComponents[name]; !ok {
			resp.Diagnostics.AddAttributeError(
				path.Root("components"),
				"Check component",
				fmt.Sprintf("Сomponent %s is not installed in cluster, but it is planned in the regions."+
					"If you want to delete %s in cluster, first disable it in regions.", name, name),
			)
			return
		}
	}

	tsdsInstalled := clusterComponents.TsdsOperator != nil && clusterComponents.TsdsOperator.Installed.ValueBool()
	istioNotInstalled := clusterComponents.Istio == nil || (clusterComponents.Istio != nil && !clusterComponents.Istio.Installed.ValueBool())

	if tsdsInstalled && istioNotInstalled {
		resp.Diagnostics.AddAttributeError(
			path.Root("components"),
			"Missing Istio",
			"To install TSDS operator, component Istio must be installed in the cluster.",
		)
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}

	var controlPlaneRawList []basetypes.ObjectValue
	diags = req.Config.GetAttribute(ctx, path.Root("control_plane"), &controlPlaneRawList)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if len(controlPlaneRawList) != 3 {
		resp.Diagnostics.AddAttributeError(
			path.Root("control_plane"),
			"Missing component",
			"You must define exactly 3 control plane components: master, infra, monitoring",
		)
	}

	var gslb types.Bool
	diags = req.Config.GetAttribute(ctx, path.Root("gslb_only"), &gslb)
	resp.Diagnostics.Append(diags...)

	var version K8sClusterVersionModel
	diags = req.Config.GetAttribute(ctx, path.Root("version"), &version)
	resp.Diagnostics.Append(diags...)

	versionCheck, err := utils.IsVersionOlder(version.ProductVersion.ValueString(), "1.21.0")

	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("version"),
			"Invalid version format",
			fmt.Sprintf("Failed to parse version: %v", err),
		)
		return
	}

	gslbCheck := versionCheck && !gslb.IsNull()
	if gslbCheck {
		resp.Diagnostics.AddAttributeError(
			path.Root("gslb_only"),
			"Invalid attribute gslb_only",
			"GSLB Only balancing is not available for configuration in versions earlier than 1.21.0",
		)
		return
	}

	if !versionCheck {
		if gslb.IsNull() || gslb.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("gslb_only"),
				"Missing required attribute",
				"gslb_only parameter is required for versions 1.21.0 and higher",
			)
		}
	}
}

func validateRegionsIsCodes(
	planRegions []K8sClusterRegionModel,
	resp *resource.ModifyPlanResponse,
) {
	for i, region := range planRegions {
		nonNullCodes := 0
		for _, code := range region.IsCodes.Elements() {
			if !code.IsNull() && !code.IsUnknown() {
				nonNullCodes++
			}
		}

		if region.Name.IsNull() || region.Name.IsUnknown() {
			if i == 0 && nonNullCodes > 1 {
				resp.Diagnostics.AddAttributeError(
					path.Root("regions").AtListIndex(i),
					"Too many IS codes",
					"First region cannot have more than one IS code.",
				)
			}
			continue
		}

		nameStr := region.Name.ValueString()

		if !strings.Contains(nameStr, "genr") && nonNullCodes > 1 {
			resp.Diagnostics.AddAttributeError(
				path.Root("regions").AtListIndex(i),
				"Too many IS codes",
				fmt.Sprintf("Region '%s' has more than one IS code, but it is not a 'genr' region.", nameStr),
			)
			return
		}
	}
}

func getAllRegionEnabledComponents(regions []RegionModel) []string {

	unique := make(map[string]struct{})
	for _, region := range regions {
		for _, name := range getEnabledComponents(region.Components) {
			unique[name] = struct{}{}
		}
	}

	var result []string
	for name := range unique {
		result = append(result, name)
	}
	return result
}

func getClusterEnabledComponents(planComponents K8sClusterComponentsModel) map[string]struct{} {

	result := make(map[string]struct{})
	for _, name := range getEnabledComponents(&planComponents) {
		result[name] = struct{}{}
	}
	return result
}

func setComputedNames(
	plan *K8sClusterModel,
	order *orders.K8sClusterOrder,
) (
	newRegionsPlan, newRegionsState []K8sClusterRegionModel,
	newIngressPlan, newIngressState []K8sClusterIngressModel,
	err error,
) {

	orderItem, err := order.GetParentItem()
	if err != nil {
		return nil, nil, nil, nil,
			fmt.Errorf("can't get k8s cluster item from portal.\nError:%s", err.Error())
	}

	itemDataConfig, ok := orderItem.Data.Config.(entities.K8sClusterConfig)
	if !ok {
		return nil, nil, nil, nil,
			fmt.Errorf("item data type conversion error")
	}

	regions := getListRegionsFromData(itemDataConfig.Regions)
	for i := 0; i < len(plan.Regions); i++ {
		oldName := plan.Regions[i].Name
		if i < len(regions) {
			plan.Regions[i].Name = regions[i].Name
			// plan.Regions[i].VMs = regions[i].VMs
			setDefaultRatios(&plan.Regions[i])
		} else {
			plan.Regions[i].Name = types.StringNull()
		}
		if oldName.ValueString() == "" && !plan.Regions[i].Name.IsNull() {
			newRegionsPlan = append(newRegionsPlan, plan.Regions[i])
			newRegionsState = append(newRegionsState, regions[i])
		}
	}
	ingresses := getListIngressFromData(itemDataConfig.Ingress)
	for i := 0; i < len(plan.Ingress); i++ {
		oldName := plan.Ingress[i].Name
		if i < len(ingresses) {
			plan.Ingress[i].Name = ingresses[i].Name
		} else {
			plan.Ingress[i].Name = types.StringNull()
		}
		if oldName.ValueString() == "" && !plan.Ingress[i].Name.IsNull() {
			newIngressPlan = append(newIngressPlan, plan.Ingress[i])
			newIngressState = append(newIngressState, ingresses[i])
		}
	}
	if itemDataConfig.Components.Istio == nil {
		return
	}
	controlPlanes := ConvertIstioCPToModel(itemDataConfig.Components.Istio.ControlPlanes)
	for i := 0; i < len(plan.Components.Istio.ControlPlanes); i++ {
		if i < len(controlPlanes) {
			plan.Components.Istio.ControlPlanes[i].Name = controlPlanes[i].Name
		} else {
			plan.Components.Istio.ControlPlanes[i].Name = types.StringNull()
		}
	}
	return newRegionsPlan, newRegionsState, newIngressPlan, newIngressState, err
}

func setDefaultRatios(region *K8sClusterRegionModel) {
	if region.ContainerCPURatio.IsNull() || region.ContainerCPURatio.IsUnknown() {
		region.ContainerCPURatio = types.Int64Value(1)
	}
	if region.ContainerMemoryRatio.IsNull() || region.ContainerMemoryRatio.IsUnknown() {
		region.ContainerMemoryRatio = types.Int64Value(1)
	}
	if region.PodCPUMax.IsNull() || region.PodCPUMax.IsUnknown() {
		region.PodCPUMax = types.Int64Value(region.Flavor.Cores.ValueInt64() - 1)
	}
	if region.PodMemoryMax.IsNull() || region.PodMemoryMax.IsUnknown() {
		region.PodMemoryMax = types.Int64Value(region.Flavor.Memory.ValueInt64() - 1)
	}
}

func getListRegionsFromData(clusterRegions []entities.K8sConfigRegion) []K8sClusterRegionModel {

	if len(clusterRegions) == 0 {
		return nil
	}

	var regionsList []K8sClusterRegionModel

	for _, region := range clusterRegions {
		regionData := ConvertRegionToModel(region, true)
		regionsList = append(regionsList, regionData)
	}
	return regionsList
}

func getListIngressFromData(clusterIngress []entities.K8sConfigIngress) []K8sClusterIngressModel {

	if len(clusterIngress) == 0 {
		return nil
	}

	var ingressList []K8sClusterIngressModel

	for _, ingress := range clusterIngress {
		ingressData := ConvertIngressToModel(ingress, true)
		ingressList = append(ingressList, ingressData)
	}
	return ingressList
}

func getListProductsFromData(clusterProducts []string) []types.String {

	if len(clusterProducts) == 0 {
		return nil
	}

	var productsList []types.String

	for _, product := range clusterProducts {
		productsList = append(productsList, types.StringValue(product))
	}
	return productsList
}

func (r K8sClusterResource) updateK8sCluster(
	order *orders.K8sClusterOrder,
	plan K8sClusterModel,
) (diags diag.Diagnostics) {
	err := order.K8sClusterUpdate(
		ConvertModelToVersion(&plan.Version),
	)
	if err != nil {
		diags.AddError(
			"Update k8s cluster",
			fmt.Sprintf(
				"Update k8s cluster ended with error.\nError message: %s", err.Error(),
			),
		)
	}
	return
}

func (r K8sClusterResource) addRegionsK8sCluster(
	order *orders.K8sClusterOrder,
	plan K8sClusterModel,
	config orders.CommonActionParams,
) (diags diag.Diagnostics) {
	for i := 1; i < len(plan.Regions); i++ {
		setDefaultRatios(&plan.Regions[i])
		region := plan.Regions[i]
		err := order.K8sClusterAddRegion(ConvertModelToRegion(region), config)
		setComputedNames(&plan, order)
		if err != nil {
			diags.AddError(
				"Adding region",
				fmt.Sprintf("Adding region ended with error.\nError message: %s", err.Error()),
			)
			return
		}
		region = plan.Regions[i]
		if region.Size.ValueInt64() > config.RegionIngressAddMax {
			if diags = r.resizeNewRegionK8sCluster(order, region, config); diags.HasError() {
				return
			}
		}
		componentsEnabled := getEnabledComponents(region.Components)
		for _, component := range componentsEnabled {
			action := fmt.Sprintf("region_configure_%s", component)
			if err := order.K8sClusterRegionConfigureComponents(ConvertModelToRegion(region), action); err != nil {
				diags.AddError(
					fmt.Sprintf("Configure %s", component),
					fmt.Sprintf("Configure %s in %s ended with error.\nError message: %s", component, region.Name, err.Error()),
				)
				return
			}
		}
	}
	return
}

func (r K8sClusterResource) resizeNewRegionK8sCluster(
	order *orders.K8sClusterOrder,
	region K8sClusterRegionModel,
	config orders.CommonActionParams,
) (diags diag.Diagnostics) {
	newNodesCount := region.Size.ValueInt64() - config.RegionIngressAddMax
	err := order.K8sClusterAddNodes(region.Name.ValueString(), newNodesCount, config, "region")
	if err != nil {
		diags.AddError(
			"Adding region",
			fmt.Sprintf("Adding region ended with error.\nError message: %s", err.Error()),
		)
		return
	}
	err = order.K8sClusterConfigureRegionIngress("uncordon", "region", region.Name.ValueString())
	if err != nil {
		diags.AddError(
			"Adding region",
			fmt.Sprintf("Uncordon nodes ended with error.\nError message: %s", err.Error()),
		)
		return
	}
	return
}

func (r K8sClusterResource) addIngressK8sCluster(
	order *orders.K8sClusterOrder,
	plan K8sClusterModel,
	config orders.CommonActionParams,
) (diags diag.Diagnostics) {
	for i := 1; i < len(plan.Ingress); i++ {
		ingress := plan.Ingress[i]
		err := order.K8sClusterAddIngress(ConvertModelToIngress(ingress), config)
		setComputedNames(&plan, order)
		if err != nil {
			diags.AddError(
				"Adding ingress",
				fmt.Sprintf("Adding ingress ended with error.\nError message: %s", err.Error()),
			)
			return
		}
		ingress = plan.Ingress[i]
		if ingress.Size.ValueInt64() > config.RegionIngressAddMax {
			newNodesCount := ingress.Size.ValueInt64() - config.RegionIngressAddMax
			err := order.K8sClusterAddNodes(ingress.Name.ValueString(), newNodesCount, config, "ingress")
			if err != nil {
				diags.AddError(
					"Adding ingress",
					fmt.Sprintf("Adding ingress ended with error.\nError message: %s", err.Error()),
				)
				return
			}
		}
		if IsIngressFeaturesChangedFromDefault(ingress) || len(ingress.TcpUdpSettings) > 0 {
			if err := order.K8sClusterIngressSettings(ConvertModelToIngress(ingress)); err != nil {
				diags.AddError(
					fmt.Sprintf("Configure %s settings", ingress.Name),
					fmt.Sprintf("Configure %s settings ended with error.\nError message: %s", ingress.Name, err.Error()),
				)
				return
			}
		}
	}
	return
}

func (r K8sClusterResource) resizeNewIngressK8sCluster(
	order *orders.K8sClusterOrder,
	ingress K8sClusterIngressModel,
	config orders.CommonActionParams,
) (diags diag.Diagnostics) {
	newNodesCount := ingress.Size.ValueInt64() - config.RegionIngressAddMax
	err := order.K8sClusterAddNodes(ingress.Name.ValueString(), newNodesCount, config, "ingress")
	if err != nil {
		diags.AddError(
			"Adding ingress",
			fmt.Sprintf("Adding ingress ended with error.\nError message: %s", err.Error()),
		)
		return
	}
	return
}

func (r K8sClusterResource) addIstioK8sCluster(
	ctx context.Context,
	order *orders.K8sClusterOrder,
	plan *K8sClusterModel,
) (diags diag.Diagnostics) {
	istioData, istioDiags := ConvertModelToIstio(ctx, plan.Components.Istio)
	diags.Append(istioDiags...)
	if istioDiags.HasError() {
		return
	}
	err := order.K8sClusterAddIstio(istioData)
	if err != nil {
		diags.AddError(
			"Add istio",
			fmt.Sprintf(
				"Add istio ended with error. Error message: %s", err.Error(),
			),
		)
	}
	return
}

func (r K8sClusterResource) configureIstioOptionsK8sCluster(
	ctx context.Context,
	order *orders.K8sClusterOrder,
	plan *K8sClusterModel,
) (diags diag.Diagnostics) {
	options, optionsDiags := ConvertModelToIstioOptions(ctx, plan.Components.Istio.Options)
	diags.Append(optionsDiags...)
	if optionsDiags.HasError() {
		return diags
	}
	err := order.K8sClusterConfigureIstioOptions(
		&options,
		plan.Components.Istio.OptionsFlavor.ValueString(),
	)
	if err != nil {
		diags.AddError(
			"Configure istio options",
			fmt.Sprintf(
				"Configure istio options ended with error.\nError message: %s", err.Error(),
			),
		)
	}
	return
}

func (r K8sClusterResource) configureIstioCPsK8sCluster(
	order *orders.K8sClusterOrder,
	plan *K8sClusterModel,
	action string,
) (diags diag.Diagnostics) {
	for i := 1; i < len(plan.Components.Istio.ControlPlanes); i++ {
		err := order.K8sClusterConfigureIstioCP(
			ConvertModelToIstioCP(plan.Components.Istio.ControlPlanes[i]),
			action,
		)
		if err != nil {
			diags.AddError(
				"Add istio control plane",
				fmt.Sprintf(
					"Add istio control plane ended with error.\nError message: %s", err.Error(),
				),
			)
			return
		}
	}
	return
}

func (r K8sClusterResource) configureRatioK8sCluster(
	order *orders.K8sClusterOrder,
	plan K8sClusterModel,
) (diags diag.Diagnostics) {
	limits, err := references.GetK8sClusterRatioData(
		r.client.Creds,
		plan.NetSegment.ValueString(),
		utils.GetMinorVersion(plan.Version.ProductVersion.ValueString()),
	)
	if err != nil {
		diags.AddError(
			"Can't get data from references",
			fmt.Sprintf(
				"Can't get limits data from references\nError message: %s", err.Error(),
			),
		)
		return
	}
	err = order.K8sClusterRequestsRatioConfig(ConvertModelToLimits(plan), limits)
	if err != nil {
		diags.AddError(
			"Configure requests ratio",
			fmt.Sprintf(
				"Configure requests ratio ended with error.\nError message: %s", err.Error(),
			),
		)
	}
	return
}

func (r K8sClusterResource) configureProductsK8sCluster(
	order *orders.K8sClusterOrder,
	plan K8sClusterModel,
) (diags diag.Diagnostics) {
	products := func(list []types.String) []string {
		r := make([]string, len(list))
		for i, v := range list {
			r[i] = v.ValueString()
		}
		return r
	}(plan.Products)
	err := order.K8sClusterProducts(products)
	if err != nil {
		diags.AddError(
			"Allow products deployment",
			fmt.Sprintf(
				"Allow products deployment ended with error.\nError message: %s", err.Error(),
			),
		)
	}
	return
}

func (r K8sClusterResource) configureVisibilityK8sCluster(
	order *orders.K8sClusterOrder,
) (diags diag.Diagnostics) {
	err := order.K8sClusterManageComponent("", "visibility")
	if err != nil {
		diags.AddError(
			"Change cluster visibility",
			fmt.Sprintf(
				"hange cluster visibility ended with error.\nError message: %s", err.Error(),
			),
		)
	}
	return
}

func (r K8sClusterResource) handleControlPlaneAction(
	planCP, stateCP []K8sClusterControlPlaneModel,
	order *orders.K8sClusterOrder,
	config orders.CommonActionParams,
) (diags diag.Diagnostics) {

	modifiedCPComponents := findCPComponentDiff(planCP, stateCP)

	type cpComponentsAction struct {
		components []K8sClusterControlPlaneModel
		action     func(component K8sClusterControlPlaneModel) error
		errMsg     string
	}

	actions := []cpComponentsAction{}

	if len(modifiedCPComponents) > 0 {
		for _, m := range modifiedCPComponents {
			modified := m
			switch modified.Change {
			case "flavor":
				actions = append(actions, cpComponentsAction{
					components: []K8sClusterControlPlaneModel{modified.Component.(K8sClusterControlPlaneModel)},
					action: func(region K8sClusterControlPlaneModel) error {
						return order.K8sClusterChangeFlavor(
							ConvertModelToCPComponent(modified.Component.(K8sClusterControlPlaneModel)), modified.Action)
					},
					errMsg: fmt.Sprintf("change %s %s", modified.Action, modified.Change),
				})
			case "nodes":
				if modified.Action == "add" {
					actions = append(actions, cpComponentsAction{
						components: []K8sClusterControlPlaneModel{modified.Component.(K8sClusterControlPlaneModel)},
						action: func(component K8sClusterControlPlaneModel) error {
							return order.K8sClusterAddNodes(component.Role.ValueString(), modified.Count, config, component.Role.ValueString())
						},
						errMsg: fmt.Sprintf("%s %s in %s", modified.Action, modified.Change, modified.Component.(K8sClusterControlPlaneModel).Role.ValueString()),
					})
				}
			}
		}
	}

	for _, act := range actions {
		for _, component := range act.components {
			err := act.action(component)
			if err != nil {
				diags.AddError(
					act.errMsg,
					fmt.Sprintf("%s ended with error.\nError message: %s", act.errMsg, err.Error()),
				)
				return
			}
		}
	}
	return
}

func (r K8sClusterResource) handleClusterComponentAction(
	planComponents, stateComponents *K8sClusterComponentsModel,
	order *orders.K8sClusterOrder,
) (diags diag.Diagnostics) {

	toAdd, toDelete := findClusterComponentsDiff(planComponents, stateComponents)

	for _, component := range toAdd {
		action := "add"
		if err := order.K8sClusterManageComponent(action, component.Name); err != nil {
			diags.AddError(
				fmt.Sprintf("%s %s", action, component.Name),
				fmt.Sprintf("%sing %s ended with error.\nError message: %s", action, component.Name, err.Error()),
			)
			return
		}
	}

	for _, component := range toDelete {
		action := "delete"
		if err := order.K8sClusterManageComponent(action, component.Name); err != nil {
			diags.AddError(
				fmt.Sprintf("%s %s", action, component.Name),
				fmt.Sprintf("%sing %s ended with error.\nError message: %s", action, component.Name, err.Error()),
			)
			return
		}
	}
	return
}

func (r K8sClusterResource) handleIstioCPAction(
	plan, state K8sClusterModel,
	order *orders.K8sClusterOrder,
) (diags diag.Diagnostics) {

	planIstioCP := plan.Components.Istio.ControlPlanes
	stateIstioCP := state.Components.Istio.ControlPlanes

	addedCP, _, modifiedCP := findIstioCPDiff(planIstioCP, stateIstioCP)

	type istioAction struct {
		istioCPList []K8sClusterIstioControlPlaneModel
		action      func(istioCP K8sClusterIstioControlPlaneModel) error
		errMsg      string
	}

	actions := []istioAction{}

	// if len(deletedCP) > 0 {
	// 	actions = append(actions, istioAction{
	// 		istioCPList: deletedCP,
	// 		action: func(istioCP K8sClusterIstioControlPlaneModel) error {
	// 			return order.K8sClusterConfigureIstioCP(ConvertModelToIstioCP(istioCP), "delete")
	// 		},
	// 		errMsg: "Deleting istio control plane",
	// 	})
	// }

	if len(addedCP) > 0 {
		actions = append(actions, istioAction{
			istioCPList: addedCP,
			action: func(istioCP K8sClusterIstioControlPlaneModel) error {
				return order.K8sClusterConfigureIstioCP(ConvertModelToIstioCP(istioCP), "create")
			},
			errMsg: "Adding istio control plane",
		})
	}

	if len(modifiedCP) > 0 {
		actions = append(actions, istioAction{
			istioCPList: modifiedCP,
			action: func(istioCP K8sClusterIstioControlPlaneModel) error {
				return order.K8sClusterConfigureIstioCP(ConvertModelToIstioCP(istioCP), "configure")
			},
			errMsg: "Configure istio control plane",
		})
	}

	for _, act := range actions {
		for _, istioCP := range act.istioCPList {
			err := act.action(istioCP)
			if err != nil {
				diags.AddError(
					"Configure Istio",
					fmt.Sprintf("%s ended with error.\nError message: %s", act.errMsg, err.Error()),
				)
				return
			}
		}
	}
	return
}

func (r K8sClusterResource) handleRegionsAddRemove(
	plan, state K8sClusterModel,
	order *orders.K8sClusterOrder,
	config orders.CommonActionParams,
) (diags diag.Diagnostics) {

	addedRegions, _, _ := findRegionIngressDiff(
		plan.Regions, state.Regions,
		func(r K8sClusterRegionModel) types.String {
			return r.Name
		},
		func(plan, state K8sClusterRegionModel) ([]ComponentModification, bool) {
			mods := RegionHasChanged(plan, state)
			return mods, len(mods) > 0
		},
	)

	type regionsAction struct {
		regionsList []K8sClusterRegionModel
		action      func(region K8sClusterRegionModel) error
		errMsg      string
	}

	actions := []regionsAction{}

	// if len(deletedRegions) > 0 {
	// 	actions = append(actions, regionsAction{
	// 		regionsList: deletedRegions,
	// 		action: func(region K8sClusterRegionModel) error {
	// 			return order.K8sClusterConfigureRegionIngress("delete", "region", region.Name.ValueString())
	// 		},
	// 		errMsg: "Deleting",
	// 	})
	// }

	if len(addedRegions) > 0 {
		actions = append(actions, regionsAction{
			regionsList: addedRegions,
			action: func(region K8sClusterRegionModel) error {
				setDefaultRatios(&region)
				return order.K8sClusterAddRegion(ConvertModelToRegion(region), config)
			},
			errMsg: "Adding",
		})
	}

	for _, act := range actions {
		for _, region := range act.regionsList {
			err := act.action(region)
			if err != nil {
				diags.AddError(
					"Configure regions",
					fmt.Sprintf("%s region ended with error.\nError message: %s", act.errMsg, err.Error()),
				)
				return
			}
		}
	}
	return
}

func (r K8sClusterResource) handleRegionsModifications(
	planRegions, stateRegions []K8sClusterRegionModel,
	order *orders.K8sClusterOrder,
	config orders.CommonActionParams,
) (diags diag.Diagnostics) {

	component := "region"

	_, _, modifiedRegions := findRegionIngressDiff(
		planRegions, stateRegions,
		func(r K8sClusterRegionModel) types.String {
			return r.Name
		},
		func(plan, state K8sClusterRegionModel) ([]ComponentModification, bool) {
			mods := RegionHasChanged(plan, state)
			return mods, len(mods) > 0
		},
	)

	type regionsAction struct {
		region K8sClusterRegionModel
		action func(region K8sClusterRegionModel) error
		errMsg string
	}

	actions := []regionsAction{}

	if len(modifiedRegions) > 0 {
		for _, m := range modifiedRegions {
			modified := m
			region := modified.Component.(K8sClusterRegionModel)
			switch modified.Change {
			case "flavor":
				actions = append(actions, regionsAction{
					region: region,
					action: func(region K8sClusterRegionModel) error {
						return order.K8sClusterRegionIngressChangeFlavor(ConvertModelToRegion(modified.Component.(K8sClusterRegionModel)), component)
					},
					errMsg: fmt.Sprintf("change %s %s", component, modified.Change),
				})
			case "iscodes":
				actions = append(actions, regionsAction{
					region: region,
					action: func(region K8sClusterRegionModel) error {
						return order.K8sClusterRegionSetCodes(ConvertModelToRegion(region), modified.Change)
					},
					errMsg: fmt.Sprintf("set %s %s", component, modified.Change),
				})
			case "nodes":
				if modified.Action == "add" {
					actions = append(actions, regionsAction{
						region: region,
						action: func(region K8sClusterRegionModel) error {
							if err := order.K8sClusterAddNodes(region.Name.ValueString(), modified.Count, config, component); err != nil {
								return err
							}
							return order.K8sClusterConfigureRegionIngress("uncordon", component, region.Name.ValueString())
						},
						errMsg: fmt.Sprintf("%s %s", modified.Action, modified.Change),
					})
				}
			case "requests_ratio_config":
				actions = append(actions, regionsAction{
					region: region,
					action: func(region K8sClusterRegionModel) error {
						return order.K8sClusterRegionRequestsRatioConfig(
							ConvertModelToRegion(region),
							fmt.Sprintf("%s_%s", component, modified.Change),
						)
					},
					errMsg: fmt.Sprintf("%s %s", modified.Action, modified.Change),
				})
			case "configure":
				actions = append(actions, regionsAction{
					region: region,
					action: func(region K8sClusterRegionModel) error {
						return order.K8sClusterRegionConfigureComponents(
							ConvertModelToRegion(region),
							fmt.Sprintf("%s_%s_%s", component, modified.Change, modified.Action),
						)
					},
					errMsg: fmt.Sprintf("%s %s", component, modified.Change),
				})
			}
		}
	}

	for _, act := range actions {
		if err := act.action(act.region); err != nil {
			diags.AddError(
				"Configure regions",
				fmt.Sprintf("%s ended with error.\nError message: %s", act.errMsg, err.Error()),
			)
			return
		}
	}
	return
}

func (r K8sClusterResource) handleIngressAddRemove(
	plan, state K8sClusterModel,
	order *orders.K8sClusterOrder,
	config orders.CommonActionParams,
) (diags diag.Diagnostics) {

	addedIngress, _, _ := findRegionIngressDiff(
		plan.Ingress, state.Ingress,
		func(i K8sClusterIngressModel) types.String {
			return i.Name
		},
		func(plan, state K8sClusterIngressModel) ([]ComponentModification, bool) {
			mods := IngressHasChanged(plan, state)
			return mods, len(mods) > 0
		},
	)

	type ingressAction struct {
		ingressList []K8sClusterIngressModel
		action      func(ingress K8sClusterIngressModel) error
		errMsg      string
	}

	actions := []ingressAction{}

	if len(addedIngress) > 0 {
		actions = append(actions, ingressAction{
			ingressList: addedIngress,
			action: func(ingress K8sClusterIngressModel) error {
				return order.K8sClusterAddIngress(ConvertModelToIngress(ingress), config)
			},
			errMsg: "Adding",
		})

	}

	for _, act := range actions {
		for _, ingress := range act.ingressList {
			err := act.action(ingress)
			if err != nil {
				diags.AddError(
					"Configure ingress",
					fmt.Sprintf("%s ingress ended with error.\nError message: %s", act.errMsg, err.Error()),
				)
				return
			}
		}
	}
	return
}

func (r K8sClusterResource) handleIngressModifications(
	planIngress, stateIngress []K8sClusterIngressModel,
	order *orders.K8sClusterOrder,
	config orders.CommonActionParams,
) (diags diag.Diagnostics) {

	component := "ingress"

	_, _, modifiedIngress := findRegionIngressDiff(
		planIngress, stateIngress,
		func(i K8sClusterIngressModel) types.String {
			return i.Name
		},
		func(plan, state K8sClusterIngressModel) ([]ComponentModification, bool) {
			mods := IngressHasChanged(plan, state)
			return mods, len(mods) > 0
		},
	)

	type ingressAction struct {
		ingress K8sClusterIngressModel
		action  func(ingress K8sClusterIngressModel) error
		errMsg  string
	}

	actions := []ingressAction{}

	if len(modifiedIngress) > 0 {
		for _, m := range modifiedIngress {
			modified := m
			ingress := modified.Component.(K8sClusterIngressModel)
			switch modified.Change {
			case "flavor":
				actions = append(actions, ingressAction{
					ingress: ingress,
					action: func(ingress K8sClusterIngressModel) error {
						return order.K8sClusterRegionIngressChangeFlavor(ConvertModelToIngress(modified.Component.(K8sClusterIngressModel)), component)
					},
					errMsg: fmt.Sprintf("change %s %s", component, modified.Change),
				})
			case "nodes":
				if modified.Action == "add" {
					actions = append(actions, ingressAction{
						ingress: ingress,
						action: func(ingress K8sClusterIngressModel) error {
							return order.K8sClusterAddNodes(ingress.Name.ValueString(), modified.Count, config, component)
						},
						errMsg: fmt.Sprintf("%s %s %s", modified.Action, component, modified.Change),
					})
				}
			case "settings":
				actions = append(actions, ingressAction{
					ingress: ingress,
					action: func(ingress K8sClusterIngressModel) error {
						return order.K8sClusterIngressSettings(
							ConvertModelToIngress(ingress),
						)
					},
					errMsg: fmt.Sprintf("configure %s %s", component, modified.Change),
				})
			}
		}
	}

	for _, act := range actions {
		if err := act.action(act.ingress); err != nil {
			diags.AddError(
				"Configure regions",
				fmt.Sprintf("%s ended with error.\nError message: %s", act.errMsg, err.Error()),
			)
			return
		}
	}
	return
}

type ComponentModification struct {
	Component any
	Change    string
	Action    string
	Count     int64
}

type ComponentModify struct {
	Name  string
	Value bool
}

type NamedComponent[T any] interface {
	GetName() types.String
	HasChanged(other T) ([]ComponentModification, bool)
}

func findRegionIngressDiff[T any](
	plan, state []T,
	nameFunc func(T) types.String,
	compareFunc func(plan T, state T) ([]ComponentModification, bool),
) (
	added []T,
	deleted []T,
	modified []ComponentModification,
) {
	stateMap := make(map[types.String]T)
	for _, stateItem := range state {
		stateMap[nameFunc(stateItem)] = stateItem
	}

	for name, item := range stateMap {
		found := false
		for _, planItem := range plan {
			if nameFunc(planItem) == name {
				found = true
				break
			}
		}
		if !found {
			deleted = append(deleted, item)
		}
	}

	for _, planItem := range plan {
		name := nameFunc(planItem)
		if item, exists := stateMap[name]; !exists {
			added = append(added, planItem)

			switch typedItem := any(planItem).(type) {
			case K8sClusterRegionModel:
				mods := DiffRegionComponents(typedItem, nil)
				modified = append(modified, mods...)
			case K8sClusterIngressModel:
				isDefaultFeatures := typedItem.Features.Http2Protocol == types.BoolValue(true) &&
					typedItem.Features.ProxyProtocol == types.BoolValue(false)
				if !isDefaultFeatures || len(typedItem.TcpUdpSettings) > 0 {
					modified = append(modified, ComponentModification{
						Component: typedItem,
						Change:    "settings",
						Action:    "settings",
					})
				}
			}
		} else {
			if mods, changed := compareFunc(planItem, item); changed {
				modified = append(modified, mods...)
			}
		}
	}

	return
}

func DiffRegionComponents(region K8sClusterRegionModel, stateRegion *K8sClusterRegionModel) []ComponentModification {
	var mods []ComponentModification

	check := func(name string, enabled types.Bool, changed bool) {
		if changed {
			mods = append(mods, ComponentModification{
				Component: region,
				Change:    "configure",
				Action:    name,
			})
		}
	}

	if stateRegion != nil {
		if region.Components == nil || reflect.ValueOf(region.Components).IsZero() {
			prevVal := reflect.ValueOf(stateRegion.Components)
			if prevVal.Kind() == reflect.Ptr && !prevVal.IsNil() {
				prevVal = prevVal.Elem()
			}
			if prevVal.Kind() == reflect.Struct {
				t := prevVal.Type()
				for i := 0; i < prevVal.NumField(); i++ {
					field := prevVal.Field(i)
					fieldType := t.Field(i)
					tag := fieldType.Tag.Get("tfsdk")
					if field.Type() == reflect.TypeOf(types.Bool{}) {
						if b, ok := field.Interface().(types.Bool); ok && b.ValueBool() {
							check(tag, types.BoolValue(false), true)
						}
					}
					if field.Kind() == reflect.Ptr && !field.IsNil() {
						elem := field.Elem()
						installed := elem.FieldByName("Installed")
						if installed.IsValid() && installed.Type() == reflect.TypeOf(types.Bool{}) {
							if b, ok := installed.Interface().(types.Bool); ok && b.ValueBool() {
								check(tag, types.BoolValue(false), true)
							}
						}
					}
				}
			}
			return mods
		}
		CompareWithTfsdkTags(region.Components, stateRegion.Components, check)
	} else {
		CompareWithTfsdkTags(region.Components, nil, check)
	}

	return mods
}

func RegionHasChanged(region, stateRegion K8sClusterRegionModel) []ComponentModification {
	var modifications []ComponentModification

	if region.ContainerCPURatio != stateRegion.ContainerCPURatio ||
		region.ContainerMemoryRatio != stateRegion.ContainerMemoryRatio ||
		region.PodCPUMax != stateRegion.PodCPUMax ||
		region.PodMemoryMax != stateRegion.PodMemoryMax {
		modifications = append(modifications, ComponentModification{
			Component: region,
			Change:    "requests_ratio_config",
			Action:    "change",
		})
	}

	if region.Size != stateRegion.Size {
		switch {
		case region.Size.ValueInt64() < stateRegion.Size.ValueInt64():
			modifications = append(modifications, ComponentModification{
				Component: region,
				Change:    "nodes",
				Action:    "delete",
				Count:     stateRegion.Size.ValueInt64() - region.Size.ValueInt64(),
			})
		case region.Size.ValueInt64() > stateRegion.Size.ValueInt64():
			modifications = append(modifications, ComponentModification{
				Component: region,
				Change:    "nodes",
				Action:    "add",
				Count:     region.Size.ValueInt64() - stateRegion.Size.ValueInt64(),
			})
		}
	}

	if region.Flavor != stateRegion.Flavor {
		modifications = append(modifications, ComponentModification{
			Component: region,
			Change:    "flavor",
			Action:    "change",
		})
	}

	if !reflect.DeepEqual(region.IsCodes, stateRegion.IsCodes) {
		modifications = append(modifications, ComponentModification{
			Component: region,
			Change:    "iscodes",
			Action:    "change",
		})
	}

	if region.Components != stateRegion.Components {
		mods := DiffRegionComponents(region, &stateRegion)
		modifications = append(modifications, mods...)
	}

	sort.Slice(modifications, func(i, j int) bool {
		priority := map[string]int{
			"flavor":                0,
			"nodes":                 1,
			"requests_ratio_config": 2,
			"configure":             3,
			"iscodes":               4,
		}
		return priority[modifications[i].Change] < priority[modifications[j].Change]
	})

	return modifications
}

func IngressHasChanged(ingress, stateIngress K8sClusterIngressModel) []ComponentModification {
	var modifications []ComponentModification

	if ingress.Size != stateIngress.Size {
		switch {
		case ingress.Size.ValueInt64() < stateIngress.Size.ValueInt64():
			modifications = append(modifications, ComponentModification{
				Component: ingress,
				Change:    "nodes",
				Action:    "delete",
				Count:     stateIngress.Size.ValueInt64() - ingress.Size.ValueInt64(),
			})
		case ingress.Size.ValueInt64() > stateIngress.Size.ValueInt64():
			modifications = append(modifications, ComponentModification{
				Component: ingress,
				Change:    "nodes",
				Action:    "add",
				Count:     ingress.Size.ValueInt64() - stateIngress.Size.ValueInt64(),
			})
		}
	}

	if ingress.Flavor != stateIngress.Flavor {
		modifications = append(modifications, ComponentModification{
			Component: ingress,
			Change:    "flavor",
			Action:    "change",
		})
	}

	if !reflect.DeepEqual(ingress.Features, stateIngress.Features) ||
		!reflect.DeepEqual(ingress.TcpUdpSettings, stateIngress.TcpUdpSettings) {
		modifications = append(modifications, ComponentModification{
			Component: ingress,
			Change:    "settings",
			Action:    "settings",
		})
	}

	sort.Slice(modifications, func(i, j int) bool {
		priority := map[string]int{
			"flavor":   0,
			"nodes":    1,
			"settings": 2,
		}
		return priority[modifications[i].Change] < priority[modifications[j].Change]
	})

	return modifications
}

func findCPComponentDiff(
	planCP, stateCP []K8sClusterControlPlaneModel,
) (
	modifications []ComponentModification,
) {
	stateMap := make(map[types.String]K8sClusterControlPlaneModel)
	for _, component := range stateCP {
		stateMap[component.Role] = component
	}

	for _, planComponent := range planCP {

		role := planComponent.Role
		stateComponent, exists := stateMap[role]

		if exists {
			if planComponent.Flavor.Name != stateComponent.Flavor.Name {
				modifications = append(modifications, ComponentModification{
					Component: planComponent,
					Change:    "flavor",
					Action:    planComponent.Role.ValueString(),
				})
			}

			if planComponent.Role.ValueString() == "infra" && planComponent.Size != stateComponent.Size {
				switch {
				case planComponent.Size.ValueInt64() < stateComponent.Size.ValueInt64():
					modifications = append(modifications, ComponentModification{
						Component: planComponent,
						Change:    "nodes",
						Action:    "delete",
					})
				case planComponent.Size.ValueInt64() > stateComponent.Size.ValueInt64():
					modifications = append(modifications, ComponentModification{
						Component: planComponent,
						Change:    "nodes",
						Action:    "add",
						Count:     planComponent.Size.ValueInt64() - stateComponent.Size.ValueInt64(),
					})
				}
			}
		}
	}

	sort.Slice(modifications, func(i, j int) bool {
		priority := map[string]int{
			"flavor": 0,
			"nodes":  1,
		}
		return priority[modifications[i].Change] < priority[modifications[j].Change]
	})

	return modifications
}

func findClusterComponentsDiff(
	planComponents, stateComponents *K8sClusterComponentsModel,
) (toAdd, toRemove []ComponentModify) {
	if planComponents == nil && stateComponents == nil {
		return
	}

	if planComponents == nil || reflect.ValueOf(planComponents).Elem().IsZero() {
		if stateComponents != nil {
			val := reflect.ValueOf(stateComponents).Elem()
			t := val.Type()
			for i := 0; i < val.NumField(); i++ {
				field := val.Field(i)
				fieldType := t.Field(i)
				tag := fieldType.Tag.Get("tfsdk")

				if field.Type() == reflect.TypeOf(types.Bool{}) {
					if b, ok := field.Interface().(types.Bool); ok && b.ValueBool() {
						toRemove = append(toRemove, ComponentModify{Name: tag, Value: false})
					}
				}
				if field.Kind() == reflect.Ptr && !field.IsNil() {
					elem := field.Elem()
					installed := elem.FieldByName("Installed")
					if installed.IsValid() && installed.Type() == reflect.TypeOf(types.Bool{}) {
						if b, ok := installed.Interface().(types.Bool); ok && b.ValueBool() {
							toRemove = append(toRemove, ComponentModify{Name: tag, Value: false})
						}
					}
				}
			}
		}
		return
	}

	CompareWithTfsdkTags(planComponents, stateComponents, func(name string, enabled types.Bool, check bool) {
		if enabled.IsNull() || !enabled.ValueBool() {
			toRemove = append(toRemove, ComponentModify{Name: name, Value: false})
			return
		}
		if enabled.ValueBool() {
			toAdd = append(toAdd, ComponentModify{Name: name, Value: true})
		}
	})

	return
}

func findIstioCPDiff(
	plan, state []K8sClusterIstioControlPlaneModel,
) (
	added, deleted, modified []K8sClusterIstioControlPlaneModel,
) {

	stateMap := make(map[types.String]K8sClusterIstioControlPlaneModel)
	for _, s := range state {
		stateMap[s.Name] = s
	}

	for name, s := range stateMap {
		found := false
		for _, p := range plan {
			if p.Name == name {
				found = true
				break
			}
		}
		if !found {
			deleted = append(deleted, s)
		}
	}

	for _, p := range plan {
		if s, exists := stateMap[p.Name]; !exists {
			added = append(added, p)
		} else {
			if !reflect.DeepEqual(p, s) {
				modified = append(modified, p)
			}
		}
	}

	return added, deleted, modified
}

func isCPConfigChanged(defaultCP []references.K8sClusterCPDefaults, planCP []K8sClusterControlPlaneModel) bool {
	roleMap := make(map[string]K8sClusterControlPlaneModel)
	for _, cp := range planCP {
		roleMap[cp.Role.ValueString()] = cp
	}

	for _, def := range defaultCP {
		if cp, exists := roleMap[def.Role]; exists {
			if def.Flavor != cp.Flavor.Name.ValueString() || def.Size != cp.Size.ValueInt64() {
				return true
			}
		}
	}

	return false
}

func isComponentsChanged(planComponents, stateComponents *K8sClusterComponentsModel) bool {
	planCopy := *planComponents
	stateCopy := *stateComponents

	planCopy.Istio = nil
	stateCopy.Istio = nil

	return !reflect.DeepEqual(planCopy, stateCopy)
}

func IsIngressFeaturesChangedFromDefault(ingress K8sClusterIngressModel) bool {

	def := K8sClusterIngressFeaturesModel{
		Http2Protocol: types.BoolValue(true),
		ProxyProtocol: types.BoolValue(false),
	}
	return ingress.Features.Http2Protocol.ValueBool() != def.Http2Protocol.ValueBool() ||
		ingress.Features.ProxyProtocol.ValueBool() != def.ProxyProtocol.ValueBool()
}

func isIstioOptionsChangedFromDefault(istio *K8sClusterIstioModel) bool {
	if istio.Installed == types.BoolValue(false) {
		return false
	}
	if !istio.OptionsFlavor.IsUnknown() && !istio.OptionsFlavor.IsNull() {
		return istio.OptionsFlavor != types.StringValue("tiny")
	}
	return false
}

func isIstioOptionsChanged(plan, state K8sClusterModel) bool {

	return !reflect.DeepEqual(plan.Components.Istio.Options, state.Components.Istio.Options)
}

func isIstioFlavorChanged(plan, state K8sClusterModel) bool {

	return !plan.Components.Istio.OptionsFlavor.IsUnknown() &&
		!plan.Components.Istio.OptionsFlavor.IsNull() &&
		!plan.Components.Istio.OptionsFlavor.Equal(state.Components.Istio.OptionsFlavor)
}

func getEnabledComponents(model interface{}) []string {
	if model == nil {
		return nil
	}

	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}

	var enabled []string

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := v.Type().Field(i)
		name := fieldType.Tag.Get("tfsdk")
		if name == "" {
			continue
		}

		switch field.Kind() {
		case reflect.Struct:
			if field.Type() == reflect.TypeOf(types.Bool{}) {
				b := field.Interface().(types.Bool)
				if !b.IsNull() && !b.IsUnknown() && b.ValueBool() {
					enabled = append(enabled, name)
				}
			}
		case reflect.Ptr:
			if !field.IsNil() {
				installed := field.Elem().FieldByName("Installed")
				if installed.IsValid() && installed.Type() == reflect.TypeOf(types.Bool{}) {
					b := installed.Interface().(types.Bool)
					if !b.IsNull() && !b.IsUnknown() && b.ValueBool() {
						enabled = append(enabled, name)
					}
				}
			}
		}
	}

	return enabled
}

func CompareWithTfsdkTags(
	current any,
	previous any,
	checkFunc func(fieldName string, enabled types.Bool, changed bool),
) {
	val := reflect.ValueOf(current)
	prevVal := reflect.ValueOf(previous)

	if !val.IsValid() {
		return
	}

	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return
		}
		val = val.Elem()
	}
	if prevVal.Kind() == reflect.Ptr {
		prevVal = prevVal.Elem()
	}

	if !val.IsValid() || val.Kind() != reflect.Struct {
		return
	}

	if prevVal.IsValid() && prevVal.Kind() == reflect.Ptr && !prevVal.IsNil() {
		prevVal = prevVal.Elem()
	}

	t := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := t.Field(i)
		tag := fieldType.Tag.Get("tfsdk")

		if tag == "istio" {
			continue
		}

		prevField := reflect.Zero(field.Type())
		if prevVal.IsValid() && i < prevVal.NumField() {
			prevField = prevVal.Field(i)
		}

		if reflect.DeepEqual(field.Interface(), prevField.Interface()) {
			continue
		}

		switch field.Kind() {
		case reflect.Struct:
			if field.Type() == reflect.TypeOf(types.Bool{}) {
				curr := field.Interface().(types.Bool)

				var prev types.Bool
				prevValid := false

				if prevField.IsValid() && prevField.Type() == reflect.TypeOf(types.Bool{}) {
					prev = prevField.Interface().(types.Bool)
					prevValid = true
				}

				if prevValid && prev.ValueBool() && (curr.IsNull() || !curr.ValueBool()) {
					checkFunc(tag, types.BoolValue(false), true)
				} else if !curr.IsNull() && curr.ValueBool() {
					checkFunc(tag, curr, true)
				}
			}
		case reflect.Ptr:
			var currInstalled, prevInstalled types.Bool
			currValid, prevValid := false, false

			if !field.IsNil() {
				elem := field.Elem()
				installed := elem.FieldByName("Installed")
				if installed.IsValid() && installed.Type() == reflect.TypeOf(types.Bool{}) {
					currInstalled = installed.Interface().(types.Bool)
					currValid = true
				}
			}

			if prevVal.IsValid() && i < prevVal.NumField() {
				prevField := prevVal.Field(i)
				if !prevField.IsNil() {
					elem := prevField.Elem()
					installed := elem.FieldByName("Installed")
					if installed.IsValid() && installed.Type() == reflect.TypeOf(types.Bool{}) {
						prevInstalled = installed.Interface().(types.Bool)
						prevValid = true
					}
				}
			}

			if prevValid && prevInstalled.ValueBool() && (!currValid || !currInstalled.ValueBool()) {
				checkFunc(tag, types.BoolValue(false), true)
			} else if currValid && currInstalled.ValueBool() {
				checkFunc(tag, currInstalled, true)
			}
		}
	}
}
