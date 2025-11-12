package s3ceph

import (
	"context"
	"fmt"

	//"fmt"
	"strings"

	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/pkg/client/references"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource = &S3CephImageDataSource{}
)

type S3CephImageDataSource struct {
	client *client.CloudClient
}

func NewS3CephImageDataSource() datasource.DataSource {
	return &S3CephImageDataSource{}
}

func (d S3CephImageDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_s3_ceph_image_data"
}

func (d *S3CephImageDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type S3CephImageDataSourceData struct {
	ProductID   types.String `tfsdk:"product_id"`
	StorageType types.String `tfsdk:"storage_type"`
}

func (t S3CephImageDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Type of S3Ceph instance",
		Attributes: map[string]schema.Attribute{
			// basic image data attrs
			"product_id": schema.StringAttribute{
				Computed:            true,
				Description:         "ID продукта из продуктового каталога.",
				MarkdownDescription: "ID продукта из продуктового каталога.",
			},
			// extra image data attrs
			"storage_type": schema.StringAttribute{
				Required:            true,
				Description:         "Тип хранилища S3Ceph: hdd, nvme или backup.",
				MarkdownDescription: "Тип хранилища S3Ceph: hdd, nvme или backup.",
				Validators: []validator.String{
					stringvalidator.OneOf("hdd", "nvme", "backup"),
				},
			},
		}}
}

func (d S3CephImageDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data S3CephImageDataSourceData
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if d.client.Environment != "PROD" && data.StorageType.ValueString() != "hdd" {
		resp.Diagnostics.AddAttributeError(
			path.Root("storage_type"),
			"Invalid storage type for current environment",
			"only `hdd` storage type may be choosen for non PROD environment",
		)
		return
	}

	var image *references.S3CephImageData
	var err error
	if strings.ToLower(d.client.EnvironmentName) == "lt" {
		image, err = references.GetS3CephImageData(
			d.client.Creds,
			d.client.Organization,
			d.client.EnvironmentName,
		)
	} else {
		image, err = references.GetS3CephImageData(
			d.client.Creds,
			d.client.Organization,
			d.client.Environment,
		)
	}
	if err != nil {
		resp.Diagnostics.AddError("Get Image data from references", err.Error())
		return
	}

	// set state data
	data.ProductID = types.StringValue(image.ProductID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
