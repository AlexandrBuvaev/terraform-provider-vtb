package vtbartemis

import (
	"context"
	"fmt"
	"slices"
	"sort"

	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/consts"
	"terraform-provider-vtb/internal/utils"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &ArtemisRolesResource{}
	_ resource.ResourceWithImportState = &ArtemisRolesResource{}
)

type ArtemisRolesResource struct {
	client *client.CloudClient
}

func NewArtemisRolesResource() resource.Resource {
	return &ArtemisRolesResource{}
}

func (r ArtemisRolesResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_artemis_roles"
}

func (r *ArtemisRolesResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type ArtemisRolesResourceModel struct {
	OrderID  types.String       `tfsdk:"vtb_artemis_order_id"`
	RoleList []ArtemisRoleModel `tfsdk:"role_list"`
}

type ArtemisRoleModel struct {
	SecurityPolicyName types.String `tfsdk:"security_policy_name"`
	Role               types.String `tfsdk:"role"`
	UserNames          types.Set    `tfsdk:"user_names"`
}

func (r ArtemisRolesResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"vtb_artemis_order_id": schema.StringAttribute{
				Required:            true,
				Description:         "Идентификатор заказа кластера VTB Artemis.",
				MarkdownDescription: "Идентификатор заказа кластера VTB Artemis.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"role_list": schema.SetNestedAttribute{
				Required:            true,
				Description:         "Список ролей.",
				MarkdownDescription: "Список ролей.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"security_policy_name": schema.StringAttribute{
							Required:            true,
							Description:         "Наименование политики безопасности.",
							MarkdownDescription: "Наименование политики безопасности.",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},

						"role": schema.StringAttribute{
							Required:            true,
							Description:         "Наименование роли.",
							MarkdownDescription: "Наименование роли.",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},

						"user_names": schema.SetAttribute{
							Required:            true,
							ElementType:         types.StringType,
							Description:         "Имена пользователей (ТУЗ)",
							MarkdownDescription: "Имена пользователей (ТУЗ)",
							PlanModifiers: []planmodifier.Set{
								setplanmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
		},
	}
}

func (r ArtemisRolesResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("vtb_artemis_order_id"), req, resp)
}

func (r ArtemisRolesResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan ArtemisRolesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	artemis, err := orders.GetArtemisOrder(
		r.client.Creds,
		r.client.ProjectName,
		plan.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(consts.CREATE_RES_FAIL, err.Error())
		return
	}

	for _, role := range plan.RoleList {

		var userNames []string
		role.UserNames.ElementsAs(ctx, &userNames, false)

		if len(userNames) > 0 {
			roleToAdd := orders.ChangeRoleAttrs{
				Role:                role.Role.ValueString(),
				SecurityPolicyNames: role.SecurityPolicyName.ValueString(),
				UserNames:           userNames,
			}
			err = artemis.ChangeUserInRole(roleToAdd)
			if err != nil {
				resp.Diagnostics.AddWarning(
					consts.CREATE_RES_FAIL,
					fmt.Sprintf("change user in role failed: %s", err.Error()),
				)
			}
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r ArtemisRolesResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var orderID types.String
	req.State.GetAttribute(ctx, path.Root("vtb_artemis_order_id"), &orderID)

	artemis, err := orders.GetArtemisOrder(
		r.client.Creds,
		r.client.ProjectName,
		orderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(consts.READ_RES_FAIL, err.Error())
		return
	}

	parentItem, err := artemis.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(consts.READ_RES_FAIL, err.Error())
		return
	}
	ssRolesList := parentItem.Data.Config.(entities.VTBArtemisItemConfig).RoleList

	state := ArtemisRolesResourceModel{
		OrderID:  orderID,
		RoleList: []ArtemisRoleModel{},
	}

	for _, role := range ssRolesList {
		userNames, _ := types.SetValueFrom(ctx, types.StringType, role.UserNames)
		if len(role.UserNames) > 0 {
			roleToAdd := ArtemisRoleModel{
				Role:               types.StringValue(role.Role),
				SecurityPolicyName: types.StringValue(role.SecurityPolicyName),
				UserNames:          userNames,
			}
			state.RoleList = append(state.RoleList, roleToAdd)
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r ArtemisRolesResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state ArtemisRolesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	artemis, err := orders.GetArtemisOrder(
		r.client.Creds,
		r.client.ProjectName,
		plan.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(consts.CREATE_RES_FAIL, err.Error())
		return
	}

	var rolesToUpdate []ArtemisRoleModel
	for _, stateRole := range state.RoleList {
		markUpdated := false
		for _, planRole := range plan.RoleList {
			if stateRole.Role.Equal(planRole.Role) {
				markUpdated = true
				var sortedPlanUserNames, sortedStateUserNames []string
				planRole.UserNames.ElementsAs(ctx, &sortedPlanUserNames, false)
				stateRole.UserNames.ElementsAs(ctx, &sortedStateUserNames, false)
				sort.Strings(sortedPlanUserNames)
				sort.Strings(sortedStateUserNames)
				if slices.Equal(sortedPlanUserNames, sortedStateUserNames) {
					continue
				}
				rolesToUpdate = append(rolesToUpdate, planRole)
			}
		}
		if !markUpdated {
			stateRole.UserNames = types.SetNull(types.StringType)
			rolesToUpdate = append(rolesToUpdate, stateRole)
		}
	}
	for _, planRole := range plan.RoleList {
		markExisted := false
		for _, stateRole := range state.RoleList {
			if stateRole.Role.Equal(planRole.Role) {
				markExisted = true
				break
			}
		}
		if !markExisted {
			rolesToUpdate = append(rolesToUpdate, planRole)
		}
	}

	for _, roleToUpdate := range rolesToUpdate {
		var inputUsers []string
		roleToUpdate.UserNames.ElementsAs(ctx, &inputUsers, false)
		inputRole := orders.ChangeRoleAttrs{
			Role:                roleToUpdate.Role.ValueString(),
			SecurityPolicyNames: roleToUpdate.SecurityPolicyName.ValueString(),
			UserNames:           inputUsers,
		}
		err = artemis.ChangeUserInRole(inputRole)
		if err != nil {
			resp.Diagnostics.AddWarning(
				consts.CREATE_RES_FAIL,
				fmt.Sprintf("update user in role failed: %s", err.Error()),
			)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r ArtemisRolesResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state ArtemisRolesResourceModel
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
		resp.Diagnostics.AddError(consts.CREATE_RES_FAIL, err.Error())
		return
	}

	emptyUsers := make([]string, 0)

	for _, stateRole := range state.RoleList {
		var stateUserNames []string
		stateRole.UserNames.ElementsAs(ctx, &stateUserNames, false)
		if len(stateUserNames) == 0 {
			continue
		}
		err = artemis.ChangeUserInRole(orders.ChangeRoleAttrs{
			Role:                stateRole.Role.ValueString(),
			SecurityPolicyNames: stateRole.SecurityPolicyName.ValueString(),
			UserNames:           emptyUsers,
		})
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("user_names"),
				consts.DELETE_RES_FAIL,
				fmt.Sprintf("can't delete users from role: %s", err.Error()),
			)
			return
		}
	}
}

func checkExcludes(checkRole ArtemisRoleModel) error {
	excludedRoles := []string{"sla", "amq", "DLQ_producer", "ExpiryQueue_producer"}
	for _, excRole := range excludedRoles {
		if excRole == checkRole.Role.ValueString() {
			return fmt.Errorf("you are using an exception role %s, it cannot be used", excRole)
		}
	}

	switch checkRole.Role.ValueString() {
	case "cluster_manager":
		if checkRole.SecurityPolicyName.ValueString() != "all" {
			return fmt.Errorf("you are using an specific role %s, security_policy_name should be 'all'", checkRole.Role.ValueString())
		}
		return nil
	case "temp_producer":
		if checkRole.SecurityPolicyName.ValueString() != "temporary" {
			return fmt.Errorf("you are using an specific role %s, security_policy_name should be 'temporary'", checkRole.Role.ValueString())
		}
		return nil
	case "temp_consumer":
		if checkRole.SecurityPolicyName.ValueString() != "temporary" {
			return fmt.Errorf("you are using an specific role %s, security_policy_name should be 'temporary'", checkRole.Role.ValueString())
		}
		return nil
	}

	switch checkRole.SecurityPolicyName.ValueString() {
	case "all":
		if checkRole.Role.ValueString() != "cluster_manager" {
			return fmt.Errorf("you are using an specific security_policy_name %s, role should be 'cluster_manager'", checkRole.SecurityPolicyName.ValueString())
		}
		return nil
	case "temporary":
		if checkRole.Role.ValueString() != "temp_producer" || checkRole.Role.ValueString() != "temp_consumer" {
			return fmt.Errorf("you are using an specific security_policy_name %s, role should be 'temp_consumer' or 'temp_producer'", checkRole.SecurityPolicyName.ValueString())
		}
		return nil
	}

	return nil
}

func (r ArtemisRolesResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan ArtemisRolesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	r.checkRoles(ctx, &plan, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r ArtemisRolesResource) checkRoles(
	ctx context.Context,
	plan *ArtemisRolesResourceModel,
	diags *diag.Diagnostics,
) {
	if plan.OrderID.IsUnknown() {
		return
	}

	artemis, err := orders.GetArtemisOrder(
		r.client.Creds,
		r.client.ProjectName,
		plan.OrderID.ValueString(),
	)
	if err != nil {
		diags.AddError(consts.MODIFY_PLAN_FAIL, err.Error())
		return
	}

	parentItem, err := artemis.GetParentItem()
	if err != nil {
		diags.AddError(consts.MODIFY_PLAN_FAIL, err.Error())
		return
	}
	ssRolesList := parentItem.Data.Config.(entities.VTBArtemisItemConfig).RoleList
	ssTuzList := parentItem.Data.Config.(entities.VTBArtemisItemConfig).TuzList

	var planRoleNames, planUserNames, ssRoleNames, ssUserNames []string
	for _, ssRoleItem := range ssRolesList {
		ssRoleNames = append(ssRoleNames, ssRoleItem.Role)
	}
	for _, ssTuzItem := range ssTuzList {
		ssUserNames = append(ssUserNames, ssTuzItem.UserName)
	}

	for _, planRoleItem := range plan.RoleList {
		if !slices.Contains(ssRoleNames, planRoleItem.Role.ValueString()) {
			excludeRoles := []string{"cluster_manager"}
			if !slices.Contains(excludeRoles, planRoleItem.Role.ValueString()) {
				diags.AddError(
					consts.MODIFY_PLAN_FAIL,
					fmt.Sprintf("undefined role: %s", planRoleItem.Role.ValueString()),
				)
			}
		}
		var chkUserNames []string
		planRoleItem.UserNames.ElementsAs(ctx, &chkUserNames, false)

		for _, chkUserName := range chkUserNames {
			if !slices.Contains(ssUserNames, chkUserName) {
				diags.AddError(
					consts.MODIFY_PLAN_FAIL,
					fmt.Sprintf("undefined user name: %s", chkUserName),
				)
			}
		}

		if utils.ContainsDuplicate(chkUserNames) {
			diags.AddError(
				consts.MODIFY_PLAN_FAIL,
				fmt.Sprintf("role must be a unique user names, problem in role: %s", planRoleItem.Role.ValueString()),
			)
		}

		planRoleItem.UserNames.ElementsAs(ctx, &planUserNames, false)
		if len(planUserNames) == 0 {
			diags.AddError(
				consts.MODIFY_PLAN_FAIL,
				fmt.Sprintf("role must be user_names, role: %s", planRoleItem.Role.ValueString()),
			)
		}
		planRoleNames = append(planRoleNames, planRoleItem.Role.ValueString())
		err := checkExcludes(planRoleItem)
		if err != nil {
			diags.AddError(consts.MODIFY_PLAN_FAIL, err.Error())
		}

	}

	if utils.ContainsDuplicate(planRoleNames) {
		diags.AddError(
			consts.MODIFY_PLAN_FAIL,
			fmt.Sprintf("role must be a unique, roles: %s", planRoleNames),
		)
		return
	}
}
