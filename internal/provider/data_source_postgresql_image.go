package provider

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-vtb/pkg/client/references"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource = PostgresqlImageDataSource{}
)

type PostgresqlImageDataSource struct {
	provider *VTBCloudProvider
}

func NewPostgresqlImageDataSource(p *VTBCloudProvider) PostgresqlImageDataSource {
	return PostgresqlImageDataSource{
		provider: p,
	}
}

func (d PostgresqlImageDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = "vtb_postgresql_image_data"
}

type PostgresqlImageDataSourceData struct {
	ProductType                 types.String `tfsdk:"product_type"`
	Distribution                types.String `tfsdk:"distribution"`
	Version                     types.String `tfsdk:"version"`
	OnSupport                   types.Bool   `tfsdk:"on_support"`
	ProductID                   types.String `tfsdk:"product_id"`
	ADIntegration               types.Bool   `tfsdk:"ad_integration"`
	LoadProfile                 types.String `tfsdk:"load_profile"`
	PostgresqlVersion           types.String `tfsdk:"postgresql_version"`
	DefaultTransactionIsolation types.String `tfsdk:"default_transaction_isolation"`
}

func (t PostgresqlImageDataSource) Schema(
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
				Description:         "Name of distribution (RHEL or Astra).",
				MarkdownDescription: "Name of distribution (RHEL or Astra).",
				Validators: []validator.String{
					stringvalidator.OneOf("rhel", "astra"),
				},
			},
			"version": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
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
				Description:         "Product ID from product catalog.",
				MarkdownDescription: "Product ID from product catalog.",
			},
			"ad_integration": schema.BoolAttribute{
				Computed:            true,
				Description:         "Flag for integration with Active Directory.",
				MarkdownDescription: "Flag for integration with Active Directory.",
			},

			// extra image data attrs
			"product_type": schema.StringAttribute{
				Required:            true,
				Description:         "Type of postgresql installation: stand-alone or cluster.",
				MarkdownDescription: "Type of postgresql installation: stand-alone or cluster.",
				Validators: []validator.String{
					stringvalidator.OneOf("stand-alone", "cluster"),
				},
			},
			"load_profile": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Load profile for PostgreSQL installation.",
				MarkdownDescription: "Load profile for PostgreSQL installation.",
				Validators: []validator.String{
					stringvalidator.OneOf("OLTP", "OLAP"),
				},
			},
			"postgresql_version": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "PostgreSQL version.",
				MarkdownDescription: "PostgreSQL version.",
				Validators: []validator.String{
					stringvalidator.OneOf("11", "12", "13", "14", "15"),
				},
			},
			"default_transaction_isolation": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Default Transaction Isolation parameter for PostgreSQL configuration.",
				MarkdownDescription: "Default Transaction Isolation parameter for PostgreSQL configuration.",
				Validators: []validator.String{
					stringvalidator.OneOf("READ COMMITTED", "REPEATABLE READ"),
				},
			},
		}}
}

func (d PostgresqlImageDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data PostgresqlImageDataSourceData
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.provider.Environment == "PROD" && data.ProductType.ValueString() != "cluster" {
		resp.Diagnostics.AddAttributeError(
			path.Root("product_type"),
			"Invalid product type for current environment",
			"only `cluster` product may be choosen for PROD",
		)
		return
	}

	var image *references.PostgresImageData
	var err error
	if strings.EqualFold(d.provider.EnvironmentName, "lt") && strings.EqualFold(data.ProductType.ValueString(), "cluster") {
		image, err = references.GetPostgresImageData(
			d.provider.Creds,
			data.Distribution.ValueString(),
			data.ProductType.ValueString(),
			d.provider.Organization,
			d.provider.EnvironmentName,
			data.Version.ValueString(),
		)
	} else {
		image, err = references.GetPostgresImageData(
			d.provider.Creds,
			data.Distribution.ValueString(),
			data.ProductType.ValueString(),
			d.provider.Organization,
			d.provider.Environment,
			data.Version.ValueString(),
		)
	}
	if err != nil {
		resp.Diagnostics.AddError("Get Image data from references", err.Error())
		return
	}

	if !strings.Contains(image.Version, data.Version.ValueString()) {
		resp.Diagnostics.AddError(
			"Validate OS Version",
			fmt.Sprintf("Unknown OS version %s", data.Version.ValueString()),
		)
		return
	}

	if data.PostgresqlVersion.IsNull() {
		data.PostgresqlVersion = types.StringValue(image.PostgresqlVersion)
	}
	if data.LoadProfile.IsNull() {
		data.LoadProfile = types.StringValue(image.LoadProfile)
	}
	if data.DefaultTransactionIsolation.IsNull() {
		data.DefaultTransactionIsolation = types.StringValue(image.DefaultTransactionIsolation)
	}

	// set state data
	data.Version = types.StringValue(image.Version)
	data.OnSupport = types.BoolValue(image.OnSupport)
	data.ProductID = types.StringValue(image.ProductID)
	data.ADIntegration = types.BoolValue(image.ADIntegration)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
