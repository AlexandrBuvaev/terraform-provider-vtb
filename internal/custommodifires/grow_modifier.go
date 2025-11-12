package custommodifires

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

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
