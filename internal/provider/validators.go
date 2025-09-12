package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
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

// NamespaceNotReservedPrefixValidator ensures the namespace part of <namespace>/<name>:<port>
// does not start with cluster-, kube-, or default.
type NamespaceNotReservedPrefixValidator struct{}

func NamespaceNotReservedPrefix() NamespaceNotReservedPrefixValidator {
	return NamespaceNotReservedPrefixValidator{}
}

func (v NamespaceNotReservedPrefixValidator) Description(_ context.Context) string {
	return "The namespace must not start with 'cluster-', 'kube-', or 'default'."
}

func (v NamespaceNotReservedPrefixValidator) MarkdownDescription(_ context.Context) string {
	return "The namespace must not start with cluster-, kube-, or default."
}

func (v NamespaceNotReservedPrefixValidator) ValidateString(
	ctx context.Context,
	req validator.StringRequest,
	resp *validator.StringResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()

	parts := strings.SplitN(value, "/", 2)
	if len(parts) != 2 {
		return
	}

	ns := parts[0]
	badPrefixes := []string{"cluster-", "kube-", "default"}
	for _, prefix := range badPrefixes {
		if strings.HasPrefix(ns, prefix) {
			resp.Diagnostics.AddAttributeError(
				req.Path,
				"Not valid namespace prefix",
				fmt.Sprintf("Namespace '%s' must not start with '%s'", ns, prefix),
			)
			return
		}
	}
}
