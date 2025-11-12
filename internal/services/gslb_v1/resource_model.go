package gslbv1

import (
	"terraform-provider-vtb/internal/common"
	"terraform-provider-vtb/internal/services/core"
	"terraform-provider-vtb/internal/services/flavor"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type GSLBV1ResourceModel struct {
	Label            types.String                      `tfsdk:"label"`
	Lifetime         types.Int64                       `tfsdk:"lifetime"`
	OrderID          types.String                      `tfsdk:"order_id"`
	ItemID           types.String                      `tfsdk:"item_id"`
	Core             core.CoreModel                    `tfsdk:"core"`
	Image            GSLBV1ImageDataSourceModel        `tfsdk:"image"`
	FinancialProject types.String                      `tfsdk:"financial_project"`
	Layout           types.String                      `tfsdk:"layout"`
	Access           map[string][]types.String         `tfsdk:"access"`
	Flavor           flavor.FlavorModel                `tfsdk:"flavor"`
	ExtraMounts      map[string]common.ExtraMountModel `tfsdk:"extra_mounts"`

	ApiPassword    types.String `tfsdk:"api_password"`
	NginxPassword  types.String `tfsdk:"nginx_password"`
	DesiredVersion types.String `tfsdk:"desired_version"`
	DNSZone        types.String `tfsdk:"dns_zone"`
	BgPaas         types.Bool   `tfsdk:"bgpaas"`
	GSLBHosts      types.Set    `tfsdk:"gslb_hosts"`
}

type GSLBHosts struct {
	Name            types.String `tfsdk:"name"`
	MaintenanceMode types.String `tfsdk:"maintenance_mode"`
}

func (m GSLBHosts) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"name":             types.StringType,
		"maintenance_mode": types.StringType,
	}
}

func (m GSLBHosts) ToObjectValue() (types.Object, diag.Diagnostics) {
	return types.ObjectValue(
		m.AttributeTypes(),
		map[string]attr.Value{
			"name":             m.Name,
			"maintenance_mode": m.MaintenanceMode,
		},
	)
}
