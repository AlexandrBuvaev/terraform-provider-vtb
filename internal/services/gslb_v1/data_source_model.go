package gslbv1

import "github.com/hashicorp/terraform-plugin-framework/types"

type GSLBV1ImageDataSourceModel struct {
	Distribution types.String `tfsdk:"distribution"`
	OsVersion    types.String `tfsdk:"os_version"`
	OnSupport    types.Bool   `tfsdk:"on_support"`
	ProductID    types.String `tfsdk:"product_id"`

	GeoDistribution types.Bool   `tfsdk:"geo_distribution"`
	ProductVersion  types.String `tfsdk:"product_version"`
}
