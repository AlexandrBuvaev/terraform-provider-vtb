package custommodifires

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

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
