package k8sproject

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"time"

	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/consts"
	"terraform-provider-vtb/internal/custommodifires"
	"terraform-provider-vtb/internal/utils"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"
	"terraform-provider-vtb/pkg/client/references"
	"terraform-provider-vtb/pkg/client/sources"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                = &K8sProjectResource{}
	_ resource.ResourceWithModifyPlan  = &K8sProjectResource{}
	_ resource.ResourceWithImportState = &K8sProjectResource{}
)

type K8sProjectResource struct {
	client *client.CloudClient
}

func NewK8sProjectResource() resource.Resource {
	return &K8sProjectResource{}
}

func (r K8sProjectResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_k8sproject_instance"
}

func (r *K8sProjectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type K8sProjectQuotaData struct {
	CPU    types.Int64 `tfsdk:"cpu"`
	Memory types.Int64 `tfsdk:"memory"`
}

type K8sProjectFullComponentData struct {
	Roles []entities.RolesK8sProject `tfsdk:"roles"`
}

type K8sProjectIstioData struct {
	ControlPlane types.String               `tfsdk:"control_plane"`
	Roles        []entities.RolesK8sProject `tfsdk:"roles"`
}

type K8sProjectComponentData struct {
	Namespace basetypes.StringValue `tfsdk:"namespace"`
}

type K8sProjectOmniData struct {
	AppName    string                `tfsdk:"app_name"`
	ClientName string                `tfsdk:"client_name"`
	Name       basetypes.StringValue `tfsdk:"name"`
}

type K8sProjectPoolData struct {
	ID          types.String `tfsdk:"id"`
	NameCluster types.String `tfsdk:"name_cluster"`
}

type K8sProjectModel struct {
	Label            types.String        `tfsdk:"label"`
	OrderID          types.String        `tfsdk:"order_id"`
	ItemID           types.String        `tfsdk:"item_id"`
	Access           map[string][]string `tfsdk:"access"`
	FinancialProject types.String        `tfsdk:"financial_project"`
	Lifetime         types.Int64         `tfsdk:"lifetime"`

	NetSegment       types.String                 `tfsdk:"net_segment"`
	ProjectName      types.String                 `tfsdk:"project_name"`
	FullProjectName  types.String                 `tfsdk:"full_project_name"`
	Ingress          types.String                 `tfsdk:"ingress"`
	Quota            K8sProjectQuotaData          `tfsdk:"quota"`
	Region           types.String                 `tfsdk:"region"`
	DataCenter       types.String                 `tfsdk:"data_center"`
	ClusterName      types.String                 `tfsdk:"cluster_name"`
	ClusterID        types.String                 `tfsdk:"cluster_id"`
	Istio            *K8sProjectIstioData         `tfsdk:"istio"`
	Tyk              *K8sProjectFullComponentData `tfsdk:"tyk"`
	TslgOperator     *K8sProjectFullComponentData `tfsdk:"tslg_operator"`
	TsamOperator     *K8sProjectFullComponentData `tfsdk:"tsam_operator"`
	TsdsOperator     *K8sProjectComponentData     `tfsdk:"tsds_operator"`
	OmniCertificates []K8sProjectOmniData         `tfsdk:"omni_certificates"`
	ChaosMesh        *K8sProjectComponentData     `tfsdk:"chaos_mesh"`
}

func (t K8sProjectResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "VTB.Cloud ресурс Kubernetes project",
		Attributes: map[string]schema.Attribute{
			"order_id": schema.StringAttribute{
				MarkdownDescription: "Идентификатор заказа. Становится известен после создания заказа",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"item_id": schema.StringAttribute{
				MarkdownDescription: "Идентификатор проекта в заказе. Становится известен после создания заказа",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_name": schema.StringAttribute{
				MarkdownDescription: "Пользовательское имя проекта",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(4, 32),
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$`),
						"must contain only lowercase alphanumeric characters",
					),
				},
			},
			"full_project_name": schema.StringAttribute{
				MarkdownDescription: "Полное имя проекта с именем кластера",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ingress": schema.StringAttribute{
				MarkdownDescription: "Название Ingress кластера Kubernetes",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"quota": schema.SingleNestedAttribute{
				MarkdownDescription: "Квота проекта",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"cpu": schema.Int64Attribute{
						MarkdownDescription: "Количество ЦПУ",
						Required:            true,
						Validators: []validator.Int64{
							int64validator.AtMost(1024),
							int64validator.AtLeast(1),
						},
					},
					"memory": schema.Int64Attribute{
						MarkdownDescription: "Память (GB)",
						Required:            true,
						Validators: []validator.Int64{
							int64validator.AtMost(1024),
							int64validator.AtLeast(1),
						},
					},
				},
			},
			"access": schema.MapAttribute{
				MarkdownDescription: "Права. Словарь, где ключом является роль, а значением - список групп, которым назначается эта роль",
				Required:            true,
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
			},
			"net_segment": schema.StringAttribute{
				MarkdownDescription: "Сетевой сегмент размещения заказа",
				Required:            true,
			},
			"data_center": schema.StringAttribute{
				MarkdownDescription: "Дата-центр размещения заказа",
				Required:            true,
			},
			"cluster_id": schema.StringAttribute{
				MarkdownDescription: "Идентификатор кластера Kubernetes. Определяется в процессе заказа",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_name": schema.StringAttribute{
				MarkdownDescription: "Имя кластера Kubernetes",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "Название региона в кластере Kubernetes",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"label": schema.StringAttribute{
				MarkdownDescription: "Метка заказа",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(3, 64),
				},
			},
			"istio": &schema.SingleNestedAttribute{
				MarkdownDescription: "Компонент Istio",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"roles": schema.ListNestedAttribute{
						MarkdownDescription: "Права. Словарь, где ключом является роль, а значением - список групп, которым назначается эта роль",
						Required:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"role": schema.StringAttribute{
									MarkdownDescription: "Название роли",
									Optional:            true,
									Validators: []validator.String{
										stringvalidator.RegexMatches(regexp.MustCompile(`\S+`), "must not be empty"),
									},
								},
								"groups": schema.ListAttribute{
									MarkdownDescription: "Список групп, которым назначена роль",
									ElementType:         types.StringType,
									Optional:            true,
									Validators: []validator.List{
										listvalidator.SizeAtLeast(1),
									},
								},
							},
						},
					},
					"control_plane": schema.StringAttribute{
						MarkdownDescription: "Название контрольной панели Istio",
						Required:            true,
					},
				},
			},
			"tyk": &schema.SingleNestedAttribute{
				MarkdownDescription: "Компонент TYK",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"roles": schema.ListNestedAttribute{
						MarkdownDescription: "Права. Словарь, где ключом является роль, а значением - список групп, которым назначается эта роль",
						Required:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"role": schema.StringAttribute{
									MarkdownDescription: "Название роли",
									Optional:            true,
									Validators: []validator.String{
										stringvalidator.RegexMatches(regexp.MustCompile(`\S+`), "must not be empty"),
									},
								},
								"groups": schema.ListAttribute{
									MarkdownDescription: "Список групп, которым назначена роль",
									ElementType:         types.StringType,
									Optional:            true,
									Validators: []validator.List{
										listvalidator.SizeAtLeast(1),
									},
								},
							},
						},
					},
				},
			},
			"tslg_operator": &schema.SingleNestedAttribute{
				MarkdownDescription: "Компонент TSLG",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"roles": schema.ListNestedAttribute{
						MarkdownDescription: "Права. Словарь, где ключом является роль, а значением - список групп, которым назначается эта роль",
						Required:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"role": schema.StringAttribute{
									MarkdownDescription: "Название роли",
									Optional:            true,
									Validators: []validator.String{
										stringvalidator.RegexMatches(regexp.MustCompile(`\S+`), "must not be empty"),
									},
								},
								"groups": schema.ListAttribute{
									MarkdownDescription: "Список групп, которым назначена роль",
									ElementType:         types.StringType,
									Optional:            true,
									Validators: []validator.List{
										listvalidator.SizeAtLeast(1),
									},
								},
							},
						},
					},
				},
			},
			"tsam_operator": &schema.SingleNestedAttribute{
				MarkdownDescription: "Компонент TSAM",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"roles": schema.ListNestedAttribute{
						MarkdownDescription: "Права. Словарь, где ключом является роль, а значением - список групп, которым назначается эта роль",
						Required:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"role": schema.StringAttribute{
									MarkdownDescription: "Название роли",
									Optional:            true,
									Validators: []validator.String{
										stringvalidator.RegexMatches(regexp.MustCompile(`\S+`), "must not be empty"),
									},
								},
								"groups": schema.ListAttribute{
									MarkdownDescription: "Список групп, которым назначена роль",
									ElementType:         types.StringType,
									Optional:            true,
									Validators: []validator.List{
										listvalidator.SizeAtLeast(1),
									},
								},
							},
						},
					},
				},
			},
			"tsds_operator": &schema.SingleNestedAttribute{
				MarkdownDescription: "Компонент TSDS",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"namespace": schema.StringAttribute{
						MarkdownDescription: "Namespace компонента. Передается дефолтное значение.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
			},
			"omni_certificates": &schema.ListNestedAttribute{
				MarkdownDescription: "Список сертификатов OMNI",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"app_name": schema.StringAttribute{
							MarkdownDescription: "Название приложения, для которого выпущен сертификат",
							Required:            true,
							PlanModifiers: []planmodifier.String{
								custommodifires.IgnoreChangesModifier{},
							},
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Название сертификата",
							Required:            true,
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"client_name": schema.StringAttribute{
							MarkdownDescription: "Пользовательское имя сертификата. Используется как суффикс в CN сертификата",
							Required:            true,
							PlanModifiers: []planmodifier.String{
								custommodifires.IgnoreChangesModifier{},
							},
							Validators: []validator.String{
								stringvalidator.LengthBetween(1, 10),
								stringvalidator.RegexMatches(
									regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-.]*$`),
									"must start with a letter or digit and can contain only letters, digits, dots, and hyphens",
								),
							},
						},
					},
				},
			},
			"chaos_mesh": &schema.SingleNestedAttribute{
				MarkdownDescription: "Компонент Chaos Mesh",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"namespace": schema.StringAttribute{
						MarkdownDescription: "Namespace компонента. Передается дефолтное значение.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
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
		},
	}
}

func (r K8sProjectResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {

	var plan K8sProjectModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	var Roles []entities.RolesK8sProject
	for role, groups := range plan.Access {
		Roles = append(Roles, entities.RolesK8sProject{
			Role:   role,
			Groups: groups,
		})
	}

	image, err := references.GetK8sProjectImageData(
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

	var clusterID string
	timeout := time.After(20 * time.Minute)
	loopBreak := false

	for !loopBreak {
		select {
		case <-ctx.Done():
			resp.Diagnostics.AddError("Canceled", "Operation canceled by user or context timeout")
			return

		case <-timeout:
			resp.Diagnostics.AddError(
				"Cluster or region is not available",
				"Check cluster availability on portal or check cluster and region name input value",
			)
			return

		default:
			clusters, err := sources.GetK8sClusters(
				r.client.Creds.AccessToken,
				r.client.ProjectName,
				plan.Quota.CPU.ValueInt64(),
				plan.Quota.Memory.ValueInt64(),
				plan.DataCenter.ValueString(),
				plan.NetSegment.ValueString(),
			)
			if err != nil {
				resp.Diagnostics.AddError("Can't get clusters", err.Error())
				return
			}

			clusterMatched := false

			for _, cluster := range clusters {
				if cluster.Name == plan.ClusterName.ValueString() {
					if cluster.Region == plan.Region.ValueString() {
						clusterID = cluster.ID
						clusterMatched = true
						loopBreak = true
						break
					}
				}
			}

			if !clusterMatched {
				time.Sleep(1 * time.Minute)
			}
		}
	}

	finProj, err := sources.GetFinancialProjectByName(
		r.client.Creds,
		r.client.ProjectName,
		plan.FinancialProject.ValueString(),
	)

	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("financial_project"), consts.CREATE_RES_FAIL, err.Error())
	}

	k8sProjectAttrs := orders.K8sProjectAttrs{
		Quota: entities.QuotaK8sProject{
			CPU:    plan.Quota.CPU.ValueInt64(),
			Memory: plan.Quota.Memory.ValueInt64(),
		},
		Roles:      Roles,
		Ingress:    plan.Ingress.ValueString(),
		UserMark:   plan.ProjectName.ValueString(),
		DataCenter: plan.DataCenter.ValueString(),
		NetSegment: plan.NetSegment.ValueString(),
		ResourcePool: entities.ResourcePoolK8sProject{
			ID:     clusterID,
			Name:   plan.ClusterName.ValueString(),
			Region: plan.Region.ValueString(),
		},
		CreatedWithOpenTofu: true,
	}

	order := orders.NewK8sProjectOrder(
		r.client.Creds,
		r.client.ProjectName,
		image.ProductId,
		k8sProjectAttrs,
	)

	err = order.CreateOrder(orders.CreateOrderPayload{
		Label:        plan.Label.ValueString(),
		Lifetime:     int(plan.Lifetime.ValueInt64()),
		FinProjectID: finProj.ID,
	})

	if err != nil {
		resp.Diagnostics.AddError("Can't create k8s project: ", err.Error())
		return
	}

	err = order.Sync()
	if err != nil {
		diags.AddError("Can't create k8s project: ", err.Error())
		resp.Diagnostics.Append(diags...)
		return
	}

	err = order.Sync()
	if err != nil {
		diags.AddError("Error while update state: ", err.Error())
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
	config := itemData.Config.(entities.K8sProjectItemConfig)
	projectname := config.ProjectName
	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(orderItem.ID)
	plan.FullProjectName = types.StringValue(projectname)
	plan.ClusterID = types.StringValue(clusterID)

	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	components := map[string]*K8sProjectComponentData{
		"tsds_operator": plan.TsdsOperator,
		"chaos_mesh":    plan.ChaosMesh,
	}

	for name, planComponent := range components {
		if planComponent != nil {
			if err := order.K8sProjectAddComponent(ctx, name); err != nil {
				resp.Diagnostics.AddError("Add "+name, err.Error())
				return
			}
		}
	}

	if plan.OmniCertificates != nil {
		for _, omniCert := range plan.OmniCertificates {
			if !omniCert.Name.IsNull() {
				if err := order.K8sProjectAddOmniCertificate(ctx, omniCert.AppName, omniCert.ClientName, "omni_certificate"); err != nil {
					resp.Diagnostics.AddError("Add omni_certificate", err.Error())
					return
				}
			}
		}
	}

	fullComponents := map[string]*K8sProjectFullComponentData{
		"tsam_operator": plan.TsamOperator,
		"tslg_operator": plan.TslgOperator,
		"tyk":           plan.Tyk,
	}

	var component ComponentData
	for name, planComponent := range fullComponents {
		if planComponent != nil {
			component = planComponent
			if err := order.K8sProjectAddFullComponent(ctx, component, name); err != nil {
				resp.Diagnostics.AddError("Add "+name, err.Error())
				return
			}
		}
	}

	if plan.Istio != nil {
		component = plan.Istio
		if err := order.K8sProjectAddFullComponent(ctx, component, "istio"); err != nil {
			resp.Diagnostics.AddError("Add istio", err.Error())
			return
		}
	}
}

func (r K8sProjectResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state K8sProjectModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	order, err := orders.GetK8sProjectOrder(
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

	err = order.DeleteOrder()
	if err != nil {
		diags.AddError(consts.DELETE_RES_FAIL, err.Error())
		resp.Diagnostics.Append(diags...)
	}

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r K8sProjectResource) Read(
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

	order, err := orders.GetK8sProjectOrder(
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
				"Can't get k8s project item from portal with order_id '%s'.\nError:%s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}

	config := item.Data.Config.(entities.K8sProjectItemConfig)

	state := K8sProjectModel{
		Label:            types.StringValue(order.Label),
		OrderID:          orderID,
		ItemID:           types.StringValue(item.ID),
		Access:           GetAccess(config.Roles),
		FinancialProject: types.StringValue(order.FinancialSource.Name),
		NetSegment:       types.StringValue(config.Cluster.Segment),
		DataCenter:       types.StringValue(order.Attrs.DataCenter),
		ProjectName:      types.StringValue(order.Attrs.UserMark),
		FullProjectName:  types.StringValue(config.ProjectName),
		Ingress:          types.StringValue(order.Attrs.Ingress),
		Quota: K8sProjectQuotaData{
			CPU:    types.Int64Value(int64(config.Quota.CPU)),
			Memory: types.Int64Value(int64(config.Quota.Memory)),
		},
		Region:           types.StringValue(config.Region),
		ClusterName:      types.StringValue(config.Cluster.Name),
		ClusterID:        types.StringValue(config.Cluster.UUID),
		Tyk:              GetFullComponentData(config.Tyk),
		TslgOperator:     GetFullComponentData(config.TslgOperator),
		TsamOperator:     GetFullComponentData(config.TsamOperator),
		TsdsOperator:     GetComponentData(config.TsdsOperator),
		Istio:            GetIstioData(config.Istio),
		OmniCertificates: GetOmniData(config.OmniCertificates),
		ChaosMesh:        GetComponentData(config.ChaosMesh),
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

func (r K8sProjectResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r K8sProjectResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan K8sProjectModel
	var state K8sProjectModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = resp.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	order, err := orders.GetK8sProjectOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		diags.AddError("Can't get order from portal: ", err.Error())
		resp.Diagnostics.Append(diags...)
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

	labelChanged := !plan.Label.Equal(state.Label)
	quotaChanged := !reflect.DeepEqual(plan.Quota, state.Quota)
	accessChanged := !reflect.DeepEqual(plan.Access, state.Access)
	financialProjectChanged := !plan.FinancialProject.Equal(state.FinancialProject)
	componentsChanged, changes := GetChangedComponents(&plan, &state)
	tsdsChanged := IsComponentChanged(plan.TsdsOperator, state.TsdsOperator)
	omniChanged := IsComponentChanged(plan.OmniCertificates, state.OmniCertificates)
	chaosMeshChanged := IsComponentChanged(plan.ChaosMesh, state.ChaosMesh)

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
		err = order.ChangeLabel(plan.Label.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Change order label",
				fmt.Sprintf(
					"Changing order label ended with error.\nError message: %s", err.Error(),
				),
			)
			return
		}
		resp.State.SetAttribute(ctx, path.Root("label"), plan.Label)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	if quotaChanged || accessChanged {
		diags := ChangeProject(ctx, order, &plan)
		resp.Diagnostics.Append(diags...)

		if resp.Diagnostics.HasError() {
			return
		}
	}

	if tsdsChanged {
		component := "tsds_operator"
		HandleComponentAction(ctx, &plan, &state, order, resp, component)

		if resp.Diagnostics.HasError() {
			return
		}
	}

	if chaosMeshChanged {
		component := "chaos_mesh"
		HandleComponentAction(ctx, &plan, &state, order, resp, component)

		if resp.Diagnostics.HasError() {
			return
		}
	}

	if componentsChanged {
		HandleFullComponentAction(ctx, &plan, &state, changes, order, resp)

		if resp.Diagnostics.HasError() {
			return
		}
	}

	if omniChanged {
		HandleOmniCertificateAction(ctx, &plan, &state, order, resp)

		if resp.Diagnostics.HasError() {
			return
		}
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func ChangeProject(
	ctx context.Context,
	order *orders.K8sProjectOrder,
	planResource *K8sProjectModel,
) diag.Diagnostics {

	var diags diag.Diagnostics

	quotas := entities.QuotaK8sProject{
		CPU:    planResource.Quota.CPU.ValueInt64(),
		Memory: planResource.Quota.Memory.ValueInt64(),
	}

	var roles []entities.RolesK8sProject

	for role, groups := range planResource.Access {
		roles = append(roles, entities.RolesK8sProject{
			Role:   role,
			Groups: groups,
		})
	}

	err := order.UpdateK8sProject(quotas, roles)
	if err != nil {
		diags.AddError("Failed to update K8s project", err.Error())
	}

	return diags
}

func (r K8sProjectResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan K8sProjectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	var state K8sProjectModel
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	checkOrderIsDeleted := utils.CheckOrderIsDeleted(r.client, plan.OrderID, "project", "kubernetes")
	resp.Diagnostics.Append(checkOrderIsDeleted.Diagnostics...)

	utils.HandleExternalDeletionRecreate(ctx, resp, &plan, checkOrderIsDeleted.IsDeleted, checkOrderIsDeleted.Diagnostics)

	var clusterName string
	diags := req.Plan.GetAttribute(ctx, path.Root("cluster_name"), &clusterName)

	if diags.HasError() {
		return
	}

	// r.ClusterDataPlan(ctx, &plan, &state, resp)
	r.NetSegmentModifyPlan(ctx, &plan, &state, resp)
	r.DataCenterModifyPlan(ctx, &plan, &state, resp)
	r.RolesModifyPlan(ctx, &plan, &state, resp)

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)

}

func (r K8sProjectResource) ClusterDataPlan(
	ctx context.Context,
	plan,
	state *K8sProjectModel,
	resp *resource.ModifyPlanResponse,
) {
	clusters, err := sources.GetK8sClusters(
		r.client.Creds.AccessToken,
		r.client.ProjectName,
		plan.Quota.CPU.ValueInt64(),
		plan.Quota.Memory.ValueInt64(),
		plan.DataCenter.ValueString(),
		plan.NetSegment.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Can't get clusters", err.Error())
		return
	}

	clusterAvailable := false
	regionAvaiable := false
	for _, cluster := range clusters {
		if cluster.Name == plan.ClusterName.ValueString() {
			clusterAvailable = true
			if cluster.Region == plan.Region.ValueString() {
				regionAvaiable = true
			}
		}
	}

	if !regionAvaiable {
		resp.Diagnostics.AddError(
			"Region in cluster is not available",
			"Check region input value",
		)
	}

	if !clusterAvailable {
		resp.Diagnostics.AddError(
			"Cluster is not available",
			"Check cluster_name input value",
		)
	}
}

func (r K8sProjectResource) NetSegmentModifyPlan(
	ctx context.Context,
	plan,
	state *K8sProjectModel,
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

func (r K8sProjectResource) DataCenterModifyPlan(
	ctx context.Context,
	plan,
	state *K8sProjectModel,
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

func (r K8sProjectResource) RolesModifyPlan(
	ctx context.Context,
	plan,
	state *K8sProjectModel,
	resp *resource.ModifyPlanResponse,
) {
	projectEnv, _ := sources.GetK8sProjectEnviroment(
		r.client.Creds,
		r.client.ProjectName,
	)

	roles, err := sources.GetK8sRoles(r.client.Creds, projectEnv)
	if err != nil {
		resp.Diagnostics.AddError("Get roles data from portal", err.Error())
		return
	}

	var rolesNames []string
	rolesAvailable := false
	for _, role := range roles {
		rolesNames = append(rolesNames, role.Name)
	}

	for planRole := range plan.Access {
		for _, role := range rolesNames {
			if planRole == role {
				rolesAvailable = true
				break
			}
		}
		if !rolesAvailable {
			resp.Diagnostics.AddError(
				"This role is not available",
				"Check role name in input value",
			)
		}
	}

	accessGroups, err := sources.GetK8sGroups(
		r.client.Creds,
		r.client.ProjectName,
	)
	if err != nil {
		resp.Diagnostics.AddError("Get group data from portal", err.Error())
		return
	}

	var groupNames []string
	for _, group := range accessGroups {
		groupNames = append(groupNames, group.Name)
	}

	for _, planRole := range plan.Access {
		exists := false
		for planGroup := range planRole {
			for _, group := range groupNames {
				if planRole[planGroup] == group {
					exists = true
					break
				}
			}
		}
		if !exists {
			resp.Diagnostics.AddError(
				"This group is not available",
				"Check group name in input information",
			)
		}
	}
}

func GetAccess(roles []entities.RolesK8sProject) map[string][]string {
	newAccess := make(map[string][]string)

	for _, role := range roles {
		if _, ok := newAccess[role.Role]; !ok {
			newAccess[role.Role] = role.Groups
		} else {
			for _, member := range role.Groups {
				if !slices.Contains(newAccess[role.Role], member) {
					newAccess[role.Role] = append(newAccess[role.Role], member)
				}
			}
		}
	}
	return newAccess
}

func GetIstioData(component entities.K8sProjectIstioData) *K8sProjectIstioData {

	createIstioData := func(component entities.K8sProjectIstioData) *K8sProjectIstioData {
		if component.ControlPlane == "" {
			return nil
		}
		var rolesList []entities.RolesK8sProject
		for _, role := range component.Roles {
			rolesList = append(rolesList, entities.RolesK8sProject{
				Role:   role.Role,
				Groups: role.Groups,
			})
		}
		return &K8sProjectIstioData{
			ControlPlane: types.StringValue(component.ControlPlane),
			Roles:        rolesList,
		}
	}

	return createIstioData(component)
}

func GetComponentData(component entities.K8sProjectComponentData) *K8sProjectComponentData {

	createComponentData := func(component entities.K8sProjectComponentData) *K8sProjectComponentData {
		if component.Namespace == "" {
			return nil
		}

		return &K8sProjectComponentData{
			Namespace: basetypes.NewStringValue(component.Namespace),
		}
	}

	return createComponentData(component)
}

func GetFullComponentData(component entities.K8sProjectFullComponentData) *K8sProjectFullComponentData {

	createFullComponentData := func(component entities.K8sProjectFullComponentData) *K8sProjectFullComponentData {
		if component.Namespace == "" {
			return nil
		}
		rolesList := make([]entities.RolesK8sProject, 0)
		for _, role := range component.Roles {
			rolesList = append(rolesList, entities.RolesK8sProject{
				Role:   role.Role,
				Groups: role.Groups,
			})
		}
		return &K8sProjectFullComponentData{Roles: rolesList}
	}

	return createFullComponentData(component)
}

func GetOmniData(component []entities.K8sProjectOmniData) []K8sProjectOmniData {

	if len(component) == 0 {
		return nil
	}

	var omniCertificatesList []K8sProjectOmniData

	for _, cert := range component {
		omniCertificate := K8sProjectOmniData{
			AppName:    cert.AppName,
			ClientName: cert.ClientName,
			Name:       basetypes.NewStringValue(cert.Name),
		}
		omniCertificatesList = append(omniCertificatesList, omniCertificate)
	}
	return omniCertificatesList
}

func GetChangedComponents(plan, state K8sComponentData) (bool, map[string]bool) {
	changedComponents := map[string]bool{
		"tyk":           IsComponentChanged(plan.GetFullComponent("tyk"), state.GetFullComponent("tyk")),
		"tslg_operator": IsComponentChanged(plan.GetFullComponent("tslg_operator"), state.GetFullComponent("tslg_operator")),
		"tsam_operator": IsComponentChanged(plan.GetFullComponent("tsam_operator"), state.GetFullComponent("tsam_operator")),
		"istio":         IsComponentChanged(plan.GetIstio(), state.GetIstio()),
	}

	for _, changed := range changedComponents {
		if changed {
			return true, changedComponents
		}
	}

	return false, map[string]bool{}
}

type ComponentData interface {
	GetRoles() []entities.RolesK8sProject
	GetControlPlane() string
}

func (c *K8sProjectIstioData) GetRoles() []entities.RolesK8sProject {
	if c == nil {
		return nil
	}
	return c.Roles
}

func (c *K8sProjectIstioData) GetControlPlane() string {
	if c == nil {
		return ""
	}
	return c.ControlPlane.ValueString()
}

func (c *K8sProjectFullComponentData) GetRoles() []entities.RolesK8sProject {
	if c == nil {
		return nil
	}
	return c.Roles
}

func (c *K8sProjectFullComponentData) GetControlPlane() string {
	return ""
}

type K8sComponentData interface {
	GetFullComponent(component string) *K8sProjectFullComponentData
	GetComponent(component string) *K8sProjectComponentData
	GetIstio() *K8sProjectIstioData
	GetOmni() []K8sProjectOmniData
	GetProjectName() string
}

func (k *K8sProjectModel) GetFullComponent(component string) *K8sProjectFullComponentData {

	var data *K8sProjectFullComponentData
	switch component {
	case "tyk":
		data = k.Tyk
	case "tslg_operator":
		data = k.TslgOperator
	case "tsam_operator":
		data = k.TsamOperator
	}
	return data
}

func (k *K8sProjectModel) GetComponent(component string) *K8sProjectComponentData {

	var data *K8sProjectComponentData
	switch component {
	case "chaos_mesh":
		data = k.ChaosMesh
	case "tsds_operator":
		data = k.TsdsOperator
	}
	return data
}

func (k *K8sProjectModel) GetIstio() *K8sProjectIstioData {
	return k.Istio
}

func (k *K8sProjectModel) GetOmni() []K8sProjectOmniData {
	return k.OmniCertificates
}

func (k *K8sProjectModel) GetProjectName() string {
	return k.ProjectName.ValueString()
}

type OrderHandler interface {
	GetOrderID() string
	K8sProjectAddComponent(ctx context.Context, component string) error
	K8sProjectDeleteComponent(ctx context.Context, component string) error
	K8sProjectAddFullComponent(ctx context.Context, planComponent any, component string) error
	K8sProjectUpdateFullComponent(ctx context.Context, roles []entities.RolesK8sProject, component string) error
	K8sProjectAddOmniCertificate(ctx context.Context, appName, certName, component string) error
	K8sProjectDeleteOmniCertificate(ctx context.Context, name, component string) error
}

func HandleFullComponentAction(
	ctx context.Context,
	plan, state K8sComponentData,
	componentsChanged map[string]bool,
	order OrderHandler,
	resp *resource.UpdateResponse,
) {

	for component, changed := range componentsChanged {
		if !changed {
			continue
		}

		var planComponent, stateComponent ComponentData

		if component == "tyk" || component == "tslg_operator" || component == "tsam_operator" {
			planComponent, stateComponent = plan.GetFullComponent(component), state.GetFullComponent(component)
		}
		if component == "istio" {
			planComponent, stateComponent = plan.GetIstio(), state.GetIstio()
		}

		planIsNil := reflect.ValueOf(planComponent).IsNil()
		stateIsNil := reflect.ValueOf(stateComponent).IsNil()

		if component == "istio" {
			if planIsNil && !stateIsNil {
				if err := order.K8sProjectDeleteComponent(ctx, component); err != nil {
					resp.Diagnostics.AddError(
						fmt.Sprintf("Delete %s", component),
						fmt.Sprintf("Deleting %s ended with error.\nError message: %s", component, err.Error()),
					)
					return
				}
				resp.State.SetAttribute(ctx, path.Root(component), planComponent)
				continue
			}

			if stateIsNil && !planIsNil {
				if err := order.K8sProjectAddFullComponent(ctx, planComponent, component); err != nil {
					resp.Diagnostics.AddError(
						fmt.Sprintf("Add %s", component),
						fmt.Sprintf("Adding %s ended with error.\nError message: %s", component, err.Error()),
					)
					return
				}
				resp.State.SetAttribute(ctx, path.Root(component), planComponent)
				continue
			}

			controlPlaneChanged := planComponent.GetControlPlane() != stateComponent.GetControlPlane()
			rolesChanged := IsComponentRolesChanged(planComponent.GetRoles(), stateComponent.GetRoles())

			if !stateIsNil && controlPlaneChanged {
				if err := order.K8sProjectDeleteComponent(ctx, component); err != nil {
					resp.Diagnostics.AddError(
						fmt.Sprintf("Delete %s", component),
						fmt.Sprintf("Deleting %s ended with error.\nError message: %s", component, err.Error()),
					)
					return
				}
				if err := order.K8sProjectAddFullComponent(ctx, planComponent, component); err != nil {
					resp.Diagnostics.AddError(
						fmt.Sprintf("Add %s", component),
						fmt.Sprintf("Adding %s ended with error.\nError message: %s", component, err.Error()),
					)
					return
				}
				resp.State.SetAttribute(ctx, path.Root(component), planComponent)
				continue
			}

			if !stateIsNil && rolesChanged {
				if err := order.K8sProjectUpdateFullComponent(ctx, planComponent.GetRoles(), component); err != nil {
					resp.Diagnostics.AddError(
						fmt.Sprintf("Update %s", component),
						fmt.Sprintf("Updating %s ended with error.\nError message: %s", component, err.Error()),
					)
					return
				}
				resp.State.SetAttribute(ctx, path.Root(component), planComponent)
			}
			continue
		}

		if stateIsNil {
			if !planIsNil {
				if err := order.K8sProjectAddFullComponent(ctx, planComponent, component); err != nil {
					resp.Diagnostics.AddError(
						fmt.Sprintf("Add %s", component),
						fmt.Sprintf("Adding %s ended with error.\nError message: %s", component, err.Error()),
					)
					return
				}
				resp.State.SetAttribute(ctx, path.Root(component), planComponent)
			}
		} else if planIsNil {
			if err := order.K8sProjectDeleteComponent(ctx, component); err != nil {
				resp.Diagnostics.AddError(
					fmt.Sprintf("Delete %s", component),
					fmt.Sprintf("Deleting %s ended with error.\nError message: %s", component, err.Error()),
				)
				return
			}
			resp.State.SetAttribute(ctx, path.Root(component), planComponent)

		} else if IsComponentRolesChanged(planComponent.GetRoles(), stateComponent.GetRoles()) {
			if err := order.K8sProjectUpdateFullComponent(ctx, planComponent.GetRoles(), component); err != nil {
				resp.Diagnostics.AddError(
					fmt.Sprintf("Update %s", component),
					fmt.Sprintf("Updating %s ended with error.\nError message: %s", component, err.Error()),
				)
				return
			}
			resp.State.SetAttribute(ctx, path.Root(component), planComponent)
		}
	}
}

func HandleOmniCertificateAction(
	ctx context.Context,
	plan, state K8sComponentData,
	order OrderHandler,
	resp *resource.UpdateResponse,
) {

	component := "omni_certificate"

	planComponent, stateComponent := plan.GetOmni(), state.GetOmni()

	addedCerts, deletedCerts := FindCertsDiff(planComponent, stateComponent)

	type certAction struct {
		certs  []K8sProjectOmniData
		action func(cert K8sProjectOmniData) error
		errMsg string
	}

	actions := []certAction{}

	if len(deletedCerts) > 0 {
		actions = append(actions, certAction{
			certs: deletedCerts,
			action: func(cert K8sProjectOmniData) error {
				return order.K8sProjectDeleteOmniCertificate(ctx, cert.Name.ValueString(), component)
			},
			errMsg: "Deleting",
		})
	}

	if len(addedCerts) > 0 {
		actions = append(actions, certAction{
			certs: addedCerts,
			action: func(cert K8sProjectOmniData) error {
				return order.K8sProjectAddOmniCertificate(ctx, cert.AppName, cert.ClientName, component)
			},
			errMsg: "Adding",
		})

	}

	for _, act := range actions {
		for _, cert := range act.certs {
			err := act.action(cert)
			if err != nil {
				resp.Diagnostics.AddError(
					fmt.Sprintf("%s %s", act.errMsg, component),
					fmt.Sprintf("%s %s ended with error.\nError message: %s", act.errMsg, component, err.Error()),
				)
				return
			}
		}
	}
	resp.State.SetAttribute(ctx, path.Root(component), planComponent)
}

func HandleComponentAction(
	ctx context.Context,
	plan, state K8sComponentData,
	order OrderHandler,
	resp *resource.UpdateResponse,
	component string,
) {
	planComponent, stateComponent := plan.GetComponent(component), state.GetComponent(component)

	if stateComponent == nil {
		if planComponent != nil {
			if err := order.K8sProjectAddComponent(ctx, component); err != nil {
				resp.Diagnostics.AddError(
					fmt.Sprintf("Add %s", component),
					fmt.Sprintf("Adding %s ended with error.\nError message: %s", component, err.Error()),
				)
				return
			}
		}
	} else if planComponent == nil {
		if err := order.K8sProjectDeleteComponent(ctx, component); err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("Delete %s", component),
				fmt.Sprintf("Deleting %s ended with error.\nError message: %s", component, err.Error()),
			)
			return
		}
		resp.State.SetAttribute(ctx, path.Root(component), planComponent)
	}
}

func IsComponentChanged(planComponent, stateComponent interface{}) bool {
	return !reflect.DeepEqual(planComponent, stateComponent)
}

func IsComponentRolesChanged(planRoles, stateRoles interface{}) bool {
	if (stateRoles == nil) != (planRoles == nil) {
		return true
	}

	if stateRoles != nil && planRoles != nil {
		if !reflect.DeepEqual(planRoles, stateRoles) {
			return true
		}
	}
	return false
}

func FindCertsDiff(plan, state []K8sProjectOmniData) (added, deleted []K8sProjectOmniData) {
	stateMap := make(map[string]K8sProjectOmniData)
	for _, cert := range state {
		stateMap[cert.AppName] = cert
	}

	planMap := make(map[string]K8sProjectOmniData)
	for _, cert := range plan {
		planMap[cert.AppName] = cert
	}

	for name, cert := range stateMap {
		if _, exists := planMap[name]; !exists {
			deleted = append(deleted, cert)
		}
	}

	for name, cert := range planMap {
		if _, exists := stateMap[name]; !exists {
			added = append(added, cert)
		}
	}

	return added, deleted
}
