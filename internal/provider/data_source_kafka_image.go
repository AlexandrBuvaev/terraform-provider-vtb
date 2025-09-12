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
	_ datasource.DataSource = KafkaImageDataSource{}
)

type KafkaImageDataSource struct {
	provider *VTBCloudProvider
}

func NewKafkaImageDataSource(p *VTBCloudProvider) KafkaImageDataSource {
	return KafkaImageDataSource{
		provider: p,
	}
}

func (d KafkaImageDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = "vtb_kafka_image_data"
}

type KafkaImageModel struct {
	Distribution  types.String `tfsdk:"distribution"`
	OsVersion     types.String `tfsdk:"os_version"`
	OnSupport     types.Bool   `tfsdk:"on_support"`
	ProductID     types.String `tfsdk:"product_id"`
	ADIntegration types.Bool   `tfsdk:"ad_integration"`

	DefaultKafkaVersion types.String `tfsdk:"default_kafka_version"`
	GeoDistribution     types.Bool   `tfsdk:"geo_distribution"`
}

func (t KafkaImageDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description:         "Image data from reference service needed to create order",
		MarkdownDescription: "Image data from reference service needed to create order",
		Attributes: map[string]schema.Attribute{
			"distribution": schema.StringAttribute{
				Required:            true,
				Description:         "Name of distribution (RHEL or Astra).",
				MarkdownDescription: "Name of distribution (RHEL or Astra).",
				Validators: []validator.String{
					stringvalidator.OneOf("rhel", "astra"),
				},
			},
			"os_version": schema.StringAttribute{
				Computed:            true,
				Optional:            true,
				Description:         "Verison of distribution image.",
				MarkdownDescription: "Verison of distribution image.",
			},
			"on_support": schema.BoolAttribute{
				Computed:            true,
				Description:         "Flag mean, what the product will supported by sup group.",
				MarkdownDescription: "Flag mean, what the product will supported by sup group.",
			},
			"product_id": schema.StringAttribute{
				Computed:            true,
				Description:         "Product id of kafka cluster instance.",
				MarkdownDescription: "Product id of kafka cluster instance.",
			},
			"ad_integration": schema.BoolAttribute{
				Computed:            true,
				Description:         "Flag for integration compute instance with Active Directory.",
				MarkdownDescription: "Flag for integration compute instance with Active Directory.",
			},

			// extra image data attrs
			"default_kafka_version": schema.StringAttribute{
				Computed:            true,
				Description:         "Verison of kafka distribution image.",
				MarkdownDescription: "Verison of kafka distribution image.",
				Validators: []validator.String{
					stringvalidator.OneOf("2.13-2.4.1", "2.13-2.8.2"),
				},
			},
			"geo_distribution": schema.BoolAttribute{
				Computed:            true,
				Description:         "Flag for using geo distribution of cluster's nodes",
				MarkdownDescription: "Flag for using geo distribution of cluster's nodes",
			},
		},
	}
}

func (d KafkaImageDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data KafkaImageModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	image, err := references.GetKafkaImageData(
		d.provider.Creds,
		data.Distribution.ValueString(),
		d.provider.Organization,
		d.provider.Environment,
		data.OsVersion.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Get defaults data from references", err.Error())
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
	data.GeoDistribution = types.BoolValue(image.GeoDistribution)
	data.DefaultKafkaVersion = types.StringValue(image.DefaultKafkaVersion)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
