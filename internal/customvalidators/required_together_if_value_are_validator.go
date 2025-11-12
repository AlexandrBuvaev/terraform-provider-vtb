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

// RequiredTogetherIfValueAreValidator проверяет, что при указании значения FieldValue
// в поле FieldName требуется указание следующих полей в конфигурации: RequiredToghererFields.
type RequiredTogetherIfValueAreValidator struct {
	FieldName           string   // Имя проверяемого поля
	FieldValue          string   // Значение, при котором происходит проверка
	OneOfRequiredFields []string // Поля, которые должны быть указаны при соблюдении условия
}

func AtLeastOneOfIfValueAre(fieldName, fieldValue string, OneOfRequiredFields []string) RequiredTogetherIfValueAreValidator {
	return RequiredTogetherIfValueAreValidator{
		FieldName:           fieldName,
		FieldValue:          fieldValue,
		OneOfRequiredFields: OneOfRequiredFields,
	}
}
func (v RequiredTogetherIfValueAreValidator) Description(ctx context.Context) string {
	return "Use schema validator AtLeastOneOf for field with some specified value"
}
func (v RequiredTogetherIfValueAreValidator) MarkdownDescription(ctx context.Context) string {
	return "Use schema validator AtLeastOneOf for field with some specified value"
}
func (v RequiredTogetherIfValueAreValidator) ValidateMap(
	ctx context.Context,
	req validator.MapRequest,
	resp *validator.MapResponse,
) {
	type PathObject struct {
		MainFieldPath       path.Path
		RequiredFieldsPaths []path.Path
	}

	var pathsObjects []PathObject
	for key := range req.ConfigValue.Elements() {

		elemPath := req.Path.AtMapKey(key)
		var tmp PathObject
		tmp.MainFieldPath = elemPath.AtName(v.FieldName)

		for _, pName := range v.OneOfRequiredFields {
			tmp.RequiredFieldsPaths = append(tmp.RequiredFieldsPaths, elemPath.AtName(pName))
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
			for _, attrPath := range pObject.RequiredFieldsPaths {
				var value attr.Value
				req.Config.GetAttribute(ctx, attrPath, &value)
				if value != nil && !value.IsNull() {
					count++
				}
			}
			if count == 0 {
				resp.Diagnostics.AddError(
					"cleanup_policy validation error",
					fmt.Sprintf(
						"Field '%s' with value '%s' required at least one of [%s]",
						v.FieldName,
						v.FieldValue,
						strings.Join(v.OneOfRequiredFields, ", "),
					),
				)
			}
		}
	}
}
