package gslbv1

import (
	"context"
	"fmt"
	"reflect"
	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/common"
	"terraform-provider-vtb/internal/consts"
	"terraform-provider-vtb/internal/services/core"
	"terraform-provider-vtb/internal/services/flavor"
	"terraform-provider-vtb/internal/utils"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"
	"terraform-provider-vtb/pkg/client/sources"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var GSLB_DESIRED_VERSIONS = map[string]int64{
	"latest":  10000000,
	"1.1.11":  26,
	"1.10.10": 24,
	"1.10.9":  23,
	"1.10.8":  22,
	"1.10.6":  20,
	"1.10.3":  18,
	"1.9.1":   15,
	"1.9.0":   14,
	"1.8.4":   13,
	"1.8.3":   12,
	"1.6.1":   9,
}

var AvailaiableGSLBV1InstallationVersions = utils.GetMapKeys(GSLB_DESIRED_VERSIONS)

var (
	_ resource.Resource                = &GSLBV1Resource{}
	_ resource.ResourceWithImportState = &GSLBV1Resource{}
	_ resource.ResourceWithModifyPlan  = &GSLBV1Resource{}
)

type GSLBV1Resource struct {
	client *client.CloudClient
}

func NewGSLBV1Resource() resource.Resource {
	return &GSLBV1Resource{}
}

func (r *GSLBV1Resource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_gslb_v1_cluster"
}

func (r *GSLBV1Resource) Configure(
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
			fmt.Sprintf("Expected *client.CloudClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

func (r *GSLBV1Resource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan, state GSLBV1ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	checkOrderIsDeleted := utils.CheckOrderIsDeleted(
		r.client,
		plan.OrderID,
		"cluster",
		"gslb_cluster_v1",
	)
	resp.Diagnostics.Append(checkOrderIsDeleted.Diagnostics...)

	utils.HandleExternalDeletionRecreate(
		ctx,
		resp,
		&plan,
		checkOrderIsDeleted.IsDeleted,
		checkOrderIsDeleted.Diagnostics,
	)

	// проверка хостов GSLB в плане
	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		r.validateGSLBHosts(ctx, plan, state, resp)
	}

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *GSLBV1Resource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r *GSLBV1Resource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan GSLBV1ResourceModel
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
			fmt.Sprintf("Get financial project ended with error.\nError: %s", err.Error()),
		)
		return
	}

	attrs := r.prepareOrderAttrs(plan)

	order := orders.NewGSLBV1(
		r.client.Creds,
		r.client.ProjectName,
		plan.Image.ProductID.ValueString(),
		attrs,
	)

	err = order.Create(orders.CreateOrderPayload{
		Label:        plan.Label.ValueString(),
		Lifetime:     int(plan.Lifetime.ValueInt64()),
		FinProjectID: finProj.ID,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Create order on portal ended with error.\nError: %s", err.Error()),
		)
		return
	}

	gslbItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Get order item from portal ended with error.\nError: %s", err.Error()),
		)
		return
	}

	gslbItemConfg, ok := gslbItem.Data.Config.(entities.GSLBV1ItemData)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Invalid type for `GSLB Item Config`",
		)
		return
	}

	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(gslbItem.ID)

	plan.BgPaas = types.BoolValue(gslbItemConfg.Config.AnycastEnabled)

	tfGslbHosts, diag := getGslbHostsFromPortal(order)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.GSLBHosts = tfGslbHosts

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GSLBV1Resource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var orderID, nginxPassword, apiPassword types.String
	req.State.GetAttribute(ctx, path.Root("nginx_password"), &nginxPassword)
	req.State.GetAttribute(ctx, path.Root("api_password"), &apiPassword)
	req.State.GetAttribute(ctx, path.Root("order_id"), &orderID)
	if orderID.IsNull() {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Resource hasn't `order_id`. Without `order_id` can't get order from portal",
		)
		return
	}

	order, err := orders.GetGSLBV1Order(
		r.client.Creds,
		r.client.ProjectName,
		orderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("order_id"),
			consts.READ_RES_FAIL,
			fmt.Sprintf("Get order data from portal ended with error.\nError: %s", err.Error()),
		)
		return
	}

	gslbItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Get order item from portal ended with error.\nError: %s",
				err.Error(),
			),
		)
		return
	}

	vmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Get VM Items from portal ended with error.\nError: %s",
				err.Error(),
			),
		)
		return
	}

	vmItem := vmItems[0]

	vmConfig, ok := vmItem.Data.Config.(entities.VMItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Invalid type for vmConfig",
		)
		return
	}

	appExtraMount, err := order.GetExtraMount("/app")
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get vm extra mount `app/` from portal with `order_id`: %s.\nError: %s",
				orderID, err.Error(),
			),
		)
		return
	}

	gslbItemConfg, ok := gslbItem.Data.Config.(entities.GSLBV1ItemData)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Invalid type for `GSLB Item Config`",
		)
		return
	}

	tfGslbHosts, diag := getGslbHostsFromPortal(order)
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := GSLBV1ResourceModel{
		OrderID:          types.StringValue(order.ID),
		ItemID:           types.StringValue(gslbItem.ID),
		FinancialProject: types.StringValue(order.FinancialSource.Name),
		Layout:           types.StringValue(order.Attrs.Layout),
		Label:            types.StringValue(order.Label),
		Core: core.CoreModel{
			Platform:       types.StringValue(order.Attrs.Platform),
			Domain:         types.StringValue(order.Attrs.Domain),
			NetSegmentCode: types.StringValue(vmConfig.DefaultNic.NetSegment),
			Zone:           types.StringValue(order.Attrs.AvailabilityZone),
		},
		Image: GSLBV1ImageDataSourceModel{
			Distribution:    types.StringValue(vmConfig.Image.Os.Distribution),
			OsVersion:       types.StringValue(vmConfig.Image.Os.Version),
			OnSupport:       types.BoolValue(order.Attrs.OnSupport),
			ProductID:       types.StringValue(order.ProductID),
			GeoDistribution: types.BoolValue(order.Attrs.GeoDistribution),
			ProductVersion:  types.StringValue(gslbItem.Data.Provider),
		},
		Flavor: flavor.FlavorModel{
			Cores:  types.Int64Value(vmConfig.Flavor.Cores),
			Memory: types.Int64Value(vmConfig.Flavor.Memory),
			Name:   types.StringValue(vmConfig.Flavor.Name),
			UUID:   types.StringValue(vmConfig.Flavor.UUID),
		},
		Access: utils.ReadAccessMapV2(vmItem.Data.ACLs),
		ExtraMounts: map[string]common.ExtraMountModel{
			"/app": {
				Size:       types.Int64Value(appExtraMount.Size),
				FileSystem: types.StringValue(appExtraMount.FileSystem),
			},
		},

		ApiPassword:   apiPassword,
		NginxPassword: nginxPassword,
		DNSZone:       types.StringValue(order.Attrs.DNSZone),
		BgPaas:        types.BoolValue(gslbItemConfg.Config.AnycastEnabled),
		GSLBHosts:     tfGslbHosts,
	}

	if desiredVersion, found := utils.GetKeyByValue(GSLB_DESIRED_VERSIONS, order.Attrs.DesiredVersion); found {
		state.DesiredVersion = types.StringValue(desiredVersion)
	} else {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("error for getting `desired_version`, availiable_versions: %v", AvailaiableGSLBV1InstallationVersions),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *GSLBV1Resource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state GSLBV1ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetGSLBV1Order(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Get GSLBv1 order ended with error.\nError: %s", err.Error()),
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
			fmt.Sprintf("Get financial project ended with error.\nError: %s", err.Error()),
		)
		return
	}

	labelChanged := !plan.Label.Equal(state.Label)
	finProjChanged := !plan.FinancialProject.Equal(state.FinancialProject)
	flavorChanged := plan.Flavor != state.Flavor
	nginxPasswordChanged := !plan.NginxPassword.Equal(state.NginxPassword)
	apiPasswordChanged := !plan.ApiPassword.Equal(state.ApiPassword)
	extraMountsChanged := !reflect.DeepEqual(plan.ExtraMounts, state.ExtraMounts)
	bgPaaSChanged := !plan.BgPaas.Equal(state.BgPaas)
	gslbHostsChanged := !plan.GSLBHosts.Equal(state.GSLBHosts)

	if labelChanged {
		utils.ChangeOrderLabel(order, plan.Label.ValueString(), resp)
	}

	if finProjChanged {
		r.changeFinancialProject(order, finProj.ID, resp)
	}

	if nginxPasswordChanged {
		r.changeNginxPassword(order, plan.NginxPassword, resp)
	}

	if apiPasswordChanged {
		r.changeApiPassword(order, plan.ApiPassword, resp)
	}

	if bgPaaSChanged {
		r.switchBgPaas(plan.BgPaas, order, resp)
	}

	if gslbHostsChanged {
		r.applyGslbHosts(ctx, plan, state, order, resp)
	}

	if extraMountsChanged {
		r.changeExtraMountsGSLBV1(order, plan, resp)
	}

	if flavorChanged {
		r.changeFlavor(order, plan.Flavor, resp)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GSLBV1Resource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state GSLBV1ResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetGSLBV1Order(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf("Get GSLBv1 order ended with error.\nError: %s", err.Error()),
		)
		return
	}

	orderState, err := order.GetState()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf("Get State order ended with error.\nError: %s", err.Error()),
		)
		return
	}

	if orderState == "deleted" || orderState == "deprovisioned" {
		resp.Diagnostics.AddWarning(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf(
				"Order with id '%s' was deleted by another way. Not from OpenTofu. "+
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

	err = order.DeleteCluster()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf(
				"Delete GSLBv1 ended with error.\nError: %s", err.Error(),
			),
		)
		return
	}
}
