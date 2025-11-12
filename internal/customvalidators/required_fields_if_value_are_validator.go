package customvalidators

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// RequireFieldsIfValueAreValidator проверяет, что при указаном значении FieldValue для поля, в котором объявлен валидатор,
// пользователь объявил в схеме поля, указанные в RequiredFields.
// Требуемые поля перечасляются в списке. Если поле входит в состав другого атрибута, то нужно указать путь до него через точку,
// например, []string{"rabbitmq_config","image.airflow_version"}
type RequireFieldsIfValueAreValidator struct {
	FieldValue     string   // Значение поля, при котором происходит проверка
	RequiredFields []string // Поля, которые должны быть указаны в схеме
}

func RequireIfValueAre(fieldValue string, RequiredFields []string) RequireFieldsIfValueAreValidator {
	return RequireFieldsIfValueAreValidator{
		FieldValue:     fieldValue,
		RequiredFields: RequiredFields,
	}
}
func (v RequireFieldsIfValueAreValidator) Description(ctx context.Context) string {
	return "Use schema validator RequireIfValueAre for field, when it's certain value requires other fields existance"
}
func (v RequireFieldsIfValueAreValidator) MarkdownDescription(ctx context.Context) string {
	return "Use schema validator RequireIfValueAre for field, when it's certain value requires other fields existance"
}

func (v RequireFieldsIfValueAreValidator) ValidateString(
	ctx context.Context,
	req validator.StringRequest,
	resp *validator.StringResponse,
) {
	buildPath := func(requiredField string) path.Path {
		requiredFieldPath := strings.Split(requiredField, ".")
		schemaPath := path.Root(requiredFieldPath[0])
		for _, step := range requiredFieldPath[1:] {
			schemaPath = schemaPath.AtName(step)
		}
		return schemaPath
	}

	// If the value is unknown or null, there is nothing to validate.
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}
	actualFieldValue := req.ConfigValue.ValueString()
	if actualFieldValue != v.FieldValue {
		return
	}

	for _, requiredField := range v.RequiredFields {

		schemaPath := buildPath(requiredField)

		var targetValue attr.Value
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, schemaPath, &targetValue)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if targetValue.IsUnknown() || targetValue.IsNull() {
			elemPath := req.Path.AtMapKey(actualFieldValue)
			resp.Diagnostics.AddError(
				"Field Validation Error",
				fmt.Sprintf(
					"Field '%s' with value '%s' requires not empty fields: %s",
					elemPath.String(),
					v.FieldValue,
					strings.Join(v.RequiredFields, ", "),
				),
			)
			return
		}
	}
}
