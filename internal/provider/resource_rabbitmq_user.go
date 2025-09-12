package provider

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource = RabbitMQUserResource{}
)

type RabbitMQUserResource struct {
	provider *VTBCloudProvider
}

func NewRabbitMQUserResource(p *VTBCloudProvider) RabbitMQUserResource {
	return RabbitMQUserResource{
		provider: p,
	}
}

type RabbitMQUserModel struct {
	RabbitMQOrderID types.String `tfsdk:"rabbitmq_order_id"`
	Username        types.String `tfsdk:"username"`
	VhostsAccess    types.List   `tfsdk:"vhosts_access"`
}

type VhostAccessModel struct {
	Name        types.String `tfsdk:"name"`
	Permissions types.List   `tfsdk:"permissions"`
}

func (v VhostAccessModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":        types.StringType,
		"permissions": types.ListType{ElemType: types.StringType},
	}
}

// Модель для редактирования прав доступа на вирутальных хостах
type VhostPermissionsModel struct {
	UserName       types.String `tfsdk:"username"`
	VhostRead      types.Set    `tfsdk:"vhost_read"`
	VhostWrite     types.Set    `tfsdk:"vhost_write"`
	VhostConfigure types.Set    `tfsdk:"vhost_configure"`
}

func (v VhostPermissionsModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"username":        types.StringType,
		"vhost_read":      types.SetType{ElemType: types.StringType},
		"vhost_write":     types.SetType{ElemType: types.StringType},
		"vhost_configure": types.SetType{ElemType: types.StringType},
	}
}

func (r RabbitMQUserResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = "vtb_rabbitmq_user"
}

func (t RabbitMQUserResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description:         "Управление правами пользователей для виртуальных хостов в кластере RabbitMQ.",
		MarkdownDescription: "Управление правами пользователей для виртуальных хостов в кластере RabbitMQ.",
		Attributes: map[string]schema.Attribute{

			"rabbitmq_order_id": schema.StringAttribute{
				Required:            true,
				Description:         "Идентификатор заказа кластера RabbitMQ.",
				MarkdownDescription: "Идентификатор заказа кластера RabbitMQ.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"username": schema.StringAttribute{
				Required:            true,
				Description:         "Имя пользователя",
				MarkdownDescription: "Имя пользователя",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vhosts_access": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"username": schema.StringAttribute{
							Required:            true,
							Description:         "Имя пользователя, которому будут предоставлены доступы к вирутальным хостам.",
							MarkdownDescription: "Имя пользователя, которому будут предоставлены доступы к вирутальным хостам.",
						},
						"vhost_read": schema.SetAttribute{
							Optional:            true,
							ElementType:         types.StringType,
							Description:         "Список вирутальных хостов (vhosts) с правами на чтение для пользователя.",
							MarkdownDescription: "Список вирутальных хостов (vhosts) с правами на чтение для пользователя.",
						},
						"vhost_write": schema.SetAttribute{
							Optional:            true,
							ElementType:         types.StringType,
							Description:         "Список виртуальных хостов (vhosts) с правами на запись для пользователя",
							MarkdownDescription: "Спико виртуальных хостов (vhosts) с правами на запись для пользователя",
						},
						"vhost_configure": schema.SetAttribute{
							Optional:            true,
							ElementType:         types.StringType,
							Description:         "Список виртуальных хостов (vhosts) с правами на конфигурирование для пользователя",
							MarkdownDescription: "Список виртуальных хостов (vhosts) с правами на конфигурирование для пользователя",
						},
					},
				},
			},
		},
	}
}

func (r RabbitMQUserResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("rabbitmq_order_id"), req, resp)
}

func (r RabbitMQUserResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan RabbitMQUserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	r.checkVhostExist(ctx, &plan, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r RabbitMQUserResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan RabbitMQUserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetRabbitMQOrder(
		r.provider.Creds,
		r.provider.ProjectName,
		plan.RabbitMQOrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("rabbitmq_order_id"),
			fmt.Sprintln(CREATE_RES_FAIL, "fetch order from portal API ended with error"),
			err.Error(),
		)
		return
	}
	cluster, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("rabbitmq_order_id"),
			fmt.Sprintln(READ_RES_FAIL, "can't get cluster item"),
			err.Error(),
		)
		return
	}

	fullClusterNameParts := strings.Split(cluster.Data.Config.(entities.RabbitMQItemConfig).FullClusterName, "-")
	env_prefix := fullClusterNameParts[2]

	usrPayload, err := r.preparePayload(&plan, env_prefix)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("username"), CREATE_RES_FAIL, err.Error())
		return
	}

	err = order.CreateUsers(usrPayload)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			fmt.Sprintln(CREATE_RES_FAIL, "user creation ended with error"),
			err.Error(),
		)
		return
	}

	if !plan.VhostsAccess.IsNull() {
		payload := r.prepareUpdateVhostsAccessMultiplyPayload(ctx, &plan)
		err = order.UpdateVhostAccessMultiply(payload)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("vhosts_access"),
				fmt.Sprintln(CREATE_RES_FAIL, "Update vhosts access permissions ended with error"),
				err.Error(),
			)
			return
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r RabbitMQUserResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state RabbitMQUserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetRabbitMQOrder(
		r.provider.Creds,
		r.provider.ProjectName,
		state.RabbitMQOrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("rabbitmq_order_id"),
			fmt.Sprintln(READ_RES_FAIL, "fetch order from portal API ended with error:"),
			err.Error(),
		)
		return
	}

	cluster, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("rabbitmq_order_id"),
			fmt.Sprintln(READ_RES_FAIL, "can't get cluster item"),
			err.Error(),
		)
		return
	}

	vhostAccesses := cluster.Data.Config.(entities.RabbitMQItemConfig).VhostAccess

	var actualAccessList []VhostPermissionsModel
	var vhostRead, vhostWrite, vhostConfigure []string
	for _, vhost := range vhostAccesses {
		for _, permission := range vhost.Permissions {
			switch permission {
			case "READ":
				vhostRead = append(vhostRead, vhost.VhostName)
			case "WRITE":
				vhostWrite = append(vhostWrite, vhost.VhostName)
			case "CONFIGURE":
				vhostConfigure = append(vhostConfigure, vhost.VhostName)
			}
		}
	}
	tfVhostRead, _ := types.SetValueFrom(ctx, types.StringType, vhostRead)
	tfVhostWrite, _ := types.SetValueFrom(ctx, types.StringType, vhostWrite)
	tfVhostConfigure, _ := types.SetValueFrom(ctx, types.StringType, vhostConfigure)
	actualAccessList = append(actualAccessList, VhostPermissionsModel{
		UserName:       state.Username,
		VhostRead:      tfVhostRead,
		VhostWrite:     tfVhostWrite,
		VhostConfigure: tfVhostConfigure,
	})
	state.VhostsAccess, _ = types.ListValueFrom(ctx, types.ObjectType{AttrTypes: VhostPermissionsModel{}.AttributeTypes()}, actualAccessList)

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r RabbitMQUserResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state RabbitMQUserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetRabbitMQOrder(
		r.provider.Creds,
		r.provider.ProjectName,
		plan.RabbitMQOrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("rabbitmq_order_id"),
			fmt.Sprintln(UPDATE_RES_FAIL, "fetch order from portal API ended with error"),
			err.Error(),
		)
		return
	}
	if plan.VhostsAccess.IsNull() {
		var planVhostAccesses []VhostPermissionsModel
		plan.VhostsAccess.ElementsAs(ctx, &planVhostAccesses, false)

		var stateVhostAccesses []VhostPermissionsModel
		state.VhostsAccess.ElementsAs(ctx, &stateVhostAccesses, false)

		deleteVhostAccess(order, &plan, stateVhostAccesses, planVhostAccesses, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !plan.VhostsAccess.Equal(state.VhostsAccess) && !plan.VhostsAccess.IsNull() {
		var planVhostAccesses []VhostPermissionsModel
		plan.VhostsAccess.ElementsAs(ctx, &planVhostAccesses, false)

		var stateVhostAccesses []VhostPermissionsModel
		state.VhostsAccess.ElementsAs(ctx, &stateVhostAccesses, false)

		deleteVhostAccess(order, &plan, stateVhostAccesses, planVhostAccesses, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
		payload := r.prepareUpdateVhostsAccessMultiplyPayload(ctx, &plan)
		err = order.UpdateVhostAccessMultiply(payload)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("vhosts_access"),
				fmt.Sprintln(CREATE_RES_FAIL, "Update vhosts access permissions ended with error"),
				err.Error(),
			)
			return
		}

	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r RabbitMQUserResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state RabbitMQUserModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetRabbitMQOrder(
		r.provider.Creds,
		r.provider.ProjectName,
		state.RabbitMQOrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("rabbitmq_order_id"),
			fmt.Sprintln(DELETE_RES_FAIL, "fetch order from portal API ended with error"),
			err.Error(),
		)
		return
	}

	err = order.DeleteUser(state.Username.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("rabbitmq_order_id"),
			fmt.Sprintln(DELETE_RES_FAIL, "user deletion ended with error"),
			err.Error(),
		)
		return
	}
}

func (r RabbitMQUserResource) checkVhostExist(
	ctx context.Context,
	plan *RabbitMQUserModel,
	diags *diag.Diagnostics,
) {
	if plan.RabbitMQOrderID.IsUnknown() {
		diags.AddWarning(
			"Can't validate vhosts:",
			"order_id is unknown, so vhosts wont be validated",
		)
		return
	}

	order, err := orders.GetRabbitMQOrder(
		r.provider.Creds,
		r.provider.ProjectName,
		plan.RabbitMQOrderID.ValueString(),
	)
	if err != nil {
		diags.AddAttributeError(
			path.Root("rabbitmq_order_id"),
			fmt.Sprintln(MODIFY_PLAN_FAIL, "fetch order from portal API ended with error"),
			err.Error(),
		)
		return
	}

	cluster, err := order.GetParentItem()
	if err != nil {
		diags.AddError(
			fmt.Sprintln(MODIFY_PLAN_FAIL, "can't get cluster item"),
			err.Error(),
		)
	}

	clusterVhosts := cluster.Data.Config.(entities.RabbitMQItemConfig).Vhosts

	var existedVhosts []string
	for _, vhost := range clusterVhosts {
		existedVhosts = append(existedVhosts, vhost.Name)
	}
	var planVhostsPermissions []VhostPermissionsModel
	plan.VhostsAccess.ElementsAs(ctx, &planVhostsPermissions, false)
	uniq_plan_vhosts := getUniqueVhosts(planVhostsPermissions)
	for _, vhost := range uniq_plan_vhosts {
		if !slices.Contains(existedVhosts, vhost) {
			diags.AddWarning(
				MODIFY_PLAN_FAIL,
				fmt.Sprintf(
					"RabbitMQ cluster item has no vhost=`%s`. Available: %s",
					vhost, strings.Join(existedVhosts, ", "),
				),
			)
		}
	}

}

func (r RabbitMQUserResource) prepareUpdateVhostsAccessMultiplyPayload(
	ctx context.Context,
	plan *RabbitMQUserModel,
) []entities.RabbitMQUserInputPermissions {
	var inputPermissions []entities.RabbitMQUserInputPermissions

	var planVhostAccesses []VhostPermissionsModel
	plan.VhostsAccess.ElementsAs(ctx, &planVhostAccesses, false)

	for _, vhostPermission := range planVhostAccesses {
		var vhostRead, vhostWrite, vhostConfigure []string

		vhostPermission.VhostRead.ElementsAs(ctx, &vhostRead, false)
		vhostPermission.VhostWrite.ElementsAs(ctx, &vhostWrite, false)
		vhostPermission.VhostConfigure.ElementsAs(ctx, &vhostConfigure, false)

		inputPermissions = append(inputPermissions, entities.RabbitMQUserInputPermissions{
			UserName:       vhostPermission.UserName.ValueString(),
			VhostRead:      vhostRead,
			VHostWrite:     vhostWrite,
			VhostConfigure: vhostConfigure,
		})
	}

	return inputPermissions
}

func (r RabbitMQUserResource) preparePayload(plan *RabbitMQUserModel, envPrefix string) (*orders.RabbitqMQCreateUserAttrs, error) {

	parts := strings.Split(plan.Username.ValueString(), "-")
	if len(parts) < 5 {
		return nil, fmt.Errorf(
			"invalid username format: expected <RIS код>-rbmq-%s-client-<уникальное имя клиента>, got %s",
			envPrefix, plan.Username.ValueString(),
		)
	}
	// Проверка, что вторая подстрока равна "rbmq"
	if parts[1] != "rbmq" {
		return nil, fmt.Errorf(
			"invalid username format: expected <RIS код>-rbmq-%s-client-<уникальное имя клиента>, got %s",
			envPrefix, plan.Username.ValueString(),
		)
	}
	if parts[2] != envPrefix {
		return nil, fmt.Errorf(
			"invalid username format: expected <RIS код>-rbmq-%s-client-<уникальное имя клиента>, got %s",
			envPrefix, plan.Username.ValueString(),
		)
	}

	risCode := parts[0]
	uniqueName := parts[4]

	p := &orders.RabbitqMQCreateUserAttrs{
		RabbitMQUsers: []orders.RabbitMQUsers{
			{
				RisCode: risCode,
				Name:    uniqueName,
			},
		},
		EnvPrefix: envPrefix,
		Precheck:  true,
	}
	return p, nil
}

func deleteVhostAccess(
	order *orders.RabbitMQ,
	plan *RabbitMQUserModel,
	stateVhostAccess []VhostPermissionsModel,
	planVhostAccess []VhostPermissionsModel,
	diags *diag.Diagnostics,
) {
	stateUniqVhosts := getUniqueVhosts(stateVhostAccess)
	planUniqVhosts := getUniqueVhosts(planVhostAccess)
	planUniqVhostsMap := make(map[string]struct{})
	for _, elem := range planUniqVhosts {
		planUniqVhostsMap[elem] = struct{}{}
	}
	var toDelete []entities.RabbitMQVhostAccess
	for _, vhost := range stateUniqVhosts {
		if _, exsits := planUniqVhostsMap[vhost]; !exsits {
			// если vhosts отустутвуют в plan, но при это присутсвуют в state
			// то добаляем их в список на удаление
			toDelete = append(toDelete, entities.RabbitMQVhostAccess{
				UserName:  plan.Username.ValueString(),
				VhostName: vhost,
			})
		}
	}

	if len(toDelete) > 0 {
		for _, access := range toDelete {
			err := order.DeleteVhostAccess(access)
			if err != nil {
				diags.AddError(
					"Update resource failed: delete vhost access ended with error",
					err.Error(),
				)
			}
		}
	}
}

// utils
func getUniqueElements(vhosts []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(vhosts))

	for _, vhost := range vhosts {
		if !seen[vhost] {
			seen[vhost] = true
			result = append(result, vhost)
		}
	}
	return result
}

// Функция получает уникальный список vhosts, которые переданы в форме terrafrom
// Необходима для получения и сравнения списка vhosts хранимых в state и полученных в plan
func getUniqueVhosts(
	vhostsAccessList []VhostPermissionsModel,
) []string {
	var vhostsList []string
	for _, planPermission := range vhostsAccessList {
		for _, elem := range planPermission.VhostRead.Elements() {
			if str, ok := elem.(types.String); ok {
				vhostsList = append(vhostsList, str.ValueString())
			}
		}
		for _, elem := range planPermission.VhostWrite.Elements() {
			if str, ok := elem.(types.String); ok {
				vhostsList = append(vhostsList, str.ValueString())
			}
		}
		for _, elem := range planPermission.VhostConfigure.Elements() {
			if str, ok := elem.(types.String); ok {
				vhostsList = append(vhostsList, str.ValueString())
			}
		}
	}
	uniq_vhosts := getUniqueElements(vhostsList)
	return uniq_vhosts
}
