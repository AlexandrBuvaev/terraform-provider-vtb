package gslbv1

import (
	"context"
	"fmt"
	"terraform-provider-vtb/internal/consts"
	"terraform-provider-vtb/internal/services/flavor"
	"terraform-provider-vtb/internal/utils"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// custom logic
func (r GSLBV1Resource) prepareOrderAttrs(
	plan GSLBV1ResourceModel,
) (attrs orders.GSLBV1Attrs) {
	return orders.GSLBV1Attrs{
		ADLogonGrants:    utils.PrepareADLogonGrants(plan.Access),
		OnSupport:        plan.Image.OnSupport.ValueBool(),
		OsVersion:        plan.Image.OsVersion.ValueString(),
		AvailabilityZone: plan.Core.Zone.ValueString(),
		Domain:           plan.Core.Domain.ValueString(),
		Platform:         plan.Core.Platform.ValueString(),
		DefaultNic: entities.DefaultNic{
			NetSegment: plan.Core.NetSegmentCode.ValueString(),
		},
		Flavor: entities.Flavor{
			Cores:  plan.Flavor.Cores.ValueInt64(),
			Memory: plan.Flavor.Memory.ValueInt64(),
			Name:   plan.Flavor.Name.ValueString(),
			UUID:   plan.Flavor.UUID.ValueString(),
		},
		Layout:          plan.Layout.ValueString(),
		GeoDistribution: plan.Image.GeoDistribution.ValueBool(),
		ExtraMounts:     utils.PrepareExtraMountsAttrs(plan.ExtraMounts),
		ExtraNics:       []interface{}{},

		DesiredVersion: GSLB_DESIRED_VERSIONS[plan.DesiredVersion.ValueString()],
		DNSZone:        plan.DNSZone.ValueString(),
		OrderCredentials: orders.OrderCredentials{
			NginxLogin:    "nginx_system",
			NginxPassword: plan.NginxPassword.ValueString(),
			ApiPassword:   plan.ApiPassword.ValueString(),
		},
	}
}

// actions
func (r GSLBV1Resource) changeFinancialProject(
	order *orders.GSLBV1,
	finProjectID string,
	resp *resource.UpdateResponse,
) {
	err := order.ChangeFinancialProject(finProjectID)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("financial_project"),
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Change financial project for order ended with error.\nError: %s", err.Error()),
		)
		return
	}
}

func (r GSLBV1Resource) changeFlavor(
	order *orders.GSLBV1,
	flavor flavor.FlavorModel,
	resp *resource.UpdateResponse,
) {
	err := order.VerticalScaling(
		entities.Flavor{
			Cores:  flavor.Cores.ValueInt64(),
			Memory: flavor.Memory.ValueInt64(),
			UUID:   flavor.UUID.ValueString(),
			Name:   flavor.Name.ValueString(),
		},
	)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Vertical scaling ended with error.\nError: %s", err.Error()),
		)
		return
	}
}

func (r GSLBV1Resource) changeNginxPassword(
	order *orders.GSLBV1,
	nginxPassword types.String,
	resp *resource.UpdateResponse,
) {
	err := order.ChangePasswords("nginx", nginxPassword.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("nginx_password"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Change `nginx_password` ended with error.\nError: %s", err.Error()),
		)
		return
	}
}

func (r GSLBV1Resource) changeApiPassword(
	order *orders.GSLBV1,
	apiPassword types.String,
	resp *resource.UpdateResponse,
) {
	err := order.ChangePasswords("api", apiPassword.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_password"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Change `api_password` ended with error.\nError: %s", err.Error()),
		)
		return
	}
}

func (r *GSLBV1Resource) changeExtraMountsGSLBV1(
	order *orders.GSLBV1,
	plan GSLBV1ResourceModel,
	resp *resource.UpdateResponse,
) {
	for path, planExtraMount := range plan.ExtraMounts {
		err := order.ExpandMountPoint(entities.ExtraMount{
			Path:       path,
			Size:       planExtraMount.Size.ValueInt64(),
			FileSystem: planExtraMount.FileSystem.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError(
				consts.UPDATE_RES_FAIL,
				fmt.Sprintf("Expand mount point ended with error.\nError: %s", err.Error()),
			)
		}
		return
	}
}

func (r *GSLBV1Resource) switchBgPaas(
	planBgPaas types.Bool,
	order *orders.GSLBV1,
	resp *resource.UpdateResponse,
) {
	err := order.SwitchBgPaaS(planBgPaas.ValueBool())
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("bgpaas"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Switch BgPaaS ended with error.\nError: %s", err.Error()),
		)
		return
	}
}

func (r *GSLBV1Resource) applyGslbHosts(
	ctx context.Context,
	plan, state GSLBV1ResourceModel,
	order *orders.GSLBV1,
	resp *resource.UpdateResponse,
) {
	planHosts, diags := r.gslbHostsSetToMap(ctx, plan.GSLBHosts)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	stateHosts, diags := r.gslbHostsSetToMap(ctx, state.GSLBHosts)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var toEnable, toDisable []string

	for hostname, planHost := range planHosts {
		stateHost, exists := stateHosts[hostname]
		if !exists {
			continue
		}

		planMode := planHost.MaintenanceMode.ValueString()
		stateMode := stateHost.MaintenanceMode.ValueString()
		if planMode != stateMode {
			switch planMode {
			case "on":
				toEnable = append(toEnable, hostname)
			case "off":
				toDisable = append(toDisable, hostname)
			}
		}
	}

	if len(toEnable) > 0 {
		resp.Diagnostics.Append(r.enableGSLBHosts(order, toEnable, plan.Core.Domain.ValueString())...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if len(toDisable) > 0 {
		resp.Diagnostics.Append(r.disableGSLBHosts(order, toDisable, plan.Core.Domain.ValueString())...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}

func (r *GSLBV1Resource) enableGSLBHosts(
	order *orders.GSLBV1,
	hostnames []string,
	domain string) diag.Diagnostics {
	var diags diag.Diagnostics
	err := order.EnableMMHosts(hostnames, domain)
	if err != nil {
		diags.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Error enabling hosts %v: %s", hostnames, err.Error()),
		)
	}
	return diags
}

func (r *GSLBV1Resource) disableGSLBHosts(
	order *orders.GSLBV1,
	hostnames []string,
	domain string) diag.Diagnostics {
	var diags diag.Diagnostics
	err := order.DisableMMHosts(hostnames, domain)
	if err != nil {
		diags.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Error disabling hosts %v: %s", hostnames, err.Error()),
		)
	}
	return diags
}

// helpers
func getGslbHostsFromPortal(
	order *orders.GSLBV1,
) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	var tfGslbHosts types.Set
	gslbAppInfoItems, err := order.GetGSLBV1AppConfigs()
	if err != nil {
		diags.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("get gslb app items ended with error.\nError: %s", err.Error()),
		)
		return tfGslbHosts, diags
	}

	tfGslbHosts, diag := convertGslbHostsToModel(gslbAppInfoItems)
	diags.Append(diag...)
	if diags.HasError() {
		return tfGslbHosts, diags
	}

	return tfGslbHosts, diags
}

func convertGslbHostsToModel(hosts []entities.GSLBAppItemConfig) (types.Set, diag.Diagnostics) {
	objType := types.ObjectType{AttrTypes: GSLBHosts{}.AttributeTypes()}

	if hosts == nil {
		return types.SetNull(objType), nil
	}

	var diags diag.Diagnostics
	elements := make([]attr.Value, len(hosts))

	for i, host := range hosts {
		hostModel := GSLBHosts{
			Name:            types.StringValue(host.Name),
			MaintenanceMode: types.StringValue(host.MaintenanceMode),
		}
		objVal, objDiags := hostModel.ToObjectValue()
		diags.Append(objDiags...)
		if diags.HasError() {
			return types.SetNull(objType), diags
		}
		elements[i] = objVal
	}

	setVal, setDiags := types.SetValue(objType, elements)
	diags.Append(setDiags...)
	return setVal, diags
}

func (r *GSLBV1Resource) validateGSLBHosts(
	ctx context.Context,
	plan, state GSLBV1ResourceModel,
	resp *resource.ModifyPlanResponse,
) {
	// Если в состоянии нет gslb_hosts — пропускаем валидацию (например, при создании)
	if state.GSLBHosts.IsNull() || state.GSLBHosts.IsUnknown() {
		return
	}

	// Преобразуем Set в map для удобного сравнения по имени
	stateHosts, diags := r.gslbHostsSetToMap(ctx, state.GSLBHosts)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	planHosts, diags := r.gslbHostsSetToMap(ctx, plan.GSLBHosts)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if len(stateHosts) != len(planHosts) {
		resp.Diagnostics.AddAttributeError(
			path.Root("gslb_hosts"),
			consts.MODIFY_PLAN_FAIL,
			fmt.Sprintf("The number of GSLB hosts cannot be changed. "+
				"State has %d hosts, but plan has %d. "+
				"Only 'maintenance_mode' values may be updated.",
				len(stateHosts), len(planHosts),
			),
		)
		return
	}

	for name := range stateHosts {
		if _, exists := planHosts[name]; !exists {
			resp.Diagnostics.AddAttributeError(
				path.Root("gslb_hosts"),
				consts.MODIFY_PLAN_FAIL,
				fmt.Sprintf("Host '%s' is missing in the planned configuration. "+
					"Host names and count must remain identical to state. "+
					"Only 'maintenance_mode' may be modified.", name),
			)
			return
		}
	}

	for name := range planHosts {
		if _, exists := stateHosts[name]; !exists {
			resp.Diagnostics.AddAttributeError(
				path.Root("gslb_hosts"),
				consts.MODIFY_PLAN_FAIL,
				fmt.Sprintf("Host '%s' was added in the planned configuration. "+
					"Adding or removing hosts is not allowed. "+
					"Only 'maintenance_mode' may be updated.", name),
			)
			return
		}
	}

}

func (r *GSLBV1Resource) gslbHostsSetToMap(
	ctx context.Context,
	set types.Set,
) (map[string]GSLBHosts, diag.Diagnostics) {
	var diags diag.Diagnostics
	result := make(map[string]GSLBHosts)

	if set.IsNull() || set.IsUnknown() {
		return result, diags
	}

	var hosts []GSLBHosts
	d := set.ElementsAs(ctx, &hosts, false) // false = не разрешать null-элементы
	diags.Append(d...)
	if diags.HasError() {
		return nil, diags
	}

	for _, host := range hosts {
		if host.Name.IsNull() || host.Name.IsUnknown() {
			diags.AddError(consts.MODIFY_PLAN_FAIL, "Host name cannot be null or unknown")
			return nil, diags
		}
		name := host.Name.ValueString()
		result[name] = host
	}

	return result, diags
}
