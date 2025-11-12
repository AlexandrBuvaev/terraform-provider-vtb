package rqaas

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/consts"
	"terraform-provider-vtb/internal/utils"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"
	"terraform-provider-vtb/pkg/client/productcatalog"
	"terraform-provider-vtb/pkg/client/sources"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &RQaaSResource{}
	_ resource.ResourceWithImportState = &RQaaSResource{}
	_ resource.ResourceWithModifyPlan  = &RQaaSResource{}
)

type RQaaSResource struct {
	client *client.CloudClient
}

func NewRQaaSResource() resource.Resource {
	return &RQaaSResource{}
}

func (r RQaaSResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_rqaas_instance"
}

func (r *RQaaSResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type RQaaSResourceModel struct {
	Label            types.String                `tfsdk:"label"`
	Name             types.String                `tfsdk:"name"`
	Cluster          RQaaSClusterDataSourceModel `tfsdk:"cluster"`
	Lifetime         types.Int64                 `tfsdk:"lifetime"`
	OrderID          types.String                `tfsdk:"order_id"`
	ItemID           types.String                `tfsdk:"item_id"`
	FinancialProject types.String                `tfsdk:"financial_project"`
	QueueUsers       types.Set                   `tfsdk:"queue_users"`
}

type QueueUserModel struct {
	UserName types.String `tfsdk:"username"`
	Read     types.Bool   `tfsdk:"read"`
	Write    types.Bool   `tfsdk:"write"`
}

func (m QueueUserModel) AttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"username": types.StringType,
		"read":     types.BoolType,
		"write":    types.BoolType,
	}
}

var clusterSchema = map[string]schema.Attribute{
	"name": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Имя кластера RabbitMQ для RQaaS",
		Description:         "Имя кластера RabbitMQ для RQaaS",
	},
	"hosts": schema.ListAttribute{
		Required:            true,
		Description:         "Список хостов кластера",
		MarkdownDescription: "Список хостов кластера",
		ElementType:         types.StringType,
	},
	"platform": schema.StringAttribute{
		Required:            true,
		Description:         "Название платформы",
		MarkdownDescription: "Название платформы",
	},
	"domain": schema.StringAttribute{
		Required:            true,
		Description:         "Название доменной зоны",
		MarkdownDescription: "Название доменной зоны",
	},
	"net_segment": schema.StringAttribute{
		Required:            true,
		Description:         "Название сетевого сегмента",
		MarkdownDescription: "Название сетевого сегмента",
	},
	"zone": schema.StringAttribute{
		Required:            true,
		Description:         "Зона доступности",
		MarkdownDescription: "Зона доступности",
	},
}

var queueUserSchema = map[string]schema.Attribute{
	"username": schema.StringAttribute{
		Required:            true,
		Description:         "Имя пользователя",
		MarkdownDescription: "Имя пользователя",
	},
	"read": schema.BoolAttribute{
		Required:            true,
		Description:         "Флаг предоставляющий пользователю права на чтение",
		MarkdownDescription: "Флаг предостваляющий пользователю права на чтение",
	},
	"write": schema.BoolAttribute{
		Required:            true,
		Description:         "Флаг предоставляющий пользователю права на запись",
		MarkdownDescription: "Флаг предоставляющий пользователю права на запись",
	},
}

func (r RQaaSResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "Имя очереди",
				MarkdownDescription: "Имя очереди",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"label": schema.StringAttribute{
				Required:            true,
				Description:         "Метка заказа",
				MarkdownDescription: "Метка заказа",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"cluster": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          clusterSchema,
				Description:         "Параметры коммунального кластера RabbitMQ для RQaaS",
				MarkdownDescription: "Параметры коммунального кластера RabbitMQ для RQaaS",
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
			"order_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Идентификатор заказа, известен после создания заказа",
				MarkdownDescription: "Идентификатор заказа, известен после создания заказа",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"item_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Идентификатор сущности ВМ в заказе. Известен после создания заказа",
				MarkdownDescription: "Идентификатор сущности ВМ в заказе. Известен после создания заказа",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"financial_project": schema.StringAttribute{
				Required:            true,
				Description:         "Источник финансирования",
				MarkdownDescription: "Источник финансирования",
			},
			"queue_users": schema.SetNestedAttribute{
				Optional:            true,
				Description:         "Пользователи очереди",
				MarkdownDescription: "Пользователи очереди",
				NestedObject: schema.NestedAttributeObject{
					Attributes: queueUserSchema,
				},
			},
		},
	}
}

func (r RQaaSResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r RQaaSResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan RQaaSResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	checkOrderIsDeleted := utils.CheckOrderIsDeleted(
		r.client,
		plan.OrderID,
		"saas",
		"rqaas",
	)

	utils.HandleExternalDeletionRecreate(ctx, resp, &plan, checkOrderIsDeleted.IsDeleted, checkOrderIsDeleted.Diagnostics)

	if !plan.QueueUsers.IsNull() || !plan.QueueUsers.IsUnknown() {
		var planUsersList []QueueUserModel
		diags := plan.QueueUsers.ElementsAs(ctx, &planUsersList, false)
		if diags.HasError() {
			return
		}
		for _, user := range planUsersList {
			err := validateRQaasUsername(user.UserName.ValueString(), r.client.EnvPrefix)
			if err != nil {
				resp.Diagnostics.AddError(
					consts.MODIFY_PLAN_FAIL,
					fmt.Sprintf("invalid QueueUser.username.\nError: %v", err.Error()),
				)
				return
			}

		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r RQaaSResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan RQaaSResourceModel
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
	productImageData, err := productcatalog.GetProductImageData(
		r.client.Creds,
		"rqaas",
		r.client.Environment,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("get product image data ended with error.\nError: %v", err.Error()),
		)
		return
	}
	hosts := []string{}
	diag := plan.Cluster.Hosts.ElementsAs(ctx, &hosts, false)

	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	attrs := orders.RQaaSAttrs{
		Name: plan.Name.ValueString(),
		Cluster: entities.RQaaSCluster{
			ClusterName:      plan.Cluster.Name.ValueString(),
			NetSegment:       plan.Cluster.NetSegment.ValueString(),
			Platform:         plan.Cluster.Platform.ValueString(),
			Domain:           plan.Cluster.Domain.ValueString(),
			AvailabilityZone: plan.Cluster.Zone.ValueString(),
			Hosts:            hosts,
		},
		Platform: plan.Cluster.Platform.ValueString(),
		DefaultNic: entities.DefaultNic{
			NetSegment: plan.Cluster.NetSegment.ValueString(),
		},
		AvailabilityZone:    plan.Cluster.Zone.ValueString(),
		CreatedWithOpenTofu: true,
	}

	order := orders.NewRQaaS(
		r.client.Creds,
		r.client.ProjectName,
		productImageData.ID,
		attrs,
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
			fmt.Sprintf("Create RQaaS instance ended with error.\nError: %v", err.Error()),
		)
		return
	}

	if !plan.QueueUsers.IsNull() || !plan.QueueUsers.IsUnknown() {

		stateQueueUsers := types.SetNull(plan.QueueUsers.ElementType(ctx))
		diags := r.applyQueueUsers(ctx, order, plan.QueueUsers, stateQueueUsers)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	parentItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Get parent item ended with error.\nError: %v", err.Error()),
		)
		return
	}
	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(parentItem.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r RQaaSResource) Read(
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

	order, err := orders.GetRQaaSOrder(
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
	config, ok := item.Data.Config.(entities.RQaaSItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Ivalid type for item config",
		)
		return
	}

	tfhosts, diag := types.ListValueFrom(ctx, types.StringType, order.Attrs.Cluster.Hosts)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}
	tfQueueUsers, diag := types.SetValueFrom(ctx, types.ObjectType{AttrTypes: QueueUserModel{}.AttrTypes()}, config.QueueUsers)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := RQaaSResourceModel{
		OrderID:          orderID,
		ItemID:           types.StringValue(item.ID),
		Label:            types.StringValue(order.Label),
		Name:             types.StringValue(config.Name),
		FinancialProject: types.StringValue(order.FinancialSource.Name),
		Cluster: RQaaSClusterDataSourceModel{
			Name:       types.StringValue(order.Attrs.Cluster.ClusterName),
			Domain:     types.StringValue(order.Attrs.Cluster.Domain),
			NetSegment: types.StringValue(order.Attrs.Cluster.NetSegment),
			Platform:   types.StringValue(order.Attrs.Cluster.Platform),
			Zone:       types.StringValue(order.Attrs.Cluster.AvailabilityZone),
			Hosts:      tfhosts,
		},
		QueueUsers: tfQueueUsers,
	}

	var lifetime types.Int64
	req.State.GetAttribute(ctx, path.Root("lifetime"), &lifetime)
	if !lifetime.IsNull() {
		state.Lifetime = lifetime
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r RQaaSResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state RQaaSResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetRQaaSOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Get RQaaS order ended with error.\nError: %v", err.Error()),
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
	usersChanged := !plan.QueueUsers.Equal(state.QueueUsers)

	if labelChanged {
		utils.ChangeOrderLabel(order, plan.Label.ValueString(), resp)
	}

	if finProjChanged {
		err := order.ChangeFinancialProject(finProj.ID)
		if err != nil {
			resp.Diagnostics.AddError(
				consts.CREATE_RES_FAIL,
				fmt.Sprintf("Change financial project for order ended with error.\nError: %v", err.Error()),
			)
			return
		}
	}

	if usersChanged {
		diags := r.applyQueueUsers(ctx, order, plan.QueueUsers, state.QueueUsers)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

}

func (r RQaaSResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state RQaaSResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetRQaaSOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf("Get RQaaS order ended with error.\nError: %v", err.Error()),
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

	err = order.DeleteRQaaS()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf(
				"Delete RQaaS ended with error.\nError: %v", err.Error(),
			),
		)
		return
	}
}

// custom logic
func (r RQaaSResource) compareQueueUsers(
	ctx context.Context,
	planQueueUsers, stateQueueUsers types.Set,
) (toAdd, toUpdate, toDelete []QueueUserModel, diags diag.Diagnostics) {

	planMap := make(map[string]QueueUserModel)
	stateMap := make(map[string]QueueUserModel)

	if !planQueueUsers.IsNull() && !planQueueUsers.IsUnknown() {
		var planUsersList []QueueUserModel
		diags = planQueueUsers.ElementsAs(ctx, &planUsersList, false)
		if diags.HasError() {
			return
		}

		for _, user := range planUsersList {
			planMap[user.UserName.ValueString()] = user
		}
	}

	if !stateQueueUsers.IsNull() && !stateQueueUsers.IsUnknown() {
		var stateUsersList []QueueUserModel
		diags = stateQueueUsers.ElementsAs(ctx, &stateUsersList, false)
		if diags.HasError() {
			return
		}

		for _, user := range stateUsersList {
			stateMap[user.UserName.ValueString()] = user
		}
	}

	for username, planUser := range planMap {
		if stateUser, exists := stateMap[username]; !exists {
			toAdd = append(toAdd, planUser)
		} else {
			if !planUser.Read.Equal(stateUser.Read) ||
				!planUser.Write.Equal(stateUser.Write) {
				toUpdate = append(toUpdate, planUser)
			}
		}
	}

	for username, stateUser := range stateMap {
		if _, exists := planMap[username]; !exists {
			toDelete = append(toDelete, stateUser)
		}
	}

	return toAdd, toUpdate, toDelete, diags
}

func (r RQaaSResource) applyQueueUsers(
	ctx context.Context,
	order *orders.RQaaS,
	planQueueUsers, stateQueueUsers types.Set,
) (diags diag.Diagnostics) {
	toAdd, toUpdate, toDelete, diags := r.compareQueueUsers(ctx, planQueueUsers, stateQueueUsers)

	for _, user := range toAdd {
		attrs := prepareQueueUserAttrs(user)
		err := order.CreateUser(*attrs)
		if err != nil {
			diags.AddError(consts.UPDATE_RES_FAIL, err.Error())
			return diags
		}
	}

	for _, user := range toUpdate {
		attrs := orders.QueueUserAttrs{
			UserName: user.UserName.ValueString(),
			Read:     user.Read.ValueBool(),
			Write:    user.Write.ValueBool(),
		}
		err := order.UpdateUserPermissions(attrs)
		if err != nil {
			diags.AddError(consts.UPDATE_RES_FAIL, err.Error())
			return diags
		}
	}

	for _, user := range toDelete {
		attrs := orders.QueueUserAttrs{
			UserName: user.UserName.ValueString(),
			Read:     false,
			Write:    false,
		}
		err := order.UpdateUserPermissions(attrs)
		if err != nil {
			diags.AddError(consts.UPDATE_RES_FAIL, err.Error())
			return diags
		}
	}

	return diags

}

func prepareQueueUserAttrs(user QueueUserModel) *orders.QueueUserAttrs {
	APD, risCode, name := ParseRQaasUsername(user.UserName.ValueString())
	checkAccess := fmt.Sprintf("%v %v", user.Read.ValueBool(), user.Write.ValueBool())
	attrs := &orders.QueueUserAttrs{
		UserName:    user.UserName.ValueString(),
		Read:        user.Read.ValueBool(),
		Write:       user.Write.ValueBool(),
		RisCode:     risCode,
		APDCode:     APD,
		Name:        name,
		CheckAccess: &checkAccess,
	}
	return attrs
}

func ParseRQaasUsername(userName string) (apdCode, risCode, name *string) {
	parts := strings.Split(userName, "-")
	strPtr := func(s string) *string { return &s }

	risCode = strPtr(parts[len(parts)-5])
	name = strPtr(parts[len(parts)-1])

	if len(parts) >= 6 && parts[0] != "" {
		apdCode = strPtr(parts[0] + "-")
	}

	return apdCode, risCode, name
}

func validateRQaasUsername(userName, envPrefix string) error {
	parts := strings.Split(userName, "-")

	if len(parts) < 5 {
		return fmt.Errorf(
			"invalid format: expected '[APD<code>-]<RIS_code>-rbmq-%s-client-<name>', got '%s' (example: 'APD12.34-567-rbmq-%s-client-myapp1')",
			envPrefix, userName, envPrefix,
		)
	}

	name := parts[len(parts)-1]
	if !regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9\-.]{0,61}[a-zA-Z0-9]$`).MatchString(name) {
		return fmt.Errorf(
			"invalid client name: '%s' (2-63 chars, alphanumeric with dots/dashes inside, example: 'app1' or 'app-1.2')",
			name,
		)
	}

	if parts[len(parts)-3] != envPrefix {
		return fmt.Errorf(
			"invalid environment prefix: expected '%s', found '%s' in username '%s'",
			envPrefix, parts[len(parts)-3], userName,
		)
	}

	risPart := parts[len(parts)-5]
	if !regexp.MustCompile(`^\d{1,64}$`).MatchString(risPart) {
		return fmt.Errorf(
			"invalid RIS code: '%s' (1-64 digits, example: '567')",
			risPart,
		)
	}

	if len(parts) >= 6 && parts[0] != "" {
		apdPart := parts[0]
		if !regexp.MustCompile(`^APD\d[\d.]{3,60}$`).MatchString(apdPart) {
			return fmt.Errorf(
				"invalid APD code: '%s' (5-64 chars: 'APD' followed by digits/dots, example: 'APD12.34')",
				apdPart,
			)
		}
	}

	return nil
}
