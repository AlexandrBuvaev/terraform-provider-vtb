package syncxpert

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/consts"
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
	_ resource.Resource               = &SyncXpertConnectorResource{}
	_ resource.ResourceWithModifyPlan = &SyncXpertConnectorResource{}
)

type SyncXpertConnectorResource struct {
	client *client.CloudClient
}

func NewSyncXpertConnectorResource() resource.Resource {
	return &SyncXpertConnectorResource{}
}

func (r *SyncXpertConnectorResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_sync_xpert_connector"
}

func (r *SyncXpertConnectorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	DbTopicPrefix   types.String `tfsdk:"db_topic_prefix"`
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

func (r *SyncXpertConnectorResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{

			"order_id": schema.StringAttribute{
				Required:            true,
				Description:         "Идентификатор заказа кластера SyncXpert",
				MarkdownDescription: "Идентификатор заказа кластера SyncXpert",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"name": schema.StringAttribute{
				Required:            true,
				Description:         "Удобочитаемое название коннектора",
				MarkdownDescription: "Удобочитаемое название коннектора",
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
						MarkdownDescription: "Список (через запятую) кластера серверов PostgreSQL в формате `адрес:порт`",
						Description:         "Список (через запятую) кластера серверов PostgreSQL в формате `адрес:порт`",
					},

					"name": schema.StringAttribute{
						Required:            true,
						Description:         "Имя базы данных, из которой будут перенесены изменения данных",
						MarkdownDescription: "Имя базы данных, из которой будут перенесены изменения данных",
						Validators: []validator.String{
							stringvalidator.RegexMatches(
								regexp.MustCompile(`^[A-Za-z0-9_-]*$`),
								"may contain uppercase/lowercase Latin letters, numbers, "+
									"punctuation marks ['.' '-' '_']. (cannot start or end with punctuation)",
							),
						},
					},
					"db_topic_prefix": schema.StringAttribute{
						Required:            true,
						Description:         "Префикс топика базы данных",
						MarkdownDescription: "Префикс топика базы данных",
						Validators: []validator.String{
							stringvalidator.RegexMatches(
								regexp.MustCompile(`^[a-z0-9][a-z0-9\\-]*[a-z0-9]$`),
								"must match regex: ^[a-z0-9][a-z0-9-]*[a-z0-9]$ (lowercase alphanumeric, hyphens allowed but not at start/end)",
							),
						},
					},

					"user": schema.StringAttribute{
						Required:            true,
						Description:         "Имя пользователя PostgreSQL для подключения к базе данных",
						MarkdownDescription: "Имя пользователя PostgreSQL для подключения к базе данных",
						Validators: []validator.String{
							stringvalidator.OneOf("debezium", "connector"),
						},
					},

					"password": schema.StringAttribute{
						Required:            true,
						Sensitive:           true,
						Description:         "Пароль пользователя PostgreSQL для подключения к базе данных",
						MarkdownDescription: "Пароль пользователя PostgreSQL для подключения к базе данных",
					},

					"slot_name": schema.StringAttribute{
						Required: true,
						Description: "Имя логического слота репликации PostgreSQL " +
							"который передает изменения данных в Debezium connector",
						MarkdownDescription: "Имя логического слота репликации PostgreSQL " +
							"который передает изменения данных в Debezium connector",
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
						Description: "Название публикации PostgreSQL " +
							"созданной для потоковой передачи изменений при использовании плагина pg output",
						MarkdownDescription: "Название публикации PostgreSQL " +
							"созданной для потоковой передачи изменений при использовании плагина pg output",
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
						Description: "Список включенный в отслеживание - на выбор доступны 2 типа: " +
							"`schema.include.list` и `table.include.list`",
						MarkdownDescription: "Список включенный в отслеживание - на выбор доступны 2 типа: " +
							"`schema.include.list` и `table.include.list`",
						Validators: []validator.String{
							stringvalidator.OneOf(
								"schema.include.list",
								"table.include.list",
							),
						},
					},

					"include_list": schema.StringAttribute{
						Required:            true,
						Description:         "Список (через запятую) регулярных выражений",
						MarkdownDescription: "Список (через запятую) регулярных выражений",
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
						Description:         "Следует ли использовать зашифрованное соединение с PostgreSQL или нет",
						MarkdownDescription: "Следует ли использовать зашифрованное соединение с PostgreSQL или нет",
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
						Description:         "Полный путь к клиентскому сертификату",
						MarkdownDescription: "Полный путь к клиентскому сертификату",
					},

					"root_cert": schema.StringAttribute{
						Optional:            true,
						Description:         "Сертификат доверенного центра сертификации",
						MarkdownDescription: "Сертификат доверенного центра сертификации",
					},

					"key": schema.StringAttribute{
						Optional:            true,
						Description:         "Полный путь к файлу ключа сертификата клиента",
						MarkdownDescription: "Полный путь к файлу ключа сертификата клиента",
					},

					"password": schema.StringAttribute{
						Optional:            true,
						Description:         "Пароль сертификата клиента",
						MarkdownDescription: "Пароль сертификата клиента",
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
						Description: "Частота операций Heartbeat. " +
							"По умолчанию: 60000",
						MarkdownDescription: "Частота операций Heartbeat " +
							"По умолчанию: 60000",
						Validators: []validator.Int64{
							int64validator.AtLeast(0),
						},
					},

					"action_query": schema.StringAttribute{
						Optional: true,
						Computed: true,
						Default:  stringdefault.StaticString("insert into hb_debezium.hb_table (id) values (1)"),
						Description: "Запрос, который выполняется для heartbeat. " +
							"По умолчанию: insert into hb_debezium.hb_table (id) values (1)",
						MarkdownDescription: "Request that is made for heartbeat. " +
							"По умолчанию: insert into hb_debezium.hb_table (id) values (1)",
					},

					"topic_prefix": schema.StringAttribute{
						Optional: true,
						Computed: true,
						Default:  stringdefault.StaticString("debezium-heartbeat"),
						Description: "Окончательное название топика будет <heartbeat.topics.prefix>.<topic.prefix>. " +
							"По умолчанию: debezium-heartbeat",
						MarkdownDescription: "Окончательное название топика будет <heartbeat.topics.prefix>.<topic.prefix>. " +
							"По умолчанию: debezium-heartbeat",
					},
				},
			},
		},
	}
}

func (r *SyncXpertConnectorResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var connectorName types.String
	req.Plan.GetAttribute(ctx, path.Root("name"), &connectorName)

	var connectedDbName types.String
	req.Plan.GetAttribute(ctx, path.Root("database").AtName("name"), &connectedDbName)
	pgPrefix := fmt.Sprintf("pg-%s-%s", r.client.RisShortName, r.client.EnvPrefix)

	isConnectorNameInvalid := !strings.HasPrefix(connectorName.ValueString(), pgPrefix)

	if isConnectorNameInvalid {
		resp.Diagnostics.AddAttributeError(
			path.Root("name"),
			consts.VALIDATION_FAIL,
			fmt.Sprintf("name of the connector must start with %s", pgPrefix),
		)
	}
}

func (r *SyncXpertConnectorResource) Create(
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
		r.client.Creds,
		r.client.ProjectName,
		plan.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("debezium_order_id"),
			fmt.Sprintln(consts.UPDATE_RES_FAIL, "fetch order from portal API ended with error"),
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
		DbTopicPref:       plan.Database.DbTopicPrefix.ValueString(),

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
		resp.Diagnostics.AddError(consts.CREATE_RES_FAIL, err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SyncXpertConnectorResource) Read(
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

func (r *SyncXpertConnectorResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	resp.Diagnostics.AddWarning(
		"Invalid resource attributes changes: ",
		"vtb_sync_xpert_connector do not support partial update. Resource must be recreated",
	)
}

func (r *SyncXpertConnectorResource) Delete(
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
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("debezium_order_id"),
			fmt.Sprintln(consts.DELETE_RES_FAIL, "fetch order from portal API ended with error"),
			err.Error(),
		)
		return
	}

	err = order.DeleteConnector(state.ConnectorName.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(consts.DELETE_RES_FAIL, "debezium_delete_connector ended with error"),
			err.Error(),
		)
	}
}
