package custommodifires

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

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
