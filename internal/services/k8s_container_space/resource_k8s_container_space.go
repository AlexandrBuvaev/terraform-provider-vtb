package k8scontainerspace

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"regexp"
	"strings"

	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/common"
	"terraform-provider-vtb/internal/consts"
	"terraform-provider-vtb/internal/custommodifires"
	"terraform-provider-vtb/internal/customvalidators"
	k8scluster "terraform-provider-vtb/internal/services/k8s_cluster"
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource               = &K8sContainerSpaceResource{}
	_ resource.ResourceWithModifyPlan = &K8sContainerSpaceResource{}
)

type K8sContainerSpaceResource struct {
	client *client.CloudClient
}

func NewK8sContainerSpaceResource() resource.Resource {
	return &K8sContainerSpaceResource{}
}

func (r K8sContainerSpaceResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_k8scontainer_space"
}

func (r *K8sContainerSpaceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type K8sContainerSpaceModel struct {
	OrderID          types.String `tfsdk:"order_id"`
	ItemID           types.String `tfsdk:"item_id"`
	FinancialProject types.String `tfsdk:"financial_project"`
	Lifetime         types.Int64  `tfsdk:"lifetime"`
	Label            types.String `tfsdk:"label"`

	Name         types.String                       `tfsdk:"name"`
	Domain       types.String                       `tfsdk:"domain"`
	Platform     types.String                       `tfsdk:"platform"`
	DataCenter   types.String                       `tfsdk:"data_center"`
	NetSegment   types.String                       `tfsdk:"net_segment"`
	ControlPlane K8sContainerSpaceControlPlaneModel `tfsdk:"control_plane"`
	Region       k8scluster.K8sClusterRegionModel   `tfsdk:"region"`
	Ingress      k8scluster.K8sClusterIngressModel  `tfsdk:"ingress"`
}

type K8sContainerSpaceControlPlaneModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (t K8sContainerSpaceResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "VTB.Cloud ресурс Пространство контейнеров Kubernetes",
		Attributes: map[string]schema.Attribute{
			"order_id": schema.StringAttribute{
				MarkdownDescription: "Идентификатор заказа. Становится известен после создания заказа",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"item_id": schema.StringAttribute{
				MarkdownDescription: "Идентификатор пространства контейнеров в заказе. Становится известен после создания заказа",
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
				MarkdownDescription: "Имя пространства контейнеров Kubernetes. Становится известно после создания заказа",
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
			"control_plane": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Контрольная панель",
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Имя контрольной панели",
						Required:            true,
					},
					"id": schema.StringAttribute{
						MarkdownDescription: "Идентификатор контрольной панели. Определяется в процессе заказа",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"region": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Регион рабочих узлов",
				Attributes: map[string]schema.Attribute{
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
						Optional:            true,
						MarkdownDescription: "Коды ИС",
						ElementType:         types.StringType,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
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
						Required:            true,
						MarkdownDescription: "Отношение лимитов к реквестам по цпу для контейнера",
					},
					"container_memory_ratio": schema.Int64Attribute{
						Required:            true,
						MarkdownDescription: "Отношение лимитов к реквестам по памяти для контейнера",
					},
					"pod_cpu_max": schema.Int64Attribute{
						Required:            true,
						MarkdownDescription: "Максимальный размер пода по цпу",
					},
					"pod_memory_max": schema.Int64Attribute{
						Required:            true,
						MarkdownDescription: "Максимальный размер пода по памяти",
					},
					"components": schema.SingleNestedAttribute{
						Optional:            true,
						MarkdownDescription: "Компоненты в регионе",
						Attributes:          k8scluster.K8sClusterRegionComponentsSchema,
					},
				},
			},
			"ingress": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Ingress",
				Attributes: map[string]schema.Attribute{
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
						Optional:            true,
						MarkdownDescription: "Код ИС",
						ElementType:         types.StringType,
						PlanModifiers: []planmodifier.List{
							listplanmodifier.UseStateForUnknown(),
						},
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
						Attributes:          k8scluster.K8sClusterIngressFeaturesSchema,
					},
				},
			},
		},
		Blocks:             map[string]schema.Block{},
		Description:        "",
		DeprecationMessage: "",
		Version:            0,
	}
}

func (r K8sContainerSpaceResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r K8sContainerSpaceResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {

	var plan K8sContainerSpaceModel
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
	}

	image, err := references.GetK8sSpaceImageData(
		r.client.Creds,
		r.client.Organization,
		r.client.Environment,
	)
	if err != nil {
		resp.Diagnostics.AddError("Get image data from references", err.Error())
		return
	}

	if image.ProductId == "" {
		resp.Diagnostics.AddError("Product ID not found", "check product id")
		return
	}

	var controlPlaneID string
	timeout := time.After(20 * time.Minute)
	loopBreak := false

	for !loopBreak {
		select {
		case <-ctx.Done():
			resp.Diagnostics.AddError("Canceled", "Operation canceled by user or context timeout")
			return

		case <-timeout:
			resp.Diagnostics.AddError(
				"Control plane is not available",
				"Check control plane availability on portal or check input values: control plane name, data center, net segment",
			)
			return

		default:
			nodesRegion := plan.Region.Size.ValueInt64()
			nodesIngress := plan.Ingress.Size.ValueInt64()
			CPU := nodesRegion*plan.Region.Flavor.Cores.ValueInt64() +
				nodesIngress*plan.Ingress.Flavor.Cores.ValueInt64()
			memory := nodesRegion*plan.Region.Flavor.Memory.ValueInt64() +
				nodesIngress*plan.Ingress.Flavor.Memory.ValueInt64()

			controlPlanes, err := sources.GetK8sControlPlane(
				r.client.Creds.AccessToken,
				r.client.ProjectName,
				plan.DataCenter.ValueString(),
				plan.NetSegment.ValueString(),
				nodesRegion+nodesIngress,
				CPU,
				memory,
			)
			if err != nil {
				resp.Diagnostics.AddError("Can't get control planes", err.Error())
				return
			}

			controlPlaneAvailable := false
			for _, cp := range controlPlanes {
				if cp.Name == plan.ControlPlane.Name.ValueString() {
					controlPlaneID = cp.ID
					controlPlaneAvailable = true
					loopBreak = true
					break
				}
			}

			if !controlPlaneAvailable {
				time.Sleep(1 * time.Minute)
			}
		}
	}

	envConfig, err := sources.GetK8sEnvConfig(
		r.client.Creds,
		plan.NetSegment.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Get ratio data from portal", err.Error())
		return
	}

	k8sSpaceAttrs := orders.K8sContainerSpaceAttrs{
		Ratio: entities.RatioK8sContainerSpace{
			ID:                   envConfig[0].ID,
			ContainerCPURatio:    envConfig[0].Data.ContainerCPURatio,
			ContainerMemoryRatio: envConfig[0].Data.ContainerMemoryRatio,
		},
		Domain: plan.Domain.ValueString(),
		Worker: entities.WorkerK8sContainerSpace{
			Size: setRegionIngressSize(plan.Region.Size, image.RegionIngressAddMax),
			Flavor: entities.K8sConfigNodeFlavor{
				Cpus:   plan.Region.Flavor.Cores.ValueInt64(),
				Memory: plan.Region.Flavor.Memory.ValueInt64(),
				Name:   plan.Region.Flavor.Name.ValueString(),
				UUID:   plan.Region.Flavor.UUID.ValueString(),
			},
			PodCPUMax:            plan.Region.PodCPUMax.ValueInt64(),
			PodMemoryMax:         plan.Region.PodMemoryMax.ValueInt64(),
			ContainerCPURatio:    plan.Region.ContainerCPURatio.ValueInt64(),
			ContainerMemoryRatio: plan.Region.ContainerMemoryRatio.ValueInt64(),
		},
		Platform:   plan.Platform.ValueString(),
		DataCenter: plan.DataCenter.ValueString(),
		DefaultNic: entities.DefaultNic{
			NetSegment: plan.NetSegment.ValueString(),
		},
		ControlPlane: entities.ControlPlaneK8sContainerSpace{
			ID:   controlPlaneID,
			Name: plan.ControlPlane.Name.ValueString(),
		},
		IngressEnabled: true,
		Ingress: entities.IngressK8sContainerSpace{
			Size: setRegionIngressSize(plan.Ingress.Size, image.RegionIngressAddMax),
			Flavor: entities.K8sConfigNodeFlavor{
				Cpus:   plan.Ingress.Flavor.Cores.ValueInt64(),
				Memory: plan.Ingress.Flavor.Memory.ValueInt64(),
				Name:   plan.Ingress.Flavor.Name.ValueString(),
				UUID:   plan.Ingress.Flavor.UUID.ValueString(),
			},
		},
		CreatedWithOpenTofu: true,
	}

	order := orders.NewK8sContainerSpaceOrder(
		r.client.Creds,
		r.client.ProjectName,
		image.ProductId,
		k8sSpaceAttrs,
	)

	err = order.Create(orders.CreateOrderPayload{
		Label:        plan.Label.ValueString(),
		Lifetime:     int(plan.Lifetime.ValueInt64()),
		FinProjectID: finProj.ID,
	})

	if err != nil {
		resp.Diagnostics.AddError("Can't create k8s container space ", err.Error())
		return
	}

	err = order.Sync()
	if err != nil {
		diags.AddError("Can't create k8s container space: ", err.Error())
		resp.Diagnostics.Append(diags...)
		return
	}

	orderItem, err := order.GetParentItem()
	if err != nil {
		diags.AddError("Error while update state: ", err.Error())
		resp.Diagnostics.Append(diags...)
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(diags...)

	itemData := orderItem.Data
	itemDataConfig := itemData.Config.(entities.K8sContainerSpaceConfig)
	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(orderItem.ID)
	plan.Name = types.StringValue(itemDataConfig.Name)
	plan.ControlPlane.ID = types.StringValue(itemDataConfig.ControlPlane.ID)
	plan.Region.Name = types.StringValue(itemDataConfig.Region.Name)
	plan.Ingress.Name = types.StringValue(itemDataConfig.Ingress.Name)

	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	config := orders.CommonActionParams{
		Platform:            plan.Platform.ValueString(),
		NetSegment:          plan.NetSegment.ValueString(),
		DataCenter:          plan.DataCenter.ValueString(),
		RegionAddNodesMax:   image.RegionAddNodesMax,
		IngressAddNodesMax:  image.IngressAddNodesMax,
		RegionIngressAddMax: image.RegionIngressAddMax,
	}

	stateRegion := k8scluster.ConvertRegionToModel(itemDataConfig.Region, false)
	if plan.Region.Size.ValueInt64() > stateRegion.Size.ValueInt64() {
		newNodesCount := plan.Region.Size.ValueInt64() - stateRegion.Size.ValueInt64()
		if err := order.K8sContainerSpaceAddNodes(
			k8scluster.ConvertModelToRegion(plan.Region).Name,
			newNodesCount, config, "region",
		); err != nil {
			diags.AddError(
				consts.CREATE_RES_FAIL,
				fmt.Sprintf("Resize region ended with error.\nError message: %s", err.Error()),
			)
			return
		}
		if err = order.K8sContainerSpaceConfigureRegionIngress(
			"uncordon", "region",
			k8scluster.ConvertModelToRegion(plan.Region).Name,
		); err != nil {
			diags.AddError(
				consts.CREATE_RES_FAIL,
				fmt.Sprintf("Uncordon nodes ended with error.\nError message: %s", err.Error()),
			)
			return
		}
	}

	resp.Diagnostics.Append(r.handleRegionComponentsModifications(plan.Region, stateRegion, order)...)
	if resp.Diagnostics.HasError() {
		return
	}

	stateIngress := k8scluster.ConvertIngressToModel(itemDataConfig.Ingress, false)
	if plan.Ingress.Size.ValueInt64() > stateIngress.Size.ValueInt64() {
		newNodesCount := plan.Ingress.Size.ValueInt64() - stateIngress.Size.ValueInt64()
		if err := order.K8sContainerSpaceAddNodes(
			k8scluster.ConvertModelToIngress(plan.Ingress).Name,
			newNodesCount, config, "ingress",
		); err != nil {
			diags.AddError(
				consts.CREATE_RES_FAIL,
				fmt.Sprintf("Configure ingress ended with error.\nError message: %s", err.Error()),
			)
			return
		}
	}

	resp.Diagnostics.Append(r.handleIngressSettingsModifications(plan.Ingress, order)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r K8sContainerSpaceResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state K8sContainerSpaceModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	order, err := orders.GetK8sContainerSpaceOrder(
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

func (r K8sContainerSpaceResource) Read(
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

	order, err := orders.GetK8sContainerSpaceOrder(
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

	item, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get k8s container space item from portal with order_id '%s'.\nError:%s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}

	config := item.Data.Config.(entities.K8sContainerSpaceConfig)

	state := K8sContainerSpaceModel{
		OrderID:          orderID,
		ItemID:           types.StringValue(item.ID),
		FinancialProject: types.StringValue(order.FinancialSource.Name),
		Label:            types.StringValue(order.Label),
		Name:             types.StringValue(config.Name),
		Domain:           types.StringValue(config.Domain),
		Platform:         types.StringValue(config.Platform),
		DataCenter:       types.StringValue(order.Attrs.DataCenter),
		NetSegment:       types.StringValue(config.NetSegment),
		ControlPlane: K8sContainerSpaceControlPlaneModel{
			Name: types.StringValue(config.ControlPlane.Name),
			ID:   types.StringValue(config.ControlPlane.ID),
		},
		Region:  k8scluster.ConvertRegionToModel(config.Region, false),
		Ingress: k8scluster.ConvertIngressToModel(config.Ingress, false),
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

func (r K8sContainerSpaceResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan K8sContainerSpaceModel
	var state K8sContainerSpaceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetK8sContainerSpaceOrder(
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

	image, err := references.GetK8sSpaceImageData(
		r.client.Creds,
		r.client.Organization,
		r.client.Environment,
	)

	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("get k8s container space image data"), consts.UPDATE_RES_FAIL, err.Error())
		return
	}

	labelChanged := !plan.Label.Equal(state.Label)
	financialProjectChanged := !plan.FinancialProject.Equal(state.FinancialProject)
	regionChanged := !reflect.DeepEqual(plan.Region, state.Region)
	ingressChanged := !reflect.DeepEqual(plan.Ingress, state.Ingress)

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

	spaceItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get k8s container space item from portal with order_id '%s'.\nError: %s",
				order.ID, err.Error(),
			),
		)
		return
	}

	itemDataConfig, ok := spaceItem.Data.Config.(entities.K8sContainerSpaceConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Item data type conversion error",
		)
		return
	}

	ratio, err := references.GetK8sClusterRatioData(
		r.client.Creds,
		plan.NetSegment.ValueString(),
		utils.GetMinorVersion(itemDataConfig.ControlPlane.ProductVersion),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Can't get data from references",
			fmt.Sprintf(
				"Can't get ratio data from references\nError message: %s", err.Error(),
			),
		)
		return
	}

	if regionChanged {
		resp.Diagnostics.Append(r.handleRegionModifications(plan.Region, state.Region, order, config, ratio)...)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.State.SetAttribute(ctx, path.Root("region"), plan.Region)
	}

	if ingressChanged {
		resp.Diagnostics.Append(r.handleIngressModifications(plan.Ingress, state.Ingress, order, config)...)
		if resp.Diagnostics.HasError() {
			return
		}
		resp.State.SetAttribute(ctx, path.Root("ingress"), plan.Ingress)
	}
}

func (r K8sContainerSpaceResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {

	var plan, state K8sContainerSpaceModel

	if req.Plan.Raw.IsNull() {
		return
	}
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	}

	r.NetSegmentModifyPlan(ctx, &plan, &state, resp)
	r.DataCenterModifyPlan(ctx, &plan, &state, resp)
	r.DomainModifyPlan(ctx, &plan, resp)

	checkOrderIsDeleted := utils.CheckOrderIsDeleted(r.client, plan.OrderID, "container_space", "kubernetes")
	resp.Diagnostics.Append(checkOrderIsDeleted.Diagnostics...)

	utils.HandleExternalDeletionRecreate(ctx, resp, &plan, checkOrderIsDeleted.IsDeleted, checkOrderIsDeleted.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r K8sContainerSpaceResource) NetSegmentModifyPlan(
	ctx context.Context,
	plan,
	state *K8sContainerSpaceModel,
	resp *resource.ModifyPlanResponse,
) {
	netSegments, err := sources.GetNetSegments(
		r.client.Creds,
		r.client.ProjectName,
	)
	if err != nil {
		resp.Diagnostics.AddError("Get network segments data from portal", err.Error())
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
			"NetSegment is not available",
			fmt.Sprintf(
				"`net_segment=%s` not found.\nAvailable network segments is [%s]",
				plan.NetSegment.ValueString(),
				strings.Join(segmentsCodes, ", "),
			),
		)
	}
}

func (r K8sContainerSpaceResource) DataCenterModifyPlan(
	ctx context.Context,
	plan,
	state *K8sContainerSpaceModel,
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
		resp.Diagnostics.AddError("Get data centers data from portal", err.Error())
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
			"DataCenter is not available",
			fmt.Sprintf(
				"You can't choose data center=`%s`"+
					"\nAvailable data centers: %v",
				plan.DataCenter.ValueString(),
				strings.Join(dcCodes, ", "),
			),
		)
	}
}

func (r K8sContainerSpaceResource) DomainModifyPlan(
	ctx context.Context,
	plan *K8sContainerSpaceModel,
	resp *resource.ModifyPlanResponse,
) {

	availableDomains, err := sources.GetDomainsByProjectName(r.client.Creds, r.client.ProjectName)
	if err != nil {
		resp.Diagnostics.AddError(
			"Get domains", err.Error(),
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
			"Domain is not available",
			fmt.Sprintf("Available domains [%s].",
				strings.Join(domains, ", ")),
		)
		return
	}
}

func (r K8sContainerSpaceResource) handleRegionModifications(
	planRegion, stateRegion k8scluster.K8sClusterRegionModel,
	order *orders.K8sContainerSpaceOrder,
	config orders.CommonActionParams,
	ratio *references.K8sClusterRatioData,
) (diags diag.Diagnostics) {

	modifications := k8scluster.RegionHasChanged(planRegion, stateRegion)
	actions := []regionsAction{}
	for _, modify := range modifications {
		collectRegionActions(order, &actions, modify, config, ratio)
	}
	for _, act := range actions {
		if err := act.action(act.region); err != nil {
			diags.AddError(
				"Configure region",
				fmt.Sprintf("%s ended with error.\nError message: %s", act.errMsg, err.Error()),
			)
			return
		}
	}
	return
}

func (r K8sContainerSpaceResource) handleIngressModifications(
	planIngress, stateIngress k8scluster.K8sClusterIngressModel,
	order *orders.K8sContainerSpaceOrder,
	config orders.CommonActionParams,
) (diags diag.Diagnostics) {

	modifications := k8scluster.IngressHasChanged(planIngress, stateIngress)
	actions := []ingressAction{}
	for _, modify := range modifications {
		collectIngressActions(order, &actions, modify, config)
	}
	for _, act := range actions {
		if err := act.action(act.ingress); err != nil {
			diags.AddError(
				"Configure ingress",
				fmt.Sprintf("%s ended with error.\nError message: %s", act.errMsg, err.Error()),
			)
			return
		}
	}
	return
}

func (r K8sContainerSpaceResource) handleRegionComponentsModifications(
	planRegion, stateRegion k8scluster.K8sClusterRegionModel,
	order *orders.K8sContainerSpaceOrder,
) (diags diag.Diagnostics) {

	modifications := k8scluster.DiffRegionComponents(planRegion, &stateRegion)
	if len(modifications) > 0 {
		for _, m := range modifications {
			modified := m
			region := modified.Component.(k8scluster.K8sClusterRegionModel)
			if modified.Change == "configure" {
				if err := order.K8sContainerSpaceRegionConfigureComponents(
					k8scluster.ConvertModelToRegion(region),
					fmt.Sprintf("%s_%s_%s", "region", modified.Change, modified.Action),
				); err != nil {
					diags.AddError(
						consts.CREATE_RES_FAIL,
						fmt.Sprintf("Configure region ended with error.\nError message: %s", err.Error()),
					)
					return
				}
			}
		}
	}
	return
}

func (r K8sContainerSpaceResource) handleIngressSettingsModifications(
	planIngress k8scluster.K8sClusterIngressModel,
	order *orders.K8sContainerSpaceOrder,
) (diags diag.Diagnostics) {

	if k8scluster.IsIngressFeaturesChangedFromDefault(planIngress) || len(planIngress.TcpUdpSettings) > 0 {
		if err := order.K8sContainerSpaceIngressSettings(
			k8scluster.ConvertModelToIngress(planIngress),
		); err != nil {
			diags.AddError(
				consts.CREATE_RES_FAIL,
				fmt.Sprintf("Configure ingress settings ended with error.\nError message: %s", err.Error()),
			)
			return
		}
	}
	return
}

type regionsAction struct {
	region k8scluster.K8sClusterRegionModel
	action func(region k8scluster.K8sClusterRegionModel) error
	errMsg string
}

type ingressAction struct {
	ingress k8scluster.K8sClusterIngressModel
	action  func(ingress k8scluster.K8sClusterIngressModel) error
	errMsg  string
}

func collectRegionActions(
	order *orders.K8sContainerSpaceOrder,
	actions *[]regionsAction,
	modified k8scluster.ComponentModification,
	config orders.CommonActionParams,
	ratio *references.K8sClusterRatioData,
) {
	component := "region"
	region := modified.Component.(k8scluster.K8sClusterRegionModel)
	switch modified.Change {
	case "flavor":
		*actions = append(*actions, regionsAction{
			region: region,
			action: func(region k8scluster.K8sClusterRegionModel) error {
				return order.K8sContainerSpaceRegionIngressChangeFlavor(k8scluster.ConvertModelToRegion(modified.Component.(k8scluster.K8sClusterRegionModel)), component)
			},
			errMsg: fmt.Sprintf("change %s %s", component, modified.Change),
		})
	case "nodes":
		if modified.Action == "add" {
			*actions = append(*actions, regionsAction{
				region: region,
				action: func(region k8scluster.K8sClusterRegionModel) error {
					if err := order.K8sContainerSpaceAddNodes(region.Name.ValueString(), modified.Count, config, component); err != nil {
						return err
					}
					return order.K8sContainerSpaceConfigureRegionIngress("uncordon", component, region.Name.ValueString())
				},
				errMsg: fmt.Sprintf("%s %s", modified.Action, modified.Change),
			})
		}
	case "requests_ratio_config":
		*actions = append(*actions, regionsAction{
			region: region,
			action: func(region k8scluster.K8sClusterRegionModel) error {
				return order.K8sContainerSpaceRegionRequestsRatioConfig(
					k8scluster.ConvertModelToRegion(region),
					fmt.Sprintf("%s_%s", component, modified.Change),
					ratio, config,
				)
			},
			errMsg: fmt.Sprintf("%s %s", modified.Action, modified.Change),
		})
	case "configure":
		*actions = append(*actions, regionsAction{
			region: region,
			action: func(region k8scluster.K8sClusterRegionModel) error {
				return order.K8sContainerSpaceRegionConfigureComponents(
					k8scluster.ConvertModelToRegion(region),
					fmt.Sprintf("%s_%s_%s", component, modified.Change, modified.Action),
				)
			},
			errMsg: fmt.Sprintf("%s %s", component, modified.Change),
		})
	}
}

func collectIngressActions(
	order *orders.K8sContainerSpaceOrder,
	actions *[]ingressAction,
	modified k8scluster.ComponentModification,
	config orders.CommonActionParams,
) {
	component := "ingress"
	ingress := modified.Component.(k8scluster.K8sClusterIngressModel)
	switch modified.Change {
	case "flavor":
		*actions = append(*actions, ingressAction{
			ingress: ingress,
			action: func(ingress k8scluster.K8sClusterIngressModel) error {
				return order.K8sContainerSpaceRegionIngressChangeFlavor(k8scluster.ConvertModelToIngress(modified.Component.(k8scluster.K8sClusterIngressModel)), component)
			},
			errMsg: fmt.Sprintf("change %s %s", component, modified.Change),
		})
	case "nodes":
		if modified.Action == "add" {
			*actions = append(*actions, ingressAction{
				ingress: ingress,
				action: func(ingress k8scluster.K8sClusterIngressModel) error {
					return order.K8sContainerSpaceAddNodes(ingress.Name.ValueString(), modified.Count, config, component)
				},
				errMsg: fmt.Sprintf("%s %s %s", modified.Action, component, modified.Change),
			})
		}
	case "settings":
		*actions = append(*actions, ingressAction{
			ingress: ingress,
			action: func(ingress k8scluster.K8sClusterIngressModel) error {
				return order.K8sContainerSpaceIngressSettings(
					k8scluster.ConvertModelToIngress(ingress),
				)
			},
			errMsg: fmt.Sprintf("configure %s %s", component, modified.Change),
		})
	}
}

func setRegionIngressSize(size types.Int64, limit int64) int64 {
	if size.ValueInt64() > limit {
		return limit
	}
	return size.ValueInt64()
}
