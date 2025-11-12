package custommodifires

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

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
