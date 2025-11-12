package rabbitmquser

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"terraform-provider-vtb/internal/consts"
	"terraform-provider-vtb/internal/utils"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func (r *RabbitMQUserResource) checkVhostExist(
	ctx context.Context,
	plan *RabbitMQUserModel,
	resp *resource.ModifyPlanResponse,
) {
	if plan.RabbitMQOrderID.IsUnknown() {
		resp.Diagnostics.AddWarning(
			consts.MODIFY_PLAN_FAIL,
			fmt.Sprintln("Can't validate vhosts, `order_id` is unknown, so vhosts won't be validated"),
		)
		return
	}

	order, err := orders.GetRabbitMQOrder(
		r.client.Creds,
		r.client.ProjectName,
		plan.RabbitMQOrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("rabbitmq_order_id"),
			consts.MODIFY_PLAN_FAIL,
			fmt.Sprintf("fetch order from portal API ended with error.\nError: %s", err.Error()),
		)
		return
	}

	clusterItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.MODIFY_PLAN_FAIL,
			fmt.Sprintf("Get order item from portal ended with error.\nError: %s", err.Error()),
		)
		return
	}

	clusterConfig, ok := clusterItem.Data.Config.(entities.RabbitMQItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.MODIFY_PLAN_FAIL,
			fmt.Sprintln("Invalid type for Item cluster Config"),
		)
		return
	}

	if plan.VhostsAccess.IsNull() || plan.VhostsAccess.IsUnknown() {
		return
	}

	var planVhostsPermissions *VhostPermissionsModel
	diag := plan.VhostsAccess.As(ctx, &planVhostsPermissions, basetypes.ObjectAsOptions{})
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() || planVhostsPermissions == nil {
		return
	}

	var existedVhosts []string
	for _, vhost := range clusterConfig.Vhosts {
		existedVhosts = append(existedVhosts, vhost.Name)
	}

	uniqPlanVhosts := getUniqueVhosts(*planVhostsPermissions)

	for _, vhost := range uniqPlanVhosts {
		if !slices.Contains(existedVhosts, vhost) {
			resp.Diagnostics.AddWarning(
				consts.MODIFY_PLAN_FAIL,
				fmt.Sprintf(
					"RabbitMQ cluster item has no vhost=`%s`. Available: %s",
					vhost, strings.Join(existedVhosts, ", "),
				),
			)
		}
	}
}

func (r *RabbitMQUserResource) deleteVhostAccess(
	ctx context.Context,
	order *orders.RabbitMQ,
	plan, state RabbitMQUserModel,
	resp *resource.UpdateResponse,
) {
	var planVhostAccess, stateVhostAccess *VhostPermissionsModel

	if !plan.VhostsAccess.IsNull() && !plan.VhostsAccess.IsUnknown() {
		diags := plan.VhostsAccess.As(ctx, &planVhostAccess, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !state.VhostsAccess.IsNull() && !state.VhostsAccess.IsUnknown() {
		diags := state.VhostsAccess.As(ctx, &stateVhostAccess, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if planVhostAccess == nil && stateVhostAccess == nil {
		return
	}

	var stateUniqVhosts, planUniqVhosts []string
	if stateVhostAccess != nil {
		stateUniqVhosts = getUniqueVhosts(*stateVhostAccess)
	}
	if planVhostAccess != nil {
		planUniqVhosts = getUniqueVhosts(*planVhostAccess)
	}

	planUniqVhostsMap := make(map[string]struct{})
	for _, elem := range planUniqVhosts {
		planUniqVhostsMap[elem] = struct{}{}
	}

	var toDelete []entities.RabbitMQVhostAccess
	for _, vhost := range stateUniqVhosts {
		if _, exists := planUniqVhostsMap[vhost]; !exists {
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
				resp.Diagnostics.AddError(
					consts.UPDATE_RES_FAIL,
					fmt.Sprintf("Delete vhost access ended with error.\nError: %s", err.Error()),
				)
				return
			}
		}
	}
}

func (r *RabbitMQUserResource) updateVhostAccess(
	ctx context.Context,
	plan RabbitMQUserModel,
	order *orders.RabbitMQ,
) diag.Diagnostics {
	var diags diag.Diagnostics
	payload, diag := r.prepareUpdateVhostsAccessMultiplyPayload(ctx, plan)
	diags.Append(diag...)
	if diags.HasError() {
		return diags
	}

	err := order.UpdateVhostAccessMultiply(payload)
	if err != nil {
		diags.AddAttributeError(
			path.Root("vhosts_access"),
			"Update vhost access:",
			fmt.Sprintf("Create vhost access ended with error.\nError: %s", err.Error()),
		)
		return diags
	}
	return diags
}

func (r RabbitMQUserResource) prepareUpdateVhostsAccessMultiplyPayload(
	ctx context.Context,
	plan RabbitMQUserModel,
) ([]entities.RabbitMQUserInputPermissions, diag.Diagnostics) {
	var inputPermissions []entities.RabbitMQUserInputPermissions
	var diags diag.Diagnostics

	var planVhostsPermissions *VhostPermissionsModel
	diag := plan.VhostsAccess.As(ctx, &planVhostsPermissions, basetypes.ObjectAsOptions{})
	diags.Append(diag...)
	if diags.HasError() {
		return inputPermissions, diags
	}

	var vhostRead, vhostWrite, vhostConfigure []string

	planVhostsPermissions.VhostRead.ElementsAs(ctx, &vhostRead, false)
	planVhostsPermissions.VhostWrite.ElementsAs(ctx, &vhostWrite, false)
	planVhostsPermissions.VhostConfigure.ElementsAs(ctx, &vhostConfigure, false)

	inputPermissions = append(inputPermissions, entities.RabbitMQUserInputPermissions{
		UserName:       plan.Username.ValueString(),
		VhostRead:      vhostRead,
		VHostWrite:     vhostWrite,
		VhostConfigure: vhostConfigure,
	})

	return inputPermissions, diags
}

func (r RabbitMQUserResource) preparePayload(
	plan *RabbitMQUserModel,
	envPrefix string,
) (*orders.RabbitqMQCreateUserAttrs, error) {

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

// utils

func getUniqueVhosts(
	vhostsAccess VhostPermissionsModel,
) []string {
	var vhostsList []string
	for _, elem := range vhostsAccess.VhostRead.Elements() {
		if str, ok := elem.(types.String); ok {
			vhostsList = append(vhostsList, str.ValueString())
		}
	}
	for _, elem := range vhostsAccess.VhostWrite.Elements() {
		if str, ok := elem.(types.String); ok {
			vhostsList = append(vhostsList, str.ValueString())
		}
	}
	for _, elem := range vhostsAccess.VhostConfigure.Elements() {
		if str, ok := elem.(types.String); ok {
			vhostsList = append(vhostsList, str.ValueString())
		}
	}
	uniq_vhosts := utils.GetUniqueElements(vhostsList)
	return uniq_vhosts
}
