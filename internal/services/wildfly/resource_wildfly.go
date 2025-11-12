package wildfly

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/common"
	"terraform-provider-vtb/internal/consts"
	"terraform-provider-vtb/internal/custommodifires"
	"terraform-provider-vtb/internal/customvalidators"
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
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	// ISO8601Format представляет формат даты и времени в стандарте ISO 8601
	ISO8601Format = "2006-01-02T15:04:05.000Z"
	// DisplayDateFormat представляет формат даты и времени для отображения в main.tf/tfstate файлах.
	DisplayDateFormat = "2006-01-02 15:04"
)

var (
	_ resource.Resource                = WildflyResource{}
	_ resource.ResourceWithImportState = WildflyResource{}
	_ resource.ResourceWithModifyPlan  = WildflyResource{}
)

type WildflyResource struct {
	client *client.CloudClient
}

func NewWildflyResource() resource.Resource {
	return &WildflyResource{}
}

func (r WildflyResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_wildfly_instance"
}

func (r *WildflyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type MmMode struct {
	DateEnd types.String `tfsdk:"date_end"`
}

type WildflyResourceModel struct {
	Lifetime              types.Int64                       `tfsdk:"lifetime"`
	Label                 types.String                      `tfsdk:"label"`
	OrderID               types.String                      `tfsdk:"order_id"`
	ItemID                types.String                      `tfsdk:"item_id"`
	Core                  core.CoreModel                    `tfsdk:"core"`
	Flavor                flavor.FlavorModel                `tfsdk:"flavor"`
	Image                 common.ImageStandardModel         `tfsdk:"image"`
	ActiveDirectoryAccess map[string][]string               `tfsdk:"access"`
	ExtraMounts           map[string]common.ExtraMountModel `tfsdk:"extra_mounts"`
	ServiceStatus         types.String                      `tfsdk:"service_status"`
	Hostname              types.String                      `tfsdk:"hostname"`
	JavaVersion           types.String                      `tfsdk:"java_version"`
	StandaloneType        types.String                      `tfsdk:"standalone_type"`
	WildflyVersion        types.String                      `tfsdk:"wildfly_version"`
	WildflyAccess         map[string][]string               `tfsdk:"wildfly_access"`
	CertAltNames          types.List                        `tfsdk:"cert_alt_names"`
	ClientCert            types.Bool                        `tfsdk:"client_cert"`
	FinancialProject      types.String                      `tfsdk:"financial_project"`
	BuildVersion          types.String                      `tfsdk:"build_version"`
	MmModeEndDate         types.String                      `tfsdk:"mm_mode_end_date"`
}

func (r WildflyResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Управление облачным продуктом Wildfly Astra.",
		Attributes: map[string]schema.Attribute{
			// basic schema
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
				MarkdownDescription: "Идентификатор заказа. Становится известен после создания заказа.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
				Attributes:          common.CoreSchema,
				MarkdownDescription: "Основные параметры для вирутальных машин в заказе.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"flavor": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.FlavorSchema,
				MarkdownDescription: "Кол-во CPU/RAM для виртуальных машин.",
			},
			"image": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.ImageStandardSchema,
				MarkdownDescription: "Тип вычислительного экземпляра.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
			},
			"access": schema.MapAttribute{
				Required: true,
				Description: "Карта, где ключом является роль, а значением - список групп, " +
					"который предоставит доступ для входа в Active Directory",
				MarkdownDescription: "Карта, где ключом является роль, а значением - список групп, " +
					"который предоставит доступ для входа в Active Directory",
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
				Validators: []validator.Map{
					customvalidators.EmptyAccessGroupListValidator{},
				},
			},
			"extra_mounts": schema.MapNestedAttribute{
				MarkdownDescription: "Дополнительные точки монтирования для ВМ.",
				Required:            true,
				Validators: []validator.Map{
					mapvalidator.KeysAre(
						stringvalidator.OneOf(
							"/app/app",
							"/app/logs",
						),
					),
					mapvalidator.SizeBetween(2, 2),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"size": schema.Int64Attribute{
							MarkdownDescription: "Размер точки монтирования (в ГБ.)",
							Required:            true,
							Validators: []validator.Int64{
								int64validator.AtMost(2048),
								int64validator.AtLeast(10),
							},
							PlanModifiers: []planmodifier.Int64{
								custommodifires.GrowOnly(),
								custommodifires.MininalGrow(10),
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

			// extra product schema
			"service_status": schema.StringAttribute{
				MarkdownDescription: "Статус обслуживания ",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("on", "off"),
				},
			},
			"wildfly_version": schema.StringAttribute{
				MarkdownDescription: "Версия Wildfly",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"java_version": schema.StringAttribute{
				MarkdownDescription: "Версия Java",
				Required:            true,
			},
			"hostname": schema.StringAttribute{
				MarkdownDescription: "Имя хоста виртуальной машины",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"standalone_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Конфигурация файла stanadalone.xml type (standard, full)",
				Validators: []validator.String{
					stringvalidator.OneOf("standard", "full"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"wildfly_access": schema.MapAttribute{
				MarkdownDescription: "Выберите роли для групп разработчиков",
				Required:            true,
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
				Validators: []validator.Map{
					customvalidators.EmptyAccessGroupListValidator{},
				},
			},

			"cert_alt_names": schema.ListAttribute{
				ElementType:         types.StringType,
				Description:         "Альтернативные имена в сертификате.",
				MarkdownDescription: "Альтернативные имена в сертификате.",
				Optional:            true,
			},
			"client_cert": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				Description:         "Клиенсткий сертификат",
				MarkdownDescription: "Клиенсткий сертификат",
			},
			"mm_mode_end_date": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Дата окончания ММ.",
				Description:         "Дата окончания ММ.",
			},

			"build_version": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Версия релиза продукта",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"financial_project": schema.StringAttribute{
				Required:            true,
				Description:         "Источник финансирования.",
				MarkdownDescription: "Источник финансирования.",
			},
		},
	}
}

func (r WildflyResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r WildflyResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	var plan, state WildflyResourceModel

	if req.Plan.Raw.IsNull() {
		return
	}

	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	}

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	checkOrderIsDeleted := utils.CheckOrderIsDeleted(
		r.client,
		plan.OrderID,
		"app",
		"wildfly",
	)
	resp.Diagnostics.Append(checkOrderIsDeleted.Diagnostics...)

	utils.HandleExternalDeletionRecreate(
		ctx,
		resp,
		&plan,
		checkOrderIsDeleted.IsDeleted,
		checkOrderIsDeleted.Diagnostics,
	)

	validateAccessRolesDiags := utils.ValidateAccessRolesV1(
		r.client,
		"app:wildfly",
		plan.ActiveDirectoryAccess,
	)
	resp.Diagnostics.Append(validateAccessRolesDiags...)

	r.validateWildflyAndJavaVesion(&plan, resp)
	r.validateManagmentGroup(&plan, resp)
	r.validateMmModeDate(&plan, &state, resp)

	if resp.Diagnostics.HasError() {
		return
	}
}

func (r WildflyResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan WildflyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wildflyAccessGroup, attrs := r.prepareAtts(&plan)

	finProj, err := sources.GetFinancialProjectByName(
		r.client.Creds,
		r.client.ProjectName,
		plan.FinancialProject.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("financial_project"), consts.CREATE_RES_FAIL, err.Error())
		return
	}

	order := orders.NewWildfly(
		r.client.Creds,
		r.client.ProjectName,
		plan.Image.ProductID.ValueString(),
		attrs,
	)

	err = order.Create(orders.CreateOrderPayload{
		Label:        plan.Label.ValueString(),
		Lifetime:     int(plan.Lifetime.ValueInt64()),
		FinProjectID: finProj.ID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Can't create virtual machine: ", err.Error())
		return
	}

	orderItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError("Error while getting item: ", err.Error())
		return
	}

	vmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError("Error while update state: ", err.Error())
		return
	}

	if len(vmItems) != 1 {
		resp.Diagnostics.AddError(
			"Error while updating state:",
			"Length vmItems more then 1",
		)
		return
	}

	itemData := vmItems[0].Data
	config := itemData.Config.(entities.VMItemConfig)

	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(vmItems[0].ID)
	plan.Hostname = types.StringValue(config.Hostname)
	plan.BuildVersion = types.StringValue(orderItem.Data.Build.SetupVersion)

	if !plan.CertAltNames.IsNull() {
		r.updateCert(ctx, &plan, order, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	if !plan.ClientCert.Equal(types.BoolValue(false)) {
		r.SwitchClientCert(order, &plan, &resp.Diagnostics)
	}

	if !plan.MmModeEndDate.IsNull() {
		r.SwitchMmMode(order, &plan, &resp.Diagnostics)
	}

	applyManagmentGroup(wildflyAccessGroup, order, &plan, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.ServiceStatus.Equal(types.StringValue("off")) {
		r.SwitchServiceState(order, &plan, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}

	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r WildflyResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var orderID types.String
	req.State.GetAttribute(ctx, path.Root("order_id"), &orderID)
	if orderID.IsNull() {
		resp.Diagnostics.AddError(
			"Read order from portal",
			"Resource hasn't order_id. Without order_id can't get order from portal",
		)
		return
	}

	order, err := orders.GetWildflyOrder(
		r.client.Creds,
		r.client.ProjectName,
		orderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Read order from portal",
			fmt.Sprintf(
				"Can't read order with order_id equal '%s'.\nError:%s",
				orderID.ValueString(), err.Error(),
			),
		)
		return
	}

	wildflyItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(
			"Read order from portal",
			fmt.Sprintf("Can't get wildfly item.\nError: %s", err.Error()),
		)
		return
	}

	wildflyVMs, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			"Read order from portal",
			fmt.Sprintf("Can't get wildfly VM item.\nError: %s", err.Error()),
		)
		return
	}

	appApp, err := order.GetExtraMount("/app/app")
	if err != nil {
		resp.Diagnostics.AddError(
			"Read order from portal",
			fmt.Sprintf(
				"Can't get vm extra mount '/app/app' from portal with order_id '%s'.\nError:%s",
				orderID, err.Error(),
			),
		)
		return
	}

	appLogs, err := order.GetExtraMount("/app/logs")
	if err != nil {
		resp.Diagnostics.AddError(
			"Read order from portal",
			fmt.Sprintf(
				"Can't get vm extra mount '/app/logs' from portal with order_id '%s'.\nError:%s",
				orderID, err.Error(),
			),
		)
		return
	}

	vmItem := wildflyVMs[0]
	vmConfig := vmItem.Data.Config.(entities.VMItemConfig)
	wildflyConfig := wildflyItem.Data.Config.(entities.WildflyItemConfig)

	var certAltNames basetypes.ListValue
	if len(wildflyConfig.Certificate.AltNames) == 0 {
		certAltNames = types.ListNull(types.StringType)
	} else {
		certAltNames, _ = types.ListValueFrom(ctx, types.StringType, wildflyConfig.Certificate.AltNames)
	}
	var mmModeEndDate basetypes.StringValue
	if wildflyItem.Data.MmMode.DateEnd == "" {
		mmModeEndDate = types.StringNull()
	} else {
		formattedDate, err := prepareStateDateFormat(wildflyItem.Data.MmMode.DateEnd)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("mm_mode_end_date"),
				consts.READ_RES_FAIL,
				fmt.Sprintf("Error while parsing mm_mode_end_date: %s", err.Error()),
			)
		}
		mmModeEndDate = types.StringValue(formattedDate)
	}

	state := WildflyResourceModel{
		OrderID:               orderID,
		ActiveDirectoryAccess: utils.ReadAccessMapVV1(vmItem.Data.ACLs),
		ServiceStatus:         types.StringValue(wildflyItem.Data.State),
		WildflyVersion:        types.StringValue(wildflyConfig.WildflyVersion),
		JavaVersion:           types.StringValue(wildflyConfig.JavaVersion),
		StandaloneType:        types.StringValue(wildflyConfig.StandaloneType),
		Label:                 types.StringValue(order.Label),
		ItemID:                types.StringValue(vmItem.ID),
		Hostname:              types.StringValue(vmConfig.Hostname),
		BuildVersion:          types.StringValue(wildflyItem.Data.Build.SetupVersion),
		MmModeEndDate:         mmModeEndDate,
		Flavor: flavor.FlavorModel{
			Cores:  types.Int64Value(vmConfig.Flavor.Cores),
			Memory: types.Int64Value(vmConfig.Flavor.Memory),
			UUID:   types.StringValue(vmConfig.Flavor.UUID),
			Name:   types.StringValue(vmConfig.Flavor.Name),
		},
		ExtraMounts: map[string]common.ExtraMountModel{
			"/app/app": {
				Size:       types.Int64Value(appApp.Size),
				FileSystem: types.StringValue(appApp.FileSystem),
			},
			"/app/logs": {
				Size:       types.Int64Value(appLogs.Size),
				FileSystem: types.StringValue(appLogs.FileSystem),
			},
		},
		Core: core.CoreModel{
			Platform:       types.StringValue(utils.SelectPlatform(order.Attrs.Platform)),
			Domain:         types.StringValue(vmConfig.Domain),
			NetSegmentCode: types.StringValue(vmConfig.DefaultNic.NetSegment),
			Zone:           types.StringValue(order.Attrs.AvailabilityZone),
		},
		Image: common.ImageStandardModel{
			Distribution:  types.StringValue(vmConfig.Image.Os.Distribution),
			OsVersion:     types.StringValue(vmConfig.Image.Os.Version),
			OnSupport:     types.BoolValue(vmConfig.OnSupport),
			ProductID:     types.StringValue(order.ProductID),
			ADIntegration: types.BoolValue(vmConfig.AdIntegration),
		},
		FinancialProject: types.StringValue(order.FinancialSource.Name),
	}

	// read info about wildfly access groups
	var wildflyAccess = make(map[string][]string)
	for _, group := range wildflyConfig.Group {
		if _, ok := wildflyAccess[group.Role]; !ok {
			wildflyAccess[group.Role] = make([]string, 0)
		}
		wildflyAccess[group.Role] = append(wildflyAccess[group.Role], group.Name)
	}
	state.WildflyAccess = wildflyAccess

	// Read info about cert alt names
	cert := wildflyConfig.Certificate
	clientCert := types.BoolValue(cert.ClientCert)
	state.CertAltNames = certAltNames
	state.ClientCert = clientCert

	var lifetime types.Int64
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("lifetime"), &lifetime)...)
	if !lifetime.IsNull() {
		state.Lifetime = lifetime
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r WildflyResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state WildflyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetWildflyOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Can't get order from portal: ", err.Error())
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

	labelChanged := plan.Label != state.Label
	mountChanged := utils.IsExtraMountChanged(state.ExtraMounts, plan.ExtraMounts)
	flavorChanged := plan.Flavor != state.Flavor
	javaVersionChanged := plan.JavaVersion != state.JavaVersion
	accessChanged := !reflect.DeepEqual(plan.ActiveDirectoryAccess, state.ActiveDirectoryAccess)
	managmentGroupChanged := !reflect.DeepEqual(plan.WildflyAccess, state.WildflyAccess)
	altNamesChanged := !plan.CertAltNames.Equal(state.CertAltNames)
	clientCertChanged := !plan.ClientCert.Equal(state.ClientCert)
	mmModeChanged := !plan.MmModeEndDate.Equal(state.MmModeEndDate)

	if labelChanged {
		utils.ChangeOrderLabel(order, plan.Label.ValueString(), resp)
		resp.State.SetAttribute(ctx, path.Root("label"), plan.Label.ValueString())
	}

	if javaVersionChanged {
		resp.Diagnostics.Append(changeJavaVerison(order, &plan)...)
	}

	if managmentGroupChanged {
		resp.Diagnostics.Append(changeWildflyManagmentGroups(order, &state, &plan)...)
	}

	if mountChanged {
		resp.Diagnostics.Append(changeExtraMountsWildfly(order, &plan)...)
	}

	if accessChanged {
		resp.Diagnostics.Append(r.changeAccess(order, &state, &plan)...)
	}

	if flavorChanged {
		resp.Diagnostics.Append(verticalScalingWildfly(order, &plan)...)
	}

	if altNamesChanged {
		r.updateCert(ctx, &plan, order, &resp.Diagnostics)
	}

	if clientCertChanged {
		r.SwitchClientCert(order, &plan, &resp.Diagnostics)
	}

	if mmModeChanged {
		r.SwitchMmMode(order, &plan, &resp.Diagnostics)
	}

	if !plan.ServiceStatus.Equal(state.ServiceStatus) {
		r.SwitchServiceState(order, &plan, &resp.Diagnostics)
	}

	if !plan.FinancialProject.Equal(state.FinancialProject) {
		order.ChangeFinancialProject(finProj.ID)
	}

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r WildflyResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state WildflyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetWildflyOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Can't get order from portal: ", err.Error())
	}

	orderState, err := order.GetState()
	if err != nil {
		resp.Diagnostics.AddError(
			"Delete resource",
			fmt.Sprintf("Can't get state from portal.\nError:%s", err.Error()),
		)
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
			r.client.Organization,
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
	if err := order.DeleteTwoLayer(false); err != nil {
		resp.Diagnostics.AddError("While process delete VM: ", err.Error())
		return
	}
}

// * Custom logic *

func (r WildflyResource) validateWildflyAndJavaVesion(
	plan *WildflyResourceModel,
	resp *resource.ModifyPlanResponse,
) {
	FlyJavaVersions, err := references.GetWildflyAndJavaVersions(r.client.Creds)
	if err != nil {
		resp.Diagnostics.AddError(
			"Get wildfly's versions from portal", err.Error(),
		)
		return
	}

	for _, item := range FlyJavaVersions {
		if item.WildflyVersion == plan.WildflyVersion.ValueString() {
			for _, version := range item.JavaVersions {
				if version == plan.JavaVersion.ValueString() {
					return
				}
			}
		}
	}

	resp.Diagnostics.AddError(
		"Wildfly version not available for resource",
		fmt.Sprintf(
			"Available wildfly versions with supported java versions [%s].",
			FlyJavaVersions,
		),
	)
}

func (r WildflyResource) validateManagmentGroup(
	plan *WildflyResourceModel,
	resp *resource.ModifyPlanResponse,
) {
	availableRoles, err := references.GetWildflyManagmentRoles(
		r.client.Creds,
		r.client.Environment,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Get wildfly's managment roles from portal",
			err.Error(),
		)
		return
	}

	for role := range plan.WildflyAccess {
		if !slices.Contains(availableRoles, role) {

			resp.Diagnostics.AddAttributeError(
				path.Root("wildfly_access").AtMapKey(role),
				"Get wildfly's management roles from portal",
				fmt.Sprintln(
					"The following role are unavailable or don't exist."+
						"\nAvailable roles is:", strings.Join(availableRoles, ", "),
				),
			)
		}
	}
}

func (r WildflyResource) updateCert(
	ctx context.Context,
	plan *WildflyResourceModel,
	order *orders.Wildfly,
	diags *diag.Diagnostics,
) {
	altNames := []string{}
	if plan.CertAltNames.IsNull() {
		err := order.UpdateCert(altNames)
		if err != nil {
			diags.AddAttributeError(
				path.Root("cert_alt_names"),
				"Cant't perform action update_cert",
				err.Error(),
			)
		}
	}
	plan.CertAltNames.ElementsAs(ctx, &altNames, true)
	if len(altNames) > 0 {
		err := order.UpdateCert(altNames)
		if err != nil {
			diags.AddAttributeError(
				path.Root("cert_alt_names"),
				"Can't perform action  update_cert",
				err.Error(),
			)
		}
	}
}

func (r WildflyResource) SwitchServiceState(
	order *orders.Wildfly,
	plan *WildflyResourceModel,
	diags *diag.Diagnostics,
) {
	if plan.ServiceStatus.ValueString() == "off" {
		err := order.StopService()
		if err != nil {
			diags.AddError("Change wildfly service state to stop", err.Error())
		}
		return
	}

	if plan.ServiceStatus.ValueString() == "on" {
		err := order.StartService()
		if err != nil {
			diags.AddError("Change wildfly service state to start", err.Error())
		}
		return
	}
}

func (r WildflyResource) SwitchClientCert(
	order *orders.Wildfly,
	plan *WildflyResourceModel,
	diags *diag.Diagnostics,
) {
	if plan.ClientCert.Equal(types.BoolValue(true)) {
		err := order.AddClientCert()
		if err != nil {
			diags.AddError(
				consts.UPDATE_RES_FAIL,
				fmt.Sprintf("Add client cert ended with error: %s", err.Error()),
			)
		}
	}

	if plan.ClientCert.Equal(types.BoolValue(false)) || plan.ClientCert.IsNull() {
		err := order.DeleteClientCert()
		if err != nil {
			diags.AddError(
				consts.UPDATE_RES_FAIL,
				fmt.Sprintf("Delete client cert ended with error: %s", err.Error()),
			)
		}
	}
}

func (r WildflyResource) SwitchMmMode(
	order *orders.Wildfly,
	plan *WildflyResourceModel,
	diags *diag.Diagnostics,
) {
	if !plan.MmModeEndDate.IsNull() {
		endDate, err := preparePlanDateFormat(plan.MmModeEndDate.ValueString())
		if err != nil {
			diags.AddAttributeError(
				path.Root("mm_mode_end_date"),
				"Invalid input date, error while parsing date",
				err.Error(),
			)
			return
		}
		err = order.SetMaintanceModeOn(endDate)
		if err != nil {
			diags.AddError(
				consts.UPDATE_RES_FAIL,
				fmt.Sprintf(`Set maintance mode "on" ended with error: %s`, err.Error()),
			)
			return
		}
	}

	if plan.MmModeEndDate.IsNull() {
		err := order.SetMaintanceModeOff()
		if err != nil {
			diags.AddError(
				consts.UPDATE_RES_FAIL,
				fmt.Sprintf(`Set maintance mode "off" ended with error: %s`, err.Error()),
			)
			return
		}
	}
}

func (r WildflyResource) prepareAtts(plan *WildflyResourceModel) (string, orders.WildflyAttrs) {

	var ADLogonGrants []entities.ADLogonGrants
	for role, groups := range plan.ActiveDirectoryAccess {
		var groupsNames []string
		groupsNames = append(groupsNames, groups...)
		ADLogonGrants = append(ADLogonGrants, entities.ADLogonGrants{
			Role:   role,
			Groups: groupsNames,
		})
	}

	var wildflyAccessGroups string
	for _, groups := range plan.WildflyAccess {
		if len(groups) > 0 {
			wildflyAccessGroups = groups[0]
			break
		}
	}

	devOpsGroupName := fmt.Sprintf("cloud-%s-wildfly-devops-%s", r.client.RisShortName, r.client.EnvPrefix)

	attrs := orders.WildflyAttrs{
		BasicAttrs: orders.BasicAttrs{
			ADLogonGrants:    ADLogonGrants,
			ADIntegration:    plan.Image.ADIntegration.ValueBool(),
			ExtraMounts:      utils.PrepareExtraMountsAttrs(plan.ExtraMounts),
			AvailabilityZone: plan.Core.Zone.ValueString(),
			Domain:           plan.Core.Domain.ValueString(),
			Flavor: entities.Flavor{
				Cores:  plan.Flavor.Cores.ValueInt64(),
				Memory: plan.Flavor.Memory.ValueInt64(),
				Name:   string(plan.Flavor.Name.ValueString()),
				UUID:   string(plan.Flavor.UUID.ValueString()),
			},
			Platform:  plan.Core.Platform.ValueString(),
			OnSupport: plan.Image.OnSupport.ValueBool(),
			OsVersion: plan.Image.OsVersion.ValueString(),
			DefaultNic: entities.DefaultNic{
				NetSegment: plan.Core.NetSegmentCode.ValueString(),
			},
			CreatedWithOpenTofu: true,
		},
		ExtraNics:       []interface{}{},
		StandaloneType:  plan.StandaloneType.ValueString(),
		AccessGroups:    []string{wildflyAccessGroups},
		JavaVersion:     plan.JavaVersion.ValueString(),
		WildflyVersion:  plan.WildflyVersion.ValueString(),
		InfoAd:          map[string]interface{}{},
		InfoVm:          map[string]interface{}{},
		InfoOs:          map[string]interface{}{},
		InfoDevOps:      map[string]interface{}{},
		InfoAccessGroup: map[string]interface{}{},
		DevopsGroup:     devOpsGroupName,
	}
	return wildflyAccessGroups, attrs
}

func (r WildflyResource) validateMmModeDate(
	plan, state *WildflyResourceModel,
	resp *resource.ModifyPlanResponse,
) {
	if !plan.MmModeEndDate.IsNull() {
		parsePlanDate, err := time.Parse(time.DateTime, string(plan.MmModeEndDate.ValueString())+":00")
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("mm_mode_date_end"),
				consts.MODIFY_PLAN_FAIL,
				fmt.Sprintf("Error while parsing date: %s", err.Error()),
			)
		}

		if parsePlanDate.UTC().Before(time.Now().UTC()) || parsePlanDate.Equal(time.Now().UTC()) {
			resp.Diagnostics.AddAttributeError(
				path.Root("mm_mode_date_end"),
				consts.MODIFY_PLAN_FAIL,
				fmt.Sprintln("mm_mode_end_date, must be a greather current date."),
			)
		}
	}

	if !state.MmModeEndDate.IsNull() && !plan.MmModeEndDate.IsNull() && !plan.MmModeEndDate.Equal(state.MmModeEndDate) {
		resp.Diagnostics.AddAttributeWarning(
			path.Root("mm_mode_date_end"),
			consts.MODIFY_PLAN_FAIL,
			fmt.Sprintln("MM mode already is active. For update mm_mode end date, delete current."),
		)
	}
}

func verticalScalingWildfly(order *orders.Wildfly, plan *WildflyResourceModel) (diags diag.Diagnostics) {
	flavor := entities.Flavor{
		Cores:  plan.Flavor.Cores.ValueInt64(),
		Memory: plan.Flavor.Memory.ValueInt64(),
		UUID:   plan.Flavor.UUID.ValueString(),
		Name:   plan.Flavor.Name.ValueString(),
	}
	err := order.VerticalScaling(flavor)
	if err != nil {
		diags.AddError("Changing VM flavor", err.Error())
	}
	return
}

func changeExtraMountsWildfly(order *orders.Wildfly, plan *WildflyResourceModel) (diags diag.Diagnostics) {
	for path, planExtraMount := range plan.ExtraMounts {
		err := order.ExpandMountPoint(entities.ExtraMount{
			Path:       path,
			Size:       planExtraMount.Size.ValueInt64(),
			FileSystem: planExtraMount.FileSystem.ValueString(),
		})
		if err != nil {
			diags.AddError("Incorrent changes in extra_mounts parameters", err.Error())
		}
	}
	return
}

func changeJavaVerison(order *orders.Wildfly, plan *WildflyResourceModel) (diags diag.Diagnostics) {

	wildflyVersion := plan.WildflyVersion.ValueString()
	javaVersion := plan.JavaVersion.ValueString()

	err := order.ChangeJavaVersion(wildflyVersion, javaVersion)
	if err != nil {
		diags.AddError("Cant change java version", err.Error())
	}
	return
}

func (r WildflyResource) changeAccess(
	order *orders.Wildfly,
	state,
	plan *WildflyResourceModel,
) (diags diag.Diagnostics) {

	addedRoles, changedRoles, deletedRoles := r.compareADRoles(state, plan)

	if len(addedRoles) > 0 {
		rolesFromCloud, err := references.GetRoles(
			r.client.Creds,
			r.client.Environment,
			"app:wildfly",
		)
		if err != nil {
			diags.AddError("Receiving roles from Cloud", err.Error())
			return
		}

		permissionsMap := make(map[string][]string)
		for _, value := range rolesFromCloud {
			permissionsMap[value.Data.GroupName] = value.Data.Permissions
		}

		for _, roles := range addedRoles {
			err := order.AddAccessGroup(roles, permissionsMap[roles.Role])
			if err != nil {
				diags.AddError("Adding new VM roles", err.Error())
			}
		}
	}

	if len(changedRoles) > 0 {
		for _, roles := range changedRoles {
			err := order.ChangeAccessGroup(roles)
			if err != nil {
				diags.AddError("Changing groups in VM roles", err.Error())
				return
			}
		}
	}

	if len(deletedRoles) > 0 {
		for _, roles := range deletedRoles {
			err := order.DeleteAccessGroup(roles)
			if err != nil {
				diags.AddError("Deleting VM roles", err.Error())
			}
		}
	}
	return
}

func (r WildflyResource) compareADRoles(
	state, plan *WildflyResourceModel,
) (addedRoles, changedRoles, deletedRoles []entities.ADLogonGrants) {
	for role, groups := range state.ActiveDirectoryAccess {
		if _, ok := plan.ActiveDirectoryAccess[role]; !ok {
			deletedRoles = append(
				deletedRoles,
				entities.ADLogonGrants{Role: role, Groups: groups},
			)
		} else {
			if len(plan.ActiveDirectoryAccess[role]) == len(state.ActiveDirectoryAccess[role]) {
				if utils.DifferenceLen(plan.ActiveDirectoryAccess[role], state.ActiveDirectoryAccess[role]) {
					changedRoles = append(
						changedRoles,
						entities.ADLogonGrants{Role: role, Groups: plan.ActiveDirectoryAccess[role]},
					)
				}
			} else {
				changedRoles = append(
					changedRoles,
					entities.ADLogonGrants{Role: role, Groups: plan.ActiveDirectoryAccess[role]},
				)
			}
		}
	}

	for role, groups := range plan.ActiveDirectoryAccess {
		if _, ok := state.ActiveDirectoryAccess[role]; !ok {
			addedRoles = append(
				addedRoles,
				entities.ADLogonGrants{Role: role, Groups: groups},
			)
		}
	}
	return addedRoles, changedRoles, deletedRoles
}

func changeWildflyManagmentGroups(
	order *orders.Wildfly,
	state,
	plan *WildflyResourceModel,
) (diags diag.Diagnostics) {

	deletedRoles, addedRoles := compareWildflyGroups(state, plan)

	if len(deletedRoles) > 0 {
		for role, groups := range deletedRoles {
			for _, group := range groups {
				err := order.DeleteManagmentGroup(role, group)
				if err != nil {
					diags.AddError("Deleting wildfly managment access", err.Error())
				}
			}
		}
	}

	if len(addedRoles) > 0 {
		for role, groups := range addedRoles {
			for _, group := range groups {
				err := order.AddManagmentGroup(role, group)
				if err != nil {
					diags.AddError("Adding wildfly managment access", err.Error())
				}
			}
		}
	}
	return
}

func compareWildflyGroups(
	state, plan *WildflyResourceModel,
) (deletedRoles, addedRoles map[string][]string) {

	deletedRoles = make(map[string][]string)
	addedRoles = make(map[string][]string)

	for role, groups := range state.WildflyAccess {
		if _, ok := plan.WildflyAccess[role]; !ok {
			deletedRoles[role] = groups
		} else {
			for _, groupState := range groups {
				if !slices.Contains(plan.WildflyAccess[role], groupState) {
					deletedRoles[role] = append(deletedRoles[role], groupState)
				}
			}
			for _, groupPlan := range plan.WildflyAccess[role] {
				if !slices.Contains(state.WildflyAccess[role], groupPlan) {
					addedRoles[role] = append(addedRoles[role], groupPlan)
				}
			}
		}
	}

	for role, group := range plan.WildflyAccess {
		if _, ok := state.WildflyAccess[role]; !ok {
			addedRoles[role] = group
		}
	}
	return deletedRoles, addedRoles
}

func applyManagmentGroup(
	accessGroup string,
	order *orders.Wildfly,
	plan *WildflyResourceModel,
	resp *resource.CreateResponse,
) {
	wildflyRole := ""
	switch plan.Core.Domain.ValueString() {
	case "region.vtb.ru":
		wildflyRole = "Deployer"
	case "test.vtb.ru":
		wildflyRole = "Maintainer"
	case "corp.dev.vtb":
		wildflyRole = "SuperUser"
	}

	err := order.DeleteManagmentGroup(wildflyRole, accessGroup)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf(
				"Error when deleted wildfly managment access for %s:%s",
				wildflyRole, accessGroup,
			),
			err.Error(),
		)
		return
	}

	for role, groups := range plan.WildflyAccess {
		for _, group := range groups {
			err := order.AddManagmentGroup(role, group)
			if err != nil {
				resp.Diagnostics.AddError(
					fmt.Sprintf(
						"Error when added managment access for wildfly server for %s:%s",
						role, group,
					),
					err.Error(),
				)
			}
		}
	}
}

func preparePlanDateFormat(date string) (string, error) {
	parsedDate, err := time.Parse(DisplayDateFormat, date)
	if err != nil {
		return "", err
	}
	parsedDate = parsedDate.UTC()
	parsedDate = parsedDate.Add(-3 * time.Hour)
	formattedDate := parsedDate.Format(ISO8601Format)
	return formattedDate, nil
}

func prepareStateDateFormat(date string) (string, error) {
	parsedDate, err := time.Parse(ISO8601Format, date)
	if err != nil {
		return "", err
	}
	parsedDate = parsedDate.UTC()
	parsedDate = parsedDate.Add(3 * time.Hour)
	formattedDate := parsedDate.Format(DisplayDateFormat)
	return formattedDate, nil
}
