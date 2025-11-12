package etcd

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/common"
	"terraform-provider-vtb/internal/consts"
	"terraform-provider-vtb/internal/custommodifires"
	"terraform-provider-vtb/internal/services/core"
	"terraform-provider-vtb/internal/services/flavor"
	"terraform-provider-vtb/internal/utils"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"
	"terraform-provider-vtb/pkg/client/references"
	"terraform-provider-vtb/pkg/client/sources"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &EtcdResource{}
	_ resource.ResourceWithImportState = &EtcdResource{}
	_ resource.ResourceWithModifyPlan  = &EtcdResource{}
)

type EtcdResource struct {
	client *client.CloudClient
}

func NewEtcdResource() resource.Resource {
	return &EtcdResource{}
}

func (r EtcdResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_etcd_instance"
}

func (r *EtcdResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type EtcdResourceModel struct {
	Core             core.CoreModel                    `tfsdk:"core"`
	Flavor           flavor.FlavorModel                `tfsdk:"flavor"`
	Lifetime         types.Int64                       `tfsdk:"lifetime"`
	Label            types.String                      `tfsdk:"label"`
	ItemID           types.String                      `tfsdk:"item_id"`
	OrderID          types.String                      `tfsdk:"order_id"`
	Access           map[string][]string               `tfsdk:"access"`
	ExtraMounts      map[string]common.ExtraMountModel `tfsdk:"extra_mounts"`
	FinancialProject types.String                      `tfsdk:"financial_project"`

	Image            EtcdImageDataSourceData `tfsdk:"image"`
	EtcdUserName     types.String            `tfsdk:"etcd_user_name"`
	EtcdUserPassword types.String            `tfsdk:"etcd_user_password"`
	ClusterName      types.String            `tfsdk:"cluster_name"`
	EtcdVersion      types.String            `tfsdk:"etcd_version"`
	NodesCount       types.Int64             `tfsdk:"nodes_count"`
	Layout           types.String            `tfsdk:"layout"`
}

func (r EtcdResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Etcd Resource",
		Attributes: map[string]schema.Attribute{
			// basic schema
			"core": schema.SingleNestedAttribute{
				Required:            true,
				Description:         "Основные параметры для ВМ в заказе.",
				MarkdownDescription: "Основные параметры для ВМ в заказе.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: common.CoreSchema,
			},
			"flavor": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.FlavorSchema,
				Description:         "Кол-во CPU/RAM для ВМ.",
				MarkdownDescription: "Кол-во CPU/RAM для ВМ.",
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
				Required:            true,
				Description:         "Метка заказа",
				MarkdownDescription: "Метка заказа",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"order_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Идентификатор заказа. Становиться известным после создания заказа.",
				MarkdownDescription: "Идентификатор заказа. Становиться известным после создания заказа.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"item_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Идентификатор родительского итема заказа",
				MarkdownDescription: "Идентификатор родительского итема заказа",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"extra_mounts": schema.MapNestedAttribute{
				Required:            true,
				Description:         "Дополнительные точки монтирования ВМ",
				MarkdownDescription: "Дополнительные точки монтирования ВМ",
				Validators: []validator.Map{
					mapvalidator.KeysAre(
						stringvalidator.OneOf(
							"/app/etcd/",
							"/app/logs/",
							"/app/backup/",
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
					custommodifires.EmptyAccessGroup(),
				},
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
			},
			"financial_project": schema.StringAttribute{
				Required:            true,
				Description:         "Источник финансорования заказа",
				MarkdownDescription: "Источник финансорования заказа",
			},

			// image schema
			"image": schema.SingleNestedAttribute{
				Required:            true,
				Description:         "Тип вычислительного ресурса",
				MarkdownDescription: "Тип вычислительного ресурса",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"distribution": schema.StringAttribute{
						Required:            true,
						Description:         "Имя дистрибутива (RHEL или Astra)",
						MarkdownDescription: "Имя дистрибутива (RHEL или Astra)",
						Validators: []validator.String{
							stringvalidator.OneOf("rhel", "astra"),
						},
					},
					"os_version": schema.StringAttribute{
						Required:            true,
						Description:         "Версия образа",
						MarkdownDescription: "Версия образа",
					},
					"on_support": schema.BoolAttribute{
						Required:            true,
						Description:         "Флаг обозначающий, осуществляется ли поддержка группой сопровождения",
						MarkdownDescription: "Флаг обозначающий, осуществляется ли поддержка группой сопровождения",
					},
					"product_id": schema.StringAttribute{
						Required:            true,
						Description:         "Идентификатор продукта в продуктовом каталоге",
						MarkdownDescription: "Идентификатор продукта в продуктовом каталоге",
					},
					"ad_integration": schema.BoolAttribute{
						Required:            true,
						Description:         "Флаг обозначающий, включена ли интеграция Active Directory.",
						MarkdownDescription: "Флаг обозначающий, включена ли интеграция Active Directory.",
					},

					// extra image data attrs
					"use_ssl": schema.BoolAttribute{
						Required:            true,
						Description:         "Использовать SSL",
						MarkdownDescription: "Использовать SSL",
					},
					"on_backup": schema.BoolAttribute{
						Required:            true,
						Description:         "Использовать резервное копирование",
						MarkdownDescription: "Использовать резервное копирование",
					},
					"geo_distribution": schema.BoolAttribute{
						Required:            true,
						Description:         "Использовать геораспределение",
						MarkdownDescription: "Использовать геораспределение",
					},
				},
			},
			"etcd_version": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Версия Etcd",
				MarkdownDescription: "Версия Etcd",
				Default:             stringdefault.StaticString("3.5.9"),
				Validators: []validator.String{
					stringvalidator.OneOf("3.5.9"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"nodes_count": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "Количество нод в кластере",
				MarkdownDescription: "Количество нод в кластере",
				Default:             int64default.StaticInt64(3),
				Validators: []validator.Int64{
					int64validator.OneOf(3, 5, 7),
				},
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"layout": schema.StringAttribute{
				Computed:            true,
				Description:         "Схема геораспределения",
				MarkdownDescription: "Схема геораспределения",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"etcd_user_name": schema.StringAttribute{
				Required:            true,
				Description:         "Имя пользователя Etcd",
				MarkdownDescription: "Имя пользователя Etcd",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"etcd_user_password": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				Description:         "Пароль пользователя Etcd",
				MarkdownDescription: "Пароль пользователя Etcd",
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile("[A-Z]"),
						"Password must have 1 uppercase letter"),
					stringvalidator.RegexMatches(regexp.MustCompile("[a-z]"),
						"Password must have 1 lowercase letter"),
					stringvalidator.RegexMatches(regexp.MustCompile("[0-9]"),
						"Password must have 1 numeric character"),
					stringvalidator.RegexMatches(regexp.MustCompile(`^[a-zA-Z0-9\\._-]{20,64}$`),
						"Password must match pattern ^[a-zA-Z0-9\\._-]{20,64}$"),
				},
			},
			"cluster_name": schema.StringAttribute{
				Required:            true,
				Description:         "Имя кластера",
				MarkdownDescription: "Имя кластера",
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile("^[a-z][a-z0-9_]{0,23}[a-z0-9]$"),
						"Password must contain lowercase latin letters, numeric values and the \"_\" symbol, "+
							"but cannot start or end with it, cannot contain the \"-\" symbol,"+
							"and cannot begin with a number. The name length is from 2 to 23 characters."),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
	}
}

func (r EtcdResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r EtcdResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan, state EtcdResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	}

	checkOrderIsDeleted := utils.CheckOrderIsDeleted(r.client, plan.OrderID, "cluster", "etcd")
	resp.Diagnostics.Append(checkOrderIsDeleted.Diagnostics...)

	utils.HandleExternalDeletionRecreate(ctx, resp, &plan, checkOrderIsDeleted.IsDeleted, checkOrderIsDeleted.Diagnostics)

	validateAccessForEnvDiags := utils.ValidateActiveDirectoryAccessForEnvironmentsV2(r.client, plan.Access, "DEV")
	resp.Diagnostics.Append(validateAccessForEnvDiags...)

	validateAccessRolesDiags := utils.ValidateAccessRolesV1(r.client, "cluster:etcd", plan.Access)
	resp.Diagnostics.Append(validateAccessRolesDiags...)

	var extraMountsDev = []string{"/app/etcd/", "/app/logs/"}

	_, exist := plan.ExtraMounts["/app/backup/"]
	if exist && r.client.Environment != "PROD" {
		resp.Diagnostics.AddError(
			"/app/backup/ in nonProd environment",
			"'extra_mounts' can contain '/app/backup/' only in PROD",
		)
		return

	} else if !exist && r.client.Environment == "PROD" {
		resp.Diagnostics.AddError(
			"/app/backup in PROD environment",
			"'extra_mounts' must contain '/app/backup/' in PROD",
		)
		return

	}
	for _, extraMount := range extraMountsDev {
		_, exist := plan.ExtraMounts[extraMount]
		if !exist {
			resp.Diagnostics.AddError(
				fmt.Sprintf("'%s' is not in 'extra_mounts'", extraMount),
				fmt.Sprintf("'%s' extra mount must be in 'extra_mounts'", extraMount),
			)
			return
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r EtcdResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan EtcdResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if r.client.Environment == "PROD" && plan.Image.GeoDistribution.ValueBool() {
		layout, err := references.GetGeoPageByLayout(r.client.Creds, fmt.Sprintf("etcd:%d", plan.NodesCount.ValueInt64()))
		if err != nil {
			resp.Diagnostics.AddError(consts.CREATE_RES_FAIL, fmt.Sprintf("Get layout data from reference: %v", err.Error()))
			return
		}
		plan.Layout = types.StringValue(layout.ID)
	} else {
		plan.Layout = types.StringNull()
	}

	attrs := prepareAttrs(plan)

	order := orders.NewEtcdOrder(
		r.client.Creds,
		r.client.ProjectName,
		plan.Image.ProductID.ValueString(),
		attrs,
	)

	finProj, err := sources.GetFinancialProjectByName(
		r.client.Creds,
		r.client.ProjectName,
		plan.FinancialProject.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("financial_project"), consts.CREATE_RES_FAIL, err.Error())
		return
	}

	err = order.Create(orders.CreateOrderPayload{
		Label:        plan.Label.ValueString(),
		Lifetime:     int(plan.Lifetime.ValueInt64()),
		FinProjectID: finProj.ID,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Create order ended with error.\nError: %v", err.Error()),
		)
		return
	}

	err = order.Sync()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Sync order state ended with error.\nError: %v", err.Error()),
		)
		return
	}

	orderItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Get order item ended with error.\nError: %v", err.Error()),
		)
		return
	}

	err = order.Sync()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.CREATE_RES_FAIL,
			fmt.Sprintf("Sync order state ended with error.\nError: %v", err.Error()),
		)
		return
	}

	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(orderItem.ID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r EtcdResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data EtcdResourceModel

	var orderID, etcdUserPassword types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("order_id"), &orderID)...)
	req.State.GetAttribute(ctx, path.Root("etcd_user_password"), &etcdUserPassword)

	if orderID.IsNull() {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Resource hasn't order_id. Without order_id can't get order from portal",
		)
		return
	}

	data.OrderID = orderID
	order, err := orders.GetEtcdOrder(
		r.client.Creds,
		r.client.ProjectName,
		data.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't read order with order_id equal '%s'.\nError:%s",
				data.OrderID.ValueString(), err.Error(),
			),
		)
		return
	}

	item, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf(
				"Can't get parent item from portal with order_id '%s'. \nError:%s",
				data.OrderID.ValueString(), err.Error(),
			),
		)
		return
	}

	config, ok := item.Data.Config.(entities.EtcdlItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintln("invalid type for etcd config item"),
		)
		return
	}

	etcdUser, err := order.GetDefaultUser()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Can't get etcd default user.\nError:%s", err.Error()),
		)
		return
	}

	vmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Error:%s", err.Error()),
		)
		return
	}
	data.NodesCount = types.Int64Value(int64(len(vmItems)))

	vmItem := vmItems[0]
	vmConfig := vmItem.Data.Config.(entities.VMItemConfig)

	data.Label = types.StringValue(order.Label)
	data.ItemID = types.StringValue(item.ID)
	data.Layout = types.StringValue(order.Attrs.Layout)
	data.ClusterName = types.StringValue(config.ClusterName)
	data.EtcdUserName = types.StringValue(etcdUser.Username)
	data.EtcdUserPassword = etcdUserPassword
	data.Access = utils.ReadAccessMapVV1(vmItem.Data.ACLs)
	data.FinancialProject = types.StringValue(order.FinancialSource.Name)
	data.EtcdVersion = types.StringValue(strings.Split(config.Version, "v")[1])
	data.Flavor = flavor.FlavorModel{
		Cores:  types.Int64Value(vmConfig.Flavor.Cores),
		Memory: types.Int64Value(vmConfig.Flavor.Memory),
		Name:   types.StringValue(vmConfig.Flavor.Name),
		UUID:   types.StringValue(vmConfig.Flavor.UUID),
	}

	data.ExtraMounts = make(map[string]common.ExtraMountModel)
	orderExtraMounts, err := order.GetExtraMounts()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Can't get etcd extra mounts.\nError:%s", err.Error()),
		)
		return
	}

	for _, extraMount := range orderExtraMounts {
		data.ExtraMounts[extraMount.Path+"/"] = common.ExtraMountModel{
			Size:       types.Int64Value(extraMount.Size),
			FileSystem: types.StringValue(extraMount.FileSystem),
		}
	}

	data.Image = EtcdImageDataSourceData{
		Distribution:    types.StringValue(vmConfig.Image.Os.Distribution),
		OSVersion:       types.StringValue(vmConfig.Image.Os.Version),
		OnSupport:       types.BoolValue(order.Attrs.OnSupport),
		GeoDistribution: types.BoolValue(order.Attrs.GeoDistribution),
		OnBackup:        types.BoolValue(order.Attrs.OnBackup),
		ProductID:       types.StringValue(order.ProductID),
		UseSsl:          types.BoolValue(config.UseSsl),
		ADIntegration:   types.BoolValue(order.Attrs.ADIntegration),
	}
	data.Core = core.CoreModel{
		Platform:       types.StringValue(utils.SelectPlatform(order.Attrs.Platform)),
		Domain:         types.StringValue(order.Attrs.Domain),
		NetSegmentCode: types.StringValue(vmConfig.DefaultNic.NetSegment),
		Zone:           types.StringValue(order.Attrs.AvailabilityZone),
	}

	var lifetime types.Int64
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("lifetime"), &lifetime)...)
	if !lifetime.IsNull() {
		data.Lifetime = lifetime
	}

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r EtcdResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state EtcdResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetEtcdOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)

	if err != nil {
		resp.Diagnostics.AddError(
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Can't get order data from portal.\nError: %v", err.Error()),
		)
		return
	}

	finProj, err := sources.GetFinancialProjectByName(
		r.client.Creds,
		r.client.ProjectName,
		plan.FinancialProject.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("financial_project"), consts.UPDATE_RES_FAIL, err.Error())
		return
	}

	mountChanged := utils.IsExtraMountChanged(state.ExtraMounts, plan.ExtraMounts)
	passwordChanged := state.EtcdUserPassword != plan.EtcdUserPassword
	flavorChanged := state.Flavor != plan.Flavor
	accessChanged := !reflect.DeepEqual(state.Access, plan.Access)
	financialProjectChanged := !plan.FinancialProject.Equal(state.FinancialProject)
	labelChanged := !plan.Label.Equal(state.Label)

	if labelChanged {
		utils.ChangeOrderLabel(order, plan.Label.ValueString(), resp)
	}

	if financialProjectChanged {
		err = order.ChangeFinancialProject(finProj.ID)
		if err != nil {
			resp.Diagnostics.AddError(
				consts.UPDATE_RES_FAIL,
				fmt.Sprintf("Change financial source ended with error.\nError: %v", err.Error()),
			)
			return
		}
	}

	if passwordChanged {
		err = order.ChangeUserPassword(state.EtcdUserName.ValueString(), plan.EtcdUserPassword.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				consts.UPDATE_RES_FAIL,
				fmt.Sprintf("Change password failed: %v", err.Error()))
			return
		}
	}

	if mountChanged {
		diags := changeEtcdExtraMounts(order, &plan)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if flavorChanged {
		diags := changeEtcdFlavor(order, &plan)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if accessChanged {
		diags := r.changeAccess(order, &plan)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

}

func (r EtcdResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state EtcdResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetEtcdOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(consts.DELETE_RES_FAIL, err.Error())
		return
	}

	orderState, err := order.GetState()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf("Can't get state from portal.\nError: %s", err.Error()),
		)
		return
	}

	if orderState == "deleted" || orderState == "deprovisioned" {
		resp.Diagnostics.AddWarning(
			consts.DELETE_RES_FAIL,
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
			r.client.Organization,
		)

		resp.Diagnostics.AddError(
			consts.DELETE_RES_FAIL,
			fmt.Sprintf(
				"Order with order_id '%s' isn't deletable. "+
					"Please, make your order is deletable by the portal VTB.Cloud, "+
					"and repeat again.\nURL reference: %s",
				order.ID, orderURL,
			),
		)
		return
	}

	err = order.Delete()
	if err != nil {
		resp.Diagnostics.AddError(consts.DELETE_RES_FAIL, err.Error())
		return
	}

}

func changeEtcdExtraMounts(
	order *orders.EtcdOrder,
	planResource *EtcdResourceModel,
) (diags diag.Diagnostics) {
	vmItems, _ := order.GetVMItems()
	for _, vmItem := range vmItems {
		config := vmItem.Data.Config.(entities.VMItemConfig)
		for _, vmExtraMount := range config.ExtraMounts {
			planExtraMountSize := planResource.ExtraMounts[vmExtraMount.Mount+"/"].Size.ValueInt64()
			if planExtraMountSize != int64(vmExtraMount.Size) && planExtraMountSize > int64(vmExtraMount.Size) {
				err := order.ExpandMountPointForVm(vmItem.ID, vmExtraMount.Mount, planExtraMountSize-int64(vmExtraMount.Size))

				if err != nil {
					diags.AddError("Expand mount point error", err.Error())
					return diags
				}
			}
		}
	}
	return diags
}

func changeEtcdFlavor(
	order *orders.EtcdOrder,
	plan *EtcdResourceModel,
) (diags diag.Diagnostics) {
	flavor := entities.Flavor{
		Cores:  plan.Flavor.Cores.ValueInt64(),
		Memory: plan.Flavor.Memory.ValueInt64(),
		Name:   plan.Flavor.Name.ValueString(),
		UUID:   plan.Flavor.UUID.ValueString(),
	}
	vmItems, _ := order.GetVMItems()
	for _, vmItem := range vmItems {
		config := vmItem.Data.Config.(entities.VMItemConfig)
		if config.Flavor != flavor {
			err := order.ChangeFlavorForVm(vmItem.ID, flavor)
			if err != nil {
				diags.AddError("Change flavor for VM Error", err.Error())
				return diags
			}
		}
	}
	return diags
}

func (r EtcdResource) changeAccess(
	order *orders.EtcdOrder,
	plan *EtcdResourceModel,
) (diags diag.Diagnostics) {
	vmItems, err := order.GetVMItems()
	if err != nil {
		diags.AddError(consts.READ_RES_FAIL, err.Error())
		return diags
	}

	for _, vmItem := range vmItems {
		stateAccess := utils.ReadAccessMapVV1(vmItem.Data.ACLs)
		deletedRoles, changedRoles, addedRoles := r.compareADRoles(plan.Access, stateAccess)

		if len(addedRoles) > 0 {
			rolesFromCloud, err := references.GetRoles(
				r.client.Creds,
				r.client.Environment,
				"cluster:etcd",
			)
			if err != nil {
				diags.AddError(consts.UPDATE_RES_FAIL, err.Error())
				return diags
			}

			permissionsMap := make(map[string][]string)
			for _, value := range rolesFromCloud {
				permissionsMap[value.Data.GroupName] = value.Data.Permissions
			}

			for _, roles := range addedRoles {
				err := order.AddAccessGroupForVm(vmItem.ID, roles, permissionsMap[roles.Role])
				if err != nil {
					diags.AddError(consts.UPDATE_RES_FAIL, err.Error())
					return diags
				}
			}
		}

		if len(changedRoles) > 0 {
			for _, roles := range changedRoles {
				err := order.ChangeAccessGroupForVm(vmItem.ID, roles)
				if err != nil {
					diags.AddError(consts.UPDATE_RES_FAIL, err.Error())
					return diags
				}
			}
		}

		if len(deletedRoles) > 0 {
			for _, roles := range deletedRoles {
				err := order.DeleteAccessGroupForVm(vmItem.ID, roles)
				if err != nil {
					diags.AddError(consts.UPDATE_RES_FAIL, err.Error())
					return diags
				}
			}
		}
	}
	return diags
}

func (r EtcdResource) compareADRoles(
	planAccess, stateAccess map[string][]string) (
	deletedRoles, changedRoles, addedRoles []entities.ADLogonGrants) {
	for role, groups := range stateAccess {
		if _, ok := planAccess[role]; !ok {
			deletedRoles = append(
				deletedRoles,
				entities.ADLogonGrants{Role: role, Groups: groups},
			)
		} else {
			if len(planAccess[role]) == len(stateAccess[role]) {
				if utils.DifferenceLen(planAccess[role], stateAccess[role]) {
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
	return
}

func prepareAttrs(
	plan EtcdResourceModel,

) orders.EtcdAttrs {

	ADLogonGrants := []entities.ADLogonGrants{}
	for role, groups := range plan.Access {
		var groupsNames []string
		groupsNames = append(groupsNames, groups...)
		ADLogonGrants = append(ADLogonGrants, entities.ADLogonGrants{
			Role:   role,
			Groups: groupsNames,
		})
	}

	basicAttrs := orders.BasicAttrs{
		ADIntegration:    plan.Image.ADIntegration.ValueBool(),
		ADLogonGrants:    ADLogonGrants,
		Platform:         plan.Core.Platform.ValueString(),
		AvailabilityZone: plan.Core.Zone.ValueString(),
		Domain:           plan.Core.Domain.ValueString(),
		OnSupport:        plan.Image.OnSupport.ValueBool(),
		OsVersion:        plan.Image.OSVersion.ValueString(),
		ExtraMounts:      utils.PrepareExtraMountsAttrs(plan.ExtraMounts),
		DefaultNic: entities.DefaultNic{
			NetSegment: plan.Core.NetSegmentCode.ValueString(),
		},
		Flavor: entities.Flavor{
			Cores:  plan.Flavor.Cores.ValueInt64(),
			Memory: plan.Flavor.Memory.ValueInt64(),
			Name:   string(plan.Flavor.Name.ValueString()),
			UUID:   string(plan.Flavor.UUID.ValueString()),
		},
		CreatedWithOpenTofu: true,
	}
	return orders.EtcdAttrs{
		BasicAttrs:      basicAttrs,
		GeoDistribution: plan.Image.GeoDistribution.ValueBool(),
		OnBackup:        plan.Image.OnBackup.ValueBool(),
		EtcdUser:        plan.EtcdUserName.ValueString(),
		EtcdPassword:    plan.EtcdUserPassword.ValueString(),
		EtcdVersion:     plan.EtcdVersion.ValueString(),
		ClusterName:     plan.ClusterName.ValueString(),
		Layout:          plan.Layout.ValueString(),
		NodesCount:      int(plan.NodesCount.ValueInt64()),
	}
}
