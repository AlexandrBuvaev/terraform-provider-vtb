package utils

import (
	"context"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type OrderAwareResource interface {
	GetOrderID() types.String
	SetOrderID(types.String)
}

func HandleExternalDeletionRecreate(
	ctx context.Context,
	resp *resource.ModifyPlanResponse,
	plan interface{},
	isDeleted bool,
	diagnostics diag.Diagnostics,
) {
	resp.Diagnostics.Append(diagnostics...)
	if resp.Diagnostics.HasError() || !isDeleted {
		return
	}

	// Проверяем, что план реализует наш интерфейс
	if resource, ok := plan.(OrderAwareResource); ok {
		resource.SetOrderID(types.StringValue(""))
		resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
	} else {
		// Fallback: пытаемся очистить поле через рефлексию
		clearOrderIDViaReflection(plan)
		resp.Diagnostics.Append(resp.Plan.Set(ctx, plan)...)
	}

	resp.RequiresReplace.Append(path.Root("order_id"))
	resp.Diagnostics.AddWarning(
		"External deletion",
		"Resource will be recreated due to external deletion",
	)
}

func clearOrderIDViaReflection(plan interface{}) {
	val := reflect.ValueOf(plan).Elem()
	if val.Kind() == reflect.Struct {
		field := val.FieldByName("OrderID")
		if field.IsValid() && field.CanSet() {
			if field.Type() == reflect.TypeOf(types.StringValue("")) {
				field.Set(reflect.ValueOf(types.StringValue("")))
			}
		}
	}
}
