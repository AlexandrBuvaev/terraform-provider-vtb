package customvalidators

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
)

// Валидация на пустой список групп доступа для роли в AD. Пример не допустимого значения: "role" = []
type EmptyAccessGroupListValidator struct{}

func (v EmptyAccessGroupListValidator) EmptyAccessGroupListValidator() EmptyAccessGroupListValidator {
	return EmptyAccessGroupListValidator{}
}

func (v EmptyAccessGroupListValidator) Description(ctx context.Context) string {
	return "Empty access group list validation."
}

func (v EmptyAccessGroupListValidator) MarkdownDescription(ctx context.Context) string {
	return "Empty access group list validation."
}

func (v EmptyAccessGroupListValidator) ValidateMap(
	ctx context.Context,
	req validator.MapRequest,
	resp *validator.MapResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	for role, groupsValue := range req.ConfigValue.Elements() {
		attrPath := req.Path.AtMapKey(role)

		var groups []string
		diags := tfsdk.ValueAs(ctx, groupsValue, &groups)
		resp.Diagnostics.Append(diags...)
		if diags.HasError() {
			continue
		}

		if len(groups) == 0 {
			resp.Diagnostics.AddAttributeError(
				attrPath,
				"Empty Groups List",
				fmt.Sprintf("Groups list for role '%s' must not be empty", role),
			)
		}
	}
}
