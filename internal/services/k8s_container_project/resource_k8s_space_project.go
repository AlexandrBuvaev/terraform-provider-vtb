package k8scontainerproject

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"

	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/consts"
	"terraform-provider-vtb/internal/custommodifires"
	k8sproject "terraform-provider-vtb/internal/services/k8s_project"
	"terraform-provider-vtb/pkg/client/contextkeys"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"
	"terraform-provider-vtb/pkg/client/sources"

	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	MAX_RETRIES  = 10
	ACTION_DELAY = 15 * time.Second
	DELETE_DELAY = 20 * time.Second
)

var (
	_ resource.Resource               = &k8sproject.K8sProjectResource{}
	_ resource.ResourceWithModifyPlan = &k8sproject.K8sProjectResource{}
)

type K8sSpaceProjectResource struct {
	client *client.CloudClient
}

func NewK8sSpaceProjectResource() resource.Resource {
	return &K8sSpaceProjectResource{}
}

func (r K8sSpaceProjectResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_k8s_space_project"
}

func (r *K8sSpaceProjectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type K8sSpaceProjectModel struct {
	ItemID          types.String             `tfsdk:"item_id"`
	ProjectName     types.String             `tfsdk:"project_name"`
	FullProjectName types.String             `tfsdk:"full_project_name"`
	Access          map[string][]string      `tfsdk:"access"`
	Quota           K8sSpaceProjectQuotaData `tfsdk:"quota"`

	SpaceConfig K8sSpaceConfig `tfsdk:"space_config"`

	Istio            *k8sproject.K8sProjectIstioData         `tfsdk:"istio"`
	Tyk              *k8sproject.K8sProjectFullComponentData `tfsdk:"tyk"`
	TslgOperator     *k8sproject.K8sProjectFullComponentData `tfsdk:"tslg_operator"`
	TsamOperator     *k8sproject.K8sProjectFullComponentData `tfsdk:"tsam_operator"`
	TsdsOperator     *k8sproject.K8sProjectComponentData     `tfsdk:"tsds_operator"`
	OmniCertificates []k8sproject.K8sProjectOmniData         `tfsdk:"omni_certificates"`
	ChaosMesh        *k8sproject.K8sProjectComponentData     `tfsdk:"chaos_mesh"`
}

type K8sSpaceProjectQuotaData struct {
	CPU    types.Float64 `tfsdk:"cpu"`
	Memory types.Float64 `tfsdk:"memory"`
}

type K8sSpaceConfig struct {
	OrderID     types.String `tfsdk:"order_id"`
	ClusterName types.String `tfsdk:"cluster_name"`
	ClusterID   types.String `tfsdk:"cluster_id"`
	Region      types.String `tfsdk:"region"`
	Ingress     types.String `tfsdk:"ingress"`
}

func (k *K8sSpaceProjectModel) GetFullComponent(component string) *k8sproject.K8sProjectFullComponentData {

	var data *k8sproject.K8sProjectFullComponentData
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

func (k *K8sSpaceProjectModel) GetComponent(component string) *k8sproject.K8sProjectComponentData {

	var data *k8sproject.K8sProjectComponentData
	switch component {
	case "chaos_mesh":
		data = k.ChaosMesh
	case "tsds_operator":
		data = k.TsdsOperator
	}
	return data
}

func (k *K8sSpaceProjectModel) GetIstio() *k8sproject.K8sProjectIstioData {
	return k.Istio
}

func (k *K8sSpaceProjectModel) GetOmni() []k8sproject.K8sProjectOmniData {
	return k.OmniCertificates
}

func (k *K8sSpaceProjectModel) GetProjectName() string {
	return k.ProjectName.ValueString()
}

func (k *K8sSpaceProjectModel) GetOrderID() string {
	return k.SpaceConfig.OrderID.ValueString()
}

func (t K8sSpaceProjectResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "VTB.Cloud ресурс Kubernetes project",
		Attributes: map[string]schema.Attribute{
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
			"access": schema.MapAttribute{
				MarkdownDescription: "Права. Словарь, где ключом является роль, а значением - список групп, которым назначается эта роль",
				Required:            true,
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
			},
			"quota": schema.SingleNestedAttribute{
				MarkdownDescription: "Квота проекта",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"cpu": schema.Float64Attribute{
						MarkdownDescription: "Количество ЦПУ",
						Required:            true,
						Validators: []validator.Float64{
							float64validator.AtMost(1024),
							float64validator.AtLeast(1),
						},
					},
					"memory": schema.Float64Attribute{
						MarkdownDescription: "Память (GB)",
						Required:            true,
						Validators: []validator.Float64{
							float64validator.AtMost(1024),
							float64validator.AtLeast(1),
						},
					},
				},
			},
			"space_config": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Конфигурация пространства контейнеров",
				Attributes:          K8sSpaceConfigSchema,
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
		},
	}
}

var K8sSpaceConfigSchema = map[string]schema.Attribute{
	"order_id": schema.StringAttribute{
		MarkdownDescription: "Идентификатор заказа пространства контейнеров",
		Required:            true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	},
	"cluster_name": schema.StringAttribute{
		MarkdownDescription: "Имя контрольной панели Kubernetes",
		Required:            true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	},
	"cluster_id": schema.StringAttribute{
		MarkdownDescription: "Идентификатор контрольной панели Kubernetes",
		Required:            true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	},
	"ingress": schema.StringAttribute{
		MarkdownDescription: "Название Ingress в пространстве контейнеров Kubernetes",
		Required:            true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	},
	"region": schema.StringAttribute{
		MarkdownDescription: "Название региона в пространстве контейнеров Kubernetes",
		Required:            true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	},
}

func (r K8sSpaceProjectResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan K8sSpaceProjectModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	quota := entities.QuotaK8sSpaceProject{
		CPU:    plan.Quota.CPU.ValueFloat64(),
		Memory: plan.Quota.Memory.ValueFloat64(),
	}

	var roles []entities.RolesK8sProject
	for role, groups := range plan.Access {
		roles = append(roles, entities.RolesK8sProject{
			Role:   role,
			Groups: groups,
		})
	}

	spaceOrderID := plan.SpaceConfig.OrderID.ValueString()
	stopRetryError := stopRetryError()

	createProjectAction := r.prepareAction(spaceOrderID, func(order *orders.K8sContainerSpaceOrder) error {
		return order.K8sContainerSpaceCreateProject(
			quota,
			roles,
			plan.ProjectName.ValueString(),
		)
	})

	success, err := withRetry(ctx, MAX_RETRIES, ACTION_DELAY, createProjectAction, stopRetryError)

	if handleRetryError(
		&resp.Diagnostics, err, success,
		consts.CREATE_RES_FAIL, "project_create",
		plan.ProjectName.ValueString(),
		"", MAX_RETRIES,
	) {
		return
	}

	projectItems, diags := r.getProjectItems(plan.SpaceConfig.OrderID)

	if diags.HasError() {
		return
	}

	var projectItem entities.Item
	var itemDataConfig entities.K8sProjectItemConfig

	for _, item := range projectItems {
		config, ok := item.Data.Config.(entities.K8sProjectItemConfig)
		if !ok {
			diags.AddError(
				consts.CREATE_RES_FAIL,
				"Item data type conversion error",
			)
			return
		}
		if strings.HasSuffix(config.ProjectName, plan.ProjectName.ValueString()) {
			projectItem = item
			itemDataConfig = config
			break
		}
	}

	resp.Diagnostics.Append(diags...)

	plan.ItemID = types.StringValue(projectItem.ID)
	plan.FullProjectName = types.StringValue(itemDataConfig.ProjectName)

	if resp.Diagnostics.HasError() {
		return
	}

	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	ctxID := context.WithValue(ctx, contextkeys.ItemIDKey, plan.ItemID.ValueString())

	components := map[string]*k8sproject.K8sProjectComponentData{
		"tsds_operator": plan.TsdsOperator,
		"chaos_mesh":    plan.ChaosMesh,
	}

	for name, planComponent := range components {
		if planComponent != nil {

			addComponentAction := r.prepareAction(spaceOrderID, func(order *orders.K8sContainerSpaceOrder) error {
				return order.K8sProjectAddComponent(ctxID, name)
			})

			success, err := withRetry(ctx, MAX_RETRIES, ACTION_DELAY, addComponentAction, stopRetryError)

			if handleRetryError(
				&resp.Diagnostics, err, success,
				consts.CREATE_RES_FAIL, "component",
				plan.ProjectName.ValueString(),
				name, MAX_RETRIES,
			) {
				return
			}
		}
	}

	if plan.OmniCertificates != nil {
		for _, omniCert := range plan.OmniCertificates {
			if !omniCert.Name.IsNull() {

				addOmniAction := r.prepareAction(spaceOrderID, func(order *orders.K8sContainerSpaceOrder) error {
					return order.K8sProjectAddOmniCertificate(ctxID, omniCert.AppName, omniCert.ClientName, "omni_certificate")
				})

				success, err := withRetry(ctx, MAX_RETRIES, ACTION_DELAY, addOmniAction, stopRetryError)

				if handleRetryError(
					&resp.Diagnostics, err, success,
					consts.CREATE_RES_FAIL, "component",
					plan.ProjectName.ValueString(),
					"omni certificate", MAX_RETRIES,
				) {
					return
				}
			}
		}
	}

	fullComponents := map[string]*k8sproject.K8sProjectFullComponentData{
		"tsam_operator": plan.TsamOperator,
		"tslg_operator": plan.TslgOperator,
		"tyk":           plan.Tyk,
	}

	var component k8sproject.ComponentData
	for name, planComponent := range fullComponents {
		if planComponent != nil {
			component = planComponent
			addFullComponentAction := r.prepareAction(spaceOrderID, func(order *orders.K8sContainerSpaceOrder) error {
				return order.K8sProjectAddFullComponent(ctxID, component, name)
			})

			success, err := withRetry(ctx, MAX_RETRIES, ACTION_DELAY, addFullComponentAction, stopRetryError)

			if handleRetryError(
				&resp.Diagnostics, err, success,
				consts.CREATE_RES_FAIL, "component",
				plan.ProjectName.ValueString(),
				name, MAX_RETRIES,
			) {
				return
			}
		}
	}

	if plan.Istio != nil {
		component = plan.Istio
		addIstioAction := r.prepareAction(spaceOrderID, func(order *orders.K8sContainerSpaceOrder) error {
			return order.K8sProjectAddFullComponent(ctxID, component, "istio")
		})

		success, err := withRetry(ctx, MAX_RETRIES, ACTION_DELAY, addIstioAction, stopRetryError)

		if handleRetryError(
			&resp.Diagnostics, err, success,
			consts.CREATE_RES_FAIL, "component",
			plan.ProjectName.ValueString(),
			"istio", MAX_RETRIES,
		) {
			return
		}
	}
}

func (r K8sSpaceProjectResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state K8sSpaceProjectModel

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	spaceOrder, err := orders.GetK8sContainerSpaceOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.SpaceConfig.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(consts.DELETE_RES_FAIL, err.Error())
		return
	}

	err = spaceOrder.WaitLastActionEnded(5)
	if err != nil {
		resp.Diagnostics.AddError(consts.DELETE_RES_FAIL, err.Error())
		return
	}

	deleteProjectAction := func() error {
		return spaceOrder.K8sContainerSpaceDeleteProject(state.ItemID.ValueString())
	}

	success, err := withRetry(ctx, MAX_RETRIES, DELETE_DELAY, deleteProjectAction, nil)

	if handleRetryError(
		&resp.Diagnostics, err, success,
		consts.DELETE_RES_FAIL, "project_delete",
		fmt.Sprintf("with ID `%s`", state.ItemID.ValueString()),
		"", MAX_RETRIES,
	) {
		return
	}
}

func (r K8sSpaceProjectResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var spaceConfig K8sSpaceConfig
	var spaceOrderID types.String
	var itemID types.String

	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("item_id"), &itemID)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("space_config"), &spaceConfig)...)

	spaceOrderID = spaceConfig.OrderID
	if spaceOrderID.IsNull() {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Resource hasn't order_id. Without order_id can't get order from portal",
		)
		return
	}

	config, exists, diags := r.getProjectItemConfig(spaceOrderID, itemID)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	if !exists {
		return
	}

	projectName, err := getProjectName(config.Region, config.ProjectName)
	if err != nil {
		diags.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't read project user mark: %s",
				err.Error(),
			),
		)
		return
	}

	state := K8sSpaceProjectModel{
		SpaceConfig: K8sSpaceConfig{
			OrderID:     spaceOrderID,
			ClusterName: types.StringValue(config.Cluster.Name),
			ClusterID:   types.StringValue(config.Cluster.UUID),
			Region:      types.StringValue(config.Region),
			Ingress:     types.StringValue(config.Router),
		},
		ItemID:          types.StringValue(config.UUID),
		Access:          k8sproject.GetAccess(config.Roles),
		ProjectName:     types.StringValue(projectName),
		FullProjectName: types.StringValue(config.ProjectName),
		Quota: K8sSpaceProjectQuotaData{
			CPU:    types.Float64Value(float64(config.Quota.CPU)),
			Memory: types.Float64Value(float64(config.Quota.Memory)),
		},
		Tyk:              k8sproject.GetFullComponentData(config.Tyk),
		TslgOperator:     k8sproject.GetFullComponentData(config.TslgOperator),
		TsamOperator:     k8sproject.GetFullComponentData(config.TsamOperator),
		TsdsOperator:     k8sproject.GetComponentData(config.TsdsOperator),
		Istio:            k8sproject.GetIstioData(config.Istio),
		OmniCertificates: k8sproject.GetOmniData(config.OmniCertificates),
		ChaosMesh:        k8sproject.GetComponentData(config.ChaosMesh),
	}

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r K8sSpaceProjectResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	IDs := strings.Split(req.ID, ":")
	if len(IDs) != 2 {
		resp.Diagnostics.AddError("Invalid import ID", "Expected 'order_id:item_id'")
		return
	}

	orderID, itemID := IDs[0], IDs[1]
	_, exists, diags := r.getProjectItemConfig(
		types.StringValue(orderID),
		types.StringValue(itemID),
	)

	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	if !exists {
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("space_config").AtName("order_id"), orderID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("item_id"), itemID)...)
}

func (r K8sSpaceProjectResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan K8sSpaceProjectModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	var state K8sSpaceProjectModel
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	r.RolesModifyPlan(ctx, &plan, &state, resp)

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)

}

func (r K8sSpaceProjectResource) RolesModifyPlan(
	ctx context.Context,
	plan,
	state *K8sSpaceProjectModel,
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

func (r K8sSpaceProjectResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state K8sSpaceProjectModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	spaceOrder, err := orders.GetK8sContainerSpaceOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.SpaceConfig.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(consts.UPDATE_RES_FAIL, err.Error())
		return
	}

	quotaChanged := !reflect.DeepEqual(plan.Quota, state.Quota)
	accessChanged := !reflect.DeepEqual(plan.Access, state.Access)
	componentsChanged, changes := k8sproject.GetChangedComponents(&plan, &state)
	tsdsChanged := k8sproject.IsComponentChanged(plan.TsdsOperator, state.TsdsOperator)
	omniChanged := k8sproject.IsComponentChanged(plan.OmniCertificates, state.OmniCertificates)
	chaosMeshChanged := k8sproject.IsComponentChanged(plan.ChaosMesh, state.ChaosMesh)

	if quotaChanged || accessChanged {
		r.changeSpaceProject(ctx, spaceOrder, &plan, resp)

		if resp.Diagnostics.HasError() {
			return
		}
	}

	if tsdsChanged {
		component := "tsds_operator"
		ctxID := context.WithValue(ctx, contextkeys.ItemIDKey, state.ItemID.ValueString())
		r.handleComponentAction(ctxID, &plan, &state, spaceOrder, resp, component)

		if resp.Diagnostics.HasError() {
			return
		}
	}

	if chaosMeshChanged {
		component := "chaos_mesh"
		ctxID := context.WithValue(ctx, contextkeys.ItemIDKey, state.ItemID.ValueString())
		r.handleComponentAction(ctxID, &plan, &state, spaceOrder, resp, component)

		if resp.Diagnostics.HasError() {
			return
		}
	}

	if componentsChanged {
		ctxID := context.WithValue(ctx, contextkeys.ItemIDKey, state.ItemID.ValueString())
		r.handleFullComponentAction(ctxID, &plan, &state, changes, spaceOrder, resp)

		if resp.Diagnostics.HasError() {
			return
		}
	}

	if omniChanged {
		ctxID := context.WithValue(ctx, contextkeys.ItemIDKey, state.ItemID.ValueString())
		r.handleOmniCertificateAction(ctxID, &plan, &state, spaceOrder, resp)

		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r K8sSpaceProjectResource) handleComponentAction(
	ctx context.Context,
	plan, state k8sproject.K8sComponentData,
	order k8sproject.OrderHandler,
	resp *resource.UpdateResponse,
	component string,
) {
	planComponent, stateComponent := plan.GetComponent(component), state.GetComponent(component)

	if stateComponent == nil {
		if planComponent != nil {
			addComponentAction := r.prepareAction(order.GetOrderID(), func(order *orders.K8sContainerSpaceOrder) error {
				return order.K8sProjectAddComponent(ctx, component)
			})

			success, err := withRetry(ctx, MAX_RETRIES, ACTION_DELAY, addComponentAction, stopRetryError())

			if handleRetryError(
				&resp.Diagnostics, err, success,
				consts.UPDATE_RES_FAIL, "component",
				plan.GetProjectName(),
				component, MAX_RETRIES,
			) {
				return
			}
		}
	} else if planComponent == nil {
		deleteComponentAction := r.prepareAction(order.GetOrderID(), func(order *orders.K8sContainerSpaceOrder) error {
			return order.K8sProjectDeleteComponent(ctx, component)
		})

		success, err := withRetry(ctx, MAX_RETRIES, ACTION_DELAY, deleteComponentAction, stopRetryError())

		if handleRetryError(
			&resp.Diagnostics, err, success,
			consts.UPDATE_RES_FAIL, "component",
			plan.GetProjectName(),
			component, MAX_RETRIES,
		) {
			return
		}
	}
}

func (r K8sSpaceProjectResource) handleFullComponentAction(
	ctx context.Context,
	plan, state k8sproject.K8sComponentData,
	componentsChanged map[string]bool,
	order k8sproject.OrderHandler,
	resp *resource.UpdateResponse,
) {

	for component, changed := range componentsChanged {
		if !changed {
			continue
		}

		var planComponent, stateComponent k8sproject.ComponentData

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
				deleteComponentAction := r.prepareAction(order.GetOrderID(), func(order *orders.K8sContainerSpaceOrder) error {
					return order.K8sProjectDeleteComponent(ctx, component)
				})

				success, err := withRetry(ctx, MAX_RETRIES, ACTION_DELAY, deleteComponentAction, stopRetryError())

				if handleRetryError(
					&resp.Diagnostics, err, success,
					consts.UPDATE_RES_FAIL, "component",
					plan.GetProjectName(),
					component, MAX_RETRIES,
				) {
					return
				}
				continue
			}

			if stateIsNil && !planIsNil {
				addFullComponentAction := r.prepareAction(order.GetOrderID(), func(order *orders.K8sContainerSpaceOrder) error {
					return order.K8sProjectAddFullComponent(ctx, planComponent, component)
				})

				success, err := withRetry(ctx, MAX_RETRIES, ACTION_DELAY, addFullComponentAction, stopRetryError())

				if handleRetryError(
					&resp.Diagnostics, err, success,
					consts.UPDATE_RES_FAIL, "component",
					plan.GetProjectName(),
					component, MAX_RETRIES,
				) {
					return
				}
				continue
			}

			controlPlaneChanged := planComponent.GetControlPlane() != stateComponent.GetControlPlane()
			rolesChanged := k8sproject.IsComponentRolesChanged(planComponent.GetRoles(), stateComponent.GetRoles())

			if !stateIsNil && controlPlaneChanged {
				deleteComponentAction := r.prepareAction(order.GetOrderID(), func(order *orders.K8sContainerSpaceOrder) error {
					return order.K8sProjectDeleteComponent(ctx, component)
				})

				success, err := withRetry(ctx, MAX_RETRIES, ACTION_DELAY, deleteComponentAction, stopRetryError())

				if handleRetryError(
					&resp.Diagnostics, err, success,
					consts.UPDATE_RES_FAIL, "component",
					plan.GetProjectName(),
					component, MAX_RETRIES,
				) {
					return
				}
				addFullComponentAction := r.prepareAction(order.GetOrderID(), func(order *orders.K8sContainerSpaceOrder) error {
					return order.K8sProjectAddFullComponent(ctx, planComponent, component)
				})

				success, err = withRetry(ctx, MAX_RETRIES, ACTION_DELAY, addFullComponentAction, stopRetryError())

				if handleRetryError(
					&resp.Diagnostics, err, success,
					consts.UPDATE_RES_FAIL, "component",
					plan.GetProjectName(),
					component, MAX_RETRIES,
				) {
					return
				}
				continue
			}

			if !stateIsNil && rolesChanged {
				updateFullComponentAction := r.prepareAction(order.GetOrderID(), func(order *orders.K8sContainerSpaceOrder) error {
					return order.K8sProjectUpdateFullComponent(ctx, planComponent.GetRoles(), component)
				})

				success, err := withRetry(ctx, MAX_RETRIES, ACTION_DELAY, updateFullComponentAction, stopRetryError())

				if handleRetryError(
					&resp.Diagnostics, err, success,
					consts.UPDATE_RES_FAIL, "component",
					plan.GetProjectName(),
					component, MAX_RETRIES,
				) {
					return
				}
			}
			continue
		}

		if stateIsNil {
			if !planIsNil {
				addFullComponentAction := r.prepareAction(order.GetOrderID(), func(order *orders.K8sContainerSpaceOrder) error {
					return order.K8sProjectAddFullComponent(ctx, planComponent, component)
				})

				success, err := withRetry(ctx, MAX_RETRIES, ACTION_DELAY, addFullComponentAction, stopRetryError())

				if handleRetryError(
					&resp.Diagnostics, err, success,
					consts.UPDATE_RES_FAIL, "component",
					plan.GetProjectName(),
					component, MAX_RETRIES,
				) {
					return
				}
			}
		} else if planIsNil {
			deleteComponentAction := r.prepareAction(order.GetOrderID(), func(order *orders.K8sContainerSpaceOrder) error {
				return order.K8sProjectDeleteComponent(ctx, component)
			})

			success, err := withRetry(ctx, MAX_RETRIES, ACTION_DELAY, deleteComponentAction, stopRetryError())

			if handleRetryError(
				&resp.Diagnostics, err, success,
				consts.UPDATE_RES_FAIL, "component",
				plan.GetProjectName(),
				component, MAX_RETRIES,
			) {
				return
			}

		} else if k8sproject.IsComponentRolesChanged(planComponent.GetRoles(), stateComponent.GetRoles()) {
			updateFullComponentAction := r.prepareAction(order.GetOrderID(), func(order *orders.K8sContainerSpaceOrder) error {
				return order.K8sProjectUpdateFullComponent(ctx, planComponent.GetRoles(), component)
			})

			success, err := withRetry(ctx, MAX_RETRIES, ACTION_DELAY, updateFullComponentAction, stopRetryError())

			if handleRetryError(
				&resp.Diagnostics, err, success,
				consts.UPDATE_RES_FAIL, "component",
				plan.GetProjectName(),
				component, MAX_RETRIES,
			) {
				return
			}
		}
	}
}

func (r K8sSpaceProjectResource) handleOmniCertificateAction(
	ctx context.Context,
	plan, state k8sproject.K8sComponentData,
	order k8sproject.OrderHandler,
	resp *resource.UpdateResponse,
) {

	component := "omni_certificate"

	planComponent, stateComponent := plan.GetOmni(), state.GetOmni()

	addedCerts, deletedCerts := k8sproject.FindCertsDiff(planComponent, stateComponent)

	type certAction struct {
		certs  []k8sproject.K8sProjectOmniData
		action func(cert k8sproject.K8sProjectOmniData)
	}

	actions := []certAction{}

	if len(deletedCerts) > 0 {
		actions = append(actions, certAction{
			certs: deletedCerts,
			action: func(cert k8sproject.K8sProjectOmniData) {
				deleteComponentAction := r.prepareAction(order.GetOrderID(), func(order *orders.K8sContainerSpaceOrder) error {
					return order.K8sProjectDeleteOmniCertificate(ctx, cert.Name.ValueString(), component)
				})

				success, err := withRetry(ctx, MAX_RETRIES, ACTION_DELAY, deleteComponentAction, stopRetryError())

				if handleRetryError(
					&resp.Diagnostics, err, success,
					consts.UPDATE_RES_FAIL, "component",
					plan.GetProjectName(),
					component, MAX_RETRIES,
				) {
					return
				}
			},
		})
	}

	if len(addedCerts) > 0 {
		actions = append(actions, certAction{
			certs: addedCerts,
			action: func(cert k8sproject.K8sProjectOmniData) {
				addComponentAction := r.prepareAction(order.GetOrderID(), func(order *orders.K8sContainerSpaceOrder) error {
					return order.K8sProjectAddOmniCertificate(ctx, cert.AppName, cert.ClientName, component)
				})

				success, err := withRetry(ctx, MAX_RETRIES, ACTION_DELAY, addComponentAction, stopRetryError())

				if handleRetryError(
					&resp.Diagnostics, err, success,
					consts.UPDATE_RES_FAIL, "component",
					plan.GetProjectName(),
					component, MAX_RETRIES,
				) {
					return
				}
			},
		})
	}

	for _, act := range actions {
		for _, cert := range act.certs {
			act.action(cert)
		}
	}
}

func (r K8sSpaceProjectResource) changeSpaceProject(
	ctx context.Context,
	order *orders.K8sContainerSpaceOrder,
	plan *K8sSpaceProjectModel,
	resp *resource.UpdateResponse,
) {

	quota := entities.QuotaK8sSpaceProject{
		CPU:    plan.Quota.CPU.ValueFloat64(),
		Memory: plan.Quota.Memory.ValueFloat64(),
	}

	var roles []entities.RolesK8sProject
	for role, groups := range plan.Access {
		roles = append(roles, entities.RolesK8sProject{
			Role:   role,
			Groups: groups,
		})
	}

	updateProjectAction := r.prepareAction(order.GetOrderID(), func(order *orders.K8sContainerSpaceOrder) error {
		return order.K8sContainerSpaceUpdateProject(quota, roles, plan.ItemID.ValueString())
	})

	success, err := withRetry(ctx, MAX_RETRIES, ACTION_DELAY, updateProjectAction, stopRetryError())

	if handleRetryError(
		&resp.Diagnostics, err, success,
		consts.UPDATE_RES_FAIL, "project_update",
		plan.GetProjectName(),
		"", MAX_RETRIES,
	) {
		return
	}
}

func (r K8sSpaceProjectResource) getProjectItems(
	orderID types.String,
) ([]entities.Item, diag.Diagnostics) {
	var diags diag.Diagnostics

	spaceOrder, err := orders.GetK8sContainerSpaceOrder(
		r.client.Creds,
		r.client.ProjectName,
		orderID.ValueString(),
	)
	if err != nil {
		diags.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't read order with order_id '%s'. Error: %s",
				orderID.ValueString(),
				err.Error(),
			),
		)
		return nil, diags
	}

	projectItems, err := spaceOrder.GetActiveProjectItems()
	if err != nil {
		diags.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get project items from order '%s'. Error: %s",
				orderID.ValueString(),
				err.Error(),
			),
		)
		return nil, diags
	}
	return projectItems, diags
}

func (r K8sSpaceProjectResource) getProjectItemConfig(
	spaceOrderID types.String,
	itemID types.String,
) (entities.K8sProjectItemConfig, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	var config entities.K8sProjectItemConfig

	projectItems, diags := r.getProjectItems(spaceOrderID)

	if diags.HasError() {
		return config, false, diags
	}

	for _, item := range projectItems {
		if item.ID == itemID.ValueString() && item.Data.State == "on" {
			config, ok := item.Data.Config.(entities.K8sProjectItemConfig)
			if !ok {
				diags.AddError(
					consts.READ_RES_FAIL,
					"Item data type conversion error",
				)
				return config, false, diags
			}
			return config, true, diags
		}
	}

	diags.AddError(
		consts.READ_RES_FAIL,
		fmt.Sprintf(
			"Project with item ID '%s' not found or deleted in order '%s'",
			itemID.ValueString(),
			spaceOrderID.ValueString(),
		),
	)
	return config, false, diags
}

func (r K8sSpaceProjectResource) prepareAction(OrderID string, action func(order *orders.K8sContainerSpaceOrder) error) func() error {

	return func() error {
		spaceOrder, err := orders.GetK8sContainerSpaceOrder(
			r.client.Creds,
			r.client.ProjectName,
			OrderID,
		)
		if err != nil {
			return err
		}
		if err := spaceOrder.WaitLastActionEnded(5); err != nil {
			return err
		}
		return action(spaceOrder)
	}
}

func withRetry(
	ctx context.Context,
	maxRetries int,
	delay time.Duration,
	operation func() error,
	stopCondition func(error) bool,
) (bool, error) {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if ctx.Err() != nil {
			return false, ctx.Err()
		}

		err := operation()
		if err == nil {
			return true, nil
		}

		if stopCondition != nil && stopCondition(err) {
			return false, err
		}

		if attempt == maxRetries {
			return false, fmt.Errorf("retry limit exceeded after %d attempts", maxRetries)
		}

		select {
		case <-time.After(delay):
			continue
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}
	return false, nil
}

func stopRetryError() func(error) bool {
	return func(err error) bool {
		return !strings.Contains(err.Error(), "invalid_order_status")
	}
}

func handleRetryError(
	diags *diag.Diagnostics,
	err error,
	success bool,
	operationConst string,
	operationType string,
	projectName string,
	componentName string,
	maxRetries int,
) bool {
	if err != nil {
		if err == context.Canceled {
			diags.AddError("Aborted", "Operation canceled by context")
		} else if strings.Contains(err.Error(), "retry limit exceeded") {
			diags.AddError(
				"Retry limit exceeded",
				fmt.Sprintf("Failed after %d attempts", maxRetries),
			)
		} else {
			detail := buildErrorDetail(operationType, projectName, componentName, err.Error())
			diags.AddError(operationConst, detail)
		}
		return true
	}

	if !success {
		detail := buildErrorDetail(operationType, projectName, componentName, "")
		diags.AddError(operationConst, detail)
		return true
	}

	return false
}

func buildErrorDetail(operationType, projectName, componentName, errorMsg string) string {
	base := ""
	switch operationType {
	case "component":
		base = fmt.Sprintf("Configure '%s' in project '%s' failed", componentName, projectName)
	case "project_create":
		base = fmt.Sprintf("Project '%s' creation failed", projectName)
	case "project_update":
		base = fmt.Sprintf("Project '%s' update failed", projectName)
	case "project_delete":
		base = fmt.Sprintf("Project '%s' removing failed", projectName)
	default:
		if errorMsg != "" {
			return fmt.Sprintf("Operation failed: %s", errorMsg)
		}
		return "Operation failed without explicit error"
	}

	if errorMsg != "" {
		return fmt.Sprintf("%s. Error: %s", base, errorMsg)
	}
	return base
}

func getProjectName(regionName, fullProjectName string) (string, error) {
	trimRegionName := strings.TrimPrefix(regionName, "reg-")
	pattern := "^" + regexp.QuoteMeta(trimRegionName) + "-(.+)$"
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern")
	}
	matches := re.FindStringSubmatch(fullProjectName)

	return matches[1], nil
}
