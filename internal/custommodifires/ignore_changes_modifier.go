package custommodifires

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
