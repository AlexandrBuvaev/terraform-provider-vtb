package tarantool

import (
	"context"
	"fmt"
	"strings"
	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/pkg/client/references"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource = &TarantoolDataGridImageDataSource{}
)

type TarantoolDataGridImageDataSource struct {
	client *client.CloudClient
}

func NewTarantoolDataGridImageDataSource() datasource.DataSource {
	return &TarantoolDataGridImageDataSource{}
}

func (d TarantoolDataGridImageDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_tdg_image_data"
}

func (d *TarantoolDataGridImageDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = client
}

type TarantoolClusterImageModel struct {
	Distribution  types.String `tfsdk:"distribution"`
	OsVersion     types.String `tfsdk:"os_version"`
	OnSupport     types.Bool   `tfsdk:"on_support"`
	ProductID     types.String `tfsdk:"product_id"`
	ADIntegration types.Bool   `tfsdk:"ad_integration"`

	GeoDistribution         types.Bool   `tfsdk:"geo_distribution"`
	DefaultTarantoolVersion types.String `tfsdk:"default_tarantool_version"`
}

func (d TarantoolDataGridImageDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description:         "Данные образа из сервиса справочников необходимые для создания заказа",
		MarkdownDescription: "Данные образа из сервиса справочников необходимые для создания заказа",
		Attributes: map[string]schema.Attribute{
			"distribution": schema.StringAttribute{
				Required:            true,
				Description:         "Название дистрибутива (RHEL или Astra)",
				MarkdownDescription: "Название дистрибутива (RHEL или Astra)",
				Validators: []validator.String{
					stringvalidator.OneOf("rhel", "astra"),
				},
			},
			"os_version": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Description:         "Версия дистибутива образа",
				MarkdownDescription: "Версия дистрибутива образа",
			},
			"product_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Product id для продукта Tarantool Data Grid",
				MarkdownDescription: "Product id для продукта Tarantool Data Grid",
			},
			"on_support": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Флаг означает, что продукт будет поддерживаться группой поддержки",
			},
			"ad_integration": schema.BoolAttribute{
				Computed:            true,
				Description:         "Флаг интеграции ВМ с Active Directory",
				MarkdownDescription: "Флаг интеграции ВМ с Active Directory",
			},
			"default_tarantool_version": schema.StringAttribute{
				Computed:            true,
				Description:         "Версия Tarantool по умолчанию",
				MarkdownDescription: "Версия Tarantool по умолчанию",
			},
			"geo_distribution": schema.BoolAttribute{
				Computed:            true,
				Description:         "Флаг использования георезервирования",
				MarkdownDescription: "Флаг использования георезервирования",
			},
		},
	}
}

func (d TarantoolDataGridImageDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data TarantoolClusterImageModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	image, err := references.GetTarantoolDataGridImageData(
		d.client.Creds,
		data.Distribution.ValueString(),
		d.client.Organization,
		d.client.Environment,
		data.OsVersion.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Get defaults data from references", err.Error())
	}

	if !strings.Contains(image.Version, data.OsVersion.ValueString()) {
		resp.Diagnostics.AddError(
			"Validate Os Version",
			fmt.Sprintf("Unknown OS Version %s", data.OsVersion.ValueString()),
		)
		return
	}

	data.OsVersion = types.StringValue(image.Version)
	data.OnSupport = types.BoolValue(image.OnSupport)
	data.ADIntegration = types.BoolValue(image.ADIntegration)
	data.GeoDistribution = types.BoolValue(image.GeoDistribution)
	data.DefaultTarantoolVersion = types.StringValue(image.TarantoolVersion)
	data.ProductID = types.StringValue(image.ProductID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
