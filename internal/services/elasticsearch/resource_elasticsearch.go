package elasticsearch

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	DEFAULT_ELASTICSERACH_LAYOUT = "Elasticsearch_Master:3"
)

var (
	_ resource.Resource                = &ElasticSearchResource{}
	_ resource.ResourceWithImportState = &ElasticSearchResource{}
	_ resource.ResourceWithModifyPlan  = &ElasticSearchResource{}
)

type ElasticSearchResource struct {
	client *client.CloudClient
}

func NewElasticSearchResource() resource.Resource {
	return &ElasticSearchResource{}
}

func (r ElasticSearchResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_elasticsearch_cluster"
}

func (r *ElasticSearchResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type ElasticSearchResourceModel struct {
	Label            types.String                              `tfsdk:"label"`
	Lifetime         types.Int64                               `tfsdk:"lifetime"`
	OrderID          types.String                              `tfsdk:"order_id"`
	ItemID           types.String                              `tfsdk:"item_id"`
	FinancialProject types.String                              `tfsdk:"financial_project"`
	Access           map[string][]types.String                 `tfsdk:"access"`
	Core             core.CoreModel                            `tfsdk:"core"`
	Image            common.ClusterProductImageDataSourceModel `tfsdk:"image"`
	Layout           types.String                              `tfsdk:"layout"`

	FlavorMaster     flavor.FlavorModel `tfsdk:"flavor_master"`
	FlavorData       flavor.FlavorModel `tfsdk:"flavor_data"`
	FlavorCoordintor flavor.FlavorModel `tfsdk:"flavor_coordinator"`

	ClusterName             types.String `tfsdk:"cluster_name"`
	ElasticSearchVersion    types.String `tfsdk:"elasticsearch_version"`
	FluentdPassword         types.String `tfsdk:"fluentd_password"`
	ElasticSearchNodesCount NodesCount   `tfsdk:"nodes_count"`

	AdmAppGroups   types.Set `tfsdk:"adm_app_groups"`
	UserAppGroups  types.Set `tfsdk:"user_app_groups"`
	SystemAdmGroup types.Set `tfsdk:"system_adm_groups"`

	InstallKibana     types.Bool                        `tfsdk:"install_kibana"`
	KibanaLocation    types.String                      `tfsdk:"kibana_location"`
	KibanaExtraMounts map[string]common.ExtraMountModel `tfsdk:"kibana_extra_mounts"`
	KibanaPassword    types.String                      `tfsdk:"kibana_password"`
	FlavorKibana      *flavor.FlavorModel               `tfsdk:"flavor_kibana"`

	DataExtraMounts        map[string]common.ExtraMountModel `tfsdk:"data_extra_mounts"`
	MasterExtraMounts      map[string]common.ExtraMountModel `tfsdk:"master_extra_mounts"`
	CoordinatorExtraMounts map[string]common.ExtraMountModel `tfsdk:"coordinator_extra_mounts"`
}

type NodesCount struct {
	Data        types.Int64 `tfsdk:"data"`
	Master      types.Int64 `tfsdk:"master"`
	Coordinator types.Int64 `tfsdk:"coordinator"`
}

func (r ElasticSearchResource) Schema(
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
				MarkdownDescription: "Основные параметры для виртуальных машин в заказе",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"image": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Тип вычислительного ресурса",
				Attributes:  common.ClusterProductImageSchema,
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
				Required:            true,
				MarkdownDescription: "Метка заказа",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"order_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Идентификатор заказа, известен после создания заказа",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"item_id": schema.StringAttribute{
				MarkdownDescription: "Идентфикатор сущностей ВМ, известен после создания ВМ",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"access": schema.MapAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Словарь,где ключом является роль, а значением список групп, " +
					"которые предоставляют доступ для входа в Active Directory",
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
					mapplanmodifier.UseStateForUnknown(),
					custommodifires.EmptyAccessGroup(),
				},
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
			},
			"financial_project": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Источник финансирования",
			},
			"layout": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Схема геораспределения",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			// custom fields
			"cluster_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Имя кластера",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile("^[a-z0-9][a-z0-9-_]{0,23}[a-z0-9]$"),
						"'cluster_name' must match pattern: '^[a-z0-9][a-z0-9-_]{0,23}[a-z0-9]$'",
					),
				},
			},
			"nodes_count": schema.SingleNestedAttribute{
				Required:            true,
				MarkdownDescription: "Кол-во нод кластера по типу",
				Attributes: map[string]schema.Attribute{
					"data": schema.Int64Attribute{
						Required:            true,
						MarkdownDescription: "Кол-во нод типа Data",
					},
					"master": schema.Int64Attribute{
						Required:            true,
						MarkdownDescription: "Кол-во нод типа Master",
					},
					"coordinator": schema.Int64Attribute{
						Required:            true,
						MarkdownDescription: "Кол-во нод типа Coordinator",
					},
				},
			},
			"kibana_password": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				MarkdownDescription: "Пароль пользователя kibana",
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9~!$@#%^*_]{34,64}$`),
						"'kibana_password' must be 34-64 characters long and contain at least one digit, "+
							"one lowercase letter, one uppercase letter and one special character (~!$@#%^*_)",
					),
				},
			},
			"fluentd_password": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				MarkdownDescription: "Пароль пользователя fluentd",
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[a-zA-Z0-9~!$@#%^*_]{34,64}$`),
						"'fluentd_password' must be 34-64 characters long and contain at least one digit, "+
							"one lowercase letter, one uppercase letter and one special character (~!$@#%^*_)",
					),
				},
			},
			"install_kibana": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Установить Kibana",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"kibana_location": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Место установки Kibana",
				Validators: []validator.String{
					stringvalidator.OneOf("separate host"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"flavor_master": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.FlavorSchema,
				MarkdownDescription: "Конфигурация Core/RAM для ноды типа Master",
			},
			"flavor_data": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.FlavorSchema,
				MarkdownDescription: "Конфигурация Core/RAM для ноды типа Data",
			},
			"flavor_coordinator": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.FlavorSchema,
				MarkdownDescription: "Конфигурация Core/RAM для ноды типа Coordinator",
			},
			"flavor_kibana": schema.SingleNestedAttribute{
				Optional:            true,
				Attributes:          common.FlavorSchema,
				MarkdownDescription: "Конфигурация Core/RAM для ноды типа Kibana",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"master_extra_mounts": schema.MapNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Точки монтирования нод Master",
				Default: mapdefault.StaticValue(
					types.MapValueMust(
						types.ObjectType{
							AttrTypes: map[string]attr.Type{
								"size":        types.Int64Type,
								"file_system": types.StringType,
							},
						},
						map[string]attr.Value{
							"/app/": types.ObjectValueMust(
								map[string]attr.Type{
									"size":        types.Int64Type,
									"file_system": types.StringType,
								},
								map[string]attr.Value{
									"size":        types.Int64Value(100),
									"file_system": types.StringValue("xfs"),
								},
							),
						},
					),
				),
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"size": schema.Int64Attribute{
							Required:            true,
							MarkdownDescription: "Размер точки монтирования (размер в ГБ)",
							Validators: []validator.Int64{
								int64validator.OneOf(100),
							},
						},
						"file_system": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Тип файловой системы",
							Default:             stringdefault.StaticString("xfs"),
						},
					},
				},
			},
			"data_extra_mounts": schema.MapNestedAttribute{
				Required:            true,
				MarkdownDescription: "Точки монтирования нод Data",
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.OneOf("/app/")),
					mapvalidator.SizeBetween(1, 1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"size": schema.Int64Attribute{
							Required:            true,
							MarkdownDescription: "Размер точки монтирования (размер в ГБ)",
							Validators: []validator.Int64{
								int64validator.AtLeast(100),
							},
							PlanModifiers: []planmodifier.Int64{
								custommodifires.MininalGrow(10),
								custommodifires.GrowOnly(),
							},
						},
						"file_system": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Тип файловой системы",
							Default:             stringdefault.StaticString("xfs"),
						},
					},
				},
			},
			"coordinator_extra_mounts": schema.MapNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Точки монтирования нод Coordinator",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"size": schema.Int64Attribute{
							Required:            true,
							MarkdownDescription: "Размер точки монтирования (размер в ГБ)",
							Validators: []validator.Int64{
								int64validator.OneOf(100),
							},
						},
						"file_system": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Тип файловой системы",
							Default:             stringdefault.StaticString("xfs"),
						},
					},
				},
				Default: mapdefault.StaticValue(
					types.MapValueMust(
						types.ObjectType{
							AttrTypes: map[string]attr.Type{
								"size":        types.Int64Type,
								"file_system": types.StringType,
							},
						},
						map[string]attr.Value{
							"/app/": types.ObjectValueMust(
								map[string]attr.Type{
									"size":        types.Int64Type,
									"file_system": types.StringType,
								},
								map[string]attr.Value{
									"size":        types.Int64Value(100),
									"file_system": types.StringValue("xfs"),
								},
							),
						},
					),
				),
			},
			"kibana_extra_mounts": schema.MapNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Точки монтирования нод Kibana",
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.OneOf("/app/kibana")),
					mapvalidator.SizeBetween(1, 1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"size": schema.Int64Attribute{
							Required:            true,
							MarkdownDescription: "Размер точки монтирования (размер в ГБ)",
							Validators: []validator.Int64{
								int64validator.OneOf(100),
							},
						},
						"file_system": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Тип файловой системы",
							Default:             stringdefault.StaticString("xfs"),
						},
					},
				},
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"system_adm_groups": schema.SetAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Группы system administrator",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
			"user_app_groups": schema.SetAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Группы user application",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
			"adm_app_groups": schema.SetAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Группы admin application",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
			},
			"elasticsearch_version": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Версия Elasticsearch Opensearch",
				Validators: []validator.String{
					stringvalidator.OneOf("2.15.0 + Exporter 1.1.0"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r ElasticSearchResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r ElasticSearchResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan, state ElasticSearchResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	}

	checkOrderIsDeleted := utils.CheckOrderIsDeleted(r.client, plan.OrderID, "cluster", "elasticsearch_os")
	resp.Diagnostics.Append(checkOrderIsDeleted.Diagnostics...)

	utils.HandleExternalDeletionRecreate(ctx, resp, &plan, checkOrderIsDeleted.IsDeleted, checkOrderIsDeleted.Diagnostics)

	validateAccessForEnvDiags := utils.ValidateActiveDirectoryAccessForEnvironmentsV1(r.client, plan.Access, "DEV")
	resp.Diagnostics.Append(validateAccessForEnvDiags...)

	r.validateNodesCount(plan.ElasticSearchNodesCount, resp)
	r.validateKibanaConfiguration(plan, resp)
	r.validateSystemAdmGroups(plan, resp)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(req.Plan.Set(ctx, &plan)...)
}

func (r ElasticSearchResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan ElasticSearchResourceModel
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
		resp.Diagnostics.AddAttributeError(
			path.Root("financial_project"),
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Get financial project ended with error.\nError: %v", err.Error()),
		)
		return
	}

	if strings.EqualFold(r.client.Environment, "prod") && plan.Image.GeoDistribution.ValueBool() {
		layout, err := references.GetGeoPageByLayout(r.client.Creds, DEFAULT_ELASTICSERACH_LAYOUT)
		if err != nil {
			resp.Diagnostics.AddError(consts.CREATE_RES_FAIL, fmt.Sprintf("Get layout data from reference: %v", err.Error()))
			return
		}
		plan.Layout = types.StringValue(layout.ID)
	} else {
		plan.Layout = types.StringNull()
	}

	attrs := r.prepareElasticAttrs(ctx, plan)

	order := orders.NewElasticSearch(
		r.client.Creds,
		r.client.ProjectName,
		plan.Image.ProductID.ValueString(),
		attrs,
	)

	err = order.Create(orders.CreateOrderPayload{
		Label:        plan.Label.ValueString(),
		Lifetime:     int(plan.Lifetime.ValueInt64()),
		FinProjectID: finProj.ID,
	},
	)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Create order on portal ended with error.\nError: %v", err.Error()),
		)
		return
	}

	elasticItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Get order item from portal ended with error.\nError: %v", err.Error()),
		)
	}

	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(elasticItem.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r ElasticSearchResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var orderID types.String
	req.State.GetAttribute(ctx, path.Root("order_id"), &orderID)
	if orderID.IsNull() {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Resource hasn't order_id. Without order_id can't get order from portal",
		)
		return
	}

	var kibanaPassword, fluentdPassword types.String
	req.State.GetAttribute(ctx, path.Root("kibana_password"), &kibanaPassword)
	req.State.GetAttribute(ctx, path.Root("fluentd_password"), &fluentdPassword)

	order, err := orders.GetElasticSearchOrder(
		r.client.Creds,
		r.client.ProjectName,
		orderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("order_id"),
			consts.READ_RES_FAIL,
			err.Error(),
		)
		return
	}

	item, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Get order item from portal ended with error.\nError: %v",
				err.Error(),
			),
		)
		return
	}

	itemConfig, ok := item.Data.Config.(entities.ElasticSearchConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Invalid type for item config",
		)
		return
	}

	vmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Get VM items from portal ended with error.\nError: %v", err.Error()),
		)
		return
	}

	vmAcls, err := order.GetVMAcls()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Get VM Acls ended with error.\nError: %v", err.Error()),
		)
		return
	}

	vmConfig, ok := vmItems[0].Data.Config.(entities.VMItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Invalid type for VM config",
		)
		return
	}

	dataVmItems, err := order.GetDataVmItems()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Get Data VM items ended with error.\nError: %v", err.Error()),
		)
		return
	}

	dataVmConfig, ok := dataVmItems[0].Data.Config.(entities.VMItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Invalid type for Data vm config",
		)
		return
	}

	masterVmItems, err := order.GetMasterVmItems()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Get Master VM items ended with error.\nError: %v", err.Error()),
		)
		return
	}
	masterVmConfig, ok := masterVmItems[0].Data.Config.(entities.VMItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Invalid type for Master vm config",
		)
		return
	}

	coordinatorVmItems, err := order.GetCoordinatorVmItems()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Get Coordinator VM items ended with error.\nError: %v", err.Error()),
		)
		return
	}

	dataAppExtraMount, err := order.GetDataVmItemExtraMount("/app")
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

	tfSystemAdmGroup, diag := types.SetValueFrom(ctx, types.StringType, order.Attrs.SystemAdmGroup)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(tfSystemAdmGroup.Elements()) < 1 {
		tfSystemAdmGroup = types.SetNull(types.StringType)
	}

	tfUserAppGroups, diag := types.SetValueFrom(ctx, types.StringType, order.Attrs.UserAppGroups)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	tfAdmAppGroups, diag := types.SetValueFrom(ctx, types.StringType, order.Attrs.AdmAppGroups)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := ElasticSearchResourceModel{
		OrderID:          orderID,
		ItemID:           types.StringValue(item.ID),
		Label:            types.StringValue(order.Label),
		Access:           utils.ReadAccessMapV2(vmAcls),
		FinancialProject: types.StringValue(order.FinancialSource.Name),
		Layout:           types.StringValue(order.Attrs.Layout),
		Core: core.CoreModel{
			Platform:       types.StringValue(order.Attrs.Platform),
			Domain:         types.StringValue(order.Attrs.Domain),
			NetSegmentCode: types.StringValue(order.Attrs.DefaultNic.NetSegment),
			Zone:           types.StringValue(order.Attrs.AvailabilityZone),
		},
		Image: common.ClusterProductImageDataSourceModel{
			Distribution:    types.StringValue(vmConfig.Image.Os.Distribution),
			OsVersion:       types.StringValue(vmConfig.Image.Os.Version),
			OnSupport:       types.BoolValue(order.Attrs.OnSupport),
			ProductID:       types.StringValue(order.ProductID),
			ADIntegration:   types.BoolValue(order.Attrs.ADIntegration),
			GeoDistribution: types.BoolPointerValue(order.Attrs.GeoDistribution),
		},
		ElasticSearchNodesCount: NodesCount{
			Data:        types.Int64Value(int64(len(dataVmItems))),
			Master:      types.Int64Value(int64(len(masterVmItems))),
			Coordinator: types.Int64Value(int64(len(coordinatorVmItems))),
		},
		ElasticSearchVersion: types.StringValue(itemConfig.ElasticSearchVersion),
		ClusterName:          types.StringValue(itemConfig.ClusterName),
		KibanaPassword:       kibanaPassword,
		FluentdPassword:      fluentdPassword,
		SystemAdmGroup:       tfSystemAdmGroup,
		AdmAppGroups:         tfAdmAppGroups,
		UserAppGroups:        tfUserAppGroups,
		KibanaLocation:       types.StringPointerValue(order.Attrs.KibanaLocation),
		InstallKibana:        types.BoolValue(order.Attrs.InstallKibana),
		FlavorMaster: flavor.FlavorModel{
			Cores:  types.Int64Value(masterVmConfig.Flavor.Cores),
			Memory: types.Int64Value(masterVmConfig.Flavor.Memory),
			UUID:   types.StringValue(masterVmConfig.Flavor.UUID),
			Name:   types.StringValue(masterVmConfig.Flavor.Name),
		},
		FlavorData: flavor.FlavorModel{
			Cores:  types.Int64Value(dataVmConfig.Flavor.Cores),
			Memory: types.Int64Value(dataVmConfig.Flavor.Memory),
			Name:   types.StringValue(dataVmConfig.Flavor.Name),
			UUID:   types.StringValue(dataVmConfig.Flavor.UUID),
		},
		FlavorCoordintor: flavor.FlavorModel{
			Cores:  types.Int64Value(order.Attrs.FlavorCoordinator.Cores),
			Memory: types.Int64Value(order.Attrs.FlavorCoordinator.Memory),
			Name:   types.StringValue(order.Attrs.FlavorCoordinator.Name),
			UUID:   types.StringValue(order.Attrs.FlavorCoordinator.UUID),
		},
		DataExtraMounts: map[string]common.ExtraMountModel{
			"/app/": {
				Size:       types.Int64Value(dataAppExtraMount.Size),
				FileSystem: types.StringValue(dataAppExtraMount.FileSystem),
			},
		},
		MasterExtraMounts: map[string]common.ExtraMountModel{
			"/app/": {
				Size:       types.Int64Value(order.Attrs.MasterExtraMounts[0].Size),
				FileSystem: types.StringValue(order.Attrs.MasterExtraMounts[0].FileSystem),
			},
		},
		CoordinatorExtraMounts: map[string]common.ExtraMountModel{
			"/app/": {
				Size:       types.Int64Value(order.Attrs.CoordinatorExtraMounts[0].Size),
				FileSystem: types.StringValue(order.Attrs.CoordinatorExtraMounts[0].FileSystem),
			},
		},
	}
	if order.Attrs.FlavorKibana == nil {
		state.FlavorKibana = nil
	} else {
		state.FlavorKibana = &flavor.FlavorModel{
			Cores:  types.Int64Value(order.Attrs.FlavorKibana.Cores),
			Memory: types.Int64Value(order.Attrs.FlavorKibana.Memory),
			UUID:   types.StringValue(order.Attrs.FlavorKibana.UUID),
			Name:   types.StringValue(order.Attrs.FlavorKibana.Name),
		}
	}

	if order.Attrs.KibanaExtraMounts == nil {
		state.KibanaExtraMounts = nil
	} else {
		state.KibanaExtraMounts = map[string]common.ExtraMountModel{
			"/app/kibana": {
				Size:       types.Int64Value(order.Attrs.KibanaExtraMounts[0].Size),
				FileSystem: types.StringValue(order.Attrs.KibanaExtraMounts[0].FileSystem),
			},
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

}

func (r ElasticSearchResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state ElasticSearchResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetElasticSearchOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)

	if err != nil {
		resp.Diagnostics.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Can't get order data from portal.\nError: %v", err.Error()),
		)
		return
	}

	finProj, err := sources.GetFinancialProjectByName(
		r.client.Creds,
		r.client.ProjectName,
		plan.FinancialProject.ValueString(),
	)

	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("financial_project"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Get financial source from portal ended with error.\nError: %v", err.Error()),
		)
		return
	}

	labelChanged := plan.Label != state.Label
	finProjChanged := plan.FinancialProject != state.FinancialProject
	kibanaPassChanged := !plan.KibanaPassword.Equal(state.KibanaPassword)
	fluentdPassChanged := !plan.FluentdPassword.Equal(state.FluentdPassword)
	masterFlavorChanged := plan.FlavorMaster != state.FlavorMaster
	dataFlavorChanged := plan.FlavorData != state.FlavorData
	coordinatorFlavorChanged := plan.FlavorCoordintor != state.FlavorCoordintor
	nodesCountChanged := plan.ElasticSearchNodesCount != state.ElasticSearchNodesCount
	dataExtraMountChanged := utils.IsExtraMountChanged(state.DataExtraMounts, plan.DataExtraMounts)

	if labelChanged {
		utils.ChangeOrderLabel(order, plan.Label.ValueString(), resp)
	}

	if finProjChanged {
		r.changeFinancialProject(order, finProj.ID, resp)
	}

	if kibanaPassChanged {
		r.changeKibanaPassword(order, plan.KibanaPassword.ValueString(), resp)
	}

	if fluentdPassChanged {
		r.changeFluentdPassword(order, plan.FluentdPassword.ValueString(), resp)
	}

	if masterFlavorChanged {
		r.changeMasterFlavor(order, plan.FlavorMaster, resp)
	}

	if dataFlavorChanged {
		r.changeDataFlavor(order, plan.FlavorData, resp)
	}

	if coordinatorFlavorChanged {
		r.changeCoordinatorFlavor(order, plan.FlavorCoordintor, resp)
	}

	if nodesCountChanged {
		r.changeNodesCount(order, plan.ElasticSearchNodesCount, resp)
	}

	if dataExtraMountChanged {
		r.expandDataAppExtraMount(order, plan.DataExtraMounts["/app/"], resp)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r ElasticSearchResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state ElasticSearchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetElasticSearchOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)

	if err != nil {
		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf("Get order data from portal ended with error.\nError: %v", err.Error()),
		)
		return
	}

	orderState, err := order.GetState()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf("Get order state from portal ended with error.\nError: %v", err.Error()),
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

	err = order.DeleteElasticSearch()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf("Delete order from portal ended with error.\nError: %v", err.Error()),
		)
		return
	}
}

func (r ElasticSearchResource) prepareElasticAttrs(
	ctx context.Context,
	plan ElasticSearchResourceModel,
) (attrs orders.ElasticSearchAttrs) {

	var systemAdmGroup, userAppGroups, admAppGroups []string
	plan.SystemAdmGroup.ElementsAs(ctx, &systemAdmGroup, false)
	plan.UserAppGroups.ElementsAs(ctx, &userAppGroups, false)
	plan.AdmAppGroups.ElementsAs(ctx, &admAppGroups, false)

	attrs = orders.ElasticSearchAttrs{
		Accept:           true,
		ADIntegration:    plan.Image.ADIntegration.ValueBool(),
		OnSupport:        plan.Image.OnSupport.ValueBool(),
		ADLogonGrants:    utils.PrepareADLogonGrants(plan.Access),
		AvailabilityZone: plan.Core.Zone.ValueString(),
		Domain:           plan.Core.Domain.ValueString(),
		Platform:         plan.Core.Platform.ValueString(),
		DefaultNic: entities.DefaultNic{
			NetSegment: plan.Core.NetSegmentCode.ValueString(),
		},

		ElasticSearchVersion: plan.ElasticSearchVersion.ValueString(),
		ClusterName:          plan.ClusterName.ValueString(),
		KibanaPassword:       plan.KibanaPassword.ValueString(),
		FluentdPassword:      plan.FluentdPassword.ValueString(),
		NodesCount: orders.NodesCountElastic{
			Data:        plan.ElasticSearchNodesCount.Data.ValueInt64(),
			Coordinator: plan.ElasticSearchNodesCount.Coordinator.ValueInt64(),
			Master:      plan.ElasticSearchNodesCount.Master.ValueInt64(),
		},

		InstallKibana:  plan.InstallKibana.ValueBool(),
		KibanaLocation: plan.KibanaLocation.ValueStringPointer(),

		FlavorData: entities.Flavor{
			Cores:  plan.FlavorData.Cores.ValueInt64(),
			Memory: plan.FlavorData.Memory.ValueInt64(),
			Name:   plan.FlavorData.Name.ValueString(),
			UUID:   plan.FlavorData.UUID.ValueString(),
		},
		FlavorMaster: entities.Flavor{
			Cores:  plan.FlavorMaster.Cores.ValueInt64(),
			Memory: plan.FlavorMaster.Memory.ValueInt64(),
			Name:   plan.FlavorMaster.Name.ValueString(),
			UUID:   plan.FlavorMaster.UUID.ValueString(),
		},
		FlavorCoordinator: entities.Flavor{
			Cores:  plan.FlavorCoordintor.Cores.ValueInt64(),
			Memory: plan.FlavorCoordintor.Memory.ValueInt64(),
			Name:   plan.FlavorCoordintor.Name.ValueString(),
			UUID:   plan.FlavorCoordintor.UUID.ValueString(),
		},

		DataExtraMounts:        utils.PrepareExtraMountsAttrs(plan.DataExtraMounts),
		MasterExtraMounts:      utils.PrepareExtraMountsAttrs(plan.MasterExtraMounts),
		CoordinatorExtraMounts: utils.PrepareExtraMountsAttrs(plan.CoordinatorExtraMounts),

		SystemAdmGroup:      systemAdmGroup,
		UserAppGroups:       userAppGroups,
		AdmAppGroups:        admAppGroups,
		GeoDistribution:     plan.Image.GeoDistribution.ValueBoolPointer(),
		OsVersion:           plan.Image.OsVersion.ValueString(),
		Layout:              plan.Layout.ValueString(),
		CreatedWithOpenTofu: true,
	}

	var kibanaFlavor *entities.Flavor
	if plan.FlavorKibana != nil {
		kibanaFlavor = &entities.Flavor{
			Cores:  plan.FlavorKibana.Cores.ValueInt64(),
			Memory: plan.FlavorKibana.Memory.ValueInt64(),
			Name:   plan.FlavorKibana.Name.ValueString(),
			UUID:   plan.FlavorKibana.UUID.ValueString(),
		}
	}

	var kibanaExtraMounts []entities.ExtraMount
	if plan.KibanaExtraMounts != nil {
		kibanaExtraMounts = utils.PrepareExtraMountsAttrs(plan.KibanaExtraMounts)
	}

	attrs.FlavorKibana = kibanaFlavor
	attrs.KibanaExtraMounts = kibanaExtraMounts

	return attrs
}

func (r ElasticSearchResource) changeFinancialProject(
	order *orders.ElasticSearch,
	finProjectId string,
	resp *resource.UpdateResponse,
) {
	err := order.ChangeFinancialProject(finProjectId)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("financial_project"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Change financial source for order ender with error.\nError: %v", err.Error()),
		)
		return
	}
}

func (r ElasticSearchResource) changeKibanaPassword(
	order *orders.ElasticSearch,
	newPassword string,
	resp *resource.UpdateResponse,
) {
	err := order.ChangeOpenSearchKibanaPassword(newPassword)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("kibana_password"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Change 'kibana_password' ended with error.\nError: %v", err.Error()),
		)
		return
	}
}

func (r ElasticSearchResource) changeFluentdPassword(
	order *orders.ElasticSearch,
	newPassword string,
	resp *resource.UpdateResponse,
) {
	err := order.ChangeOpenSearchFluentdPassword(newPassword)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("fluentd_password"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Change 'fluentd_password' ended with error.\nError: %v", err.Error()),
		)
		return
	}
}

func (r ElasticSearchResource) changeMasterFlavor(
	order *orders.ElasticSearch,
	flavor flavor.FlavorModel,
	resp *resource.UpdateResponse,
) {
	err := order.ChangeFlavorMasterNodes(
		entities.Flavor{
			Cores:  flavor.Cores.ValueInt64(),
			Memory: flavor.Memory.ValueInt64(),
			UUID:   flavor.UUID.ValueString(),
			Name:   flavor.Name.ValueString(),
		},
	)

	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("flavor_master"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Change flavor for master nodes ended with error.\nError: %v", err.Error()),
		)
		return
	}
}

func (r ElasticSearchResource) changeDataFlavor(
	order *orders.ElasticSearch,
	flavor flavor.FlavorModel,
	resp *resource.UpdateResponse,
) {
	err := order.ChangeFlavorDataNodes(
		entities.Flavor{
			Cores:  flavor.Cores.ValueInt64(),
			Memory: flavor.Memory.ValueInt64(),
			UUID:   flavor.UUID.ValueString(),
			Name:   flavor.Name.ValueString(),
		},
	)

	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("flavor_data"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Change flavor for data nodes ended with error.\nError: %v", err.Error()),
		)
		return
	}
}

func (r ElasticSearchResource) changeCoordinatorFlavor(
	order *orders.ElasticSearch,
	flavor flavor.FlavorModel,
	resp *resource.UpdateResponse,
) {
	err := order.ChangeFlavorCoordinatorNodes(
		entities.Flavor{
			Cores:  flavor.Cores.ValueInt64(),
			Memory: flavor.Memory.ValueInt64(),
			Name:   flavor.Name.ValueString(),
			UUID:   flavor.UUID.ValueString(),
		},
	)

	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("flavor_coordinator"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Change flavor for coordiantor nodes ended with error.\nError: %v", err.Error()),
		)
		return
	}
}

func (r ElasticSearchResource) changeNodesCount(
	order *orders.ElasticSearch,
	nodesCount NodesCount,
	resp *resource.UpdateResponse,
) {
	err := order.ScaleOpenSearchCluster(
		orders.NodesCountElastic{
			Master:      nodesCount.Master.ValueInt64(),
			Data:        nodesCount.Data.ValueInt64(),
			Coordinator: nodesCount.Coordinator.ValueInt64(),
		},
		r.client.Environment,
	)

	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("nodes_count"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Change nodes count ended with error.\nError: %v", err.Error()),
		)
		return
	}
}

func (r ElasticSearchResource) expandDataAppExtraMount(
	order *orders.ElasticSearch,
	extraMount common.ExtraMountModel,
	resp *resource.UpdateResponse,
) {
	err := order.ExpandDataAppExtraMount(
		entities.ExtraMount{
			Path:       "/app",
			Size:       extraMount.Size.ValueInt64(),
			FileSystem: extraMount.FileSystem.ValueString(),
		},
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("data_extra_mounts"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Expand /app extra mount ended with error.\nError: %v", err.Error()),
		)
		return
	}
}

func (r ElasticSearchResource) validateNodesCount(
	nodesCount NodesCount,
	resp *resource.ModifyPlanResponse,
) {
	_, err := references.GetElasticSearchNodeCounts(
		r.client.Creds,
		nodesCount.Data.ValueInt64(),
		nodesCount.Master.ValueInt64(),
		nodesCount.Coordinator.ValueInt64(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("nodes_count"),
			consts.MODIFY_PLAN_FAIL,
			fmt.Sprintf("Get elastic nodes count ended with error.\nError: %v", err.Error()),
		)
		return
	}

	if strings.EqualFold(r.client.Environment, "prod") && nodesCount.Master.ValueInt64() < 3 {
		resp.Diagnostics.AddAttributeError(
			path.Root("nodes_count"),
			consts.MODIFY_PLAN_FAIL,
			fmt.Sprintf("Master nodes are not less 3 in 'prod' environment, got nodes: %v", nodesCount.Master.ValueInt64()),
		)
		return
	}
}

func (r ElasticSearchResource) validateKibanaConfiguration(
	plan ElasticSearchResourceModel,
	resp *resource.ModifyPlanResponse,
) {
	if plan.InstallKibana.ValueBool() {
		if plan.KibanaLocation.IsNull() || plan.KibanaLocation.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("kibana_location"),
				consts.MODIFY_PLAN_FAIL,
				"'kibana_location' must be set set when install_kibana is false",
			)
		}

		if plan.FlavorKibana == nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("flavor_kibana"),
				consts.MODIFY_PLAN_FAIL,
				"'flavor_kibana' must be set when install_kibana is true",
			)
		}

		if plan.KibanaExtraMounts == nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("kibana_extra_mounts"),
				consts.MODIFY_PLAN_FAIL,
				"'kibana_extra_mounts' must be set when install_kibana is true",
			)
		}
	} else {
		if !plan.KibanaLocation.IsNull() && !plan.KibanaLocation.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("kibana_location"),
				consts.MODIFY_PLAN_FAIL,
				"'kibana_location' does not be set when install_kibana is false",
			)
		}
		if plan.FlavorKibana != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("flavor_kibana"),
				consts.MODIFY_PLAN_FAIL,
				"'flavor_kibana' does not be set when install_kibana is false",
			)
		}

		if plan.KibanaExtraMounts != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("kibana_extra_mounts"),
				consts.MODIFY_PLAN_FAIL,
				"'kibana_extra_mounts' does not be set when install_kibana is false",
			)
		}
	}
}

func (r ElasticSearchResource) validateSystemAdmGroups(
	plan ElasticSearchResourceModel,
	resp *resource.ModifyPlanResponse,
) {
	if strings.EqualFold(r.client.Environment, "prod") || strings.EqualFold(r.client.Environment, "test") {
		if !plan.SystemAdmGroup.IsNull() && !plan.SystemAdmGroup.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("system_adm_groups"),
				consts.MODIFY_PLAN_FAIL,
				"'system_adm_groups' not availiable in 'prod' and 'test' environments",
			)
		}
	} else {
		if plan.SystemAdmGroup.IsNull() || plan.SystemAdmGroup.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root("system_adm_groups"),
				consts.MODIFY_PLAN_FAIL,
				"'system_adm_groups' is required field in 'dev' environment",
			)
		}
	}
}
