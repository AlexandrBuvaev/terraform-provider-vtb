package rabbitmquser

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/consts"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &RabbitMQUserResource{}
	_ resource.ResourceWithImportState = &RabbitMQUserResource{}
	_ resource.ResourceWithModifyPlan  = &RabbitMQUserResource{}
)

type RabbitMQUserResource struct {
	client *client.CloudClient
}

func NewRabbitMQUserResource() resource.Resource {
	return &RabbitMQUserResource{}
}

func (r *RabbitMQUserResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.CloudClient)

	if !ok {
		resp.Diagnostics.AddError(
			"unexpected resource configure type",
			fmt.Sprintf("Expected *client.CloudClient, got: %T. "+
				"Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *RabbitMQUserResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_rabbitmq_user"
}

func (r *RabbitMQUserResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID format",
			"Excepted import identifer in the form: `rabbitmq_order_id/username`",
		)
		return
	}

	orderID := parts[0]
	username := parts[1]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("rabbitmq_order_id"), orderID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("username"), username)...)
}

func (r *RabbitMQUserResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	if req.State.Raw.IsNull() {
		return
	}

	var plan RabbitMQUserModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	r.checkVhostExist(ctx, &plan, resp)

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r *RabbitMQUserResource) Create(
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
		r.client.Creds,
		r.client.ProjectName,
		plan.RabbitMQOrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("rabbitmq_order_id"),
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Get order from portal ended with error.\nError: %s", err.Error()),
		)
		return
	}

	clusterItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("rabbitmq_order_id"),
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Get order item from portal ended with error.\nError: %s", err.Error()),
		)
		return
	}

	clusterConfig, ok := clusterItem.Data.Config.(entities.RabbitMQItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			"Invalid type for cluster config",
		)
		return
	}

	fullClusterNameParts := strings.Split(clusterConfig.FullClusterName, "-")
	env_prefix := fullClusterNameParts[2]

	usrPayload, err := r.preparePayload(&plan, env_prefix)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Prepare user payload ended with error.\nError: %s", err.Error()),
		)
		return
	}

	err = order.CreateUsers(usrPayload)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("username"),
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Create user ended with error.\nError: %s", err.Error()),
		)
		return
	}

	if !plan.VhostsAccess.IsNull() {
		diags := r.updateVhostAccess(ctx, plan, order)
		resp.Diagnostics.Append(diags...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RabbitMQUserResource) Read(
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
		r.client.Creds,
		r.client.ProjectName,
		state.RabbitMQOrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("rabbitmq_order_id"),
			consts.READ_RES_FAIL,
			fmt.Sprintf("Get order from portal ended with error.\nError: %s", err.Error()),
		)
		return
	}

	clusterItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("rabbitmq_order_id"),
			consts.READ_RES_FAIL,
			fmt.Sprintf("Get order item from portal ended with error.\nError: %s", err.Error()),
		)
		return
	}

	clusterConfig, ok := clusterItem.Data.Config.(entities.RabbitMQItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.MODIFY_PLAN_FAIL,
			"Invalid cluster config type",
		)
		return
	}

	var vhostRead, vhostWrite, vhostConfigure []string

	for _, vhost := range clusterConfig.VhostAccess {
		if vhost.UserName == state.Username.ValueString() {
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
	}

	tfVhostRead, _ := types.SetValueFrom(ctx, types.StringType, vhostRead)
	tfVhostWrite, _ := types.SetValueFrom(ctx, types.StringType, vhostWrite)
	tfVhostConfigure, _ := types.SetValueFrom(ctx, types.StringType, vhostConfigure)

	isEmpty := (tfVhostRead.IsNull() || tfVhostRead.IsUnknown()) &&
		(tfVhostConfigure.IsNull() || tfVhostConfigure.IsUnknown()) &&
		(tfVhostWrite.IsNull() || tfVhostWrite.IsUnknown())

	var vhostsAccessObj types.Object
	var diags diag.Diagnostics

	if isEmpty {
		vhostsAccessObj = types.ObjectNull(VhostPermissionsModel{}.AttributeTypes())
	} else {
		permissionModels := VhostPermissionsModel{
			VhostRead:      tfVhostRead,
			VhostWrite:     tfVhostWrite,
			VhostConfigure: tfVhostConfigure,
		}

		vhostsAccessObj, diags = types.ObjectValueFrom(ctx, permissionModels.AttributeTypes(), permissionModels)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	state.VhostsAccess = vhostsAccessObj

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RabbitMQUserResource) Update(
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
		r.client.Creds,
		r.client.ProjectName,
		plan.RabbitMQOrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("rabbitmq_order_id"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Get order from portal ended with error.\nError: %s", err.Error()),
		)
		return
	}
	if plan.VhostsAccess.IsNull() {
		r.deleteVhostAccess(ctx, order, plan, state, resp)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !plan.VhostsAccess.Equal(state.VhostsAccess) && !plan.VhostsAccess.IsNull() && !plan.VhostsAccess.IsUnknown() {
		r.deleteVhostAccess(ctx, order, plan, state, resp)
		diags := r.updateVhostAccess(ctx, plan, order)
		resp.Diagnostics.Append(diags...)

	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RabbitMQUserResource) Delete(
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
		r.client.Creds,
		r.client.ProjectName,
		state.RabbitMQOrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("rabbitmq_order_id"),
			fmt.Sprintln(consts.DELETE_RES_FAIL, "fetch order from portal API ended with error"),
			err.Error(),
		)
		return
	}

	err = order.DeleteUser(state.Username.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("rabbitmq_order_id"),
			fmt.Sprintln(consts.DELETE_RES_FAIL, "user deletion ended with error"),
			err.Error(),
		)
		return
	}
}
