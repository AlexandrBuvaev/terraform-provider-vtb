package ktaas

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/consts"
	"terraform-provider-vtb/internal/custommodifires"
	"terraform-provider-vtb/internal/utils"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"
	"terraform-provider-vtb/pkg/client/productcatalog"
	"terraform-provider-vtb/pkg/client/sources"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                   = &KTaaSResource{}
	_ resource.ResourceWithImportState    = &KTaaSResource{}
	_ resource.ResourceWithValidateConfig = &KTaaSResource{}
	_ resource.ResourceWithModifyPlan     = &KTaaSResource{}
)

type KTaaSResource struct {
	client *client.CloudClient
}

func NewKTaaSResource() resource.Resource {
	return &KTaaSResource{}
}

func (r KTaaSResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_ktaas_instance"
}

func (r *KTaaSResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type KTaaSResourceModel struct {
	Label            types.String `tfsdk:"label"`
	OrderID          types.String `tfsdk:"order_id"`
	ItemID           types.String `tfsdk:"item_id"`
	Lifetime         types.Int64  `tfsdk:"lifetime"`
	FinancialProject types.String `tfsdk:"financial_project"`

	TopicName        types.String `tfsdk:"topic_name"`
	TopicFlavor      types.Int64  `tfsdk:"topic_flavor"`
	BillingFlavor    types.String `tfsdk:"billing_flavor"`
	PartitionsNumber types.Int64  `tfsdk:"partitions_number"`
	NetSegment       types.String `tfsdk:"net_segment"`
	KafkaClusterName types.String `tfsdk:"kafka_cluster_name"`
	Acls             types.Set    `tfsdk:"acls"`
	GroupAcls        types.Set    `tfsdk:"group_acls"`
}

type KTaaSAclsModel struct {
	ClientCN   types.String `tfsdk:"client_cn"`
	ClientRole types.String `tfsdk:"client_role"`
}

func (m KTaaSAclsModel) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"client_cn":   types.StringType,
		"client_role": types.StringType,
	}
}

var aclSchema = map[string]schema.Attribute{
	"client_cn": schema.StringAttribute{
		Required:            true,
		Description:         "CN сертификата клиента",
		MarkdownDescription: "CN сертификата клиента",
	},
	"client_role": schema.StringAttribute{
		Required: true,
		Validators: []validator.String{
			stringvalidator.OneOf("consumer", "producer"),
		},
		Description:         "Роль. Допустимые значения: 'producer, consumer'",
		MarkdownDescription: "Роль. Допустимые значения: 'producer, consumer'",
	},
}

type KTaaSGroupAclsModel struct {
	ClientCN  types.String `tfsdk:"client_cn"`
	GroupName types.String `tfsdk:"group_name"`
}

func (m KTaaSGroupAclsModel) AttrType() map[string]attr.Type {
	return map[string]attr.Type{
		"client_cn":  types.StringType,
		"group_name": types.StringType,
	}
}

var groupAclsSchema = map[string]schema.Attribute{
	"client_cn": schema.StringAttribute{
		Computed:            true,
		Default:             stringdefault.StaticString("*"),
		Description:         "CN сертификата клиента",
		MarkdownDescription: "CN сертификата клиента",
	},
	"group_name": schema.StringAttribute{
		Required:            true,
		Description:         "Название группы",
		MarkdownDescription: "Название группы",
	},
}

func (r KTaaSResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"label": schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
				Description:         "Метка заказа",
				MarkdownDescription: "Метка заказа",
			},
			"lifetime": schema.Int64Attribute{
				Optional: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
				Description:         "Время жизни заказа в днях(2, 7, 14, 30)",
				MarkdownDescription: "Время жизни заказа в днях(2, 7, 14, 30)",
			},
			"order_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description:         "Идентификатор заказа. Известен после создания заказа",
				MarkdownDescription: "Идентификатор заказа. Известен после создания заказа",
			},
			"item_id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description:         "Идентификатор сущности ВМ в заказе. Известен после создания заказа",
				MarkdownDescription: "Идентификатор сущности ВМ в заказа. Известен после создания заказа",
			},
			"financial_project": schema.StringAttribute{
				Required:            true,
				Description:         "Источник финансирования",
				MarkdownDescription: "Источник финансирования",
			},
			// custom fields
			"topic_name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description:         "Название топика",
				MarkdownDescription: "Название топика",
			},
			"topic_flavor": schema.Int64Attribute{
				Required:            true,
				Description:         "Размер топика",
				MarkdownDescription: "Размер топика",
			},
			"billing_flavor": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Description:         "Размер топика для расчета стоимости с учетом фактора репликации",
				MarkdownDescription: "Размер топика для расчета стоимости с учетом фактора репликации",
			},
			"partitions_number": schema.Int64Attribute{
				Required: true,
				Validators: []validator.Int64{
					int64validator.OneOf([]int64{1, 2, 4, 6, 8, 10, 12}...),
				},
				PlanModifiers: []planmodifier.Int64{
					custommodifires.GrowOnly(),
				},
				Description:         "Количество разделов",
				MarkdownDescription: "Количество разделов",
			},
			"net_segment": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description:         "Сетевой сегмент",
				MarkdownDescription: "Сетевой сегмент",
			},
			"kafka_cluster_name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description:         "Кластер Kafka для размещения топика",
				MarkdownDescription: "Кластер Kafka для размещения топика",
			},
			"acls": schema.SetNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: aclSchema,
				},
				Description:         "ACL на доступ",
				MarkdownDescription: "ACL на доступ",
			},
			"group_acls": schema.SetNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: groupAclsSchema,
				},
				Description:         "ACL на группы",
				MarkdownDescription: "ACL на группы",
			},
		},
	}
}

func (r KTaaSResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r KTaaSResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan KTaaSResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	checkOrderIsDeleted := utils.CheckOrderIsDeleted(r.client, plan.OrderID, "paas_ktaas", "ktaas")
	resp.Diagnostics.Append(checkOrderIsDeleted.Diagnostics...)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r KTaaSResource) ValidateConfig(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	var config KTaaSResourceModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if config.GroupAcls.IsUnknown() || config.GroupAcls.IsNull() {
		return
	}

	elems := config.GroupAcls.Elements()

	for _, elem := range elems {
		groupAclObj, ok := elem.(types.Object)
		if !ok {
			resp.Diagnostics.AddError(
				consts.VALIDATION_FAIL,
				"Each group ACL must be an object.",
			)
			return
		}

		groupNameVal, exists := groupAclObj.Attributes()["group_name"]
		if !exists {
			resp.Diagnostics.AddError(
				consts.VALIDATION_FAIL,
				"Each group ACL must have a 'group_name' attribute.",
			)
			return
		}

		groupName, ok := groupNameVal.(types.String)
		if !ok {
			resp.Diagnostics.AddError(
				consts.VALIDATION_FAIL,
				"The 'group_name' attribute must be a string.",
			)
			return
		}

		if groupName.IsUnknown() || groupName.IsNull() {
			continue
		}

		r.validateGroupName(config.TopicName, groupName, resp)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}

func (r KTaaSResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan KTaaSResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.BillingFlavor = r.generateBillingFlavor(plan.TopicFlavor)

	finProj, err := sources.GetFinancialProjectByName(
		r.client.Creds,
		r.client.ProjectName,
		plan.FinancialProject.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("financial_project"), consts.CREATE_RES_FAIL, err.Error(),
		)
		return
	}

	productImageData, err := productcatalog.GetProductImageData(
		r.client.Creds,
		"kafka_topic_v2",
		r.client.Environment,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("get product image data ended with error.\nError: %v", err.Error()),
		)
	}

	attrs := r.prepareAttrs(plan, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	order := orders.NewKTaaS(
		r.client.Creds,
		r.client.ProjectName,
		productImageData.ID,
		*attrs,
	)

	err = order.Create(
		orders.CreateOrderPayload{
			Label:        plan.Label.ValueString(),
			Lifetime:     int(plan.Lifetime.ValueInt64()),
			FinProjectID: finProj.ID,
		},
	)

	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Create KTaaS instance ended with error.\nError: %v", err.Error()),
		)
		return
	}

	parentItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Get parent item ended with error. Error: %v", err.Error()),
		)
		return
	}

	if !plan.Acls.IsNull() || !plan.Acls.IsUnknown() {
		stateAcls := types.SetNull(plan.Acls.ElementType(ctx))
		diags := r.applyAcls(ctx, order, plan.Acls, stateAcls)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !plan.GroupAcls.IsNull() || !plan.Acls.IsUnknown() {
		stateGroupAcls := types.SetNull(plan.GroupAcls.ElementType(ctx))
		diags := r.applyGroupAcls(ctx, order, plan.GroupAcls, stateGroupAcls)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(parentItem.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

}

func (r KTaaSResource) Read(
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

	order, err := orders.GetKTaaSOrder(
		r.client.Creds,
		r.client.ProjectName,
		orderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("order_id"), consts.READ_RES_FAIL, err.Error())
		return
	}

	item, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Get parent item ended with error.\nError: %v", err.Error()),
		)
		return
	}

	config, ok := item.Data.Config.(entities.KTaaSConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Invalid type for item config",
		)
		return
	}

	var tfAcls, tfGroupAcls basetypes.SetValue
	if len(config.ACLS) > 0 {
		var diags diag.Diagnostics
		tfAcls, diags = types.SetValueFrom(
			ctx,
			types.ObjectType{AttrTypes: KTaaSAclsModel{}.AttrType()},
			config.ACLS,
		)
		if diags.HasError() {
			return
		}
	} else {
		tfAcls = types.SetNull(types.ObjectType{AttrTypes: KTaaSAclsModel{}.AttrType()})
	}

	if len(config.GroupAcls) > 0 {
		var diags diag.Diagnostics
		tfGroupAcls, diags = types.SetValueFrom(
			ctx,
			types.ObjectType{AttrTypes: KTaaSGroupAclsModel{}.AttrType()},
			config.GroupAcls,
		)
		if diags.HasError() {
			return
		}
	} else {
		tfGroupAcls = types.SetNull(types.ObjectType{AttrTypes: KTaaSGroupAclsModel{}.AttrType()})
	}

	state := KTaaSResourceModel{
		OrderID:          orderID,
		ItemID:           types.StringValue(item.ID),
		Label:            types.StringValue(order.Label),
		FinancialProject: types.StringValue(order.FinancialSource.Name),

		TopicName:        types.StringValue(config.TopicName),
		TopicFlavor:      types.Int64Value(config.TopicFlavor),
		BillingFlavor:    types.StringValue(order.Attrs.BillingFlavor),
		PartitionsNumber: types.Int64Value(config.PartitionsNumber),
		NetSegment:       types.StringValue(config.NetSegment),
		KafkaClusterName: types.StringValue(config.ResourcePool.ResourcePoolName),
		Acls:             tfAcls,
		GroupAcls:        tfGroupAcls,
	}

	var lifetime types.Int64
	req.State.GetAttribute(ctx, path.Root("lifetime"), &lifetime)
	if !lifetime.IsNull() {
		state.Lifetime = lifetime
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r KTaaSResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state KTaaSResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetKTaaSOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Get KTaaS order ended with error.\nError: %v", err.Error()),
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
			fmt.Sprintf("Get financial project for order ended with error.\nError: %v", err.Error()),
		)
		return
	}

	labelChanged := !plan.Label.Equal(state.Label)
	finProjChanged := !plan.FinancialProject.Equal(state.FinancialProject)
	topicFlavorChanged := !plan.TopicFlavor.Equal(state.TopicFlavor)
	partitionsChanged := !plan.PartitionsNumber.Equal(state.PartitionsNumber)
	aclsChanged := !plan.Acls.Equal(state.Acls)
	groupAclsChanged := !plan.GroupAcls.Equal(state.GroupAcls)

	if labelChanged {
		utils.ChangeOrderLabel(order, plan.Label.ValueString(), resp)
	}

	if finProjChanged {
		r.changeFinancialProject(order, finProj.ID, resp)
	}

	if topicFlavorChanged {
		r.changeTopicSize(order, plan, resp)
	}

	if partitionsChanged {
		r.changePartitionsTopic(order, plan.PartitionsNumber.ValueInt64(), resp)
	}

	if aclsChanged {
		diags := r.applyAcls(ctx, order, plan.Acls, state.Acls)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if groupAclsChanged {
		diags := r.applyGroupAcls(ctx, order, plan.GroupAcls, state.GroupAcls)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r KTaaSResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state KTaaSResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetKTaaSOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf("Get KTaaS order ended with error.\nError: %v", err.Error()),
		)
		return
	}

	orderState, err := order.GetState()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf("Get State order ended with error.\nError: %v", err.Error()),
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

	err = order.DeleteKTaaS()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf(
				"Delete KTaaS ended with error.\nError: %v", err.Error(),
			),
		)
		return
	}
}

// custom logic
func (r KTaaSResource) prepareAttrs(
	plan KTaaSResourceModel,
	resp *resource.CreateResponse,
) *orders.KTaaSAttrs {

	kafkaCluster, err := sources.GetKTaaSCluster(
		r.client.Creds,
		r.client.ProjectName,
		r.client.EnvPrefix,
		r.client.Environment,
		plan.NetSegment.ValueString(),
		plan.KafkaClusterName.ValueString(),
		plan.TopicFlavor.ValueInt64(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Get Kafka Cluster for KTaaS ended with error.\nError:%v", err.Error()),
		)
		return nil
	}

	attrs := &orders.KTaaSAttrs{
		Name:    plan.TopicName.ValueString(),
		Cluster: *kafkaCluster,
		DefaultNic: entities.DefaultNic{
			NetSegment: plan.NetSegment.ValueString(),
		},
		Platform:            "kafka_topic",
		TopicFlavor:         plan.TopicFlavor.ValueInt64(),
		BillingFlavor:       plan.BillingFlavor.ValueString(),
		Accept:              true,
		PartitionsNumber:    plan.PartitionsNumber.ValueInt64(),
		CreatedWithOpenTofu: true,
	}

	return attrs
}

func (r KTaaSResource) generateBillingFlavor(
	topicFlavor types.Int64,
) (billingFlavor types.String) {
	tempFlavor := 2 * topicFlavor.ValueInt64()
	billingFlavor = types.StringValue(strconv.FormatInt(tempFlavor, 10))
	return billingFlavor
}

// Изменение источника финансирования
func (r KTaaSResource) changeFinancialProject(
	order *orders.KTaaS,
	finProjID string,
	resp *resource.UpdateResponse,
) {
	err := order.ChangeFinancialProject(finProjID)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Change financial prodject for order ended with error.\nError: %v", err.Error()),
		)
		return
	}
}

// Изменение размера топика
func (r KTaaSResource) changeTopicSize(
	order *orders.KTaaS,
	plan KTaaSResourceModel,
	resp *resource.UpdateResponse,
) {
	billingFlavor := r.generateBillingFlavor(plan.TopicFlavor)
	attrs := orders.ChangeSizeTopicAttrs{
		TopicFlavor:   plan.TopicFlavor.ValueInt64(),
		BillingFlavor: billingFlavor.ValueString(),
	}
	err := order.KTaaSChangeSizeTopic(attrs)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Change size topic ended with error.\nError: %v", err.Error()),
		)
		return
	}
}

// Изменение количества разделов
func (r KTaaSResource) changePartitionsTopic(
	order *orders.KTaaS,
	partitionsNumber int64,
	resp *resource.UpdateResponse,
) {
	err := order.KTaaSChangePartitionsTopic(partitionsNumber)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Change partitions topic ended with error.\nError: %v", err.Error()),
		)
		return
	}
}

// сравнение ACL на доступ
func (r KTaaSResource) compareAcls(
	ctx context.Context,
	planAcls, stateAcls types.Set,
) (toAdd, toDelete []KTaaSAclsModel, diags diag.Diagnostics) {

	planMap := make(map[string]KTaaSAclsModel)
	stateMap := make(map[string]KTaaSAclsModel)

	getAclKey := func(acl KTaaSAclsModel) string {
		return fmt.Sprintf("%s|%s", acl.ClientCN.ValueString(), acl.ClientRole.ValueString())
	}

	if !planAcls.IsNull() && !planAcls.IsUnknown() {
		var planAclsList []KTaaSAclsModel
		diags.Append(planAcls.ElementsAs(ctx, &planAclsList, false)...)
		if diags.HasError() {
			return
		}

		for _, acl := range planAclsList {
			planMap[getAclKey(acl)] = acl
		}
	}

	if !stateAcls.IsNull() && !stateAcls.IsUnknown() {
		var stateAclsList []KTaaSAclsModel
		diags.Append(stateAcls.ElementsAs(ctx, &stateAclsList, false)...)
		if diags.HasError() {
			return
		}

		for _, acl := range stateAclsList {
			stateMap[getAclKey(acl)] = acl
		}
	}

	for key, planAcl := range planMap {
		if _, exists := stateMap[key]; !exists {
			toAdd = append(toAdd, planAcl)
		}
	}

	for key, stateAcl := range stateMap {
		if _, exists := planMap[key]; !exists {
			toDelete = append(toDelete, stateAcl)
		}
	}

	return toAdd, toDelete, diags
}

// Применение изменений для ACL на доступ
func (r KTaaSResource) applyAcls(
	ctx context.Context,
	order *orders.KTaaS,
	planAcls, stateAcls types.Set,
) (diags diag.Diagnostics) {
	toAdd, toDelete, compareDiags := r.compareAcls(ctx, planAcls, stateAcls)
	diags.Append(compareDiags...)
	if diags.HasError() {
		return diags
	}

	if len(toAdd) > 0 {
		var aclToCreate []entities.KTaaSAcls
		for _, acl := range toAdd {
			aclToCreate = append(aclToCreate, entities.KTaaSAcls{
				ClientCN:   acl.ClientCN.ValueString(),
				ClientRole: acl.ClientRole.ValueString(),
			})
		}

		err := order.KTaaSCreateAcls(aclToCreate)
		if err != nil {
			diags.AddError(
				"Error while creating ACLs",
				err.Error(),
			)
			return diags
		}
	}

	if len(toDelete) > 0 {
		var aclToDelete []orders.DeleteAclsAttrs
		for _, acl := range toDelete {
			aclToDelete = append(
				aclToDelete,
				orders.DeleteAclsAttrs{
					RawData: entities.KTaaSAcls{
						ClientCN:   acl.ClientCN.ValueString(),
						ClientRole: acl.ClientRole.ValueString(),
					},
				},
			)
		}

		err := order.KTaaSDeleteAcls(aclToDelete)
		if err != nil {
			diags.AddError(
				"Error while deleting ACLs",
				err.Error(),
			)
			return diags
		}
	}

	return diags
}

// сравнение групповых ACL
func (r KTaaSResource) compareGroupAcls(
	ctx context.Context,
	planGroupAcls, stateGroupAcls types.Set,
) (toAdd, toDelete []KTaaSGroupAclsModel, diags diag.Diagnostics) {

	planMap := make(map[string]KTaaSGroupAclsModel)
	stateMap := make(map[string]KTaaSGroupAclsModel)

	getGroupAclKey := func(groupAcl KTaaSGroupAclsModel) string {
		return fmt.Sprintf("%s|%s", groupAcl.ClientCN.ValueString(), groupAcl.GroupName.ValueString())
	}

	if !planGroupAcls.IsNull() && !planGroupAcls.IsUnknown() {
		var planGroupAclsList []KTaaSGroupAclsModel
		diags.Append(planGroupAcls.ElementsAs(ctx, &planGroupAclsList, false)...)
		if diags.HasError() {
			return
		}

		for _, groupAcl := range planGroupAclsList {
			planMap[getGroupAclKey(groupAcl)] = groupAcl
		}
	}

	if !stateGroupAcls.IsNull() && !stateGroupAcls.IsUnknown() {
		var stateGroupAclsList []KTaaSGroupAclsModel
		diags.Append(stateGroupAcls.ElementsAs(ctx, &stateGroupAclsList, false)...)
		if diags.HasError() {
			return
		}

		for _, groupAcl := range stateGroupAclsList {
			stateMap[getGroupAclKey(groupAcl)] = groupAcl
		}
	}

	for key, planGroupAcl := range planMap {
		if _, exists := stateMap[key]; !exists {
			toAdd = append(toAdd, planGroupAcl)
		}
	}

	for key, stateGroupAcl := range stateMap {
		if _, exists := planMap[key]; !exists {
			toDelete = append(toDelete, stateGroupAcl)
		}
	}

	return toAdd, toDelete, diags
}

// Применение изменений для групповых acl
func (r KTaaSResource) applyGroupAcls(
	ctx context.Context,
	order *orders.KTaaS,
	planGroupAcls, stateGroupAcls types.Set,
) (diags diag.Diagnostics) {

	toAdd, toDelete, compareDiags := r.compareGroupAcls(ctx, planGroupAcls, stateGroupAcls)
	diags.Append(compareDiags...)
	if diags.HasError() {
		return diags
	}

	if len(toAdd) > 0 {
		var groupAclToCreate []entities.KtaaSGroupAcls
		for _, groupAcl := range toAdd {
			groupAclToCreate = append(groupAclToCreate, entities.KtaaSGroupAcls{
				ClientCN:  groupAcl.ClientCN.ValueString(),
				GroupName: groupAcl.GroupName.ValueString(),
			})
		}

		err := order.KTaaSCreateGroupAcls(groupAclToCreate)
		if err != nil {
			diags.AddError(
				"Error while creating Group ACLs",
				err.Error(),
			)
			return diags
		}
	}

	if len(toDelete) > 0 {
		var groupAclToDelete []orders.DeleteGroupAclsAttrs
		for _, groupAcl := range toDelete {
			groupAclToDelete = append(
				groupAclToDelete,
				orders.DeleteGroupAclsAttrs{
					RawData: entities.KtaaSGroupAcls{
						ClientCN:  groupAcl.ClientCN.ValueString(),
						GroupName: groupAcl.GroupName.ValueString(),
					},
				},
			)
		}

		err := order.KTaaSDeleteGroupAcls(groupAclToDelete)
		if err != nil {
			diags.AddError(
				"Error while deleting Group ACLs",
				err.Error(),
			)
			return diags
		}
	}

	return diags
}

func (r KTaaSResource) validateGroupName(
	topicName types.String,
	groupName types.String,
	resp *resource.ValidateConfigResponse,
) {
	expectedPrefix := topicName.ValueString() + "_consumergroup_"
	if !strings.HasPrefix(groupName.ValueString(), expectedPrefix) {
		resp.Diagnostics.AddError(
			"VALIDATION_FAIL",
			fmt.Sprintf("group_name must start with: '%s'", expectedPrefix),
		)
		return
	}
}
