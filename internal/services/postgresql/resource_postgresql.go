package postgresql

import (
	"context"
	"fmt"
	"math"
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
	_ resource.Resource                = PostgreSQLResource{}
	_ resource.ResourceWithImportState = PostgreSQLResource{}
	_ resource.ResourceWithModifyPlan  = PostgreSQLResource{}
)

type PostgreSQLResource struct {
	client *client.CloudClient
}

func NewPostgresqlResource() resource.Resource {
	return &PostgreSQLResource{}
}

func (r PostgreSQLResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_postgresql_instance"
}

func (r *PostgreSQLResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type DbModel struct {
	DbAdminPass types.String `tfsdk:"db_admin_pass"`
	ConnLimit   types.Int64  `tfsdk:"conn_limit"`
	DbEncoding  types.String `tfsdk:"db_encoding"`
	//DbCustomEncoding types.Bool   `tfsdk:"db_custom_encoding"`
}

type DbUserModel struct {
	UserPassword types.String `tfsdk:"user_password"`
	DbName       types.String `tfsdk:"db_name"`
	DbmsRole     types.String `tfsdk:"dbms_role"`
	Comment      types.String `tfsdk:"comment"`
}

type PostgreSQLResourceModel struct {
	Core        core.CoreModel                    `tfsdk:"core"`
	Flavor      flavor.FlavorModel                `tfsdk:"flavor"`
	Lifetime    types.Int64                       `tfsdk:"lifetime"`
	Label       types.String                      `tfsdk:"label"`
	ItemID      types.String                      `tfsdk:"item_id"`
	OrderID     types.String                      `tfsdk:"order_id"`
	Access      map[string][]types.String         `tfsdk:"access"`
	ExtraMounts map[string]common.ExtraMountModel `tfsdk:"extra_mounts"`

	Image            PostgresqlImageDataSourceData `tfsdk:"image"`
	ConnectionURL    types.String                  `tfsdk:"connection_url"`
	Databases        map[string]DbModel            `tfsdk:"dbs"`
	DatabaseUsers    map[string]DbUserModel        `tfsdk:"db_users"`
	FinancialProject types.String                  `tfsdk:"financial_project"`
}

func (r PostgreSQLResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "PostgreSQL Resource",
		Attributes: map[string]schema.Attribute{
			// basic schema
			"core": schema.SingleNestedAttribute{
				Required:            true,
				Description:         "Core parameters for VM and order.",
				MarkdownDescription: "Core parameters for VM and order.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: common.CoreSchema,
			},
			"flavor": schema.SingleNestedAttribute{
				Required:            true,
				Attributes:          common.FlavorSchema,
				Description:         "Core/memory of compute instance.",
				MarkdownDescription: "Core/memory of compute instance.",
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
				Description:         "Label of order.",
				MarkdownDescription: "Label of order.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 64),
				},
			},
			"order_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Identifiator of Order. Known after creating order.",
				MarkdownDescription: "Identifiator of Order. Known after creating order.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"item_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Item identificator of app in order.",
				MarkdownDescription: "Item identificator of app in order.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"extra_mounts": schema.MapNestedAttribute{
				Required:            true,
				Description:         "Added extra mounts in compute instance.",
				MarkdownDescription: "Added extra mounts in compute instance.",
				Validators: []validator.Map{
					mapvalidator.KeysAre(
						stringvalidator.OneOf(
							"/pg_data",
							"/pg_walarchive",
							"/pg_backup",
							"/pg_audit",
							"/app/logs",
							"/app/backup",
						),
					),
					mapvalidator.SizeBetween(1, 4),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"size": schema.Int64Attribute{
							Required:            true,
							Description:         "Mount point size (specify at GB).",
							MarkdownDescription: "Mount point size (specify at GB).",
							Validators: []validator.Int64{
								int64validator.AtMost(2048),
								//int64validator.AtLeast(10),
							},
							PlanModifiers: []planmodifier.Int64{
								custommodifires.GrowOnly(),
								custommodifires.MininalGrow(10),
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
			"access": schema.MapAttribute{
				Optional: true,
				Computed: true,
				Description: "Map, where key is role and value is list of groups, " +
					"which will grant access for Active Directory login",
				MarkdownDescription: "Map, where key is role and value is list of groups, " +
					"which will grant access for Active Directory login",
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.RequiresReplace(),
					custommodifires.EmptyAccessGroup(),
				},
				ElementType: types.SetType{
					ElemType: types.StringType,
				},
			},

			// image schema
			"image": schema.SingleNestedAttribute{
				Required:            true,
				Description:         "Type of Postgresql instance",
				MarkdownDescription: "Type of Postgresql instance",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.RequiresReplace(),
				},
				Attributes: map[string]schema.Attribute{
					"product_type": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Type of postgresql installation: stand-alone or cluster",
						Validators: []validator.String{
							stringvalidator.OneOf("stand-alone", "cluster"),
						},
					},
					"distribution": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Name of distribution (RHEL or Astra)",
						Validators: []validator.String{
							stringvalidator.OneOf("rhel", "astra"),
						},
					},
					"version": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Verison of distribution image",
					},
					"on_support": schema.BoolAttribute{
						Required:            true,
						MarkdownDescription: "Flag mean, what the product will supported by sup group",
					},
					"product_id": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Product id of compute instance",
					},
					"ad_integration": schema.BoolAttribute{
						Required:            true,
						MarkdownDescription: "Flag for integration compute instance with Active Directory",
					},
					"load_profile": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Load profile for PostgreSQL installation",
					},
					"postgresql_version": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "PostgreSQL version",
					},
					"default_transaction_isolation": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Default Transaction Isolation parameter for PostgreSQL configuration",
					},
				},
			},

			// extra product schema
			"connection_url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Connection url of ordered postgresql",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dbs": schema.MapNestedAttribute{
				Optional:            true,
				MarkdownDescription: "List of dbs to create on postgresql instance",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"db_admin_pass": schema.StringAttribute{
							Sensitive:           true,
							Required:            true,
							MarkdownDescription: "Password for admin of database",
							Validators: []validator.String{
								stringvalidator.RegexMatches(regexp.MustCompile("[A-Z]"),
									"Password must have 1 uppercase letter"),
								stringvalidator.RegexMatches(regexp.MustCompile("[a-z]"),
									"Password must have 1 lowercase letter"),
								stringvalidator.RegexMatches(regexp.MustCompile("[0-9]"),
									"Password must have 1 numeric character"),
								stringvalidator.RegexMatches(regexp.MustCompile(`^[a-zA-Z0-9\\._-]{64,128}$`),
									"Password must match pattern ^[a-zA-Z0-9\\._-]{64,128}$"),
							},
						},
						"conn_limit": schema.Int64Attribute{
							Required:            false,
							Optional:            true,
							MarkdownDescription: "Connection limit for databases",
							Validators: []validator.Int64{
								int64validator.AtLeast(10),
							},
						},
						"db_encoding": schema.StringAttribute{
							Optional: true,
							Required: false,
							Validators: []validator.String{
								stringvalidator.OneOf(
									"UTF8 C C",
									"UTF8 en_US.UTF-8 en_US.UTF-8",
									"WIN1251 ru_RU.CP1251 ru_RU.CP1251",
								),
							},
						},
						//"db_custom_encoding": schema.BoolAttribute{
						//	Required:            true,
						//	MarkdownDescription: "Database Custom Encoding",
						//},
					}},
			},
			"db_users": schema.MapNestedAttribute{
				Optional:            true,
				MarkdownDescription: "List of users to add to postgresql",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"user_password": schema.StringAttribute{
							Required:            true,
							Sensitive:           true,
							MarkdownDescription: "User's password",
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									regexp.MustCompile("[A-Z]"),
									"Password must have 1 uppercase letter",
								),
								stringvalidator.RegexMatches(
									regexp.MustCompile("[a-z]"),
									"Password must have 1 lowercase letter",
								),
								stringvalidator.RegexMatches(
									regexp.MustCompile("[0-9]"),
									"Password must have 1 numeric character",
								),
								stringvalidator.RegexMatches(
									regexp.MustCompile(`^[a-zA-Z0-9\\._-]{64,128}$`),
									"Password must match pattern ^[a-zA-Z0-9\\._-]{64,128}$",
								),
							},
						},
						"db_name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Name of DB to which user will be added",
						},
						"dbms_role": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Role of user",
							Validators: []validator.String{
								stringvalidator.OneOf("reader", "user"),
							},
						},
						"comment": schema.StringAttribute{
							Optional:            true,
							MarkdownDescription: "Comment for user",
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									regexp.MustCompile(`^[^\\\t]{5,35}$`),
									"Comment lentgt must between 5 and 35 symbols",
								),
							},
						},
					}},
			},
			"financial_project": schema.StringAttribute{
				Required:            true,
				Description:         "Financial source for order.",
				MarkdownDescription: "Financial source for order.",
			},
		},
	}
}

func (r PostgreSQLResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("order_id"), req, resp)
}

func (r PostgreSQLResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan, state PostgreSQLResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if !req.State.Raw.IsNull() {
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	if r.client.Environment == "PROD" && plan.Image.ProductType.ValueString() != "cluster" {
		resp.Diagnostics.AddAttributeError(
			path.Root("product_type"),
			"Invalid product type for current environment",
			"only `cluster` product may be choosen for PROD",
		)
		return
	}

	checkIsOrderDeleted := utils.CheckOrderIsDeleted(r.client, plan.OrderID, "app", "postgresql_v001")
	resp.Diagnostics.Append(checkIsOrderDeleted.Diagnostics...)

	utils.HandleExternalDeletionRecreate(ctx, resp, &plan, checkIsOrderDeleted.IsDeleted, checkIsOrderDeleted.Diagnostics)

	r.flavorModifyPlan(&plan, resp)
	r.availiableAccessModifyPlan(&plan, resp)
	r.dbsModifyPlan(&plan, resp)
	r.dbUsersNameModifyPlan(&plan, resp)
	r.dbNameModifyPlan(&plan, resp)
	r.extraMountsModifyPlan(&plan, resp)

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r PostgreSQLResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan PostgreSQLResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order := orders.NewPostgresqlOrder(
		r.client.Creds,
		r.client.ProjectName,
		plan.Image.ProductID.ValueString(),
		r.prepareAttrs(&plan, resp),
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
	for extraMountPath := range plan.ExtraMounts {
		if (extraMountPath == "/pg_backup") || (extraMountPath == "/pg_walarchive") || (extraMountPath == "/pg_audit") || (extraMountPath == "/app/logs") || (extraMountPath == "/app/backup") {
			err := order.AddMountPoint(extraMountPath)
			if err != nil {
				resp.Diagnostics.AddError("Error while add mount point", err.Error())
				return
			}
		}
	}

	err = order.Sync()
	if err != nil {
		resp.Diagnostics.AddError("Error while update state: ", err.Error())
		return
	}

	//create databases for postgresql instance
	for dbName, db := range plan.Databases {
		err := order.CreateDb(
			dbName, db.DbAdminPass.ValueString(),
			db.DbEncoding.ValueStringPointer(),
			//db.DbCustomEncoding.ValueBool(),
			db.ConnLimit.ValueInt64(),
			false,
		)
		if err != nil {
			resp.Diagnostics.AddError("Error while create db", err.Error())
		}
	}

	for userName, user := range plan.DatabaseUsers {

		userData := entities.PostgresqlItemDbUsers{
			UserName: removePrefix(userName, user),
			Comment:  user.Comment.ValueString(),
			DbName:   user.DbName.ValueString(),
			DbmsRole: user.DbmsRole.ValueString(),
		}

		err := order.CreateUser(
			&userData,
			user.UserPassword.ValueString(),
			false,
		)
		if err != nil {
			resp.Diagnostics.AddError("Error while create user", err.Error())
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	connectionUrl := orderItem.Data.Config.(entities.PostgresqlItemConfig).ConnectionURL
	plan.OrderID = types.StringValue(order.ID)
	plan.ItemID = types.StringValue(orderItem.ID)
	plan.ConnectionURL = types.StringValue(connectionUrl)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r PostgreSQLResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data PostgreSQLResourceModel

	var vmItem entities.Item
	var productType string
	var orderID types.String
	var image PostgresqlImageDataSourceData
	var dbs map[string]DbModel
	var dbUsers map[string]DbUserModel
	var dbsPortal []entities.Item
	var dbUsersPortal []entities.PostgresqlItemDbUsers
	var extraMounts map[string]common.ExtraMountModel
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("order_id"), &orderID)...)
	req.State.GetAttribute(ctx, path.Root("image"), &image)
	req.State.GetAttribute(ctx, path.Root("dbs"), &dbs)
	req.State.GetAttribute(ctx, path.Root("db_users"), &dbUsers)
	req.State.GetAttribute(ctx, path.Root("extra_mounts"), &extraMounts)

	if orderID.IsNull() {
		resp.Diagnostics.AddError(
			"Read order from portal",
			"Resource hasn't order_id. Without order_id can't get order from portal",
		)
		return
	}
	data.OrderID = orderID

	order, err := orders.GetPostgresqlOrder(
		r.client.Creds,
		r.client.ProjectName,
		data.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Read order from portal",
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
			"Read order from portal",
			fmt.Sprintf(
				"Can't get app item from portal with order_id '%s'.\nError:%s",
				data.OrderID.ValueString(), err.Error(),
			),
		)
		return
	}

	config := item.Data.Config.(entities.PostgresqlItemConfig)
	dbUsersPortal, err = order.GetUsers()

	if err != nil {
		resp.Diagnostics.AddError(
			"Read order from portal",
			fmt.Sprintf("Can't get postgresql users.\nError: %s", err.Error()),
		)
		return
	}

	dbsPortal, err = order.GetDBs()
	if err != nil {
		resp.Diagnostics.AddError(
			"Read order from portal",
			fmt.Sprintf("Can't get postgresql databases.\nError: %s", err.Error()),
		)
		return
	}

	vmItems, err := order.GetVMItems()
	if err != nil {
		resp.Diagnostics.AddError(
			"Get VM item of order",
			fmt.Sprintf("Error:%s", err.Error()),
		)
		return
	}

	switch item.Type {
	case "app":
		productType = "stand-alone"
		if len(vmItems) != 1 {
			resp.Diagnostics.AddError(
				"Order has too many child VM",
				"Number of child VM in order is greater than 1",
			)
			return
		}
		vmItem = vmItems[0]
	case "cluster":
		productType = "cluster"
		postgresqlVmItems, err := order.GetPostgresqlVmItems()
		if err != nil {
			resp.Diagnostics.AddError(
				"Get VM item with role 'postgresql'",
				fmt.Sprintf("Error:%s", err.Error()),
			)
		}
		vmItem = postgresqlVmItems[0]
	}

	vmConfig := vmItem.Data.Config.(entities.VMItemConfig)

	postgresqlPgdataMount, err := order.GetExtraMount("/pg_data")
	if err != nil {
		resp.Diagnostics.AddError(
			"Read order from portal",
			fmt.Sprintf("Can't get postgresql extra mounts.\nError: %s", err.Error()),
		)
		return
	}

	itemConfig := item.Data.Config.(entities.PostgresqlItemConfig)

	data.Label = types.StringValue(order.Label)
	data.ConnectionURL = types.StringValue(itemConfig.ConnectionURL)
	data.ItemID = types.StringValue(item.ID)
	data.Access = utils.ReadAccessMapV2(vmItem.Data.ACLs)
	data.DatabaseUsers = entitiesPostgreSQLDBUsersToTerraform(dbUsers, dbUsersPortal)
	data.Databases = entitiesPostgreSQLDBsToTerraform(dbs, dbsPortal)
	data.FinancialProject = types.StringValue(order.FinancialSource.Name)

	isCoresChanged := vmConfig.Flavor.Cores != data.Flavor.Cores.ValueInt64()
	isMemoryChanged := vmConfig.Flavor.Memory != data.Flavor.Memory.ValueInt64()

	if isCoresChanged || isMemoryChanged {
		data.Flavor = flavor.FlavorModel{
			Cores:  types.Int64Value(vmConfig.Flavor.Cores),
			Memory: types.Int64Value(vmConfig.Flavor.Memory),
			Name:   types.StringValue(vmConfig.Flavor.Name),
			UUID:   types.StringValue(vmConfig.Flavor.UUID),
		}
	}

	data.ExtraMounts = make(map[string]common.ExtraMountModel)
	data.ExtraMounts["/pg_data"] = common.ExtraMountModel{
		Size:       types.Int64Value(postgresqlPgdataMount.Size),
		FileSystem: types.StringValue(postgresqlPgdataMount.FileSystem),
	}
	for extraMountPath, extraMount := range extraMounts {
		extraMountOrder, err := order.GetExtraMount(extraMountPath)
		if err != nil {
			resp.Diagnostics.AddError(
				"Read order from portal",
				fmt.Sprintf("Can't get postgresql extra mounts.\nError: %s", err.Error()),
			)
			return
		}
		if extraMountOrder != nil {
			data.ExtraMounts[extraMountPath] = common.ExtraMountModel{
				Size:       extraMount.Size,
				FileSystem: extraMount.FileSystem,
			}
		}
	}

	if image.ProductID.IsNull() {
		data.Image = PostgresqlImageDataSourceData{
			Distribution:                types.StringValue(vmConfig.Image.Os.Distribution),
			Version:                     types.StringValue(vmConfig.Image.Os.Version),
			OnSupport:                   types.BoolValue(order.Attrs.OnSupport),
			ProductID:                   types.StringValue(order.ProductID),
			ADIntegration:               types.BoolValue(order.Attrs.ADIntegration),
			PostgresqlVersion:           types.StringValue(strings.Split(config.Version, ".")[0]),
			LoadProfile:                 types.StringValue(config.LoadProfile),
			DefaultTransactionIsolation: types.StringValue(strings.ToUpper(config.Configuration.DefaultTransactionIsolation)),
			ProductType:                 types.StringValue(productType),
		}
	} else {
		data.Image = image
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

func (r PostgreSQLResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan PostgreSQLResourceModel
	var state PostgreSQLResourceModel

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	order, err := orders.GetPostgresqlOrder(
		r.client.Creds,
		r.client.ProjectName,
		state.OrderID.ValueString(),
	)

	if err != nil {
		diags.AddError("Can't get order from portal: ", err.Error())
		resp.Diagnostics.Append(diags...)
	}

	finProj, err := sources.GetFinancialProjectByName(
		r.client.Creds,
		r.client.ProjectName,
		plan.FinancialProject.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("financial_project"),
			consts.UPDATE_RES_FAIL,
			fmt.Sprintf("Get financial source for project ended with error.\nError: %v", err.Error()))
		return
	}

	dbsPortal, err := order.GetDBs()
	if err != nil {
		diags.AddError(
			"Can't get list of postgresql databases from portal: ", err.Error(),
		)
		resp.Diagnostics.Append(diags...)
	}

	dbUsersPortal, err := order.GetUsers()
	if err != nil {
		diags.AddError(
			"Can't get list of postgresql users from portal: ", err.Error(),
		)
		resp.Diagnostics.Append(diags...)
	}

	dbs := entitiesPostgreSQLDBsToTerraform(state.Databases, dbsPortal)
	dbUsers := entitiesPostgreSQLDBUsersToTerraform(state.DatabaseUsers, dbUsersPortal)

	labelChanged := plan.Label != state.Label
	financialProjectChanged := !plan.FinancialProject.Equal(state.FinancialProject)
	mountChanged := utils.IsExtraMountChanged(state.ExtraMounts, plan.ExtraMounts)
	dbsChanged := validateDBs(&plan, dbs)
	dbUsersChanged := validateDBUsers(&plan, dbUsers)
	flavorChanged := plan.Flavor != state.Flavor

	if labelChanged {
		utils.ChangeOrderLabel(order, plan.Label.ValueString(), resp)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	if financialProjectChanged {
		err := order.ChangeFinancialProject(finProj.ID)
		if err != nil {
			resp.Diagnostics.AddAttributeError(
				path.Root("financial_project"),
				consts.UPDATE_RES_FAIL,
				fmt.Sprintf("Change Financial source for project ended with error.\nError: %s", err.Error()),
			)
			return
		}
	}

	if mountChanged {
		diags = changePostgresqlExtraMounts(order, &plan)
		resp.Diagnostics.Append(diags...)
	}

	if flavorChanged {
		diags = changePostgresqlFlavor(order, &plan)
		resp.Diagnostics.Append(diags...)
	}

	for extraMountPath := range plan.ExtraMounts {
		if _, exists := state.ExtraMounts[extraMountPath]; !exists {
			order.AddMountPoint(extraMountPath)
		}
	}

	if dbsChanged {
		diags = configureDBs(
			&state,
			&plan,
			order,
			strings.ToLower(r.client.Environment),
			strings.ToLower(r.client.EnvironmentName),
		)
		resp.Diagnostics.Append(diags...)
	}

	if dbUsersChanged {
		diags = configureDBUsers(&state, &plan, order)
		resp.Diagnostics.Append(diags...)
	}

	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
}

func (r PostgreSQLResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state PostgreSQLResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	order, err := orders.GetPostgresqlOrder(
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

	err = order.Delete(true)
	if err != nil {
		resp.Diagnostics.AddError(consts.DELETE_RES_FAIL, err.Error())
		return
	}
}

// * Custom logic*

func (r PostgreSQLResource) availiableAccessModifyPlan(
	plan *PostgreSQLResourceModel,
	resp *resource.ModifyPlanResponse,
) {
	if strings.ToLower(r.client.Environment) == "dev" {
		if len(plan.Access) == 0 {
			resp.Diagnostics.AddError(
				"Access Error",
				"ad_logon_grants must be specified and more than 0 at DEV segment",
			)
		}
	} else {
		if len(plan.Access) != 0 {
			resp.Diagnostics.AddError(
				"Access Error",
				"ad_logon_grants available only for DEV segment",
			)
		}
	}

	var filter string
	productType := plan.Image.ProductType.ValueString()

	if productType == "stand-alone" {
		filter = "app:postgresql_v001"
	}

	if productType == "cluster" {
		filter = "cluster:postgresql_v001"
	}

	var rolesNames []string
	roles, err := references.GetRoles(
		r.client.Creds,
		r.client.Environment,
		filter,
	)
	if err != nil {
		resp.Diagnostics.AddError("Get roles from portal", err.Error())
	}

	for _, r := range roles {
		rolesNames = append(rolesNames, r.Data.GroupName)
	}

	for key := range plan.Access {
		exists := false
		for _, roleName := range rolesNames {
			if key == roleName {
				exists = true
				break
			}
		}
		if !exists {
			resp.Diagnostics.AddError(
				"Access Roles not available for resource",
				fmt.Sprintf(
					"Available roles [%s]. Specified '%s'",
					strings.Join(rolesNames, ", "), key,
				),
			)
		}
	}
}

func (r PostgreSQLResource) extraMountsModifyPlan(
	plan *PostgreSQLResourceModel,
	resp *resource.ModifyPlanResponse,
) {
	pgDataMount, exist := plan.ExtraMounts["/pg_data"]
	if !exist {
		resp.Diagnostics.AddError(
			"No extra_mount '/pg_data'",
			"'extra_mounts' must contains '/pg_data'",
		)
		return
	}

	for extraMountPath, extraMount := range plan.ExtraMounts {
		switch extraMountPath {
		case "/pg_data":
			{
				if extraMount.Size.ValueInt64() < 50 {
					resp.Diagnostics.AddError(
						"Wrong size of '/pg_data'",
						"Size of '/pg_data' must be greater than 50",
					)
				}
			}
		case "/pg_backup":
			{
				if extraMount.Size != pgDataMount.Size {
					resp.Diagnostics.AddError(
						"Wrong size of extra_mount",
						"Size of '/pg_backup' must be equal to size of '/pg_data'",
					)
				}
			}
		case "/pg_walarchive":
			{
				if math.Ceil(float64(pgDataMount.Size.ValueInt64())/2) != float64(extraMount.Size.ValueInt64()) {
					resp.Diagnostics.AddError(
						"Wrong size of '/pg_walarchive'",
						"Size of '/pg_walarchive' must be 1/2 of size of '/pg_data'"+
							fmt.Sprintf("\nSize of '/pg_data': %v", pgDataMount.Size)+
							fmt.Sprintf("\nSize of '/pg_walarchive' should be equal %v", math.Ceil(float64(pgDataMount.Size.ValueInt64())/2)),
					)
				}
			}
		case "/pg_audit":
			{
				if extraMount.Size.ValueInt64() != 20 {
					resp.Diagnostics.AddError(
						"Wrong size of '/pg_audit'",
						"Size of '/pg_audit' must equal 20",
					)
				}
			}
		case "/app/logs":
			{
				if extraMount.Size.ValueInt64() != 10 {
					resp.Diagnostics.AddError(
						"Wrong size of '/app/logs'",
						"Size of '/app/logs' must equal 10",
					)
				}
				if plan.Image.ProductType.ValueString() != "cluster" {
					resp.Diagnostics.AddError(
						"You can't put '/app/logs' in extra_mounts while your product_type is \"stand-alone\"",
						"File system '/app/logs' available only for cluster installation",
					)
				}
			}
		case "/app/backup":
			{
				if extraMount.Size.ValueInt64() != 5 {
					resp.Diagnostics.AddError(
						"Wrong size of '/app/backup'",
						"Size of '/app/backup' must equal 5",
					)
				}
				if plan.Image.ProductType.ValueString() != "cluster" {
					resp.Diagnostics.AddError(
						"You can't put '/app/backup' in extra_mounts while your product_type is \"stand-alone\"",
						"File system '/app/backup' available only for cluster installation",
					)
				}
			}
		}
	}
}

func (r PostgreSQLResource) dbsModifyPlan(
	plan *PostgreSQLResourceModel,
	resp *resource.ModifyPlanResponse,
) {
	for userName, user := range plan.DatabaseUsers {
		if _, exist := plan.Databases[user.DbName.ValueString()]; !exist {
			resp.Diagnostics.AddError(
				"Database don't exist",
				fmt.Sprintf(
					"Database name '%s' is provided for user '%s', "+
						"but database is not described in resource",
					user.DbName.ValueString(), userName,
				),
			)
		}

		dbNamePrefixExist := strings.HasPrefix(
			userName,
			user.DbName.ValueString()+"_",
		)

		if !dbNamePrefixExist {
			resp.Diagnostics.AddError(
				"Prefix of username is not valid",
				"Prefix of username must be equal to database name",
			)
		}
	}

	environmentType := strings.ToLower(r.client.Environment)
	environment := strings.ToLower(r.client.EnvironmentName)
	for dbName, db := range plan.Databases {
		if (environmentType == "prod" || environment == "lt") && db.ConnLimit.IsNull() {
			resp.Diagnostics.AddError(
				"'conn_limit' is not set",
				fmt.Sprintf(
					"Database '%s' should have 'conn_limit' parameter",
					dbName,
				),
			)
		}
	}
}

func (r PostgreSQLResource) dbUsersNameModifyPlan(
	plan *PostgreSQLResourceModel,
	resp *resource.ModifyPlanResponse,
) {
	for userName, user := range plan.DatabaseUsers {

		userNameNoPrefix := removePrefix(userName, user)

		regex := regexp.MustCompile("^(user|admin|reader)$")
		if regex.MatchString(userNameNoPrefix) {
			resp.Diagnostics.AddError(
				"Validation username",
				"Can't use keywords 'user', 'admin' and 'reader'",
			)
		}

		regex = regexp.MustCompile("^[a-z0-9_]{0,15}[a-z0-9]$")
		if !regex.MatchString(userNameNoPrefix) {
			resp.Diagnostics.AddError(
				"Validation username",
				"Username must match pattern ^[a-z0-9_]{0,15}[a-z0-9]$",
			)
		}
	}
}

func (r PostgreSQLResource) dbNameModifyPlan(
	plan *PostgreSQLResourceModel,
	resp *resource.ModifyPlanResponse,
) {
	for dbName := range plan.Databases {

		regex := regexp.MustCompile("^([p|P][o|O][s|S][t|T][g|G][r|R][e|E][s|S]|[t|T][e|E][m|M][p|P][l|L][a|A][t|T][e|E].*)$")
		if regex.MatchString(dbName) {
			resp.Diagnostics.AddError(
				"Validation database name",
				"Database name can't have keywords 'postgres' or 'template'",
			)
		}

		regex = regexp.MustCompile("^[a-z0-9_]{1,63}$")
		if !regex.MatchString(dbName) {
			resp.Diagnostics.AddError(
				"Validation database name",
				"Database name must match pattern ^[a-z0-9_]{1,63}$",
			)
		}
	}
}

func (r PostgreSQLResource) flavorModifyPlan(
	plan *PostgreSQLResourceModel,
	resp *resource.ModifyPlanResponse,
) {
	var filter string

	productType := plan.Image.ProductType.ValueString()
	environment := strings.ToLower(r.client.Environment)

	if productType == "stand-alone" {
		filter = fmt.Sprintf("flavor:app:postgresql_v001:%s", environment)
	}

	if productType == "cluster" {
		filter = fmt.Sprintf("flavor:cluster:postgresql_v001:postgresql:%s", environment)
	}

	flavor, err := references.GetFlavorByFilter(
		r.client.Creds,
		filter,
		plan.Flavor.Memory.ValueInt64(),
		plan.Flavor.Cores.ValueInt64(),
	)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("flavor"),
			"Flavors for PostgreSQL fetch error",
			err.Error(),
		)
	}

	if plan.Flavor.UUID.IsNull() || plan.Flavor.Name.IsNull() || plan.Flavor.UUID.IsUnknown() || plan.Flavor.Name.IsUnknown() {
		plan.Flavor.UUID = types.StringValue(flavor.ID)
		plan.Flavor.Name = types.StringValue(flavor.Name)
	}
}

func (r PostgreSQLResource) prepareAttrs(
	plan *PostgreSQLResourceModel,
	resp *resource.CreateResponse,
) orders.PostgresqlAttrs {
	extraMounts := make(map[string]common.ExtraMountModel)
	extraMounts["/pg_data"] = plan.ExtraMounts["/pg_data"]

	basicAttrs := utils.PrepareBasicAttrs(
		&plan.Flavor,
		&plan.Core,
		plan.Access,
		extraMounts,
		plan.Image.Version.ValueString(),
		plan.Image.ADIntegration.ValueBool(),
		plan.Image.OnSupport.ValueBool(),
	)

	var attrs orders.PostgresqlAttrs

	environmentType := strings.ToLower(r.client.Environment)
	if environmentType == "prod" {
		layout, err := references.GetGeoPage(
			r.client.Creds,
			"postgresql",
			plan.Core.Platform.ValueString(),
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Can't geo distribution informantion", err.Error(),
			)
			return attrs
		}

		attrs = orders.PostgresqlAttrs{
			BasicAttrs:        basicAttrs,
			ExtraNics:         []interface{}{},
			GeoDistribution:   true,
			OnBackup:          true,
			CorparchCheck:     true,
			LoadProfile:       plan.Image.LoadProfile.ValueString(),
			LayoutID:          layout.ID,
			PostgresqlVersion: plan.Image.PostgresqlVersion.ValueString(),
			PostgresqlConfig: orders.PostgresqlConfig{
				DefaultTransactionIsolation: plan.Image.DefaultTransactionIsolation.ValueString(),
			},
		}
	} else {
		attrs = orders.PostgresqlAttrs{
			BasicAttrs:        basicAttrs,
			ExtraNics:         []interface{}{},
			CorparchCheck:     true,
			LoadProfile:       plan.Image.LoadProfile.ValueString(),
			PostgresqlVersion: plan.Image.PostgresqlVersion.ValueString(),
			PostgresqlConfig: orders.PostgresqlConfig{
				DefaultTransactionIsolation: plan.Image.DefaultTransactionIsolation.ValueString(),
			},
		}
	}
	return attrs
}

func validateDBs(
	plan *PostgreSQLResourceModel,
	dbsOrder map[string]DbModel,
) bool {

	if len(plan.Databases) != len(dbsOrder) {
		return true
	}

	for dbName, db := range plan.Databases {
		dbCompare, exist := dbsOrder[dbName]
		if !exist {
			return true
		}

		isAdminPassChanged := db.DbAdminPass.ValueString() != dbCompare.DbAdminPass.ValueString()
		isConnLimitChanged := !db.ConnLimit.IsNull() && db.ConnLimit.ValueInt64() != dbCompare.ConnLimit.ValueInt64()
		isConnLimitNull := db.ConnLimit.IsNull() && !dbsOrder[dbName].ConnLimit.IsNull()

		if isAdminPassChanged || isConnLimitChanged || isConnLimitNull {
			return true
		}
	}

	for dbName := range dbsOrder {
		if _, exist := plan.Databases[dbName]; !exist {
			return true
		}
	}

	return false
}

func validateDBUsers(
	planResource *PostgreSQLResourceModel,
	dbUsers map[string]DbUserModel,
) bool {

	dbUsersPlan := planResource.DatabaseUsers
	if len(planResource.DatabaseUsers) != len(dbUsers) {
		return true
	}

	for userName, user := range dbUsers {
		if _, exist := dbUsersPlan[userName]; !exist {
			return true
		}
		userPlan := dbUsersPlan[userName]
		if user.DbName.ValueString() != userPlan.DbName.ValueString() ||
			user.Comment.ValueString() != userPlan.Comment.ValueString() ||
			user.DbmsRole.ValueString() != userPlan.DbmsRole.ValueString() ||
			user.UserPassword.ValueString() != userPlan.UserPassword.ValueString() {
			return true
		}
	}
	return false
}

func configureDBs(
	state,
	plan *PostgreSQLResourceModel,
	order *orders.PostgresqlOrder,
	environmentType string,
	environment string,
) (diags diag.Diagnostics) {
	err := order.Sync()
	if err != nil {
		diags.AddWarning(
			"Sync Postgresql",
			fmt.Sprintf("Can't sync postgresql.\nError: %s", err.Error()),
		)
	}

	dbsToCreate := make(map[string]DbModel)
	dbsToUpdate := make(map[string]DbModel)
	dbsToDelete := make(map[string]DbModel)
	dbsToUpdateConnLimit := make(map[string]DbModel)
	dbsToRemoveConnLimit := make(map[string]DbModel)

	dbsExist, err := order.GetDBs()
	if err != nil {
		diags.AddWarning("Get PostgreSQL databases", err.Error())
	}

	dbsPlan := plan.Databases
	dbsState := entitiesPostgreSQLDBsToTerraform(state.Databases, dbsExist)

	for dbName, db := range dbsPlan {
		dbState, exist := dbsState[dbName]

		if !exist {
			dbsToCreate[dbName] = db
			continue
		}

		isAdminPassChanged := db.DbAdminPass.ValueString() != dbState.DbAdminPass.ValueString()
		isConnLimitChanged := !db.ConnLimit.IsNull() && db.ConnLimit.ValueInt64() != dbState.ConnLimit.ValueInt64()
		isConnLimitNull := db.ConnLimit.IsNull() && !dbState.ConnLimit.IsNull()
		isEnvironmentDevOrTest := environmentType == "dev" || environmentType == "test" && environment != "lt"

		if isAdminPassChanged {
			dbsToUpdate[dbName] = db
		}

		if isConnLimitChanged {
			dbsToUpdateConnLimit[dbName] = db
		}

		if isConnLimitNull && isEnvironmentDevOrTest {
			dbsToRemoveConnLimit[dbName] = db
		}
	}

	for dbName, db := range dbsState {
		if _, exist := dbsPlan[dbName]; !exist {
			dbsToDelete[dbName] = db
		}
	}

	for dbName, db := range dbsToCreate {
		err = order.CreateDb(
			dbName,
			db.DbAdminPass.ValueString(),
			db.DbEncoding.ValueStringPointer(),
			//db.DbCustomEncoding.ValueBool(),
			db.ConnLimit.ValueInt64(),
			false,
		)
		if err != nil {
			diags.AddError("Create database", err.Error())
			return diags
		}
	}

	for dbName, db := range dbsToUpdate {
		ownerName := dbName + "_admin"
		err = order.ChangeOwnerPassword(
			ownerName,
			db.DbAdminPass.ValueString(),
			false,
		)
		if err != nil {
			diags.AddError("Update database", err.Error())
			return diags
		}
	}

	for dbName := range dbsToDelete {
		err = order.DeleteDB(dbName, false)
		if err != nil {
			diags.AddError("Delete database", err.Error())
			return diags
		}
	}

	for dbName, db := range dbsToUpdateConnLimit {
		err = order.SetConnectionLimit(
			db.ConnLimit.ValueInt64(),
			dbName,
			false,
		)
		if err != nil {
			diags.AddError("Update connection limit for database", err.Error())
			return diags
		}
	}

	for dbName := range dbsToRemoveConnLimit {
		err = order.RemoveConnectionLimit(dbName, false)
		if err != nil {
			diags.AddError("Remove connection limit", err.Error())
			return diags
		}
	}
	return diags
}

func configureDBUsers(
	state,
	plan *PostgreSQLResourceModel,
	order *orders.PostgresqlOrder,
) (diags diag.Diagnostics) {

	err := order.Sync()
	if err != nil {
		diags.AddWarning(
			"Sync Postgresql",
			fmt.Sprintf("Can't sync postgresql.\nError: %s", err.Error()),
		)
	}

	dbUsersToCreate := make(map[string]DbUserModel)
	dbUsersToUpdate := make(map[string]DbUserModel)
	dbUsersToDelete := make(map[string]DbUserModel)

	dbUsersExist, err := order.GetUsers()
	if err != nil {
		diags.AddError("Get postgresql users", err.Error())
		return diags
	}

	dbUsersState := entitiesPostgreSQLDBUsersToTerraform(
		state.DatabaseUsers,
		dbUsersExist,
	)

	for userName, user := range plan.DatabaseUsers {
		userCompare, exist := dbUsersState[userName]
		if !exist {
			dbUsersToCreate[userName] = user
			continue
		}

		isCommentSame := userCompare.Comment.ValueString() == user.Comment.ValueString()
		isDbmsRoleSame := userCompare.DbmsRole.ValueString() == user.DbmsRole.ValueString()
		isPasswordChanged := userCompare.UserPassword.ValueString() != user.UserPassword.ValueString()

		if isCommentSame && isDbmsRoleSame && isPasswordChanged {
			dbUsersToUpdate[userName] = user
		} else if !isCommentSame || !isDbmsRoleSame {
			dbUsersToDelete[userName] = userCompare
			dbUsersToCreate[userName] = user
		}
	}

	for userName, user := range dbUsersState {
		if _, exist := plan.DatabaseUsers[userName]; !exist {
			dbUsersToDelete[userName] = user
		}
	}

	for userName, user := range dbUsersToDelete {
		_, existDb := plan.Databases[user.DbName.ValueString()]
		if existDb {
			err = order.DeleteUser(userName, false)

			if err != nil {
				diags.AddError("Delete user", err.Error())
				return diags
			}
		}
	}

	for userName, user := range dbUsersToCreate {

		userData := entities.PostgresqlItemDbUsers{
			UserName: removePrefix(userName, user),
			Comment:  user.Comment.ValueString(),
			DbmsRole: user.DbmsRole.ValueString(),
			DbName:   user.DbName.ValueString(),
		}

		err = order.CreateUser(
			&userData,
			user.UserPassword.ValueString(),
			false,
		)
		if err != nil {
			diags.AddError("Create user", err.Error())
			return diags
		}
	}

	for userName, user := range dbUsersToUpdate {
		err = order.ChangeUserPassword(
			userName,
			user.UserPassword.ValueString(),
			false,
		)
		if err != nil {
			diags.AddError("Update user password", err.Error())
			return diags
		}
	}

	return diags
}

func changePostgresqlExtraMounts(
	order *orders.PostgresqlOrder,
	planResource *PostgreSQLResourceModel,
) (diags diag.Diagnostics) {

	for path, planEM := range planResource.ExtraMounts {
		err := order.ExpandMountPoint(entities.ExtraMount{
			Path:       path,
			Size:       planEM.Size.ValueInt64(),
			FileSystem: planEM.FileSystem.ValueString(),
		}, false)
		if err != nil {
			diags.AddError("Incorrent changes in extra_mounts parameters", err.Error())
		}
	}
	return diags
}

func changePostgresqlFlavor(
	order *orders.PostgresqlOrder,
	plan *PostgreSQLResourceModel,
) (diags diag.Diagnostics) {
	flavor := entities.Flavor{
		Cores:  plan.Flavor.Cores.ValueInt64(),
		Memory: plan.Flavor.Memory.ValueInt64(),
		UUID:   plan.Flavor.UUID.ValueString(),
		Name:   plan.Flavor.Name.ValueString(),
	}
	err := order.ChangeFlavor(flavor, false)
	if err != nil {
		diags.AddError("Change app state (on changing machine state)", err.Error())
	}
	return diags
}

func FullEncodingName(encoding, lcCtype, lcCollate string) (string, bool) {
	fullencoding := fmt.Sprintf("%s %s %s", encoding, lcCtype, lcCollate)
	switch fullencoding {
	case "UTF8 C C":
		return "UTF8 C C", true
	case "UTF8 en_US.UTF-8 en_US.UTF-8":
		return "UTF8 en_US.UTF-8 en_US.UTF-8", true
	case "WIN1251 ru_RU.CP1251 ru_RU.CP1251":
		return "WIN1251 ru_RU.CP1251 ru_RU.CP1251", true
	default:
		return "", false
	}
}

func entitiesPostgreSQLDBsToTerraform(
	dbsState map[string]DbModel,
	dbsPortal []entities.Item,
) map[string]DbModel {

	dbs := make(map[string]DbModel)

	for _, dbItem := range dbsPortal {
		db := dbItem.Data.Config.(entities.PostgresqlDbItemConfig)

		var dbEncoding types.String

		if knownStr, ok := FullEncodingName(db.Encoding, db.LcCtype, db.LcCollate); ok {
			dbEncoding = types.StringValue(knownStr)
		} else {
			dbEncoding = types.StringNull()
		}
		var dbAdminPass types.String
		if existing, exists := dbsState[db.DbName]; exists {
			dbAdminPass = existing.DbAdminPass
		} else {
			dbAdminPass = types.StringValue("db_password")
		}
		var connLimit types.Int64
		if db.ConnLimit == 0 {
			connLimit = types.Int64Null()
		} else {
			connLimit = types.Int64Value(db.ConnLimit)
		}
		dbs[db.DbName] = DbModel{
			DbAdminPass: dbAdminPass,
			ConnLimit:   connLimit,
			DbEncoding:  dbEncoding,
		}
	}
	if len(dbs) == 0 {
		dbs = nil
	}
	return dbs
}

func entitiesPostgreSQLDBUsersToTerraform(
	dbUsersState map[string]DbUserModel,
	dbUsersPortal []entities.PostgresqlItemDbUsers,
) map[string]DbUserModel {

	users := make(map[string]DbUserModel)

	for _, user := range dbUsersPortal {
		if _, exist := dbUsersState[user.UserName]; exist {
			users[user.UserName] = dbUsersState[user.UserName]
		} else {
			dbmsRole := strings.Split(user.DbmsRole, "_")[len(strings.Split(user.DbmsRole, "_"))-1]
			userPortal := DbUserModel{
				DbName:       types.StringValue(user.DbName),
				DbmsRole:     types.StringValue(dbmsRole),
				Comment:      types.StringValue(user.Comment),
				UserPassword: types.StringValue("user_password"),
			}
			users[user.UserName] = userPortal
		}
	}
	if len(users) == 0 {
		return nil
	}
	return users
}

func removePrefix(userName string, user DbUserModel) string {
	return strings.Replace(
		userName,
		user.DbName.ValueString()+"_",
		"",
		1,
	)
}
