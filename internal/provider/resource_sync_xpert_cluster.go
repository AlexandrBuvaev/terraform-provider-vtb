package provider

import (
	"context"
	"fmt"
	"regexp"

	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"
	"terraform-provider-vtb/pkg/client/sources"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource = &SyncXpertClusterResource{}
)

type SyncXpertClusterResource struct {
	provider *VTBCloudProvider
}

func NewSyncXpertClusterResource(p *VTBCloudProvider) SyncXpertClusterResource {
	return SyncXpertClusterResource{
		provider: p,
	}
}

func (r SyncXpertClusterResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = "vtb_sync_xpert_cluster"
}

type SyncXpertClusterResourceModel struct {
	Core        CoreModel                          `tfsdk:"core"`
	Flavor      FlavorModel                        `tfsdk:"flavor"`
	Image       ClusterProductImageDataSourceModel `tfsdk:"image"`
	Lifetime    types.Int64                        `tfsdk:"lifetime"`
	Label       types.String                       `tfsdk:"label"`
	OrderID     types.String                       `tfsdk:"order_id"`
	ItemID      types.String                       `tfsdk:"item_id"`
	Access      map[string][]types.String          `tfsdk:"access"`
	ExtraMounts map[string]ExtraMountModel         `tfsdk:"extra_mounts"`

	APIUser            types.String `tfsdk:"api_user"`
	APIPassword        types.String `tfsdk:"api_password"`
	LayoutID           types.String `tfsdk:"layout_id"`
	DebeziumVersion    types.String `tfsdk:"debezium_version"`
	KafkaCertName      types.String `tfsdk:"kafka_cert_cname"`
	KafkaServer        types.String `tfsdk:"kafka_server"`
	ClusterGroupID     types.String `tfsdk:"cluster_group_id"`
	ConfigStorageTopic types.String `tfsdk:"config_storage_topic"`
	OffsetStorageTopic types.String `tfsdk:"offset_storage_topic"`
	StatusStorageTopic types.String `tfsdk:"status_storage_topic"`
	FinancialProject   types.String `tfsdk:"financial_project"`
}

func (r SyncXpertClusterResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"core": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          coreSchema,
				MarkdownDescription: "Core parameters for VM and order",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"flavor": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          flavorSchema,
				MarkdownDescription: "Core/memory of compute instance",
			},
			"image": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          ClusterProductImageSchema,
				MarkdownDescription: "Image data from reference_service",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
			"lifetime": schema.Int64Attribute{
				Optional:            true,
				Description:         "Order lifetime in days (2, 7, 14, 30)",
				MarkdownDescription: "Order lifetime in days (2, 7, 14, 30)",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
					int64planmodifier.RequiresReplace(),
				},
			},
			"label": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Label of order",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"order_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identifiator of Order. Known after creating order",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					CheckIsDeleted(r.provider, "vm", "openstack"),
				},
			},
			"item_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Item identificator of VM in order. Known after vm will created",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"access": schema.MapAttribute{
				Required: true,
				Description: "Map, where key is role and value is list of groups, " +
					"which will grant access for Active Directory login",
				MarkdownDescription: "Map, where key is role and value is list of groups, " +
					"which will grant access for Active Directory login",
				Validators: []validator.Map{
					mapvalidator.SizeAtLeast(1),
				},
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
			},
			"extra_mounts": schema.MapNestedAttribute{
				Required:            true,
				MarkdownDescription: "Added extra mounts in compute instance",
				Validators: []validator.Map{
					mapvalidator.KeysAre(stringvalidator.OneOf("/app")),
					mapvalidator.SizeBetween(1, 1),
				},
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"size": schema.Int64Attribute{
							Required:            true,
							MarkdownDescription: "Mount point size (specify at GB)",
							Validators: []validator.Int64{
								int64validator.AtMost(CLOUD_EXTRA_MOUNT_MAX_SIZE),
								int64validator.AtLeast(30),
							},
							PlanModifiers: []planmodifier.Int64{
								GrowOnly(),
								MininalGrow(10),
							},
						},

						"file_system": schema.StringAttribute{
							Computed:            true,
							Default:             stringdefault.StaticString("xfs"),
							Description:         "Name of file system type.",
							MarkdownDescription: "Name of file system type.",
						},
					},
				},
			},

			"layout_id": schema.StringAttribute{
				Required:            true,
				Description:         "Layout ID from geo_distribution reference.",
				MarkdownDescription: "Layout ID from geo_distribution reference.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"debezium_version": schema.StringAttribute{
				Required:            true,
				Description:         "Version of debezium application on cluster",
				MarkdownDescription: "Version of debezium application on cluster",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("1.1.0"),
				},
			},
			"api_user": schema.StringAttribute{
				Required:            true,
				Description:         "Username for REST API if necessary manually access the REST service.",
				MarkdownDescription: "Username for REST API if necessary manually access the REST service.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"api_password": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				Description:         "Password for REST API user if necessary manually access the REST service.",
				MarkdownDescription: "Password for REST API user if necessary manually access the REST service.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},

			"kafka_cert_cname": schema.StringAttribute{
				Required: true,
				Description: "Common Name (CN) with which the client certificate will" +
					"be issued to connect to the Kafka cluster",
				MarkdownDescription: "Common Name (CN) with which the client certificate will" +
					"be issued to connect to the Kafka cluster",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"kafka_server": schema.StringAttribute{
				Required:            true,
				Description:         "Servers in the format server:port, separated by commas (bootstrap_servers)",
				MarkdownDescription: "Servers in the format server:port, separated by commas (bootstrap_servers)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"cluster_group_id": schema.StringAttribute{
				Required: true,
				Description: "Group for the unique identification of the Debezium cluster, " +
					"used when working with system topics",
				MarkdownDescription: "Group for the unique identification of the Debezium cluster, " +
					"used when working with system topics",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"offset_storage_topic": schema.StringAttribute{
				Required:            true,
				Description:         "A Kafka topic for storing information about a fixed offset of replicated data.",
				MarkdownDescription: "A Kafka topic for storing information about a fixed offset of replicated data.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile("^dbzm-offset-"),
						"must start with dbzm-offset-",
					),
				},
			},
			"config_storage_topic": schema.StringAttribute{
				Required:            true,
				Description:         "A Kafka topic for storing the configuration of Debezium connectors.",
				MarkdownDescription: "A Kafka topic for storing the configuration of Debezium connectors.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile("^dbzm-config-"),
						"must start with dbzm-config-",
					),
				},
			},
			"status_storage_topic": schema.StringAttribute{
				Required: true,
				Description: "A Kafka topic, which contains information, " +
					"and on which node of the cluster Debezium is running a specific connector.",
				MarkdownDescription: "A Kafka topic, which contains information, " +
					"and on which node of the cluster Debezium is running a specific connector.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile("^dbzm-status-"),
						"must start with dbzm-status-",
					),
				},
			},
			"financial_project": schema.StringAttribute{
				Required:            true,
				Description:         "Financial source for order.",
				MarkdownDescription: "Financial source for order.",
			},
		},
	}
}

func (r SyncXpertClusterResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan SyncXpertClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	clusterConfig := orders.SyncXpertClusterConfig{
		BasicAuthUser:      plan.APIUser.ValueString(),
		BasicAuthPassword:  plan.APIPassword.ValueString(),
		DebeziumVersion:    plan.DebeziumVersion.ValueString(),
		KafkaCertCname:     plan.KafkaCertName.ValueString(),
		BootstrapServers:   plan.KafkaServer.ValueString(),
		ClusterGroupID:     plan.ClusterGroupID.ValueString(),
		ConfigStorageTopic: plan.ConfigStorageTopic.ValueString(),
		OffsetStorageTopic: plan.OffsetStorageTopic.ValueString(),
		StatusStorageTopic: plan.StatusStorageTopic.ValueString(),
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

	attrs := orders.SyncXpertAttrs{
		BasicAttrs: prepareBasicAttrs(
			&plan.Flavor,
			&plan.Core,
			plan.Access,
			plan.ExtraMounts,
			plan.Image.OsVersion.ValueString(),
			plan.Image.ADIntegration.ValueBool(),
			plan.Image.OnSupport.ValueBool(),
		),
		ExtraNics:       []interface{}{},
		Layout:          plan.LayoutID.ValueString(),
		GeoDistribution: plan.Image.GeoDistribution.ValueBool(),
		DebeziumConfig:  clusterConfig,
	}

	order := orders.NewSyncXpertCluster(
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
		resp.Diagnostics.AddError(CREATE_RES_FAIL, err.Error())
		return
	}

	orderItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError("Error while getting item", err.Error())
		return
	}

	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(orderItem.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r SyncXpertClusterResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state SyncXpertClusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r SyncXpertClusterResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state SyncXpertClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetDebeziumOrder(
		r.provider.Creds,
		r.provider.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(READ_RES_FAIL, err.Error())
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
	flavorChanged := plan.Flavor != state.Flavor

	if labelChanged {
		changeOrderLabel(order, plan.Label.ValueString(), resp)
	}

	if flavorChanged {
		r.verticalScaling(order, &plan, resp)
	}

	if !plan.FinancialProject.Equal(state.FinancialProject) {
		order.ChangeFinancialProject(finProj.ID)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r SyncXpertClusterResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state SyncXpertClusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetDebeziumOrder(
		r.provider.Creds,
		r.provider.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Can't get order from portal: ", err.Error())
		return
	}

	orderState, err := order.GetState()
	if err != nil {
		resp.Diagnostics.AddError(
			"Delete resource",
			fmt.Sprintf("Can't get state from portal.\nError:%s", err.Error()),
		)
		return
	}

	if orderState == "deleted" || orderState == "deprovisioned" {
		resp.Diagnostics.AddWarning(
			"Delete resource",
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

	err = order.DeleteTwoLayer(true)
	if err != nil {
		resp.Diagnostics.AddError(DELETE_RES_FAIL, err.Error())
		return
	}
}

func (r SyncXpertClusterResource) verticalScaling(
	order *orders.SyncXpertCluster,
	plan *SyncXpertClusterResourceModel,
	resp *resource.UpdateResponse,
) {
	flavor := entities.Flavor{
		Cores:  plan.Flavor.Cores.ValueInt64(),
		Memory: plan.Flavor.Memory.ValueInt64(),
		Name:   plan.Flavor.Name.ValueString(),
		UUID:   plan.Flavor.UUID.ValueString(),
	}
	err := order.VerticalScaling(flavor, false)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("flavor"),
			fmt.Sprintln(UPDATE_RES_FAIL, "debezium_vertical_scaling ended with error"),
			err.Error(),
		)
		return
	}
}
