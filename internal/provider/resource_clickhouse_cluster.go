package provider

import (
	"context"
	"fmt"
	"reflect"
	"regexp"

	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"
	"terraform-provider-vtb/pkg/client/references"
	"terraform-provider-vtb/pkg/client/sources"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                = &ClickHouseClusterResource{}
	_ resource.ResourceWithImportState = &ClickHouseClusterResource{}
)

type ClickHouseClusterResource struct {
	provider *VTBCloudProvider
}

func NewClickHouseClusterResource(p *VTBCloudProvider) ClickHouseClusterResource {
	return ClickHouseClusterResource{
		provider: p,
	}
}

func (r ClickHouseClusterResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = "vtb_clickhouse_cluster"
}

type ClickHouseClusterResourceModel struct {
	Core     CoreModel                          `tfsdk:"core"`
	Image    ClusterProductImageDataSourceModel `tfsdk:"image"`
	Lifetime types.Int64                        `tfsdk:"lifetime"`
	Label    types.String                       `tfsdk:"label"`
	OrderID  types.String                       `tfsdk:"order_id"`
	ItemID   types.String                       `tfsdk:"item_id"`
	Access   map[string][]types.String          `tfsdk:"access"`

	FlavorCH                   FlavorModel                `tfsdk:"flavor_ch"`
	FlavorZK                   FlavorModel                `tfsdk:"flavor_zk"`
	ClickHouseVersion          types.String               `tfsdk:"ch_version"`
	ZookeeperVersion           types.String               `tfsdk:"zk_version"`
	ClusterName                types.String               `tfsdk:"cluster_name"`
	ClickHouseUser             types.String               `tfsdk:"ch_customer_admin"`
	ClickHousePassword         types.String               `tfsdk:"ch_customer_admin_password"`
	ChCustomerPassword         types.String               `tfsdk:"ch_customer_password"`
	SystemAdmGroups            map[string][]string        `tfsdk:"system_adm_groups"`
	ClickHouseAppAdminAdGroups map[string][]string        `tfsdk:"clickhouse_app_admin_ad_groups"`
	ClickHouseUserAdGroups     map[string][]string        `tfsdk:"clickhouse_user_ad_groups"`
	FinancialProject           types.String               `tfsdk:"financial_project"`
	NodesCount                 orders.NodesCount          `tfsdk:"nodes_count"`
	CHExtraMounts              map[string]ExtraMountModel `tfsdk:"ch_extra_mounts"`
	ZKExtraMounts              map[string]ExtraMountModel `tfsdk:"zk_extra_mounts"`
	Layout                     types.String               `tfsdk:"layout"`
}

func (r ClickHouseClusterResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			// basic schema
			"core": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          coreSchema,
				Description:         "Основные параметры для виртуальных машин в заказе.",
				MarkdownDescription: "Основные параметры для виртуальных машин в заказе.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"flavor_ch": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          flavorSchema,
				Description:         "Кол-во CPU/RAM для виртуальных машин.",
				MarkdownDescription: "Кол-во CPU/RAM для виртуальных машин.",
				PlanModifiers: []planmodifier.Object{
					AvailableFlavor(r.provider, "cluster:clickhouse"),
				},
			},
			"flavor_zk": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          flavorSchema,
				Description:         "Кол-во CPU/RAM для виртуальных машин.",
				MarkdownDescription: "Кол-во CPU/RAM для виртуальных машин.",
				PlanModifiers: []planmodifier.Object{
					AvailableFlavor(r.provider, "cluster:zookeeper"),
				},
			},
			"image": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          ClusterProductImageSchema,
				MarkdownDescription: "Type of compute instance",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
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
				Description:         "Метка заказа.",
				MarkdownDescription: "Метка заказа.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"order_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Идентификатор заказа. Становиться известен после создания заказа.",
				MarkdownDescription: "Идентификатор заказа. Становиться известен после создания заказа.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					CheckIsDeleted(r.provider, "cluster", "clickhouse"),
				},
			},
			"item_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Идентификатор сущностей ВМ, известен после создания ВМ.",
				MarkdownDescription: "Идентификатор сущностей ВМ, известен после создания ВМ.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"ch_extra_mounts": schema.MapNestedAttribute{
				Required:            true,
				Description:         "Дополнительные точки монтирования ВМ",
				MarkdownDescription: "Дополнительные точки монтирования ВМ",
				Validators: []validator.Map{
					mapvalidator.KeysAre(
						stringvalidator.OneOf(
							"/app/clickhouse",
						),
					),
					mapvalidator.SizeBetween(1, 4),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"size": schema.Int64Attribute{
							Required:            true,
							Description:         "Размер точки монтированиия в ГБ",
							MarkdownDescription: "Размер точки монтированиия в ГБ",
							Validators: []validator.Int64{
								int64validator.AtMost(2048),
								int64validator.AtLeast(10),
							},
						},
						"file_system": schema.StringAttribute{
							Computed:            true,
							Default:             stringdefault.StaticString("xfs"),
							Description:         "Тип файловой системы",
							MarkdownDescription: "Тип файловой системы",
						},
					},
				},
			},
			"zk_extra_mounts": schema.MapNestedAttribute{
				Required:            true,
				Description:         "Дополнительные точки монтирования ВМ",
				MarkdownDescription: "Дополнительные точки монтирования ВМ",
				Validators: []validator.Map{
					mapvalidator.KeysAre(
						stringvalidator.OneOf(
							"/app/zookeeper",
						),
					),
					mapvalidator.SizeBetween(1, 4),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"size": schema.Int64Attribute{
							Required:            true,
							Description:         "Размер точки монтированиия в ГБ",
							MarkdownDescription: "Размер точки монтированиия в ГБ",
							Validators: []validator.Int64{
								int64validator.AtMost(2048),
								int64validator.AtLeast(10),
							},
						},
						"file_system": schema.StringAttribute{
							Computed:            true,
							Default:             stringdefault.StaticString("xfs"),
							Description:         "Тип файловой системы",
							MarkdownDescription: "Тип файловой системы",
						},
					},
				},
			},
			"access": schema.MapAttribute{
				Optional: true,
				Computed: true,
				Description: "Словарь,где ключом является роль, а значением список групп, " +
					"которые предоставляют доступ для входа в Active Directory",
				MarkdownDescription: "Словарь,где ключом является роль, а значением список групп, " +
					"которые предоставляют доступ для входа в Active Directory",
				PlanModifiers: []planmodifier.Map{
					ActiveDirectoryAccesForSpecificEnvironment(r.provider, "DEV"),
					AvailableAccessRole(r.provider, "cluster:clickhouse"),
					EmptyAccessGroup(),
				},
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
			},

			// extra product schema
			"cluster_name": schema.StringAttribute{
				MarkdownDescription: "Имя кластера",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"ch_version": schema.StringAttribute{
				MarkdownDescription: "Версия ClickHouse",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("25.3.2.39"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"zk_version": schema.StringAttribute{
				MarkdownDescription: "Версия Zookeeper",
				Computed:            true,
				Default:             stringdefault.StaticString("3.7.2"),
			},
			"ch_customer_admin": schema.StringAttribute{
				MarkdownDescription: "Имя пользователя ClickHouse",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.NoneOf("ch_admin"),
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-z0-9][a-z0-9_]{0,23}[a-z0-9]$"),
						"Username must match pattern ^(?!^ch_admin$)[a-z0-9][a-z0-9_]{0,23}[a-z0-9]$"),
				},
				PlanModifiers: []planmodifier.String{
					AvailiableStringFieldForSpecificEnvironment(r.provider, "DEV"),
				},
			},
			"ch_customer_admin_password": schema.StringAttribute{
				MarkdownDescription: "Пароля для пользователя ClickHouse",
				Sensitive:           true,
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile("[A-Z]"),
						"Password must have 1 uppercase letter"),
					stringvalidator.RegexMatches(regexp.MustCompile("[a-z]"),
						"Password must have 1 lowercase letter"),
					stringvalidator.RegexMatches(regexp.MustCompile("[0-9]"),
						"Password must have 1 numeric character"),
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-zA-Z0-9~!@#$%^*()_]{64,70}$"),
						"Password must match pattern ^[a-zA-Z0-9~!@#$%^*()_]{64,70}"),
				},
				PlanModifiers: []planmodifier.String{
					AvailiableStringFieldForSpecificEnvironment(r.provider, "DEV"),
				},
			},
			"ch_customer_password": schema.StringAttribute{
				MarkdownDescription: "Пароля для пользователя ClickHouse Customer",
				Sensitive:           true,
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile("[A-Z]"),
						"Password must have 1 uppercase letter"),
					stringvalidator.RegexMatches(regexp.MustCompile("[a-z]"),
						"Password must have 1 lowercase letter"),
					stringvalidator.RegexMatches(regexp.MustCompile("[0-9]"),
						"Password must have 1 numeric character"),
				},
			},
			"system_adm_groups": schema.MapAttribute{
				Optional:            true,
				Description:         "AD-группа с полными правами на кластер",
				MarkdownDescription: "AD-группа с полными правами на кластер",
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
				Validators: []validator.Map{
					EmptyAccessGroupListValidator{},
					mapvalidator.KeysAre(
						stringvalidator.OneOf(
							CLICKHOUSE_SYSTEM_ADM_GROUP,
						),
					),
				},
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
				},
			},
			"clickhouse_user_ad_groups": schema.MapAttribute{
				Required:            true,
				Description:         "AD-группа с правами на кластер: Select, Insert, Alter, Create table, Remote on",
				MarkdownDescription: "AD-группа с правами на кластер: Select, Insert, Alter, Create table, Remote on",
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
				Validators: []validator.Map{
					EmptyAccessGroupListValidator{},
					mapvalidator.KeysAre(
						stringvalidator.OneOf(CLICKHOUSE_APP_USER_ROLE_NAME),
					),
				},
			},
			"clickhouse_app_admin_ad_groups": schema.MapAttribute{
				Required:            true,
				Description:         "AD-группа с правами чтение таблиц",
				MarkdownDescription: "AD-группа с правами чтение таблиц",
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
				Validators: []validator.Map{
					EmptyAccessGroupListValidator{},
					mapvalidator.KeysAre(
						stringvalidator.OneOf(
							CLICKHOUSE_APP_ADMIN_ROLE_NAME,
						),
					),
				},
			},
			"nodes_count": schema.SingleNestedAttribute{
				Required:            true,
				Description:         "Кол-во нод",
				MarkdownDescription: "Кол-во нод",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"clickhouse": schema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						Description:         "Количество нод в кластере",
						MarkdownDescription: "Количество нод в кластере",
						Default:             int64default.StaticInt64(2),
						Validators: []validator.Int64{
							int64validator.OneOf(2, 4, 6, 8, 10),
						},
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.RequiresReplace(),
						},
					},
					"zookeeper": schema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						Description:         "Количество нод в кластере",
						MarkdownDescription: "Количество нод в кластере",
						Default:             int64default.StaticInt64(3),
						Validators: []validator.Int64{
							int64validator.OneOf(1, 3),
						},
						PlanModifiers: []planmodifier.Int64{
							int64planmodifier.RequiresReplace(),
						},
					},
				},
			},
			"financial_project": schema.StringAttribute{
				Required:            true,
				Description:         "Источник финансирования.",
				MarkdownDescription: "Источник финансирования.",
			},
			"layout": schema.StringAttribute{
				Computed:            true,
				Description:         "Схема геораспределения",
				MarkdownDescription: "Схема геораспределения",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r ClickHouseClusterResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r ClickHouseClusterResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan ClickHouseClusterResourceModel
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
		resp.Diagnostics.AddAttributeError(
			path.Root("financial_project"),
			CREATE_RES_FAIL,
			fmt.Sprintf("Get financial source from portal ended with error.\nError: %v", err.Error()),
		)
		return
	}

	if r.provider.Environment == "PROD" && plan.Image.GeoDistribution.ValueBool() {
		layout, err := references.GetGeoPageByLayout(r.provider.Creds, "clickhouse_zookeeper-3:clickhouse-2")
		if err != nil {
			resp.Diagnostics.AddError(CREATE_RES_FAIL, fmt.Sprintf("Get layout data from reference: %v", err.Error()))
			return
		}
		plan.Layout = types.StringValue(layout.ID)
	} else {
		plan.Layout = types.StringNull()
	}

	attrs := r.prepareClickHouseClusterAttrs(plan)

	order := orders.NewClickHouseCluster(
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
			fmt.Sprintf("Create order on portal ended with error.\nError: %v", err.Error()),
		)
		return
	}

	orderItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			CREATE_RES_FAIL,
			fmt.Sprintf("Get order item from portal ended with error.\nError: %v", err.Error()),
		)
		return
	}

	vmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			CREATE_RES_FAIL,
			fmt.Sprintf("Get order vm items from portal ended with error.\nError: %v", err.Error()),
		)
		return
	}

	if (orderItem.Type == "cluster") && (len(vmItems) == 1) {
		resp.Diagnostics.AddError(
			CREATE_RES_FAIL,
			"Error while update state: number of VM is equal 1",
		)
		return
	}

	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(orderItem.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r ClickHouseClusterResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var orderID types.String

	req.State.GetAttribute(ctx, path.Root("order_id"), &orderID)
	if orderID.IsNull() {
		resp.Diagnostics.AddError(
			READ_RES_FAIL,
			"Resource hasn't order_id. Without order_id can't get order from portal",
		)
		return
	}

	order, err := orders.GetClickhouseClusterOrder(
		r.provider.Creds,
		r.provider.ProjectName,
		orderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			READ_RES_FAIL,
			fmt.Sprintf(
				"Can't read order with order_id equal '%s'.\nError: %s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}

	appItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get app item from portal with order_id '%s'.\nError: %s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}

	vmCKItems, err := order.GetClickHouseClusterVmItems()
	if err != nil {
		resp.Diagnostics.AddError(
			READ_RES_FAIL,
			fmt.Sprintf("Get order vm items ended with error.\nError: %v", err.Error()),
		)
		return
	}

	vmItem := vmCKItems[0]

	vmZKItems, err := order.GetClickhouseClusterZookeeperVmItems()
	if err != nil {
		resp.Diagnostics.AddError(
			READ_RES_FAIL,
			fmt.Sprintf("Get zookeeper VM items ended with error.\nError: %s", err.Error()),
		)
		return
	}

	vmZKItem := vmZKItems[0]
	vmZKConfig, ok := vmZKItem.Data.Config.(entities.VMItemConfig)
	if !ok {

		resp.Diagnostics.AddError(
			READ_RES_FAIL,
			"Invalid type for vmZKConfig",
		)
		return
	}

	var ChCustomerPassword types.String
	req.State.GetAttribute(ctx, path.Root("ch_customer_password"), &ChCustomerPassword)

	var ClickHousePassword types.String
	req.State.GetAttribute(ctx, path.Root("ch_customer_admin_password"), &ClickHousePassword)

	clickHouseUser, err := order.GetDefaultClickHouseClusterUser()
	if err != nil {
		resp.Diagnostics.AddError(
			READ_RES_FAIL,
			fmt.Sprintf("Get user from order ended with error.\nError: %s", err.Error()),
		)
		return
	}

	var tfclickHouseUser basetypes.StringValue
	if clickHouseUser.UserName == "" {
		tfclickHouseUser = types.StringNull()
	} else {
		tfclickHouseUser = types.StringValue(clickHouseUser.UserName)
	}

	clickHouseConfig, ok := appItem.Data.Config.(entities.ClickhouseClusterItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			READ_RES_FAIL,
			"Invalid type for ClickHouse Config",
		)
		return
	}

	vmConfig, ok := vmItem.Data.Config.(entities.VMItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			READ_RES_FAIL,
			"Invalid type for vmConfig",
		)
		return
	}

	appClick, err := order.GetExtraMount("/app/clickhouse")
	if err != nil {
		resp.Diagnostics.AddError(
			READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get vm extra mount '/app/app' from portal with order_id '%s'.\nError:%s",
				orderID, err.Error(),
			),
		)
		return
	}

	appZookeeper, err := order.GetExtraMount("/app/zookeeper")
	if err != nil {
		resp.Diagnostics.AddError(
			READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get vm extra mount '/app/app' from portal with order_id '%s'.\nError:%s",
				orderID, err.Error(),
			),
		)
		return
	}

	systemAdmGroups, err := order.GetAppAdGroups(CLICKHOUSE_SYSTEM_ADM_GROUP)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root(CLICKHOUSE_SYSTEM_ADM_GROUP),
			READ_RES_FAIL,
			fmt.Sprintf("Get system_adm_groups from portal ended with error.\nError: %v", err.Error()),
		)
		return
	}

	if len(systemAdmGroups) < 1 {
		systemAdmGroups = nil
	}

	clickHouseUserAdGroups, err := order.GetAppAdGroups(CLICKHOUSE_APP_USER_ROLE_NAME)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root(CLICKHOUSE_APP_USER_ROLE_NAME),
			READ_RES_FAIL,
			fmt.Sprintf("Get clickhouse_user_ad_groups from portal ended with error.\nError: %v", err.Error()),
		)
		return
	}

	clickHouseAppAdminGroups, err := order.GetAppAdGroups(CLICKHOUSE_APP_ADMIN_ROLE_NAME)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root(CLICKHOUSE_APP_ADMIN_ROLE_NAME),
			READ_RES_FAIL,
			fmt.Sprintf("Get clickhouse_app_admin_ad_groups from portal ended with error.\nError: %v", err.Error()),
		)
		return
	}

	state := ClickHouseClusterResourceModel{
		OrderID:                    orderID,
		Access:                     readAccessMapV2(vmItem.Data.ACLs),
		Label:                      types.StringValue(order.Label),
		ClickHouseVersion:          types.StringValue(clickHouseConfig.Version),
		ZookeeperVersion:           types.StringValue(order.Attrs.ZookeeperVersion),
		ClusterName:                types.StringValue(clickHouseConfig.ClusterName),
		ClickHouseUser:             tfclickHouseUser,
		ChCustomerPassword:         ChCustomerPassword,
		ClickHousePassword:         ClickHousePassword,
		SystemAdmGroups:            systemAdmGroups,
		ClickHouseAppAdminAdGroups: clickHouseAppAdminGroups,
		ClickHouseUserAdGroups:     clickHouseUserAdGroups,
		NodesCount: orders.NodesCount{
			ClickHouse: len(vmCKItems),
			Zookeeper:  len(vmZKItems),
		},
		ItemID: types.StringValue(appItem.ID),
		FlavorCH: FlavorModel{
			Cores:  types.Int64Value(vmConfig.Flavor.Cores),
			Memory: types.Int64Value(vmConfig.Flavor.Memory),
			Name:   types.StringValue(vmConfig.Flavor.Name),
			UUID:   types.StringValue(vmConfig.Flavor.UUID),
		},
		FlavorZK: FlavorModel{
			Cores:  types.Int64Value(vmZKConfig.Flavor.Cores),
			Memory: types.Int64Value(vmZKConfig.Flavor.Memory),
			Name:   types.StringValue(vmZKConfig.Flavor.Name),
			UUID:   types.StringValue(vmZKConfig.Flavor.UUID),
		},
		Image: ClusterProductImageDataSourceModel{
			Distribution:    types.StringValue(vmConfig.Image.Os.Distribution),
			OsVersion:       types.StringValue(vmConfig.Image.Os.Version),
			OnSupport:       types.BoolValue(order.Attrs.OnSupport),
			ProductID:       types.StringValue(order.ProductID),
			ADIntegration:   types.BoolValue(order.Attrs.ADIntegration),
			GeoDistribution: types.BoolValue(order.Attrs.GeoDistribution),
		},
		Core: CoreModel{
			Platform:       types.StringValue(order.Attrs.Platform),
			Domain:         types.StringValue(order.Attrs.Domain),
			NetSegmentCode: types.StringValue(vmConfig.DefaultNic.NetSegment),
			Zone:           types.StringValue(order.Attrs.AvailabilityZone),
		},
		CHExtraMounts: map[string]ExtraMountModel{
			"/app/clickhouse": {
				Size:       types.Int64Value(appClick.Size),
				FileSystem: types.StringValue(appClick.FileSystem),
			},
		},
		ZKExtraMounts: map[string]ExtraMountModel{
			"/app/zookeeper": {
				Size:       types.Int64Value(appZookeeper.Size),
				FileSystem: types.StringValue(appZookeeper.FileSystem),
			},
		},
		FinancialProject: types.StringValue(order.FinancialSource.Name),
		Layout:           types.StringValue(order.Attrs.Layout),
	}

	var lifetime types.Int64
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("lifetime"), &lifetime)...)
	if !lifetime.IsNull() {
		state.Lifetime = lifetime
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r ClickHouseClusterResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state ClickHouseClusterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetClickhouseClusterOrder(
		r.provider.Creds,
		r.provider.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("order_id"),
			UPDATE_RES_FAIL,
			fmt.Sprintf("Get order from portal ended with error.\nError: %v", err.Error()),
		)
		return
	}

	finProj, err := sources.GetFinancialProjectByName(
		r.provider.Creds,
		r.provider.ProjectName,
		plan.FinancialProject.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("financial_project"),
			UPDATE_RES_FAIL,
			fmt.Sprintf("Get financial source for project ended with error.\nError: %v", err.Error()),
		)
		return
	}

	labelChanged := plan.Label != state.Label
	flavorCHChanged := plan.FlavorCH != state.FlavorCH
	flavorZKChanged := plan.FlavorZK != state.FlavorZK
	financialProjectChanged := plan.FinancialProject.Equal(state.FinancialProject)
	clickHousePasswordChanged := state.ClickHousePassword != plan.ClickHousePassword
	chCustomerPasswordChanged := state.ChCustomerPassword != plan.ChCustomerPassword
	accessChanged := !reflect.DeepEqual(state.Access, plan.Access)
	adAdminGroupChanged := !reflect.DeepEqual(state.ClickHouseAppAdminAdGroups, plan.ClickHouseAppAdminAdGroups)
	adUserGroupChanged := !reflect.DeepEqual(state.ClickHouseUserAdGroups, plan.ClickHouseUserAdGroups)

	if labelChanged {
		changeOrderLabel(order, plan.Label.ValueString(), resp)
	}

	if financialProjectChanged {
		r.changeFinancialProject(order, finProj.ID, resp)
	}

	if accessChanged {
		r.changeAccess(order, &plan, resp)
	}

	if clickHousePasswordChanged {
		r.changeUserPassword(order, &plan, &state, resp)
	}

	if chCustomerPasswordChanged {
		r.changeCHUserPassword(order, &plan, resp)
	}

	if adAdminGroupChanged {
		r.applyAdAdminGroups(order, plan, state, resp)
	}

	if adUserGroupChanged {
		r.applyAdUsersGroups(order, plan, state, resp)
	}

	if flavorCHChanged {
		r.changeFlavorClickHouseCluster(order, &plan, resp)
	}

	if flavorZKChanged {
		r.changeFlavorZKClickHouseCluster(order, &plan, resp)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r ClickHouseClusterResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data ClickHouseClusterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetClickhouseClusterOrder(
		r.provider.Creds,
		r.provider.ProjectName,
		data.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("order_id"), DELETE_RES_FAIL, err.Error())
		return
	}

	orderState, err := order.GetState()
	if err != nil {
		resp.Diagnostics.AddError(
			DELETE_RES_FAIL,
			fmt.Sprintf("Can't get state from portal.\nError:%s", err.Error()),
		)
	}

	if orderState == "deleted" || orderState == "deprovisioned" {
		resp.Diagnostics.AddWarning(
			DELETE_RES_FAIL,
			fmt.Sprintf(
				"Order with id '%s' was deleted by another way. Not from terraform. "+
					"Resource was recreated",
				data.OrderID.ValueString(),
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
			DELETE_RES_FAIL,
			fmt.Sprintf(
				"Order with order_id '%s' isn't deletable. "+
					"Please, make your order is deletable by the portal VTB.Cloud, and repeat again."+
					"\nURL reference: %s",
				order.ID, orderURL,
			),
		)
		return
	}

	if err := order.DeleteClickHouseCluster(r.provider.Environment); err != nil {
		resp.Diagnostics.AddError("While process delete Clickhouse Cluster: ", err.Error())
		return
	}
}

func (r ClickHouseClusterResource) changeFlavorClickHouseCluster(
	order *orders.ClickHouseCluster,
	plan *ClickHouseClusterResourceModel,
	resp *resource.UpdateResponse,
) {
	flavor := entities.Flavor{
		Cores:  plan.FlavorCH.Cores.ValueInt64(),
		Memory: plan.FlavorCH.Memory.ValueInt64(),
		UUID:   plan.FlavorCH.UUID.ValueString(),
		Name:   plan.FlavorCH.Name.ValueString(),
	}
	err := order.ChangeCHFlavor(flavor, false)
	if err != nil {
		resp.Diagnostics.AddError(
			UPDATE_RES_FAIL,
			fmt.Sprintf("Change clickhouse flavor ended with error.\nError: %v", err.Error()),
		)
		return
	}
}

func (r ClickHouseClusterResource) changeFlavorZKClickHouseCluster(
	order *orders.ClickHouseCluster,
	plan *ClickHouseClusterResourceModel,
	resp *resource.UpdateResponse,
) {
	flavor := entities.Flavor{
		Cores:  plan.FlavorZK.Cores.ValueInt64(),
		Memory: plan.FlavorZK.Memory.ValueInt64(),
		UUID:   plan.FlavorZK.UUID.ValueString(),
		Name:   plan.FlavorZK.Name.ValueString(),
	}
	err := order.ChangeZKFlavor(flavor, false)
	if err != nil {
		resp.Diagnostics.AddError(
			UPDATE_RES_FAIL,
			fmt.Sprintf("Change zookeper flavor ended with error.\nError: %v", err.Error()),
		)
		return
	}
}

func (r ClickHouseClusterResource) changeUserPassword(
	order *orders.ClickHouseCluster,
	plan, state *ClickHouseClusterResourceModel,
	resp *resource.UpdateResponse,
) {
	err := order.ChangeUserPassword(state.ClickHouseUser.ValueString(), plan.ClickHousePassword.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			UPDATE_RES_FAIL,
			fmt.Sprintf(
				"Change user password enden with error.\nError: %v",
				err.Error(),
			),
		)
		return
	}
}

func (r ClickHouseClusterResource) changeCHUserPassword(
	order *orders.ClickHouseCluster,
	plan *ClickHouseClusterResourceModel,
	resp *resource.UpdateResponse,
) {
	err := order.ChangeUserPassword("ch_customer", plan.ChCustomerPassword.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			UPDATE_RES_FAIL,
			fmt.Sprintf(
				"Change user password enden with error.\nError: %v",
				err.Error(),
			),
		)
		return
	}
}

func (r ClickHouseClusterResource) changeAccess(
	order *orders.ClickHouseCluster,
	plan *ClickHouseClusterResourceModel,
	resp *resource.UpdateResponse,
) {
	vmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			READ_RES_FAIL,
			fmt.Sprintf("Get VM items ended with error.\nError: %v", err.Error()))
		return
	}

	for _, vmItem := range vmItems {
		stateAccess := readAccessMapVV1(vmItem.Data.ACLs)
		planAccess := convertAccessMap(plan.Access)
		deletedRoles, changedRoles, addedRoles := r.compareADRoles(planAccess, stateAccess)

		if len(addedRoles) > 0 {
			rolesFromCloud, err := references.GetRoles(
				r.provider.Creds,
				r.provider.Environment,
				"app:clickhouse",
			)
			if err != nil {
				resp.Diagnostics.AddError(UPDATE_RES_FAIL, err.Error())
				return
			}

			permissionsMap := make(map[string][]string)
			for _, value := range rolesFromCloud {
				permissionsMap[value.Data.GroupName] = value.Data.Permissions
			}

			for _, roles := range addedRoles {
				err := order.AddAccessGroupForVm(vmItem.ID, roles, permissionsMap[roles.Role])
				if err != nil {
					resp.Diagnostics.AddError(UPDATE_RES_FAIL, err.Error())
					return
				}
			}
		}

		if len(changedRoles) > 0 {
			for _, roles := range changedRoles {
				err := order.ChangeAccessGroupForVm(vmItem.ID, roles)
				if err != nil {
					resp.Diagnostics.AddError(UPDATE_RES_FAIL, err.Error())
					return
				}
			}
		}

		if len(deletedRoles) > 0 {
			for _, roles := range deletedRoles {
				err := order.DeleteAccessGroupForVm(vmItem.ID, roles)
				if err != nil {
					resp.Diagnostics.AddError(UPDATE_RES_FAIL, err.Error())
					return
				}
			}
		}
	}
}

func (r ClickHouseClusterResource) changeFinancialProject(
	order *orders.ClickHouseCluster,
	finProjectId string,
	resp *resource.UpdateResponse,
) {
	err := order.ChangeFinancialProject(finProjectId)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("financial_project"),
			UPDATE_RES_FAIL,
			fmt.Sprintf("Change financial source for project ended wiht error: %v", err.Error()),
		)
		return
	}
}

func (r ClickHouseClusterResource) compareADRoles(planAccess, stateAccess map[string][]string) (deletedRoles, changedRoles, addedRoles []entities.ADLogonGrants) {
	for role, groups := range stateAccess {
		if _, ok := planAccess[role]; !ok {
			deletedRoles = append(
				deletedRoles,
				entities.ADLogonGrants{Role: role, Groups: groups},
			)
		} else {
			if len(planAccess[role]) == len(stateAccess[role]) {
				if differenceLen(planAccess[role], stateAccess[role]) {
					changedRoles = append(
						changedRoles,
						entities.ADLogonGrants{Role: role, Groups: planAccess[role]},
					)
				}
			} else {
				changedRoles = append(
					changedRoles,
					entities.ADLogonGrants{Role: role, Groups: planAccess[role]},
				)
			}
		}
	}

	for role, groups := range planAccess {
		if _, ok := stateAccess[role]; !ok {
			addedRoles = append(
				addedRoles,
				entities.ADLogonGrants{Role: role, Groups: groups},
			)
		}
	}
	return deletedRoles, changedRoles, addedRoles
}

func (r ClickHouseClusterResource) prepareClickHouseClusterAttrs(
	plan ClickHouseClusterResourceModel,
) (attrs orders.ClickHouseClusterAttrs) {
	var SystemAdmGroupsAttrs []orders.SystemAdmGroupsModel
	for role, groups := range plan.SystemAdmGroups {
		var groupsNames []string
		groupsNames = append(groupsNames, groups...)
		SystemAdmGroupsAttrs = append(SystemAdmGroupsAttrs, orders.SystemAdmGroupsModel{
			Role:   role,
			Groups: groupsNames,
		})
	}

	var ClickHouseAppAdminAdGroupsAttrs []orders.ClickHouseAppAdminAdGroupsModel
	for role, groups := range plan.ClickHouseAppAdminAdGroups {
		var groupsNames []string
		groupsNames = append(groupsNames, groups...)
		ClickHouseAppAdminAdGroupsAttrs = append(ClickHouseAppAdminAdGroupsAttrs, orders.ClickHouseAppAdminAdGroupsModel{
			Role:   role,
			Groups: groupsNames,
		})
	}

	var ClickHouseUserAdGroupsAttrs []orders.ClickHouseUserAdGroupsModel
	for role, groups := range plan.ClickHouseUserAdGroups {
		var groupsNames []string
		groupsNames = append(groupsNames, groups...)
		ClickHouseUserAdGroupsAttrs = append(ClickHouseUserAdGroupsAttrs, orders.ClickHouseUserAdGroupsModel{
			Role:   role,
			Groups: groupsNames,
		})
	}

	attrs = orders.ClickHouseClusterAttrs{
		ADIntegration:    plan.Image.ADIntegration.ValueBool(),
		OnSupport:        plan.Image.OnSupport.ValueBool(),
		OsVersion:        plan.Image.OsVersion.ValueString(),
		ADLogonGrants:    prepareADLogonGrants(plan.Access),
		AvailabilityZone: plan.Core.Zone.ValueString(),
		Domain:           plan.Core.Domain.ValueString(),
		Platform:         plan.Core.Platform.ValueString(),
		DefaultNic: entities.DefaultNic{
			NetSegment: plan.Core.NetSegmentCode.ValueString(),
		},
		CreatedWithOpenTofu:             true,
		Accept:                          true,
		ClickHouseVersion:               plan.ClickHouseVersion.ValueString(),
		ZookeeperVersion:                plan.ZookeeperVersion.ValueString(),
		ClusterName:                     plan.ClusterName.ValueString(),
		ClickHouseUser:                  plan.ClickHouseUser.ValueStringPointer(),
		ClickHousePassword:              plan.ClickHousePassword.ValueStringPointer(),
		ChCustomerPassword:              plan.ChCustomerPassword.ValueString(),
		SystemAdmGroupsAttrs:            SystemAdmGroupsAttrs,
		ClickHouseAppAdminAdGroupsAttrs: ClickHouseAppAdminAdGroupsAttrs,
		ClickHouseUserAdGroupsAttrs:     ClickHouseUserAdGroupsAttrs,
		FlavorCH: entities.Flavor{
			Cores:  plan.FlavorCH.Cores.ValueInt64(),
			Memory: plan.FlavorCH.Memory.ValueInt64(),
			Name:   string(plan.FlavorCH.Name.ValueString()),
			UUID:   string(plan.FlavorCH.UUID.ValueString()),
		},
		FlavorZK: entities.Flavor{
			Cores:  plan.FlavorZK.Cores.ValueInt64(),
			Memory: plan.FlavorZK.Memory.ValueInt64(),
			Name:   string(plan.FlavorZK.Name.ValueString()),
			UUID:   string(plan.FlavorZK.UUID.ValueString()),
		},
		NodesCount:      plan.NodesCount,
		CHExtraMounts:   prepareExtraMountsAttrs(plan.CHExtraMounts),
		ZKExtraMounts:   prepareExtraMountsAttrs(plan.ZKExtraMounts),
		Layout:          plan.Layout.ValueString(),
		GeoDistribution: plan.Image.GeoDistribution.ValueBool(),
	}
	return attrs
}

func (r ClickHouseClusterResource) applyAdAdminGroups(
	order *orders.ClickHouseCluster,
	plan, state ClickHouseClusterResourceModel,
	resp *resource.UpdateResponse,
) {
	toAdd, toDelete, err := r.compareAppGroups(CLICKHOUSE_APP_ADMIN_ROLE_NAME, plan, state)
	if err != nil {
		resp.Diagnostics.AddError(
			UPDATE_RES_FAIL,
			fmt.Sprintf("Compare App Admin Groups ended with error.\nError: %v", err.Error()),
		)
		return
	}

	if len(toAdd) > 0 {
		for _, groupName := range toAdd {
			err = order.CreateNewAppAdminGroupAd(groupName)
			if err != nil {
				resp.Diagnostics.AddError(
					UPDATE_RES_FAIL,
					fmt.Sprintf("Create new app admin group ended with error.\nError: %v", err.Error()),
				)
				return
			}
		}
	}

	if len(toDelete) > 0 {
		for _, groupName := range toDelete {
			err = order.RemoveNewAppAdminGroupAd(groupName)
			if err != nil {
				resp.Diagnostics.AddError(
					UPDATE_RES_FAIL,
					fmt.Sprintf("Remove new app admin group ended with error.\nError: %v", err.Error()),
				)
				return
			}
		}
	}
}

func (r ClickHouseClusterResource) applyAdUsersGroups(
	order *orders.ClickHouseCluster,
	plan, state ClickHouseClusterResourceModel,
	resp *resource.UpdateResponse,
) {
	toAdd, toDelete, err := r.compareAppGroups(CLICKHOUSE_APP_USER_ROLE_NAME, plan, state)
	if err != nil {
		resp.Diagnostics.AddError(
			UPDATE_RES_FAIL,
			fmt.Sprintf("Compare app user groups ended with error.\nError: %v", err.Error()),
		)
		return
	}

	if len(toAdd) > 0 {
		for _, groupName := range toAdd {
			err = order.CreateNewAppUserGroupAd(groupName)
			if err != nil {
				resp.Diagnostics.AddError(
					UPDATE_RES_FAIL,
					fmt.Sprintf("Create new app user group ad ended with error.\nError: %v", err.Error()),
				)
				return
			}
		}
	}

	if len(toDelete) > 0 {
		for _, groupName := range toDelete {
			err = order.RemoveNewAppUserGroupAd(groupName)
			if err != nil {
				resp.Diagnostics.AddError(
					UPDATE_RES_FAIL,
					fmt.Sprintf("Remove app user group ad ended with error.\nError: %v", err.Error()),
				)
				return
			}
		}
	}
}

func (r ClickHouseClusterResource) compareAppGroups(
	roleName string,
	plan, state ClickHouseClusterResourceModel,
) (groupsToAdd, groupsToDelete []string, err error) {

	switch roleName {
	case CLICKHOUSE_APP_ADMIN_ROLE_NAME:
		planAdminGroups, ok := plan.ClickHouseAppAdminAdGroups[CLICKHOUSE_APP_ADMIN_ROLE_NAME]
		if !ok {
			return nil, nil, fmt.Errorf("plan app admin groups not found")
		}

		stateAdminGroups, ok := state.ClickHouseAppAdminAdGroups[CLICKHOUSE_APP_ADMIN_ROLE_NAME]
		if !ok {
			return nil, nil, fmt.Errorf("state app admin groups not found")
		}

		groupsToAdd, groupsToDelete = compareSlices(stateAdminGroups, planAdminGroups)

	case CLICKHOUSE_APP_USER_ROLE_NAME:
		planAdminGroups, ok := plan.ClickHouseUserAdGroups[CLICKHOUSE_APP_USER_ROLE_NAME]
		if !ok {
			return nil, nil, fmt.Errorf("plan app user groups not found")
		}

		stateAdminGroups, ok := state.ClickHouseUserAdGroups[CLICKHOUSE_APP_USER_ROLE_NAME]
		if !ok {
			return nil, nil, fmt.Errorf("state app user groups not found")
		}

		groupsToAdd, groupsToDelete = compareSlices(stateAdminGroups, planAdminGroups)
	}

	return groupsToAdd, groupsToDelete, nil
}
