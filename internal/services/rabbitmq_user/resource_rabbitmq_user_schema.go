package rabbitmquser

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (r *RabbitMQUserResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description:         "Управление правами пользователей для виртуальных хостов в кластере RabbitMQ.",
		MarkdownDescription: "Управление правами пользователей для виртуальных хостов в кластере RabbitMQ.",
		Attributes: map[string]schema.Attribute{

			"rabbitmq_order_id": schema.StringAttribute{
				Required:            true,
				Description:         "Идентификатор заказа кластера RabbitMQ.",
				MarkdownDescription: "Идентификатор заказа кластера RabbitMQ.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"username": schema.StringAttribute{
				Required:            true,
				Description:         "Имя пользователя",
				MarkdownDescription: "Имя пользователя",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vhosts_access": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Права пользователя на виртуальные хосты",
				Attributes: map[string]schema.Attribute{
					"vhost_read": schema.SetAttribute{
						Optional:            true,
						ElementType:         types.StringType,
						Description:         "Список вирутальных хостов (vhosts) с правами на чтение для пользователя.",
						MarkdownDescription: "Список вирутальных хостов (vhosts) с правами на чтение для пользователя.",
					},
					"vhost_write": schema.SetAttribute{
						Optional:            true,
						ElementType:         types.StringType,
						Description:         "Список виртуальных хостов (vhosts) с правами на запись для пользователя",
						MarkdownDescription: "Спико виртуальных хостов (vhosts) с правами на запись для пользователя",
					},
					"vhost_configure": schema.SetAttribute{
						Optional:            true,
						ElementType:         types.StringType,
						Description:         "Список виртуальных хостов (vhosts) с правами на конфигурирование для пользователя",
						MarkdownDescription: "Список виртуальных хостов (vhosts) с правами на конфигурирование для пользователя",
					},
				},
			},
		},
	}
}
