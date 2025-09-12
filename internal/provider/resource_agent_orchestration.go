package provider

import (
	"context"
	"fmt"

	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"
	"terraform-provider-vtb/pkg/client/sources"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &AgentOrchestrationResource{}
	_ resource.ResourceWithImportState = &AgentOrchestrationResource{}
)

type AgentOrchestrationResource struct {
	provider *VTBCloudProvider
}

func NewAgentOrchestrationResource(p *VTBCloudProvider) AgentOrchestrationResource {
	return AgentOrchestrationResource{
		provider: p,
	}
}

func (r AgentOrchestrationResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = "vtb_agent_orchestration_instance"
}

type AgentOrchestrationResourceModel struct {
	Lifetime         types.Int64                `tfsdk:"lifetime"`
	Label            types.String               `tfsdk:"label"`
	OrderID          types.String               `tfsdk:"order_id"`
	ItemID           types.String               `tfsdk:"item_id"`
	Core             CoreModel                  `tfsdk:"core"`
	Flavor           FlavorModel                `tfsdk:"flavor"`
	Image            ImageStandardModel         `tfsdk:"image"`
	ExtraMounts      map[string]ExtraMountModel `tfsdk:"extra_mounts"`
	Hostname         types.String               `tfsdk:"hostname"`
	FinancialProject types.String               `tfsdk:"financial_project"`

	SferaAgent       SferaAgentModel `tfsdk:"sfera_agent"`
	AgentVersion     types.String    `tfsdk:"agent_version"`
	AgentPool        types.String    `tfsdk:"agent_pool"`
	ChannelURL       types.String    `tfsdk:"channel_url"`
	AgentInstance    types.String    `tfsdk:"agent_instance"`
	CountOfExecutors types.Int64     `tfsdk:"count_of_executors"`
}

type SferaAgentModel struct {
	JenkinsAgentExecutors types.Int64                `tfsdk:"jenkins_agent_executors"`
	JenkinsAgentSubsystem JenkinsAgentSubsystemModel `tfsdk:"jenkins_agent_subsystem"`
}

var JenkinsAgentSubsystemSchema = map[string]schema.Attribute{
	"ris_id": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Код информационной системы (RIS code)",
		Description:         "Код информационной системы (RIS code)",
	},
	"is_code": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Код информационной системы (IS code)",
		Description:         "Код информационной системы (IS code)",
	},
	"head_name": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Название головной системы",
		Description:         "Название головной системы",
	},
	"net_segment": schema.StringAttribute{
		Required:            true,
		Description:         "Сетевой сегмент",
		MarkdownDescription: "Сетевой сегмент",
	},
	"display_name": schema.StringAttribute{
		Required:            true,
		Description:         "Отоброжаемое имя подсистемы",
		MarkdownDescription: "Отображаемое имя подсистемы",
	},
	"sfera_head_url": schema.StringAttribute{
		Required:            true,
		Description:         "URL головной системы в сфере",
		MarkdownDescription: "URL головной системы в сфере",
	},
	"node_group_name": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Имя группы нод (агентов)",
		Description:         "Имя группы нод (агентов)",
	},
}

func (r AgentOrchestrationResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"lifetime": schema.Int64Attribute{
				Optional:            true,
				Description:         "Время жизни заказа в днях (2, 7, 14, 30)",
				MarkdownDescription: "Время жизни заказа в днях (2, 7, 14, 30)",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
			},
			"label": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Метка заказа.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"order_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Идентификатор заказа. Становиться известен после создания заказа.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					CheckIsDeleted(r.provider, "app", "agent_orchestration"),
				},
			},

			"item_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Идентификатор сущностей ВМ, известен после создания ВМ.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"core": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          coreSchema,
				MarkdownDescription: "Основные параметры для виртуальных машин в заказе.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},

			"flavor": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          flavorSchema,
				MarkdownDescription: "Кол-во CPU/RAM для виртуальных машин.",
			},

			"image": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          imageStandardSchema,
				MarkdownDescription: "Тип вычислительного экземпляра.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"extra_mounts": schema.MapNestedAttribute{
				Required:            true,
				MarkdownDescription: "Дополнительные точки монтирования для ВМ.",
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.OneOf("/app")),
					mapvalidator.SizeBetween(1, 1),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"size": schema.Int64Attribute{
							Required:            true,
							MarkdownDescription: "Размер точки монтирования (в ГБ.)",
							Validators: []validator.Int64{
								int64validator.AtMost(2048),
								int64validator.AtLeast(10),
							},
							PlanModifiers: []planmodifier.Int64{
								GrowOnly(),
								MininalGrow(10),
							},
						},
						"file_system": schema.StringAttribute{
							Computed:            true,
							Default:             stringdefault.StaticString("xfs"),
							Description:         "Тип файловой системы.",
							MarkdownDescription: "Тип файловой системы.",
						},
					},
				},
			},

			"hostname": schema.StringAttribute{
				Computed:            true,
				Description:         "Имя хоста виртуальной машины.",
				MarkdownDescription: "Имя хоста виртуальной машины.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"financial_project": schema.StringAttribute{
				Required:            true,
				Description:         "Источник финансирования.",
				MarkdownDescription: "Источник финансирования.",
			},

			// product fields
			"sfera_agent": schema.SingleNestedAttribute{
				Required: true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				MarkdownDescription: "Configuration of Sfera agent component",
				Attributes: map[string]schema.Attribute{
					"jenkins_agent_subsystem": schema.SingleNestedAttribute{
						Required:            true,
						MarkdownDescription: "Система / подсистема",
						Attributes:          JenkinsAgentSubsystemSchema,
					},
					"jenkins_agent_executors": schema.Int64Attribute{
						Required:            true,
						MarkdownDescription: "Количество экзекьютеров",
					},
				},
			},
			"agent_version": schema.StringAttribute{
				Computed:            true,
				Description:         "Версия Агента Оркестрации",
				MarkdownDescription: "Версия Агента Оркестрации",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"agent_pool": schema.StringAttribute{
				Description:         "Пул",
				MarkdownDescription: "Пул",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"channel_url": schema.StringAttribute{
				Description:         "Канальное приложение",
				MarkdownDescription: "Канальное приложение",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"agent_instance": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Инстанс оркестрации",
				Description:         "Инстанс оркестрации",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"count_of_executors": schema.Int64Attribute{
				Description:         "Количество экзекьютеров",
				MarkdownDescription: "Количество экзекьютеров",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r AgentOrchestrationResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r AgentOrchestrationResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan AgentOrchestrationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	finProj, err := sources.GetFinancialProjectByName(
		r.provider.Creds,
		r.provider.ProjectName,
		plan.FinancialProject.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("financial_project"), CREATE_RES_FAIL, err.Error())
		return
	}

	attrs := r.prepareAttrs(plan)

	order := orders.NewAgentOrchestration(
		r.provider.Creds,
		r.provider.ProjectName,
		plan.Image.ProductID.ValueString(),
		attrs,
	)
	err = order.Create(orders.CreateOrderPayload{
		Label:        plan.Label.ValueString(),
		Lifetime:     int(plan.Lifetime.ValueInt64()),
		FinProjectID: finProj.ID,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			CREATE_RES_FAIL,
			fmt.Sprintf("Order creating ended with error: %s", err.Error()),
		)
		return
	}

	err = order.Sync()
	if err != nil {
		resp.Diagnostics.AddError(
			CREATE_RES_FAIL,
			fmt.Sprint(
				"Can't get order's information from order service: ", err.Error(),
			),
		)
		return
	}

	agentItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			CREATE_RES_FAIL,
			fmt.Sprintf("Error while getting parent order item, error: %s", err.Error()),
		)
		return
	}

	vmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			CREATE_RES_FAIL,
			fmt.Sprintf("Error while getting vmItems, error: %s", err.Error()),
		)
		return
	}
	vmItemData := vmItems[0].Data

	vmItemConfig, ok := vmItemData.Config.(entities.VMItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			CREATE_RES_FAIL,
			fmt.Sprintln("Invalid type for vmItemConfig"),
		)
		return
	}

	agentConfig, ok := agentItem.Data.Config.(entities.AgentOrchestrationItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			CREATE_RES_FAIL,
			fmt.Sprintln("Invalid type for agentConfig"),
		)
		return
	}

	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(agentItem.ID)
	plan.Hostname = types.StringValue(vmItemConfig.Hostname)

	plan.AgentInstance = types.StringValue(agentConfig.AgentInstance)
	plan.AgentPool = types.StringValue(agentConfig.AgentPool)
	plan.AgentVersion = types.StringValue(agentConfig.Version)
	plan.ChannelURL = types.StringValue(agentConfig.ChannelURL)
	plan.CountOfExecutors = types.Int64Value(agentConfig.CountOfExecutors)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r AgentOrchestrationResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var orderID types.String
	req.State.GetAttribute(ctx, path.Root("order_id"), &orderID)
	if orderID.IsNull() {
		resp.Diagnostics.AddAttributeError(
			path.Root("order_id"),
			READ_RES_FAIL,
			"Resource hasn't order_id. Without order_id can't get order from portal",
		)
		return
	}

	order, err := orders.GetAgentOrchestrationOrder(
		r.provider.Creds,
		r.provider.ProjectName,
		orderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("order_id"),
			fmt.Sprintln(READ_RES_FAIL, "fetch order from portal API ended with error"),
			err.Error(),
		)
		return
	}

	agentItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			READ_RES_FAIL,
			fmt.Sprintf("Can't get agent item.\nError: %s", err.Error()),
		)
		return
	}

	vmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			READ_RES_FAIL,
			fmt.Sprintf("Can't get VMs items.\nError: %s", err.Error()),
		)
		return
	}
	vmItem := vmItems[0]
	vmConfig, ok := vmItem.Data.Config.(entities.VMItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			READ_RES_FAIL,
			"Invalid type for vmConfig",
		)
		return
	}
	agentConfig, ok := agentItem.Data.Config.(entities.AgentOrchestrationItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			READ_RES_FAIL,
			"Invalid type for agentConfig",
		)
		return
	}

	appExtraMount, err := order.GetExtraMount("/app")
	if err != nil {
		resp.Diagnostics.AddError(
			READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get vm extra mount '/app' from portal with order_id '%s'.\nError:%s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}

	state := AgentOrchestrationResourceModel{
		OrderID:  orderID,
		Label:    types.StringValue(order.Label),
		ItemID:   types.StringValue(vmItem.ID),
		Hostname: types.StringValue(vmConfig.Hostname),
		Flavor: FlavorModel{
			Cores:  types.Int64Value(vmConfig.Flavor.Cores),
			Memory: types.Int64Value(vmConfig.Flavor.Memory),
			UUID:   types.StringValue(vmConfig.Flavor.UUID),
			Name:   types.StringValue(vmConfig.Flavor.Name),
		},
		ExtraMounts: map[string]ExtraMountModel{
			"/app": {
				Size:       types.Int64Value(appExtraMount.Size),
				FileSystem: types.StringValue(appExtraMount.FileSystem),
			},
		},
		Image: ImageStandardModel{
			Distribution:  types.StringValue(vmConfig.Image.Os.Distribution),
			OsVersion:     types.StringValue(vmConfig.Image.Os.Version),
			OnSupport:     types.BoolValue(order.Attrs.OnSupport),
			ProductID:     types.StringValue(order.ProductID),
			ADIntegration: types.BoolValue(order.Attrs.ADIntegration),
		},
		Core: CoreModel{
			Platform:       types.StringValue(selectPlatform(order.Attrs.Platform)),
			Domain:         types.StringValue(order.Attrs.Domain),
			NetSegmentCode: types.StringValue(vmConfig.DefaultNic.NetSegment),
			Zone:           types.StringValue(order.Attrs.AvailabilityZone),
		},
		FinancialProject: types.StringValue(order.FinancialSource.Name),
		SferaAgent: SferaAgentModel{
			JenkinsAgentExecutors: types.Int64Value(order.Attrs.SferaAgent.JenkinsAgentExecutors),
			JenkinsAgentSubsystem: JenkinsAgentSubsystemModel{
				RisID:         types.StringValue(order.Attrs.SferaAgent.JenkinsAgentSubsystem.RisID.Value),
				IsCode:        types.StringValue(order.Attrs.SferaAgent.JenkinsAgentSubsystem.IsCode.Value),
				HeadName:      types.StringValue(order.Attrs.SferaAgent.JenkinsAgentSubsystem.HeadName.Value),
				NetSegment:    types.StringValue(order.Attrs.SferaAgent.JenkinsAgentSubsystem.NetSegment.Value),
				DisplayName:   types.StringValue(order.Attrs.SferaAgent.JenkinsAgentSubsystem.DisplayName.Value),
				SferaHeadURL:  types.StringValue(order.Attrs.SferaAgent.JenkinsAgentSubsystem.SferaHeadURL.Value),
				NodeGroupName: types.StringValue(order.Attrs.SferaAgent.JenkinsAgentSubsystem.NodeGroupName.Value),
			},
		},
	}
	state.AgentInstance = types.StringValue(agentConfig.AgentInstance)
	state.AgentPool = types.StringValue(agentConfig.AgentPool)
	state.AgentVersion = types.StringValue(agentConfig.Version)
	state.ChannelURL = types.StringValue(agentConfig.ChannelURL)
	state.CountOfExecutors = types.Int64Value(agentConfig.CountOfExecutors)

	var lifetime types.Int64
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("lifetime"), &lifetime)...)
	if !lifetime.IsNull() {
		state.Lifetime = lifetime
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r AgentOrchestrationResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state AgentOrchestrationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetAgentOrchestrationOrder(
		r.provider.Creds,
		r.provider.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			UPDATE_RES_FAIL,
			fmt.Sprint("Can't get order from portal: ", err.Error()),
		)
		return
	}

	finProj, err := sources.GetFinancialProjectByName(
		r.provider.Creds,
		r.provider.ProjectName,
		plan.FinancialProject.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("financial_project"), UPDATE_RES_FAIL, err.Error())
		return
	}

	labelChanged := plan.Label != state.Label
	mountChanged := isExtraMountChanged(state.ExtraMounts, plan.ExtraMounts)
	flavorChanged := plan.Flavor != state.Flavor
	financialProjectChanged := plan.FinancialProject.Equal(state.FinancialProject)

	// change label
	if labelChanged {
		changeOrderLabel(order, plan.Label.ValueString(), resp)
	}

	// changeFinancialProject
	if financialProjectChanged {
		err := order.ChangeFinancialProject(finProj.ID)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("financial_project"),
				UPDATE_RES_FAIL,
				fmt.Sprintf("Change finaciial project ended wiht error: %v", err.Error()),
			)
			return
		}
	}

	// changeMountPoint
	if mountChanged {
		resp.Diagnostics.Append(changeExtraMountsAgentOrchestration(order, &plan)...)
	}

	// changeFlavor
	if flavorChanged {
		resp.Diagnostics.Append(changeFlavorAgentOrchestration(order, &plan)...)
	}

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r AgentOrchestrationResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state AgentOrchestrationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetAgentOrchestrationOrder(
		r.provider.Creds,
		r.provider.ProjectName,
		state.OrderID.ValueString(),
	)

	if err != nil {
		resp.Diagnostics.AddError(DELETE_RES_FAIL, err.Error())
		return
	}

	orderState, err := order.GetState()
	if err != nil {
		resp.Diagnostics.AddError(DELETE_RES_FAIL, err.Error())
		return
	}

	if orderState == "deleted" || orderState == "deprovisioned" {
		resp.Diagnostics.AddWarning(
			DELETE_RES_FAIL,
			fmt.Sprintf(
				"Order with id '%s' was deleted by another way. Not from terraform. "+
					"Resource was recreated",
				state.OrderID.ValueString(),
			),
		)
		return
	}

	if !order.Deletable {
		orderURL := fmt.Sprintf(
			"https://cloud.vtb.ru/all/orders/%s/main?context=%s&type=project&org=%s",
			order.ID,
			order.ProjectName,
			r.provider.Organization,
		)

		resp.Diagnostics.AddError(
			"Deletable error",
			fmt.Sprintf(
				"Order with order_id '%s' isn't deletable. "+
					"Please, make your order is deletable by the portal VTB.Cloud, and repeat again."+
					"\nURL reference: %s",
				order.ID, orderURL,
			),
		)
		return
	}

	err = order.AgentDeleteTwoLayer()
	if err != nil {
		resp.Diagnostics.AddError(DELETE_RES_FAIL, err.Error())
		return
	}
}

func changeFlavorAgentOrchestration(
	order *orders.AgentOrchestration,
	plan *AgentOrchestrationResourceModel,
) (diags diag.Diagnostics) {

	flavor := entities.Flavor{
		Cores:  plan.Flavor.Cores.ValueInt64(),
		Memory: plan.Flavor.Memory.ValueInt64(),
		UUID:   plan.Flavor.UUID.ValueString(),
		Name:   plan.Flavor.Name.ValueString(),
	}

	err := order.ChangeFlavor(flavor)
	if err != nil {
		diags.AddError(
			UPDATE_RES_FAIL,
			fmt.Sprintf("Change flavor ended with error: %v", err.Error()),
		)
	}
	return
}

func changeExtraMountsAgentOrchestration(
	order *orders.AgentOrchestration,
	plan *AgentOrchestrationResourceModel,
) (diags diag.Diagnostics) {

	for path, planExtraMount := range plan.ExtraMounts {
		err := order.ExpandMountPoint(entities.ExtraMount{
			Path:       path,
			Size:       planExtraMount.Size.ValueInt64(),
			FileSystem: planExtraMount.FileSystem.ValueString(),
		})
		if err != nil {
			diags.AddError(
				UPDATE_RES_FAIL,
				fmt.Sprintf("Change extra_mounts ended wiht error: %v", err.Error()),
			)
		}
	}
	return
}

func (r AgentOrchestrationResource) prepareAttrs(
	plan AgentOrchestrationResourceModel,
) orders.AgentOrchestrationAttrs {

	jenkinsAgentSubsystem := entities.JenkinsAgentSubsystem{
		RisID: entities.JenkinsAgentSubsystemValue{
			Value: plan.SferaAgent.JenkinsAgentSubsystem.RisID.ValueString(),
		},
		IsCode: entities.JenkinsAgentSubsystemValue{
			Value: plan.SferaAgent.JenkinsAgentSubsystem.IsCode.ValueString(),
		},
		HeadName: entities.JenkinsAgentSubsystemValue{
			Value: plan.SferaAgent.JenkinsAgentSubsystem.HeadName.ValueString(),
		},
		NetSegment: entities.JenkinsAgentSubsystemValue{
			Value: plan.SferaAgent.JenkinsAgentSubsystem.NetSegment.ValueString(),
		},
		DisplayName: entities.JenkinsAgentSubsystemValue{
			Value: plan.SferaAgent.JenkinsAgentSubsystem.DisplayName.ValueString(),
		},
		SferaHeadURL: entities.JenkinsAgentSubsystemValue{
			Value: plan.SferaAgent.JenkinsAgentSubsystem.SferaHeadURL.ValueString(),
		},
		NodeGroupName: entities.JenkinsAgentSubsystemValue{
			Value: plan.SferaAgent.JenkinsAgentSubsystem.NodeGroupName.ValueString(),
		},
	}
	sferaAgentAttrs := orders.SferaAgentAttrs{
		JenkinsAgentInstance:  plan.SferaAgent.JenkinsAgentSubsystem.HeadName.ValueString(),
		JenkinsAgentExecutors: plan.SferaAgent.JenkinsAgentExecutors.ValueInt64(),
		JenkinsAgentSubsystem: jenkinsAgentSubsystem,
	}

	return orders.AgentOrchestrationAttrs{
		ADIntegration:    plan.Image.ADIntegration.ValueBool(),
		Platform:         plan.Core.Platform.ValueString(),
		AvailabilityZone: plan.Core.Zone.ValueString(),
		Domain:           plan.Core.Domain.ValueString(),
		OnSupport:        plan.Image.OnSupport.ValueBool(),
		OsVersion:        plan.Image.OsVersion.ValueString(),
		ExtraMounts:      prepareExtraMountsAttrs(plan.ExtraMounts),
		DefaultNic: entities.DefaultNic{
			NetSegment: plan.Core.NetSegmentCode.ValueString(),
		},
		Flavor: entities.Flavor{
			Cores:  plan.Flavor.Cores.ValueInt64(),
			Memory: plan.Flavor.Memory.ValueInt64(),
			Name:   string(plan.Flavor.Name.ValueString()),
			UUID:   string(plan.Flavor.UUID.ValueString()),
		},
		SferaAgent:          sferaAgentAttrs,
		CreatedWithOpenTofu: true,
	}
}
