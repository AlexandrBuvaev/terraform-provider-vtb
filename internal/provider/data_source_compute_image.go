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
	_ datasource.DataSource = ComputeImageDataSource{}
)

type ComputeImageDataSource struct {
	provider *VTBCloudProvider
}

func NewComputeImageDataSource(p *VTBCloudProvider) ComputeImageDataSource {
	return ComputeImageDataSource{
		provider: p,
	}
}

func (d ComputeImageDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = "vtb_compute_image_data"
}

type ImageStandardModel struct {
	Distribution  types.String `tfsdk:"distribution"`
	OsVersion     types.String `tfsdk:"os_version"`
	OnSupport     types.Bool   `tfsdk:"on_support"`
	ProductID     types.String `tfsdk:"product_id"`
	ADIntegration types.Bool   `tfsdk:"ad_integration"`
}

var dataImageStandardSchema = map[string]schema.Attribute{
	"distribution": schema.StringAttribute{
		Required:            true,
		Description:         "Название диструбутива (RHEL or Astra).",
		MarkdownDescription: "Название диструбутива (RHEL or Astra).",
		Validators: []validator.String{
			stringvalidator.OneOf("rhel", "astra", "linux_preprom"),
		},
	},
	"os_version": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Версия образа дистрибутива",
	},
	"on_support": schema.BoolAttribute{
		Computed:            true,
		MarkdownDescription: "Флаг означает, что продукт будет поддерживаться группой поддержки",
	},
	"product_id": schema.StringAttribute{
		Computed:            true,
		MarkdownDescription: "Идентификатор продукта вычислительного экземпляра",
	},
	"ad_integration": schema.BoolAttribute{
		Computed:            true,
		MarkdownDescription: "Флаг для интеграции вычислительного экземпляра с Active Directory",
	},
}

func (t ComputeImageDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes:          dataImageStandardSchema,
		Description:         "Данные образа из сервиса справочников необходимые для оформления заказа.",
		MarkdownDescription: "Данные образа из сервиса справочников необходимые для оформления заказа.",
	}
}

func (d ComputeImageDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data ImageStandardModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	image, err := references.GetComputeImageData(
		d.provider.Creds,
		data.Distribution.ValueString(),
		d.provider.Organization,
		d.provider.Environment,
		data.OsVersion.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Get Image data from references", err.Error())
		return
	}

	if !strings.Contains(image.Version, data.OsVersion.ValueString()) {
		resp.Diagnostics.AddError(
			"Validate OS Version",
			fmt.Sprintf("Unknown OS version %s", data.OsVersion.ValueString()),
		)
		return
	}

	data.OsVersion = types.StringValue(image.Version)
	data.OnSupport = types.BoolValue(image.OnSupport)
	data.ProductID = types.StringValue(image.ProductID)
	data.ADIntegration = types.BoolValue(image.ADIntegration)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
