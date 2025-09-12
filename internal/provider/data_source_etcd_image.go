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
	_ datasource.DataSource = EtcdImageDataSource{}
)

type EtcdImageDataSource struct {
	provider *VTBCloudProvider
}

func NewEtcdImageDataSource(p *VTBCloudProvider) EtcdImageDataSource {
	return EtcdImageDataSource{
		provider: p,
	}
}

func (d EtcdImageDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = "vtb_etcd_image_data"
}

type EtcdImageDataSourceData struct {
	Distribution    types.String `tfsdk:"distribution"`
	OSVersion       types.String `tfsdk:"os_version"`
	OnSupport       types.Bool   `tfsdk:"on_support"`
	ProductID       types.String `tfsdk:"product_id"`
	ADIntegration   types.Bool   `tfsdk:"ad_integration"`
	GeoDistribution types.Bool   `tfsdk:"geo_distribution"`
	OnBackup        types.Bool   `tfsdk:"on_backup"`
	UseSsl          types.Bool   `tfsdk:"use_ssl"`
}

func (t EtcdImageDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Type of Postgresql instance",
		Attributes: map[string]schema.Attribute{
			// basic image data attrs
			"distribution": schema.StringAttribute{
				Required:            true,
				Description:         "Имя дистрибутива (RHEL или Astra)",
				MarkdownDescription: "Имя дистрибутива (RHEL или Astra).",
				Validators: []validator.String{
					stringvalidator.OneOf("rhel", "astra"),
				},
			},
			"os_version": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Версия образа",
				MarkdownDescription: "Версия образа",
			},
			"on_support": schema.BoolAttribute{
				Computed:            true,
				Description:         "Флаг обозначающий, осуществляется ли поддерджка группой сопровождения",
				MarkdownDescription: "Флаг обозначающий, осуществляется ли поддерджка группой сопровождения",
			},
			"product_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Идентификатор продукта в продуктовом каталоге",
				MarkdownDescription: "Идентификатор продукта в продуктовом каталоге",
			},
			"ad_integration": schema.BoolAttribute{
				Computed:            true,
				Description:         "Флаг обозначающий, включена ли интеграция с Active Directory",
				MarkdownDescription: "Флаг обозначающий, включена ли интеграция с Active Directory",
			},

			// extra image data attrs
			"use_ssl": schema.BoolAttribute{
				Computed:            true,
				Description:         "Использовать SSL",
				MarkdownDescription: "Использовать SSL",
			},
			"on_backup": schema.BoolAttribute{
				Computed:            true,
				Description:         "Использовать резервное копирование",
				MarkdownDescription: "Использовать резервное копирование",
			},
			"geo_distribution": schema.BoolAttribute{
				Computed:            true,
				Description:         "Использовать геораспределение",
				MarkdownDescription: "Использовать геораспределение",
			},
		}}
}

func (d EtcdImageDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data EtcdImageDataSourceData
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	image, err := references.GetEtcdImageData(
		d.provider.Creds,
		data.Distribution.ValueString(),
		d.provider.Organization,
		d.provider.Environment,
		data.OSVersion.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Get image data from reference", err.Error())
		return
	}

	if !strings.Contains(image.Version, data.OSVersion.ValueString()) {
		resp.Diagnostics.AddError(
			"Validate OS Version",
			fmt.Sprintf("Unknown OS version %s", data.OSVersion.ValueString()),
		)
		return
	}

	data.OSVersion = types.StringValue(image.Version)
	data.OnSupport = types.BoolValue(image.OnSupport)
	data.ProductID = types.StringValue(image.ProductID)
	data.ADIntegration = types.BoolValue(image.ADIntegration)
	data.OnBackup = types.BoolValue(image.OnBackup)
	data.UseSsl = types.BoolValue(image.UseSsl)
	data.GeoDistribution = types.BoolValue(image.GeoDistribution)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
