package provider

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"terraform-provider-vtb/pkg/client/orders"
	"terraform-provider-vtb/pkg/client/references"
	"terraform-provider-vtb/pkg/client/sources"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// AvailableFieldForSpecificEnvironment модификатор определяет список environments, для которых возможно указать данное поле (для которого указан модификатор)
type AvailableFieldForSpecificEnvironment struct {
	Provider     *VTBCloudProvider
	Environments []string
}

func AccessForSpecificEnvironment(provider *VTBCloudProvider, environements ...string) AvailableFieldForSpecificEnvironment {
	return AvailableFieldForSpecificEnvironment{
		Provider:     provider,
		Environments: environements,
	}
}

func (m AvailableFieldForSpecificEnvironment) Description(ctx context.Context) string {
	return "Attribute must be specified only for specific environments"
}
func (m AvailableFieldForSpecificEnvironment) MarkdownDescription(ctx context.Context) string {
	return "Attribute must be specified only for specific environments"
}
func (m AvailableFieldForSpecificEnvironment) PlanModifyList(
	ctx context.Context,
	req planmodifier.ListRequest,
	resp *planmodifier.ListResponse,
) {
	if !slices.Contains(m.Environments, m.Provider.Environment) && len(req.PlanValue.Elements()) != 0 {
		resp.Diagnostics.AddError(
			fmt.Sprintln(MODIFY_PLAN_FAIL, req.Path.String()+" misconfigured"),
			"Attribute "+req.Path.String()+" available only for specific environments."+
				fmt.Sprintf("\nCurrent environment: %v\nAvailable environments: %v", m.Provider.Environment, m.Environments),
		)
		return
	}
}

// AvailableAccessForEnvironmentModifier Модификатор определяет - можно ли при создании данного продукта
// указывать группы доступа AD  (поле "access"). Для некоторых AD access доступен лишь в DEV среде.
type AvailableAccessForEnvironmentModifier struct {
	Provider *VTBCloudProvider
}

func ActiveDirectoryAccessForDevOnly(provider *VTBCloudProvider) AvailableAccessForEnvironmentModifier {
	return AvailableAccessForEnvironmentModifier{
		Provider: provider,
	}
}
func (m AvailableAccessForEnvironmentModifier) Description(ctx context.Context) string {
	return "Check AD access groups must be specified only for DEV"
}
func (m AvailableAccessForEnvironmentModifier) MarkdownDescription(ctx context.Context) string {
	return "Check AD access groups must be specified only for DEV"
}
func (m AvailableAccessForEnvironmentModifier) PlanModifyMap(
	ctx context.Context,
	req planmodifier.MapRequest,
	resp *planmodifier.MapResponse,
) {
	if m.Provider.Environment != "DEV" && len(req.PlanValue.Elements()) != 0 {
		resp.Diagnostics.AddError(
			fmt.Sprintln(MODIFY_PLAN_FAIL, "access groups misconfigured"),
			"`access = {}` (ad_logon_grants) available only for DEV environment."+
				fmt.Sprintf("\nCurrent environment: %v", m.Provider.Environment),
		)
		return
	}
}

// AvailableAccessForExactEnvironment Модификатор определяет возможно ли указать AD группу доступа при заказе продукта. Указывается список environment, в которых можно указать группу доступа
type AvailableAccessForSpecificEnvironment struct {
	Provider     *VTBCloudProvider
	Environments []string
}

func ActiveDirectoryAccesForSpecificEnvironment(provider *VTBCloudProvider, environements ...string) AvailableAccessForSpecificEnvironment {
	return AvailableAccessForSpecificEnvironment{
		Provider:     provider,
		Environments: environements,
	}
}

func (m AvailableAccessForSpecificEnvironment) Description(ctx context.Context) string {
	return "Check AD access groups must be specified only for specific environments"
}
func (m AvailableAccessForSpecificEnvironment) MarkdownDescription(ctx context.Context) string {
	return "Check AD access groups must be specified for specific environments"
}
func (m AvailableAccessForSpecificEnvironment) PlanModifyMap(
	ctx context.Context,
	req planmodifier.MapRequest,
	resp *planmodifier.MapResponse,
) {
	if !slices.Contains(m.Environments, m.Provider.Environment) && len(req.PlanValue.Elements()) != 0 {
		resp.Diagnostics.AddError(
			fmt.Sprintln(MODIFY_PLAN_FAIL, "access groups misconfigured"),
			"`access = {}` (ad_logon_grants) available only for specific environments."+
				fmt.Sprintf("\nCurrent environment: %v\nAvailable environments: %v", m.Provider.Environment, m.Environments),
		)
		return
	} else if len(req.PlanValue.Elements()) == 0 && slices.Contains(m.Environments, m.Provider.Environment) {
		resp.Diagnostics.AddError(
			fmt.Sprintln(MODIFY_PLAN_FAIL, "access groups misconfigured"),
			fmt.Sprintf("access groups (ad_logon_grants) must be specified for %v environments.", m.Environments),
		)
		return
	}
}

func AvailiableStringFieldForSpecificEnvironment(provider *VTBCloudProvider, environements ...string) AvailableStringFieldForSpecificEnvironment {
	return AvailableStringFieldForSpecificEnvironment{
		Provider:     provider,
		Environments: environements,
	}
}

type AvailableStringFieldForSpecificEnvironment struct {
	Provider     *VTBCloudProvider
	Environments []string
}

func (m AvailableStringFieldForSpecificEnvironment) Description(ctx context.Context) string {
	return "Check StingType fields in schema must be specified only for specific environments"
}

func (m AvailableStringFieldForSpecificEnvironment) MarkdownDescription(ctx context.Context) string {
	return "Check StingType fields in schema must be specified only for specific environments"
}

func (m AvailableStringFieldForSpecificEnvironment) PlanModifyString(
	ctx context.Context,
	req planmodifier.StringRequest,
	resp *planmodifier.StringResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if !slices.Contains(m.Environments, m.Provider.Environment) {
		resp.Diagnostics.AddError(
			MODIFY_PLAN_FAIL,
			fmt.Sprintf("Field is availiable for these environments: %v, for current environment: %v this field not availiable",
				m.Environments, m.Provider.Environment),
		)
	}
}

// AvailableAccessRoleModifier Модификатор определяет доступные роли для групп доступа AD.
type AvailableAccessRoleModifier struct {
	Provider *VTBCloudProvider
	Filter   string
}

func AvailableAccessRole(provider *VTBCloudProvider, filter string) AvailableAccessRoleModifier {
	return AvailableAccessRoleModifier{
		Provider: provider,
		Filter:   filter,
	}
}
func (m AvailableAccessRoleModifier) Description(ctx context.Context) string {
	return "Check available role for this product in specified project"
}
func (m AvailableAccessRoleModifier) MarkdownDescription(ctx context.Context) string {
	return "Check available role for this product in specified project"
}
func (m AvailableAccessRoleModifier) PlanModifyMap(
	ctx context.Context,
	req planmodifier.MapRequest,
	resp *planmodifier.MapResponse,
) {
	if req.PlanValue.IsUnknown() || req.PlanValue.IsNull() {
		return
	}

	roles, err := references.GetRoles(m.Provider.Creds, m.Provider.Environment, m.Filter)
	if err != nil {
		resp.Diagnostics.AddError("Get roles from portal", err.Error())
	}

	var rolesNames []string
	for _, r := range roles {
		rolesNames = append(rolesNames, r.Data.GroupName)
	}

	for key := range req.PlanValue.Elements() {
		exists := false
		for _, rname := range rolesNames {
			if key == rname {
				exists = true
				break
			}
		}

		if !exists {
			resp.Diagnostics.AddError(
				"Access Roles not available for resource",
				fmt.Sprintf(
					"Available roles [%s]. Specified '%s'",
					strings.Join(rolesNames, ", "), key,
				),
			)
		}
	}
}

type AvailableDomainsModifier struct {
	Provider *VTBCloudProvider
}

func AvailableDomains(provider *VTBCloudProvider) AvailableDomainsModifier {
	return AvailableDomainsModifier{
		Provider: provider,
	}
}
func (m AvailableDomainsModifier) Description(ctx context.Context) string {
	return "Check available domains for current project"
}
func (m AvailableDomainsModifier) MarkdownDescription(ctx context.Context) string {
	return "Check available domains for current project"
}
func (m AvailableDomainsModifier) PlanModifyString(
	ctx context.Context,
	req planmodifier.StringRequest,
	resp *planmodifier.StringResponse,
) {
	if req.PlanValue.IsUnknown() || req.PlanValue.IsNull() {
		return
	}

	domains, err := references.GetKnownDomains(m.Provider.Creds)
	if err != nil {
		resp.Diagnostics.AddError(
			"Getting known domains",
			fmt.Sprintf(
				"reference has [%s] in page or failed with error '%s'",
				domains, err.Error(),
			),
		)
	}

	for _, domainName := range domains {
		if strings.EqualFold(domainName, req.PlanValue.ValueString()) {
			return
		}
	}
	resp.Diagnostics.AddError(
		"Domain not available for resource",
		fmt.Sprintf(
			"Available domains [%s]. Specified '%s'",
			strings.Join(domains, ", "), req.PlanValue.ValueString(),
		),
	)
}

type AvailableNetSegmentsModifier struct {
	Provider *VTBCloudProvider
}

func AvailableNetSegments(provider *VTBCloudProvider) AvailableNetSegmentsModifier {
	return AvailableNetSegmentsModifier{
		Provider: provider,
	}
}
func (m AvailableNetSegmentsModifier) Description(ctx context.Context) string {
	return "Check available network segments for current project"
}
func (m AvailableNetSegmentsModifier) MarkdownDescription(ctx context.Context) string {
	return "Check available network segment for current project"
}
func (m AvailableNetSegmentsModifier) PlanModifyString(
	ctx context.Context,
	req planmodifier.StringRequest,
	resp *planmodifier.StringResponse,
) {
	if req.PlanValue.IsUnknown() || req.PlanValue.IsNull() {
		return
	}

	var segmentNames []string
	segments, err := sources.GetNetSegments(m.Provider.Creds, m.Provider.ProjectName)
	if err != nil {
		resp.Diagnostics.AddError("Get segments from portal", err.Error())
	}

	for _, s := range segments {
		segmentNames = append(segmentNames, s.Code)
	}

	for _, segmentName := range segmentNames {
		if strings.EqualFold(segmentName, req.PlanValue.ValueString()) {
			return
		}
	}
	resp.Diagnostics.AddError(
		"NetSegments not available for resource",
		fmt.Sprintf(
			"Available net_segments [%s]. Specified '%s'",
			strings.Join(segmentNames, ", "), req.PlanValue.ValueString(),
		),
	)
}

// AvailableFlavorModifier Модификатор, выполняет запрос к справочнику
// для проверки доступности указанной конфигурации RAM/Cores для конкретного продукта с учетом фильтра и среды
type AvailableFlavorModifier struct {
	Provider *VTBCloudProvider
	Filter   string // Фильтр для справочника в формате "vm:linux", "cluster:kafka", "app:nginx"
}

func AvailableFlavor(provider *VTBCloudProvider, filter string) AvailableFlavorModifier {
	return AvailableFlavorModifier{
		Provider: provider,
		Filter:   filter,
	}
}
func (m AvailableFlavorModifier) Description(ctx context.Context) string {
	return "Check available flavor for this product in specified project"
}
func (m AvailableFlavorModifier) MarkdownDescription(ctx context.Context) string {
	return "Check available flavor for this product in specified project"
}
func (m AvailableFlavorModifier) PlanModifyObject(
	ctx context.Context,
	req planmodifier.ObjectRequest,
	resp *planmodifier.ObjectResponse,
) {
	if req.PlanValue.IsUnknown() || req.PlanValue.IsNull() {
		return
	}

	var planFlavor FlavorModel
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, req.Path, &planFlavor)...)
	wishedCores := planFlavor.Cores.ValueInt64()
	wishedMemory := planFlavor.Memory.ValueInt64()

	fullFilter := fmt.Sprintf("flavor:%s:%s", m.Filter, strings.ToLower(m.Provider.Environment))

	_, err := references.GetFlavorByFilter(
		m.Provider.Creds,
		fullFilter,
		wishedMemory,
		wishedCores,
	)
	if err != nil {
		resp.Diagnostics.AddError("Flavor not available for resource", err.Error())
	}
}

type CheckIsDeletedModifier struct {
	Provider      *VTBCloudProvider
	InputType     string
	InputProvider string
}

func CheckIsDeleted(provider *VTBCloudProvider, inputType, inputProvider string) CheckIsDeletedModifier {
	return CheckIsDeletedModifier{
		Provider:      provider,
		InputType:     inputType,
		InputProvider: inputProvider,
	}
}
func (m CheckIsDeletedModifier) Description(ctx context.Context) string {
	return "Set flag for recreate if order was deleted"
}
func (m CheckIsDeletedModifier) MarkdownDescription(ctx context.Context) string {
	return "Set flag for recreate if order was deleted"
}
func (m CheckIsDeletedModifier) PlanModifyString(
	ctx context.Context,
	req planmodifier.StringRequest,
	resp *planmodifier.StringResponse,
) {
	if req.PlanValue.IsUnknown() || req.PlanValue.IsNull() {
		return
	}

	var state string

	if m.InputType == "vm" {
		order, err := orders.GetComputeOrder(
			m.Provider.Creds,
			m.Provider.ProjectName,
			req.PlanValue.ValueString(),
		)
		if err != nil {
			resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
			return
		}

		item, err := order.GetParentItem()
		if err != nil {
			resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
			return
		}
		state = item.Data.State
	}

	if m.InputType == "cluster" {
		switch m.InputProvider {
		case "kafka":
			{
				order, err := orders.GetKafkaOrder(
					m.Provider.Creds,
					m.Provider.ProjectName,
					req.PlanValue.ValueString(),
				)
				if err != nil {
					resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
					return
				}
				item, err := order.GetParentItem()
				if err != nil {
					resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
					return
				}
				state = item.Data.State
			}
		case "etcd":
			{
				order, err := orders.GetEtcdOrder(
					m.Provider.Creds,
					m.Provider.ProjectName,
					req.PlanValue.ValueString(),
				)
				if err != nil {
					resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				}
				item, err := order.GetParentItem()
				if err != nil {
					resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				}
				state = item.Data.State
			}
		case "clickhouse":
			{
				order, err := orders.GetClickhouseClusterOrder(
					m.Provider.Creds,
					m.Provider.ProjectName,
					req.PlanValue.ValueString(),
				)
				if err != nil {
					resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				}
				item, err := order.GetParentItem()
				if err != nil {
					resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				}
				state = item.Data.State
			}
		case "airflow":
			{
				order, err := orders.GetAirflowClusterOrder(
					m.Provider.Creds,
					m.Provider.ProjectName,
					req.PlanValue.ValueString(),
				)
				if err != nil {
					resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
					return
				}
				item, err := order.GetParentItem()
				if err != nil {
					resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
					return
				}
				state = item.Data.State
			}
		case "tarantool_v2":
			{
				order, err := orders.GetTarantoolClusterOrder(
					m.Provider.Creds,
					m.Provider.ProjectName,
					req.PlanValue.ValueString(),
				)
				if err != nil {
					resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
					return
				}
				item, err := order.GetParentItem()
				if err != nil {
					resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
					return
				}
				state = item.Data.State
			}
		case "balancer_v3":
			order, err := orders.GetBalancerV3Order(
				m.Provider.Creds,
				m.Provider.ProjectName,
				req.PlanValue.ValueString(),
			)
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			item, err := order.GetParentItem()
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			state = item.Data.State
		case "kubernetes":
			order, err := orders.GetK8sClusterOrder(
				m.Provider.Creds,
				m.Provider.ProjectName,
				req.PlanValue.ValueString(),
			)
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			item, err := order.GetParentItem()
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			state = item.Data.State
		case "scylladb":
			order, err := orders.GetScyllaDbClusterOrder(
				m.Provider.Creds,
				m.Provider.ProjectName,
				req.PlanValue.ValueString(),
			)
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			item, err := order.GetParentItem()
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			state = item.Data.State
		}
	}

	if m.InputType == "project" {
		if m.InputProvider == "kubernetes" {
			order, err := orders.GetK8sProjectOrder(
				m.Provider.Creds,
				m.Provider.ProjectName,
				req.PlanValue.ValueString(),
			)
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			item, err := order.GetParentItem()
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			state = item.Data.State
		}
	}

	if m.InputType == "app" {

		switch m.InputProvider {
		case "postgresql_v001":
			order, err := orders.GetPostgresqlOrder(
				m.Provider.Creds,
				m.Provider.ProjectName,
				req.PlanValue.ValueString(),
			)
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			item, err := order.GetParentItem()
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			state = item.Data.State
		case "redis":
			order, err := orders.GetRedisOrder(
				m.Provider.Creds,
				m.Provider.ProjectName,
				req.PlanValue.ValueString(),
			)
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			item, err := order.GetParentItem()
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			state = item.Data.State
		case "redis_sentinel":
			order, err := orders.GetRedisSentinelOrder(
				m.Provider.Creds,
				m.Provider.ProjectName,
				req.PlanValue.ValueString(),
			)
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			item, err := order.GetParentItem()
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			state = item.Data.State
		case "clickhouse":
			order, err := orders.GetClickHouseOrder(
				m.Provider.Creds,
				m.Provider.ProjectName,
				req.PlanValue.ValueString(),
			)
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			item, err := order.GetParentItem()
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			state = item.Data.State
		case "wildfly":
			order, err := orders.GetWildflyOrder(
				m.Provider.Creds,
				m.Provider.ProjectName,
				req.PlanValue.ValueString(),
			)
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			item, err := order.GetParentItem()
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			state = item.Data.State
		case "nginx":
			order, err := orders.GetNginxOrder(
				m.Provider.Creds,
				m.Provider.ProjectName,
				req.PlanValue.ValueString(),
			)
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			item, err := order.GetParentItem()
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			state = item.Data.State
		case "agent_orchestration":
			order, err := orders.GetAgentOrchestrationOrder(
				m.Provider.Creds,
				m.Provider.ProjectName,
				req.PlanValue.ValueString(),
			)
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			item, err := order.GetParentItem()
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			state = item.Data.State
		case "airflow":
			order, err := orders.GetAirflowStandaloneOrder(
				m.Provider.Creds,
				m.Provider.ProjectName,
				req.PlanValue.ValueString(),
			)
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			item, err := order.GetParentItem()
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			state = item.Data.State
		case "grafana":
			order, err := orders.GetGrafanaOrder(
				m.Provider.Creds,
				m.Provider.ProjectName,
				req.PlanValue.ValueString(),
			)
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			item, err := order.GetParentItem()
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			state = item.Data.State
		default:
			resp.Diagnostics.AddError(
				"Check resource is not deleted",
				fmt.Sprintf("Unknown provider == %s", m.InputProvider),
			)
			return
		}
	}

	if m.InputType == "saas" {
		switch m.InputProvider {
		case "rqaas":
			order, err := orders.GetRQaaSOrder(
				m.Provider.Creds,
				m.Provider.ProjectName,
				req.PlanValue.ValueString(),
			)
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			item, err := order.GetParentItem()
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			state = item.Data.State
		}
	}

	if m.InputType == "paas_ktaas" {
		switch m.InputProvider {
		case "ktaas":
			order, err := orders.GetKTaaSOrder(
				m.Provider.Creds,
				m.Provider.ProjectName,
				req.PlanValue.ValueString(),
			)
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			item, err := order.GetParentItem()
			if err != nil {
				resp.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return
			}
			state = item.Data.State
		}
	}

	if state == "" {
		resp.Diagnostics.AddError(
			"Check resource is not deleted",
			"Can't extract state from order",
		)
	}

	if state == "deleted" || state == "deprovisioned" {
		resp.PlanValue = types.StringValue("")
		resp.RequiresReplace = true
	}
}

// GrowModifier
// Модификатор, пресекает попытки уменьшить текущее значение поля типа int в меньшую сторону
type GrowModifier struct{}

func GrowOnly() GrowModifier {
	return GrowModifier{}
}
func (m GrowModifier) Description(ctx context.Context) string {
	return "Value may only grow, but you want reduce it."
}
func (m GrowModifier) MarkdownDescription(ctx context.Context) string {
	return "Value may only grow, but you want reduce it."
}
func (m GrowModifier) PlanModifyInt64(
	ctx context.Context,
	req planmodifier.Int64Request,
	resp *planmodifier.Int64Response,
) {
	if req.ConfigValue.IsNull() || req.PlanValue.IsNull() || req.StateValue.IsNull() {
		// shouldn't happen, but let's not panic if it does
		return
	}

	if req.State.Raw.IsNull() {
		// if we're creating the resource, no need to delete and
		// recreate it
		return
	}

	if req.Plan.Raw.IsNull() {
		// if we're deleting the resource, no need to delete and
		// recreate it
		return
	}

	state := req.StateValue
	plan := req.PlanValue
	if plan.ValueInt64() < state.ValueInt64() {
		path := req.Path.String()
		resp.Diagnostics.AddError(
			"GrowModifier error:",
			fmt.Sprintf(
				"At path:(%s)\nValue can'be grow, but not reduce. Current value:%d, plan value: %d",
				path, state.ValueInt64(), plan.ValueInt64(),
			),
		)
	}
}

// MinimalGrowModifier
// Модификатор, допускающий изменения текущего значения поля
// типа int как минимум на minimalGrowValue
type MinimalGrowModifier struct {
	minimalGrowValue int
}

func MininalGrow(value int) MinimalGrowModifier {
	return MinimalGrowModifier{
		minimalGrowValue: value,
	}
}
func (m MinimalGrowModifier) Description(ctx context.Context) string {
	return "Minimal delta for change value"
}
func (m MinimalGrowModifier) MarkdownDescription(ctx context.Context) string {
	return "Set what will minimal delta between plan and state"
}
func (m MinimalGrowModifier) PlanModifyInt64(
	ctx context.Context,
	req planmodifier.Int64Request,
	resp *planmodifier.Int64Response,
) {
	// shouldn't happen, but let's not panic if it does
	if req.ConfigValue.IsNull() || req.PlanValue.IsNull() || req.StateValue.IsNull() {
		return
	}

	// if we're creating the resource, no need to delete and
	// recreate it
	if req.State.Raw.IsNull() {
		return
	}

	// if we're deleting the resource, no need to delete and
	// recreate it
	if req.Plan.Raw.IsNull() {
		return
	}

	state := req.StateValue
	plan := req.PlanValue
	growValue := plan.ValueInt64() - state.ValueInt64()

	if state.ValueInt64() != plan.ValueInt64() && int64(m.minimalGrowValue) > growValue {
		path := req.Path.String()
		resp.Diagnostics.AddError(
			"Validate by minimal grow modifier:",
			fmt.Sprintf(
				"At path:(%s)\nMinimal value for grow is %d, current is %d",
				path,
				m.minimalGrowValue,
				growValue,
			),
		)
	}
}

type EmptyAccessGroupModifier struct{}

func EmptyAccessGroup() EmptyAccessGroupModifier {
	return EmptyAccessGroupModifier{}
}
func (m EmptyAccessGroupModifier) Description(ctx context.Context) string {
	return "If access group is unknown set null value"
}
func (m EmptyAccessGroupModifier) MarkdownDescription(ctx context.Context) string {
	return "Transformate access group from unknown to null value"
}
func (m EmptyAccessGroupModifier) PlanModifyMap(
	ctx context.Context,
	req planmodifier.MapRequest,
	resp *planmodifier.MapResponse,
) {
	if req.PlanValue.IsUnknown() {
		plan, diags := basetypes.NewMapValueFrom(
			ctx,
			basetypes.SetType{ElemType: basetypes.StringType{}},
			make(map[string][]string, 0),
		)
		resp.Diagnostics.Append(diags...)
		resp.PlanValue = plan
		return
	}
}

type DefaultEmptyStringListModifier struct{}

func DefaultEmptyStringList() DefaultEmptyStringListModifier {
	return DefaultEmptyStringListModifier{}
}
func (m DefaultEmptyStringListModifier) Description(ctx context.Context) string {
	return "If value not set, then will set empty list"
}
func (m DefaultEmptyStringListModifier) MarkdownDescription(ctx context.Context) string {
	return "For unset value will set empty list"
}
func (m DefaultEmptyStringListModifier) PlanModifySet(
	ctx context.Context,
	req planmodifier.SetRequest,
	resp *planmodifier.SetResponse,
) {
	if req.PlanValue.IsUnknown() {
		plan, diags := basetypes.NewSetValueFrom(ctx, types.StringType, make([]types.String, 0))
		resp.Diagnostics.Append(diags...)
		resp.PlanValue = plan
		return
	}
}

func AirflowClientCertModifier(
	ctx context.Context,
	req planmodifier.StringRequest,
	resp *stringplanmodifier.RequiresReplaceIfFuncResponse,
) {
	if req.PlanValue.IsUnknown() || req.PlanValue.IsNull() {
		resp.RequiresReplace = true
		return
	}
	if !(req.StateValue.IsNull() || req.StateValue.IsUnknown()) && req.StateValue != req.PlanValue {
		resp.RequiresReplace = true
		return
	}
	resp.RequiresReplace = false
}

func NginxToAngieMigrateModifier(
	ctx context.Context,
	req planmodifier.StringRequest,
	resp *stringplanmodifier.RequiresReplaceIfFuncResponse,
) {
	var distrib string
	req.State.GetAttribute(ctx, path.Root("distrib"), &distrib)
	if distrib == "Nginx" {
		resp.RequiresReplace = false
		return
	}
	resp.RequiresReplace = true
}

type IgnoreChangesModifier struct{}

func (m IgnoreChangesModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() {
		return
	}
	value := strings.Trim(req.StateValue.String(), "\"")
	resp.PlanValue = types.StringValue(value)
}

func (m IgnoreChangesModifier) Description(_ context.Context) string {
	return "Ignores changes in the field in all cases except for initialization."
}

func (m IgnoreChangesModifier) MarkdownDescription(_ context.Context) string {
	return "Ignores changes in the field in all cases except for initialization."
}

type AvailableGroupsTypeModifier struct {
	Provider  *VTBCloudProvider
	GroupType string
}

func AvailableGroupsType(provider *VTBCloudProvider, groupType string) AvailableGroupsTypeModifier {
	return AvailableGroupsTypeModifier{
		Provider:  provider,
		GroupType: groupType,
	}
}
func (m AvailableGroupsTypeModifier) Description(ctx context.Context) string {
	return "Check available groups for current project of specified accounts type (personal of service-accounts)"
}
func (m AvailableGroupsTypeModifier) MarkdownDescription(ctx context.Context) string {
	return "Check available groups for current project of specified accounts type (personal of service-accounts)"
}
func (m AvailableGroupsTypeModifier) PlanModifySet(
	ctx context.Context,
	req planmodifier.SetRequest,
	resp *planmodifier.SetResponse,
) {
	if req.PlanValue.IsUnknown() || req.PlanValue.IsNull() {
		return
	}

	planGroups := []string{}
	resp.Diagnostics.Append(req.PlanValue.ElementsAs(ctx, &planGroups, false)...)

	err := checkGroupsByAccountType(m.Provider, m.GroupType, planGroups)
	if err != nil {
		resp.Diagnostics.AddError("Unavailble groups for resource "+req.Path.String(), err.Error())
	}
}

type AvailableGroupsTypeMapModifier struct {
	Provider  *VTBCloudProvider
	GroupType string
}

func AvailableGroupsTypeMap(provider *VTBCloudProvider, groupType string) AvailableGroupsTypeMapModifier {
	return AvailableGroupsTypeMapModifier{
		Provider:  provider,
		GroupType: groupType,
	}
}
func (m AvailableGroupsTypeMapModifier) Description(ctx context.Context) string {
	return "Check available groups for current project of specified accounts type (personal of service-accounts)"
}
func (m AvailableGroupsTypeMapModifier) MarkdownDescription(ctx context.Context) string {
	return "Check available groups for current project of specified accounts type (personal of service-accounts)"
}
func (m AvailableGroupsTypeMapModifier) PlanModifyMap(
	ctx context.Context,
	req planmodifier.MapRequest,
	resp *planmodifier.MapResponse,
) {
	if req.PlanValue.IsUnknown() || req.PlanValue.IsNull() {
		return
	}

	planGroupsMap := map[string][]string{}
	resp.Diagnostics.Append(req.PlanValue.ElementsAs(ctx, &planGroupsMap, false)...)

	for _, planGroups := range planGroupsMap {
		err := checkGroupsByAccountType(m.Provider, m.GroupType, planGroups)
		if err != nil {
			resp.Diagnostics.AddError("Unavailble groups for resource "+req.Path.String(), err.Error())
		}
	}
}

func checkGroupsByAccountType(
	p *VTBCloudProvider,
	groupType string,
	planGroups []string,
) error {
	groups, err := sources.GetAccessGroupsByAccountsType(p.Creds, p.ProjectName, groupType)
	if err != nil {
		return fmt.Errorf("get groups from portal: %v", err.Error())
	}

	groupsSl := []string{}
	for _, s := range groups {
		groupsSl = append(groupsSl, s.Name)
	}

	for _, gr := range planGroups {
		if !slices.Contains(groupsSl, gr) {
			return fmt.Errorf(
				"available groups: [%s]. Specified '%s'",
				strings.Join(groupsSl, ", "), strings.Join(planGroups, ", "),
			)
		}
	}
	return nil
}

type IgnoreChangesBoolModifier struct{}

func (m IgnoreChangesBoolModifier) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
	if req.StateValue.IsNull() {
		return
	}
	resp.PlanValue = req.StateValue
}

func (m IgnoreChangesBoolModifier) Description(_ context.Context) string {
	return "Ignores changes in the field in all cases except for initialization."
}

func (m IgnoreChangesBoolModifier) MarkdownDescription(_ context.Context) string {
	return "Ignores changes in the field in all cases except for initialization."
}
