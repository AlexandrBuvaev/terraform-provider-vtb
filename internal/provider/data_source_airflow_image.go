package provider

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-vtb/pkg/client/references"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource = AirflowImageDataSource{}
)

type AirflowImageDataSource struct {
	provider *VTBCloudProvider
}

func NewAirflowImageDataSource(p *VTBCloudProvider) AirflowImageDataSource {
	return AirflowImageDataSource{
		provider: p,
	}
}

func (d AirflowImageDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = "vtb_airflow_image_data"
}

type AirflowImageDataSourceData struct {
	Distribution  types.String `tfsdk:"distribution"`
	OsVersion     types.String `tfsdk:"os_version"`
	OnSupport     types.Bool   `tfsdk:"on_support"`
	OnBackup      types.Bool   `tfsdk:"on_backup"`
	ProductID     types.String `tfsdk:"product_id"`
	ADIntegration types.Bool   `tfsdk:"ad_integration"`

	ProductType     types.String `tfsdk:"product_type"`
	AirflowVersion  types.String `tfsdk:"airflow_version"`
	GeoDistribution types.Bool   `tfsdk:"geo_distribution"`
}

func (t AirflowImageDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Image data from reference service needed to create order",
		Attributes: map[string]schema.Attribute{
			// basic image data attrs
			"distribution": schema.StringAttribute{
				Required:            true,
				Description:         "Название дистрибутива (RHEL или Astra).",
				MarkdownDescription: "Название дистрибутива (RHEL или Astra).",
				Validators: []validator.String{
					stringvalidator.OneOf("rhel", "astra"),
				},
			},
			"os_version": schema.StringAttribute{
				Required:            true,
				Description:         "Версия образа дистрибутива.",
				MarkdownDescription: "Версия образа дистрибутива.",
			},
			"on_support": schema.BoolAttribute{
				Computed:            true,
				Description:         "Флаг означает, что продукт будет поддерживаться группой сопровождения.",
				MarkdownDescription: "Флаг означает, что продукт будет поддерживаться группой сопровождения.",
			},
			"on_backup": schema.BoolAttribute{
				Computed:            true,
				Description:         "Флаг постановки на СРК RuBackup.",
				MarkdownDescription: "Флаг постановки на СРК RuBackup.",
			},
			"product_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Product ID из сервиса product_catalog.",
				MarkdownDescription: "Product ID из сервиса product_catalog.",
			},
			"ad_integration": schema.BoolAttribute{
				Computed:            true,
				Description:         "Флаг для интеграции ВМ с Active Directory.",
				MarkdownDescription: "Флаг для интеграции ВМ с Active Directory.",
			},
			"geo_distribution": schema.BoolAttribute{
				Computed:            true,
				Description:         "Флаг использования геораспределения узлов кластера.",
				MarkdownDescription: "Флаг использования геораспределения узлов кластера.",
			},

			// extra image data attrs
			"product_type": schema.StringAttribute{
				Required:            true,
				Description:         "Тип Airflow установки: stand-alone или cluster.",
				MarkdownDescription: "Тип Airflow установки: stand-alone или cluster.",
				Validators: []validator.String{
					stringvalidator.OneOf("stand-alone", "cluster"),
				},
			},
			"airflow_version": schema.StringAttribute{
				Computed:            true,
				Description:         "Версия ядра Airflow.",
				MarkdownDescription: "Версия ядра Airflow.",
			},
		}}
}

func (d AirflowImageDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data AirflowImageDataSourceData
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	image, err := references.GetAirflowImageData(
		d.provider.Creds,
		data.Distribution.ValueString(),
		data.ProductType.ValueString(),
		d.provider.Organization,
		d.provider.Environment,
		data.OsVersion.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintln(READ_RES_FAIL, "Get defaults data from references"),
			err.Error(),
		)
		return
	}

	if !strings.Contains(image.Version, data.OsVersion.ValueString()) {
		resp.Diagnostics.AddError(
			fmt.Sprintln(READ_RES_FAIL, "validate OS Version"),
			fmt.Sprintf("Unknown OS version %s", data.OsVersion.ValueString()),
		)
		return
	}

	if data.AirflowVersion.IsNull() {
		data.AirflowVersion = types.StringValue(image.AirflowVersion)
	}

	data.OsVersion = types.StringValue(image.Version)
	data.OnSupport = types.BoolValue(image.OnSupport)
	data.OnBackup = types.BoolValue(image.OnBackup)
	data.ProductID = types.StringValue(image.ProductID)
	data.ADIntegration = types.BoolValue(image.ADIntegration)
	data.GeoDistribution = types.BoolValue(image.GeoDistribution)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
