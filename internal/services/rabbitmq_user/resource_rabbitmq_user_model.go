package rabbitmquser

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type RabbitMQUserModel struct {
	RabbitMQOrderID types.String `tfsdk:"rabbitmq_order_id"`
	Username        types.String `tfsdk:"username"`
	VhostsAccess    types.Object `tfsdk:"vhosts_access"`
}

// Модель для редактирования прав доступа на вирутальных хостах
type VhostPermissionsModel struct {
	VhostRead      types.Set `tfsdk:"vhost_read"`
	VhostWrite     types.Set `tfsdk:"vhost_write"`
	VhostConfigure types.Set `tfsdk:"vhost_configure"`
}

func (v VhostPermissionsModel) AttributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"vhost_read":      types.SetType{ElemType: types.StringType},
		"vhost_write":     types.SetType{ElemType: types.StringType},
		"vhost_configure": types.SetType{ElemType: types.StringType},
	}
}
