package customvalidators

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ConflictingIfValueAreValidator проверяет, что при указании значения FieldValue
// в поле FieldName поля ConflictingFields не могут быть указаны в конфигурации
type ConflictingIfValueAreValidator struct {
	FieldName         string   // Имя проверяемого поля
	FieldValue        string   // Значение, при котором происходит проверка
	ConflictingFields []string // Поля, которые вызывают конфликт при соблюдении условия
}

func ConflictingIfValueAre(fieldName, fieldValue string, conflictingFields []string) ConflictingIfValueAreValidator {
	return ConflictingIfValueAreValidator{
		FieldName:         fieldName,
		FieldValue:        fieldValue,
		ConflictingFields: conflictingFields,
	}
}
func (v ConflictingIfValueAreValidator) Description(ctx context.Context) string {
	return "Use schema validator ForbiddenByFieldValidator for field with some specified value"
}
func (v ConflictingIfValueAreValidator) MarkdownDescription(ctx context.Context) string {
	return "Use schema validator Forbidden field with some specified value"
}
func (v ConflictingIfValueAreValidator) ValidateMap(
	ctx context.Context,
	req validator.MapRequest,
	resp *validator.MapResponse,
) {
	type PathObject struct {
		MainFieldPath path.Path
		FieldsPaths   []path.Path
	}

	var pathsObjects []PathObject
	for key := range req.ConfigValue.Elements() {

		elemPath := req.Path.AtMapKey(key)
		var tmp PathObject
		tmp.MainFieldPath = elemPath.AtName(v.FieldName)

		for _, pName := range v.ConflictingFields {
			tmp.FieldsPaths = append(tmp.FieldsPaths, elemPath.AtName(pName))
		}
		pathsObjects = append(pathsObjects, tmp)
	}

	for _, pObject := range pathsObjects {

		var targetValue types.String
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, pObject.MainFieldPath, &targetValue)...)
		if resp.Diagnostics.HasError() {
			return
		}

		if targetValue.ValueString() == v.FieldValue {
			var count uint = 0
			for _, attrPath := range pObject.FieldsPaths {
				var value attr.Value
				req.Config.GetAttribute(ctx, attrPath, &value)
				if value != nil && !value.IsNull() {
					count++
				}
			}

			if count != 0 {
				resp.Diagnostics.AddError(
					"cleanup_policy validation error",
					fmt.Sprintf(
						"Field '%s' with value '%s' can't used with fields [%s].",
						v.FieldName,
						v.FieldValue,
						strings.Join(v.ConflictingFields, ", "),
					),
				)
			}
		}
	}
}
