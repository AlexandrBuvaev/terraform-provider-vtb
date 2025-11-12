package s3ceph

import (
	"context"
	"fmt"

	"regexp"
	"strings"

	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/consts"
	"terraform-provider-vtb/internal/utils"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"

	"terraform-provider-vtb/pkg/client/sources"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &S3CephResource{}
	_ resource.ResourceWithImportState = &S3CephResource{}
	_ resource.ResourceWithModifyPlan  = &S3CephResource{}
)

type S3CephResource struct {
	client *client.CloudClient
}

func NewS3CephResource() resource.Resource {
	return &S3CephResource{}
}

func (r S3CephResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_s3_ceph_instance"
}

func (r *S3CephResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type BucketModel struct {
	Versioning types.Bool  `tfsdk:"versioning"`
	MaxSizeGb  types.Int64 `tfsdk:"max_size_gb"`
}

type S3UserModel struct {
	AccessKey types.String `tfsdk:"access_key"`
	SecretKey types.String `tfsdk:"secret_key"`
}

type S3CephResourceModel struct {
	Lifetime   types.Int64  `tfsdk:"lifetime"`
	Label      types.String `tfsdk:"label"`
	ItemID     types.String `tfsdk:"item_id"`
	OrderID    types.String `tfsdk:"order_id"`
	Zone       types.String `tfsdk:"zone"`
	NetSegment types.String `tfsdk:"net_segment"`

	Image            S3CephImageDataSourceData `tfsdk:"image"`
	UserEndpoint     types.String              `tfsdk:"user_endpoint"`
	MtlsEndpoint     types.String              `tfsdk:"mtls_endpoint"`
	Buckets          map[string]BucketModel    `tfsdk:"buckets"`
	Users            map[string]S3UserModel    `tfsdk:"users"`
	FinancialProject types.String              `tfsdk:"financial_project"`
}

func (r S3CephResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "S3 Ceph Resource",
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
				Description:         "Метка заказа.",
				MarkdownDescription: "Метка заказа.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"order_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Идентификатор заказа, известен после создания заказа",
				MarkdownDescription: "Идентификатор заказа, известен после создания заказа",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"item_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Идентфикатор сущностей инстанса, известен после создания инстанса",
				MarkdownDescription: "Идентфикатор сущностей инстанса, известен после создания инстанса",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			// image schema
			"image": schema.SingleNestedAttribute{
				Required:            true,
				Description:         "Тип инстанса S3 Ceph",
				MarkdownDescription: "Тип инстанса S3 Ceph",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"storage_type": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Тип хранилища S3 Ceph",
						Validators: []validator.String{
							stringvalidator.OneOf("hdd", "nvme", "backup"),
						},
					},
					"product_id": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "ID продукта из продуктового каталога.",
					},
				},
			},
			"net_segment": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"zone": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			// extra product schema
			"user_endpoint": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "User Endpoint заказанного инстанса S3 Ceph",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mtls_endpoint": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "mTLS Endpoint заказанного инстанса S3 Ceph",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"buckets": schema.MapNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Список бакетов для добавления в инстанс S3 Ceph",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"max_size_gb": schema.Int64Attribute{
							Required:            true,
							MarkdownDescription: "Размер бакета в гигабайтах",
							Validators: []validator.Int64{
								int64validator.AtLeast(1),
							},
						},
						"versioning": schema.BoolAttribute{
							Required:            true,
							MarkdownDescription: "Версионирование",
						},
					},
				},
			},
			"users": schema.MapNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Список пользователей для добавления в инстанс S3 Ceph",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"access_key": schema.StringAttribute{
							Required:            true,
							Sensitive:           true,
							MarkdownDescription: "Поле Access key пользователя",
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									regexp.MustCompile(`[A-Z]`),
									"Пароль должен содержать минимум один символ верхнего регистра"),
								stringvalidator.RegexMatches(
									regexp.MustCompile(`[A-Z]`),
									"Пароль должен содержать минимум один символ верхнего регистра"),
								stringvalidator.RegexMatches(
									regexp.MustCompile(`[0-9]`),
									"Пароль должен содержать минимум одну цифру"),
								stringvalidator.RegexMatches(
									regexp.MustCompile(`^[A-Z0-9]{20,30}$`),
									"Пароль должен соответствовать паттерну ^[A-Z0-9]{20,30}$",
								),
							},
						},
						"secret_key": schema.StringAttribute{
							Required:            true,
							Sensitive:           true,
							MarkdownDescription: "Поле Secret key пользователя",
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									regexp.MustCompile(`[a-z]`),
									"Пароль должен содержать минимум один символ нижнего регистра"),
								stringvalidator.RegexMatches(
									regexp.MustCompile(`[A-Z]`),
									"Пароль должен содержать минимум один символ верхнего регистра"),
								stringvalidator.RegexMatches(
									regexp.MustCompile(`[0-9]`),
									"Пароль должен содержать минимум одну цифру"),
								stringvalidator.RegexMatches(
									regexp.MustCompile(`^[a-zA-Z0-9]{40,50}$`),
									"Пароль должен соответствовать паттерну ^[a-zA-Z0-9]{40,50}$",
								),
							},
						},
					}},
			},
			"financial_project": schema.StringAttribute{
				Required:            true,
				Description:         "Источник финансирования.",
				MarkdownDescription: "Источник финансирования.",
			},
		},
	}
}

func (r S3CephResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r S3CephResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan, state S3CephResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	checkOrderIsDeleted := utils.CheckOrderIsDeleted(
		r.client,
		plan.OrderID,
		"s3",
		"ceph",
	)
	resp.Diagnostics.Append(checkOrderIsDeleted.Diagnostics...)

	utils.HandleExternalDeletionRecreate(
		ctx,
		resp,
		&plan,
		checkOrderIsDeleted.IsDeleted,
		checkOrderIsDeleted.Diagnostics,
	)

	if r.client.Environment != "PROD" && plan.Image.StorageType.ValueString() != "hdd" {
		resp.Diagnostics.AddAttributeError(
			path.Root("storage_type"),
			"Invalid storage type for current environment",
			"only `hdd` storage type may be choosen for non PROD envs",
		)
		return
	}

	r.usersNameModifyPlan(&plan, resp)
	r.bucketNameModifyPlan(&plan, resp)

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r S3CephResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan S3CephResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order := orders.NewS3CephOrder(
		r.client.Creds,
		r.client.ProjectName,
		plan.Image.ProductID.ValueString(),
		r.prepareAttrs(&plan),
	)

	var finProjectID string
	finProj, err := sources.GetFinancialProjectByName(
		r.client.Creds,
		r.client.ProjectName,
		plan.FinancialProject.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("financial_project"), consts.CREATE_RES_FAIL, err.Error())
		return
	}
	finProjectID = finProj.ID

	err = order.Create(orders.CreateOrderPayload{
		Label:        plan.Label.ValueString(),
		Lifetime:     int(plan.Lifetime.ValueInt64()),
		FinProjectID: finProjectID,
	})
	if err != nil {
		resp.Diagnostics.AddError("Can't order: ", err.Error())
		return
	}

	err = order.Sync()
	if err != nil {
		resp.Diagnostics.AddError("Can't order ", err.Error())
		return
	}

	orderItem, err := order.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError("Error while get item of order", err.Error())
		return
	}

	err = order.Sync()
	if err != nil {
		resp.Diagnostics.AddError("Error while update state: ", err.Error())
		return
	}

	//create buckets for S3 Ceph instance
	if plan.Buckets != nil {
		for name, bucket := range plan.Buckets {
			err := order.CreateBucket(
				name,
				bucket.MaxSizeGb.ValueInt64(),
				bucket.Versioning.ValueBool(),
			)
			if err != nil {
				resp.Diagnostics.AddError("Error while create bucket", err.Error())
			}
		}
	}

	//create users for S3 Ceph instance
	if plan.Users != nil {
		for userName, user := range plan.Users {
			userData := entities.S3CephTenantUsers{
				UserName:  userName,
				AccessKey: user.AccessKey.ValueString(),
			}

			err := order.CreateUser(
				&userData,
				user.SecretKey.ValueString(),
			)
			if err != nil {
				resp.Diagnostics.AddError("Error while create user", err.Error())
			}
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	userEndpoint := orderItem.Data.Config.(entities.S3CephTenantItemConfig).UserEndpoint
	mtlsEndpoint := orderItem.Data.Config.(entities.S3CephTenantItemConfig).MtlsEndpoint
	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(orderItem.ID)
	plan.UserEndpoint = types.StringValue(userEndpoint)
	plan.MtlsEndpoint = types.StringValue(mtlsEndpoint)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r S3CephResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data S3CephResourceModel

	var orderID types.String
	var buckets map[string]BucketModel
	var users map[string]S3UserModel
	var bucketsPortal []entities.Item
	var usersPortal []entities.S3CephTenantUsers
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("order_id"), &orderID)...)
	req.State.GetAttribute(ctx, path.Root("buckets"), &buckets)
	req.State.GetAttribute(ctx, path.Root("users"), &users)

	if orderID.IsNull() {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Resource hasn't order_id. Without order_id can't get order from portal",
		)
		return
	}
	data.OrderID = orderID

	order, err := orders.GetS3CephOrder(
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
				"Can't get app item from portal with order_id '%s'.\nError:%s",
				data.OrderID.ValueString(), err.Error(),
			),
		)
		return
	}

	usersPortal, err = order.GetTenantUsers()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Can't get S3 Ceph users.\nError: %s", err.Error()),
		)
		return
	}

	bucketsPortal, err = order.GetBucketItems()
	if err != nil {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Can't get S3 Ceph buckets.\nError: %s", err.Error()),
		)
		return
	}

	itemConfig, ok := item.Data.Config.(entities.S3CephTenantItemConfig)
	if !ok {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			"Can't get S3 Ceph item config.",
		)
		return
	}

	data.Label = types.StringValue(order.Label)
	data.UserEndpoint = types.StringValue(itemConfig.UserEndpoint)
	data.MtlsEndpoint = types.StringValue(itemConfig.MtlsEndpoint)
	data.ItemID = types.StringValue(item.ID)
	data.Users = entitiesS3CephUsersToTerraform(users, usersPortal)
	data.Buckets = entitiesS3CephBucketsToTerraform(buckets, bucketsPortal)
	data.Image = S3CephImageDataSourceData{
		ProductID:   types.StringValue(order.ProductID),
		StorageType: types.StringValue(order.Attrs.StorageType),
	}
	data.Zone = types.StringValue(order.Attrs.AvailabilityZone)
	data.NetSegment = types.StringValue(order.Attrs.NetSegment)
	data.FinancialProject = types.StringValue(order.FinancialSource.Name)

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

func (r S3CephResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan S3CephResourceModel
	var state S3CephResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	order, err := orders.GetS3CephOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)

	if err != nil {
		diags.AddError("Can't get order from portal: ", err.Error())
		resp.Diagnostics.Append(diags...)
	}

	bucketsPortal, err := order.GetBucketItems()
	if err != nil {
		diags.AddError(
			"Can't get list of S3 Ceph buckets from portal: ", err.Error(),
		)
		resp.Diagnostics.Append(diags...)
	}

	usersPortal, err := order.GetTenantUsers()
	if err != nil {
		diags.AddError(
			"Can't get list of S3 Ceph users from portal: ", err.Error(),
		)
		resp.Diagnostics.Append(diags...)
	}

	finProj, _ := sources.GetFinancialProjectByName(
		r.client.Creds,
		r.client.ProjectName,
		plan.FinancialProject.ValueString(),
	)

	buckets := entitiesS3CephBucketsToTerraform(state.Buckets, bucketsPortal)
	users := entitiesS3CephUsersToTerraform(state.Users, usersPortal)

	finProjChanged := plan.FinancialProject != state.FinancialProject
	labelChanged := plan.Label != state.Label
	bucketsChanged := validateBuckets(&plan, buckets)
	usersChanged := validateUsers(&plan, users)

	if finProjChanged {
		err = order.ChangeFinancialProject(finProj.ID)
		if err != nil {
			resp.Diagnostics.AddError(
				consts.UPDATE_RES_FAIL,
				fmt.Sprintf("Change financial source ended with error.\nError: %v", err.Error()),
			)
			return
		}
	}

	if labelChanged {
		utils.ChangeOrderLabel(order, plan.Label.ValueString(), resp)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	if bucketsChanged {
		diags = configureBuckets(
			&state,
			&plan,
			order,
			// strings.ToLower(r.provider.Environment),
			// strings.ToLower(r.provider.EnvironmentName),
			r.client,
		)
		resp.Diagnostics.Append(diags...)
	}

	if usersChanged {
		diags = configureUsers(&state, &plan, order)
		resp.Diagnostics.Append(diags...)
	}

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r S3CephResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state S3CephResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetS3CephOrder(
		r.client.Creds,
		r.client.ProjectName,
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
			r.client.Organization,
		)

		resp.Diagnostics.AddError(
			"Deletable error",
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

// * Custom logic*

func configureBuckets(
	state,
	plan *S3CephResourceModel,
	order *orders.S3CephOrder,
	client *client.CloudClient,
) (diags diag.Diagnostics) {
	err := order.Sync()
	if err != nil {
		diags.AddWarning(
			"Sync S3 Ceph",
			fmt.Sprintf("Can't sync S3 Ceph.\nError: %s", err.Error()),
		)
	}

	bucketsToCreate := make(map[string]BucketModel)
	bucketsToUpdate := make(map[string]BucketModel)
	bucketsToDelete := make(map[string]BucketModel)

	bucketsExist, err := order.GetBucketItems()
	if err != nil {
		diags.AddWarning("Get S3 Ceph buckets", err.Error())
	}

	bucketsPlan := plan.Buckets
	bucketsState := entitiesS3CephBucketsToTerraform(state.Buckets, bucketsExist)

	safeEnvprefix := regexp.QuoteMeta(client.EnvPrefix)
	safeRiscode := regexp.QuoteMeta(client.RisCode)

	pattern := fmt.Sprintf(`^%s-%s-[a-z0-9][a-z0-9-_]{1,30}[a-z0-9]$`, safeEnvprefix, safeRiscode)

	re := regexp.MustCompile(pattern)

	for name, bucket := range bucketsPlan {
		if !re.MatchString(name) {
			diags.AddError(
				"Неккоректное наименование бакета",
				fmt.Sprintf(
					"Имя бакета %q должно соответствовать паттерну %q.",
					name, re.String(),
				),
			)
			return diags
		}

		bucketState, exist := bucketsState[name]

		if !exist {
			bucketsToCreate[name] = bucket
			continue
		}

		isMaxSizeGbChanged := bucket.MaxSizeGb.ValueInt64() != bucketState.MaxSizeGb.ValueInt64()
		isVersioningChanged := bucket.Versioning.ValueBool() != bucketState.Versioning.ValueBool()

		if isMaxSizeGbChanged || isVersioningChanged {
			bucketsToUpdate[name] = bucket
		}
	}

	for name, bucket := range bucketsState {
		if _, exist := bucketsPlan[name]; !exist {
			bucketsToDelete[name] = bucket
		}
	}

	for name, bucket := range bucketsToCreate {
		err = order.CreateBucket(
			name,
			bucket.MaxSizeGb.ValueInt64(),
			bucket.Versioning.ValueBool(),
		)
		if err != nil {
			diags.AddError("Create bucket", err.Error())
			return diags
		}
	}

	for name, bucket := range bucketsToUpdate {
		err = order.ChangeBucket(
			name,
			bucket.MaxSizeGb.ValueInt64(),
			bucket.Versioning.ValueBool(),
		)
		if err != nil {
			diags.AddError("Update bucket", err.Error())
			return diags
		}
	}

	for name := range bucketsToDelete {
		err = order.DeleteBucket(
			name,
		)
		if err != nil {
			diags.AddError("Delete bucket", err.Error())
			return diags
		}
	}

	return diags
}

func entitiesS3CephBucketsToTerraform(
	bucketsState map[string]BucketModel,
	bucketsPortal []entities.Item,
) map[string]BucketModel {

	buckets := make(map[string]BucketModel)

	for _, bucketItem := range bucketsPortal {
		bucket := bucketItem.Data.Config.(entities.S3CephBucketItemConfig)
		if _, exist := bucketsState[bucket.Name]; exist {
			buckets[bucket.Name] = bucketsState[bucket.Name]
		} else {
			buckets[bucket.Name] = BucketModel{
				Versioning: types.BoolValue(bucket.Versioning),
				MaxSizeGb:  types.Int64Value(bucket.MaxSizeGb),
			}
		}
	}
	if len(buckets) == 0 {
		buckets = nil
	}
	return buckets
}

func entitiesS3CephUsersToTerraform(
	usersState map[string]S3UserModel,
	usersPortal []entities.S3CephTenantUsers,
) map[string]S3UserModel {

	users := make(map[string]S3UserModel)

	for _, user := range usersPortal {
		if _, exist := usersState[user.UserName]; exist {
			users[user.UserName] = usersState[user.UserName]
		} else {
			userPortal := S3UserModel{
				AccessKey: types.StringValue(user.AccessKey),
				SecretKey: types.StringValue("secret_key"),
			}
			users[user.UserName] = userPortal
		}
	}
	if len(users) == 0 {
		return nil
	}
	return users
}

func (r S3CephResource) usersNameModifyPlan(
	plan *S3CephResourceModel,
	resp *resource.ModifyPlanResponse,
) {
	for userName := range plan.Users {

		regex := regexp.MustCompile("^[a-z0-9][a-z0-9-]{0,19}[a-z0-9]$")

		if !regex.MatchString(userName) || strings.HasPrefix(userName, "-") || strings.HasPrefix(strings.ToLower(userName), "xn-") || !strings.ContainsAny(userName, "abcdefghijklmnopqrstuvwxyz") {
			resp.Diagnostics.AddError(
				"Validation username",
				"S3 username naming rules: 1 to 20 characters, must contain at least one letter. Allowed: letters, digits, and hyphens (except at the start or end).",
			)
		}
	}
}

func (r S3CephResource) bucketNameModifyPlan(
	plan *S3CephResourceModel,
	resp *resource.ModifyPlanResponse,
) {
	for name := range plan.Buckets {

		regex := regexp.MustCompile("^[a-z0-9-]{3,62}[a-z0-9]$")
		if !regex.MatchString(name) || strings.HasPrefix(name, "-") || strings.HasPrefix(strings.ToLower(name), "xn-") || !strings.ContainsAny(name, "abcdefghijklmnopqrstuvwxyz") {
			resp.Diagnostics.AddError(
				"Validation bucket name",
				"S3 bucket naming rules: 4 to 63 characters long, must contain at least one letter. Allowed characters: lowercase Latin letters, digits, and hyphens (except at the start or end of the name).",
			)
		}
	}
}

func (r S3CephResource) prepareAttrs(
	plan *S3CephResourceModel,
) orders.S3CephAttrs {

	attrs := orders.S3CephAttrs{
		AvailabilityZone:    plan.Zone.ValueString(),
		Platform:            "ceph",
		NetSegment:          plan.NetSegment.ValueString(),
		StorageType:         plan.Image.StorageType.ValueString(),
		CreatedWithOpenTofu: true,
	}
	return attrs
}

func validateBuckets(
	plan *S3CephResourceModel,
	bucketsOrder map[string]BucketModel,
) bool {

	if len(plan.Buckets) != len(bucketsOrder) {
		return true
	}

	for name, bucket := range plan.Buckets {
		bucketCompare, exist := bucketsOrder[name]
		if !exist {
			return true
		}

		isMaxSizeGbChanged := bucket.MaxSizeGb.ValueInt64() != bucketCompare.MaxSizeGb.ValueInt64()
		isVersioningChanged := bucket.Versioning.ValueBool() != bucketCompare.Versioning.ValueBool()

		if isMaxSizeGbChanged || isVersioningChanged {
			return true
		}
	}

	for name := range bucketsOrder {
		if _, exist := plan.Buckets[name]; !exist {
			return true
		}
	}

	return false
}

func validateUsers(
	plan *S3CephResourceModel,
	usersOrder map[string]S3UserModel,
) bool {

	if len(plan.Users) != len(usersOrder) {
		return true
	}

	for userName, user := range plan.Users {
		usersCompare, exist := usersOrder[userName]
		if !exist {
			return true
		}

		isAccessKeyChanged := user.AccessKey.ValueString() != usersCompare.AccessKey.ValueString()
		isSecretKeyChanged := user.SecretKey.ValueString() != usersCompare.SecretKey.ValueString()

		if isAccessKeyChanged || isSecretKeyChanged {
			return true
		}
	}

	for name := range usersOrder {
		if _, exist := plan.Users[name]; !exist {
			return true
		}
	}

	return false
}

func configureUsers(
	state,
	plan *S3CephResourceModel,
	order *orders.S3CephOrder,
) (diags diag.Diagnostics) {

	err := order.Sync()
	if err != nil {
		diags.AddWarning(
			"Sync S3 Ceph",
			fmt.Sprintf("Can't sync S3 Ceph.\nError: %s", err.Error()),
		)
	}

	usersToCreate := make(map[string]S3UserModel)
	usersToUpdate := make(map[string]S3UserModel)
	usersToDelete := make(map[string]S3UserModel)

	usersExist, err := order.GetTenantUsers()
	if err != nil {
		diags.AddError("Get S3 Ceph users", err.Error())
		return diags
	}

	usersState := entitiesS3CephUsersToTerraform(
		state.Users,
		usersExist,
	)

	for userName, user := range plan.Users {
		userCompare, exist := usersState[userName]
		if !exist {
			usersToCreate[userName] = user
			continue
		}

		isAccessKeyChanged := userCompare.AccessKey.ValueString() != user.AccessKey.ValueString()
		isSecretKeyChanged := userCompare.SecretKey.ValueString() != user.SecretKey.ValueString()

		if isAccessKeyChanged {
			usersToDelete[userName] = user
			usersToCreate[userName] = user
		}

		if isSecretKeyChanged && !isAccessKeyChanged {
			usersToUpdate[userName] = user
		}
	}

	for userName, user := range usersState {
		if _, exist := plan.Users[userName]; !exist {
			usersToDelete[userName] = user
		}
	}

	for userName := range usersToDelete {
		err = order.DeleteUser(userName)

		if err != nil {
			diags.AddError("Delete user", err.Error())
			return diags
		}
	}

	for userName, user := range usersToCreate {

		userData := entities.S3CephTenantUsers{
			UserName:  userName,
			AccessKey: user.AccessKey.ValueString(),
		}

		err = order.CreateUser(
			&userData,
			user.SecretKey.ValueString(),
		)
		if err != nil {
			diags.AddError("Create user", err.Error())
			return diags
		}
	}

	for userName, user := range usersToUpdate {

		userData := entities.S3CephTenantUsers{
			UserName:  userName,
			AccessKey: user.AccessKey.ValueString(),
		}

		err = order.UpdateUser(
			&userData,
			user.SecretKey.ValueString(),
		)
		if err != nil {
			diags.AddError("Update user", err.Error())
			return diags
		}
	}

	return diags
}
