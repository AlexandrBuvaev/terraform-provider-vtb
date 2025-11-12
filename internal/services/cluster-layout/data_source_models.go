package clusterlayout

import "github.com/hashicorp/terraform-plugin-framework/types"

type ClusterLayoutModel struct {
	ID             types.String `tfsdk:"id"`
	Layout         types.String `tfsdk:"layout"`
	NetSegmentCode types.String `tfsdk:"net_segment"`
}
