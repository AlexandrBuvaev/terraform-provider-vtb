package custommodifires

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

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
