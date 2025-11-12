package rabbitmq

import (
	"context"
	"fmt"
	"regexp"
	"slices"

	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/consts"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &RabbitMQVhostsResource{}
	_ resource.ResourceWithImportState = &RabbitMQVhostsResource{}
)

type RabbitMQVhostsResource struct {
	client *client.CloudClient
}

func NewRabbitMQVhostsResource() resource.Resource {
	return &RabbitMQVhostsResource{}
}

func (r RabbitMQVhostsResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_rabbitmq_vhosts"
}

func (r *RabbitMQVhostsResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.CloudClient)

	if !ok {
		resp.Diagnostics.AddError(
			"unexpected resource configure type",
			fmt.Sprintf("Expected *client.CloudClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

type RabbitMQVhostsModel struct {
	RabbitMQOrderID types.String `tfsdk:"rabbitmq_order_id"`
	Hostnames       types.Set    `tfsdk:"hostnames"`
}

func (t RabbitMQVhostsResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description:         "Управление виртуальными хостами в кластере RabbitMQ.",
		MarkdownDescription: "Управление виртуальными хостами в кластере RabbitMQ.",
		Attributes: map[string]schema.Attribute{

			"rabbitmq_order_id": schema.StringAttribute{
				Required:            true,
				Description:         "Идентификатор заказа кластера RabbitMQ",
				MarkdownDescription: "Идентификатор заказа кластера RabbitMQ",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"hostnames": schema.SetAttribute{
				Required:            true,
				ElementType:         types.StringType,
				Description:         "Список названий виртуальных хостов кластера RabbitMQ.",
				MarkdownDescription: "Список названий виртуальных хостов кластера RabbitMQ.",
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(
						stringvalidator.LengthAtLeast(2),
						stringvalidator.LengthAtMost(25),
						stringvalidator.NoneOf("/"),
						stringvalidator.RegexMatches(
							regexp.MustCompile(`^[A-Za-z0-9]+([._-][A-Za-z0-9]+)*$`),
							"may contain uppercase/lowercase Latin letters, numbers, "+
								"punctuation marks ['.' '-' '_']. (cannot start or end with punctuation)",
						),
					),
				},
			},
		},
	}
}

func (r RabbitMQVhostsResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("rabbitmq_order_id"), req, resp)
}

func (r RabbitMQVhostsResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan RabbitMQVhostsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetRabbitMQOrder(
		r.client.Creds,
		r.client.ProjectName,
		plan.RabbitMQOrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("rabbitmq_order_id"),
			consts.CREATE_RES_FAIL,
			fmt.Sprintln("fetch order from portal API ended with error:", err.Error()),
		)
		return
	}

	cluster, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.CREATE_RES_FAIL, "can't get cluster item"),
			err.Error(),
		)
		return
	}

	clusterConfig := cluster.Data.Config.(entities.RabbitMQItemConfig)
	existingVhosts := clusterConfig.Vhosts

	var planVhosts []string
	plan.Hostnames.ElementsAs(ctx, &planVhosts, false)

	// Создаем карту для быстрого поиска существующих виртуальных хостов
	existingVhostsMap := make(map[string]bool)
	for _, vhost := range existingVhosts {
		existingVhostsMap[vhost.Name] = true
	}

	var toAdd []entities.RabbitMQVhost
	for _, vhostName := range planVhosts {
		if !existingVhostsMap[vhostName] {
			newVhost := entities.RabbitMQVhost{Name: vhostName}
			toAdd = append(toAdd, newVhost)
		}
	}

	if len(toAdd) > 0 {
		err := order.CreateVHosts(toAdd)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("hostnames"),
				fmt.Sprintln(consts.CREATE_RES_FAIL, "vhosts creation ended with error"),
				err.Error(),
			)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r RabbitMQVhostsResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state RabbitMQVhostsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetRabbitMQOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.RabbitMQOrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("rabbitmq_order_id"),
			fmt.Sprintln(consts.READ_RES_FAIL, "fetch order from portal API ended with error"),
			err.Error(),
		)
		return
	}

	cluster, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.READ_RES_FAIL, "can't get cluster item:"),
			err.Error(),
		)
		return
	}

	clusterConfig := cluster.Data.Config.(entities.RabbitMQItemConfig)
	existingVhosts := clusterConfig.Vhosts

	var hostnames []string
	for _, vhost := range existingVhosts {
		// skip default vhost
		if vhost.Name == "/" {
			continue
		}
		hostnames = append(hostnames, vhost.Name)
	}

	state.Hostnames, _ = types.SetValueFrom(ctx, types.StringType, hostnames)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r RabbitMQVhostsResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state RabbitMQVhostsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetRabbitMQOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.RabbitMQOrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("rabbitmq_order_id"),
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "fetch order from portal API ended with error"),
			err.Error(),
		)
		return
	}

	var stateVhosts []string
	state.Hostnames.ElementsAs(ctx, &stateVhosts, false)

	var planVhosts []string
	plan.Hostnames.ElementsAs(ctx, &planVhosts, false)

	var toDelete []string
	for _, vhostName := range stateVhosts {
		if !slices.Contains(planVhosts, vhostName) {
			toDelete = append(toDelete, vhostName)
		}
	}

	var toAdd []entities.RabbitMQVhost
	for _, vhostName := range planVhosts {
		if !slices.Contains(stateVhosts, vhostName) {
			toAdd = append(toAdd, entities.RabbitMQVhost{
				Name: vhostName,
			})
		}
	}

	if len(toDelete) > 0 {
		err := order.DeleteVHosts(toDelete)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintln(consts.UPDATE_RES_FAIL, "vhosts deletion ended with error"),
				err.Error(),
			)
			return
		}
	}

	if len(toAdd) > 0 {
		err := order.CreateVHosts(toAdd)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintln(consts.UPDATE_RES_FAIL, "vhosts creation ended with error"),
				err.Error(),
			)
			return
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r RabbitMQVhostsResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state RabbitMQVhostsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetRabbitMQOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.RabbitMQOrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("rabbitmq_order_id"),
			fmt.Sprintln(consts.DELETE_RES_FAIL, "fetch order from portal API ended with error"),
			err.Error(),
		)
		return
	}

	var stateVhosts []string
	state.Hostnames.ElementsAs(ctx, &stateVhosts, false)

	err = order.DeleteVHosts(stateVhosts)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.DELETE_RES_FAIL, "vhosts deletion ended with error"),
			err.Error(),
		)
		return
	}
}
