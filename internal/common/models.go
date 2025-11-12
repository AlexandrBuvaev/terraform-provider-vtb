package common

import "github.com/hashicorp/terraform-plugin-framework/types"

type ExtraMountModel struct {
	Size       types.Int64  `tfsdk:"size"`
	FileSystem types.String `tfsdk:"file_system"`
}

type ImageStandardModel struct {
	Distribution  types.String `tfsdk:"distribution"`
	OsVersion     types.String `tfsdk:"os_version"`
	OnSupport     types.Bool   `tfsdk:"on_support"`
	ProductID     types.String `tfsdk:"product_id"`
	ADIntegration types.Bool   `tfsdk:"ad_integration"`
}

type ClusterProductImageDataSourceModel struct {
	Distribution  types.String `tfsdk:"distribution"`
	OsVersion     types.String `tfsdk:"os_version"`
	OnSupport     types.Bool   `tfsdk:"on_support"`
	ProductID     types.String `tfsdk:"product_id"`
	ADIntegration types.Bool   `tfsdk:"ad_integration"`

	GeoDistribution types.Bool `tfsdk:"geo_distribution"`
}
