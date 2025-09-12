package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"terraform-provider-vtb/pkg/client/orders"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &SyncXpertConnectorResource{}
	_ resource.ResourceWithValidateConfig = &SyncXpertConnectorResource{}
)

type SyncXpertConnectorResource struct {
	provider *VTBCloudProvider
}

func NewSyncXpertConnectorResource(p *VTBCloudProvider) SyncXpertConnectorResource {
	return SyncXpertConnectorResource{
		provider: p,
	}
}

func (r SyncXpertConnectorResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = "vtb_sync_xpert_connector"
}

type SyncXpertConnectorResourceModel struct {
	OrderID         types.String          `tfsdk:"order_id"`
	ConnectorName   types.String          `tfsdk:"name"`
	Database        ConnectedDBModel      `tfsdk:"database"`
	SSLConfig       SSLConfgiModel        `tfsdk:"ssl"`
	HeartbeatConfig *HeartbeatConfigModel `tfsdk:"heartbeat"`
}

type ConnectedDBModel struct {
	Name            types.String `tfsdk:"name"`
	User            types.String `tfsdk:"user"`
	Password        types.String `tfsdk:"password"`
	Hostname        types.String `tfsdk:"hostname"`
	IncludeListType types.String `tfsdk:"include_list_type"`
	IncludeList     types.String `tfsdk:"include_list"`
	SlotName        types.String `tfsdk:"slot_name"`
	PublicationName types.String `tfsdk:"publication_name"`
}

type SSLConfgiModel struct {
	Mode     types.String `tfsdk:"mode"`
	Password types.String `tfsdk:"password"`
	Cert     types.String `tfsdk:"cert"`
	RootCert types.String `tfsdk:"root_cert"`
	Key      types.String `tfsdk:"key"`
}

type HeartbeatConfigModel struct {
	Query       types.String `tfsdk:"action_query"`
	TopicPrefix types.String `tfsdk:"topic_prefix"`
	IntervalMS  types.Int64  `tfsdk:"interval_ms"`
}

func (r SyncXpertConnectorResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{

			"order_id": schema.StringAttribute{
				Required:            true,
				Description:         "The ID of SyncXpert cluster order.",
				MarkdownDescription: "The ID of SyncXpert cluster order.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"name": schema.StringAttribute{
				Required:            true,
				Description:         "The human-readable name of the connector",
				MarkdownDescription: "The human-readable name of the connector",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile(`^[A-Za-z0-9_-]*$`),
						"may contain uppercase/lowercase Latin letters, numbers, "+
							"punctuation marks ['.' '-' '_']. (cannot start or end with punctuation)",
					),
				},
			},

			"database": schema.SingleNestedAttribute{
				Required: true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"hostname": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "List (via commas) of the PostgreSQL server cluster in the format `address:port`",
						Description:         "List (via commas) of the PostgreSQL server cluster in the format `address:port`",
					},

					"name": schema.StringAttribute{
						Required:            true,
						Description:         "The name of the database from which the data changes will be transferred",
						MarkdownDescription: "The name of the database from which the data changes will be transferred",
						Validators: []validator.String{
							stringvalidator.RegexMatches(
								regexp.MustCompile(`^[A-Za-z0-9_-]*$`),
								"may contain uppercase/lowercase Latin letters, numbers, "+
									"punctuation marks ['.' '-' '_']. (cannot start or end with punctuation)",
							),
						},
					},

					"user": schema.StringAttribute{
						Required:            true,
						Description:         "PostgreSQL database user name to connect to the database",
						MarkdownDescription: "PostgreSQL database user name to connect to the database",
						Validators: []validator.String{
							stringvalidator.OneOf("debezium"),
						},
					},

					"password": schema.StringAttribute{
						Required:            true,
						Description:         "PostgreSQL database user password to connect to the database",
						MarkdownDescription: "PostgreSQL database user password to connect to the database",
					},

					"slot_name": schema.StringAttribute{
						Required: true,
						Description: "The name of the PostgreSQL logical replication slot " +
							"that streams data changes to the Debezium connector.",
						MarkdownDescription: "The name of the PostgreSQL logical replication slot " +
							"that streams data changes to the Debezium connector.",
						Validators: []validator.String{
							stringvalidator.RegexMatches(
								regexp.MustCompile(`^[A-Za-z0-9_-]*$`),
								"may contain uppercase/lowercase Latin letters, numbers, "+
									"punctuation marks ['.' '-' '_']. (cannot start or end with punctuation)",
							),
						},
					},

					"publication_name": schema.StringAttribute{
						Required: true,
						Description: "The name of the PostgreSQL publication " +
							"created to stream changes when using the pgoutput plugin.",
						MarkdownDescription: "The name of the PostgreSQL publication " +
							"created to stream changes when using the pgoutput plugin.",
						Validators: []validator.String{
							stringvalidator.RegexMatches(
								regexp.MustCompile(`^[A-Za-z0-9_-]*$`),
								"may contain uppercase/lowercase Latin letters, numbers, "+
									"punctuation marks ['.' '-' '_']. (cannot start or end with punctuation)",
							),
						},
					},

					"include_list_type": schema.StringAttribute{
						Required: true,
						Description: "List included in tracking - there are 2 types available to choose from: " +
							"`schema.include.list` and `table.include.list`",
						MarkdownDescription: "List included in tracking - there are 2 types available to choose from: " +
							"`schema.include.list` and `table.include.list`",
						Validators: []validator.String{
							stringvalidator.OneOf(
								"schema.include.list",
								"table.include.list",
							),
						},
					},

					"include_list": schema.StringAttribute{
						Required:            true,
						Description:         "List (via commas) of regular expressions.",
						MarkdownDescription: "List (via commas) of regular expressions.",
					},
				},
			},

			"ssl": schema.SingleNestedAttribute{
				Required: true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"mode": schema.StringAttribute{
						Required:            true,
						Description:         "Whether to use an encrypted connection with PostgreSQL or not.",
						MarkdownDescription: "Whether to use an encrypted connection with PostgreSQL or not.",
						Validators: []validator.String{
							stringvalidator.OneOf(
								"disable",
								"require",
								"verify-ca",
								"verify-full",
							),
						},
					},

					"cert": schema.StringAttribute{
						Optional:            true,
						Description:         "Full path to the client certificate.",
						MarkdownDescription: "Full path to the client certificate.",
					},

					"root_cert": schema.StringAttribute{
						Optional:            true,
						Description:         "Certificate of a trusted CA.",
						MarkdownDescription: "Certificate of a trusted CA.",
					},

					"key": schema.StringAttribute{
						Optional:            true,
						Description:         "Full path to the client certificate key file.",
						MarkdownDescription: "Full path to the client certificate key file.",
					},

					"password": schema.StringAttribute{
						Optional:            true,
						Description:         "Client certificate password.",
						MarkdownDescription: "Client certificate password.",
					},
				},
			},

			"heartbeat": schema.SingleNestedAttribute{
				Required: true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"interval_ms": schema.Int64Attribute{
						Optional: true,
						Computed: true,
						Default:  int64default.StaticInt64(60000),
						Description: "Heartbeat operation frequency. " +
							"Default to: 60000",
						MarkdownDescription: "Heartbeat operation frequency. " +
							"Default to: 60000",
						Validators: []validator.Int64{
							int64validator.AtLeast(0),
						},
					},

					"action_query": schema.StringAttribute{
						Optional: true,
						Computed: true,
						Default:  stringdefault.StaticString("insert into hb_debezium.hb_table (id) values (1)"),
						Description: "Request that is made for heartbeat. " +
							"Default to: insert into hb_debezium.hb_table (id) values (1)",
						MarkdownDescription: "Request that is made for heartbeat. " +
							"Default to: insert into hb_debezium.hb_table (id) values (1)",
					},

					"topic_prefix": schema.StringAttribute{
						Optional: true,
						Computed: true,
						Default:  stringdefault.StaticString("debezium-heartbeat"),
						Description: "The final topic name will be <heartbeat.topics.prefix>.<topic.prefix>. " +
							"Default to: debezium-heartbeat",
						MarkdownDescription: "The final topic name will be <heartbeat.topics.prefix>.<topic.prefix>. " +
							"Default to: debezium-heartbeat",
					},
				},
			},
		},
	}
}

func (r SyncXpertConnectorResource) ValidateConfig(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	var connectorName types.String
	req.Config.GetAttribute(ctx, path.Root("name"), &connectorName)

	var connectedDbName types.String
	req.Config.GetAttribute(ctx, path.Root("database").AtName("name"), &connectedDbName)

	isConnectorNameInvalid := !strings.HasPrefix(connectorName.ValueString(), "pg-soub-da")

	if isConnectorNameInvalid {
		resp.Diagnostics.AddAttributeError(
			path.Root("name"),
			VALIDATION_FAIL,
			fmt.Sprintf("name of the connector must start with %s", "pg-soub-da"),
		)
	}
}

func (r SyncXpertConnectorResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan SyncXpertConnectorResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetDebeziumOrder(
		r.provider.Creds,
		r.provider.ProjectName,
		plan.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("debezium_order_id"),
			fmt.Sprintln(UPDATE_RES_FAIL, "fetch order from portal API ended with error"),
			err.Error(),
		)
		return
	}

	connectorConfig := orders.SyncXpertDbConnectroConfig{
		ConnectorName:     plan.ConnectorName.ValueString(),
		DbHostname:        plan.Database.Hostname.ValueString(),
		DbUser:            plan.Database.User.ValueString(),
		DbName:            plan.Database.Name.ValueString(),
		DbPassword:        plan.Database.Password.ValueString(),
		DbSlotName:        plan.Database.SlotName.ValueString(),
		DbPublicationName: plan.Database.PublicationName.ValueString(),
		DbIncludeListType: plan.Database.IncludeListType.ValueString(),
		DbIncludeList:     plan.Database.IncludeList.ValueString(),

		SSLMode:     plan.SSLConfig.Mode.ValueString(),
		SSLCert:     plan.SSLConfig.Cert.ValueString(),
		SSLKey:      plan.SSLConfig.Key.ValueString(),
		SSLRootCert: plan.SSLConfig.RootCert.ValueString(),
		SSLPassword: plan.SSLConfig.Password.ValueString(),

		HeartbeatInterval: plan.HeartbeatConfig.IntervalMS.ValueInt64(),
		HeartbeatTopics:   plan.HeartbeatConfig.TopicPrefix.ValueString(),
		HeartbeatQuery:    plan.HeartbeatConfig.Query.ValueString(),
	}

	err = order.CreateConnector(connectorConfig)
	if err != nil {
		resp.Diagnostics.AddError(CREATE_RES_FAIL, err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r SyncXpertConnectorResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state SyncXpertConnectorResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r SyncXpertConnectorResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	resp.Diagnostics.AddWarning(
		"Invalid resource attributes changes: ",
		"vtb_sync_xpert_connector do not support partial update. Resource must be recreated",
	)
}

func (r SyncXpertConnectorResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state SyncXpertConnectorResourceModel
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
		resp.Diagnostics.AddAttributeError(
			path.Root("debezium_order_id"),
			fmt.Sprintln(DELETE_RES_FAIL, "fetch order from portal API ended with error"),
			err.Error(),
		)
		return
	}

	err = order.DeleteConnector(state.ConnectorName.ValueString(), false)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(DELETE_RES_FAIL, "debezium_delete_connector ended with error"),
			err.Error(),
		)
	}
}
