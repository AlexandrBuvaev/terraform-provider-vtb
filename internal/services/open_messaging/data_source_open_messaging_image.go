package openmessaging

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/common"
	"terraform-provider-vtb/pkg/client/references"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource = &OpenMessagingDataSource{}
)

type OpenMessagingDataSource struct {
	client *client.CloudClient
}

func NewOpenMessagingDataSource() datasource.DataSource {
	return &OpenMessagingDataSource{}
}

func (d OpenMessagingDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_open_messaging_image_data"
}

func (d *OpenMessagingDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (t OpenMessagingDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes:          common.DataImageStandardSchema,
		Description:         "Image data from reference service needed to create order",
		MarkdownDescription: "Image data from reference service needed to create order",
	}
}

func (d OpenMessagingDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data common.ImageStandardModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var image *references.OpenMessagingImageData
	var err error
	if strings.EqualFold(d.client.EnvironmentName, "lt") {
		image, err = references.GetOpenMessagingImageData(
			d.client.Creds,
			data.Distribution.ValueString(),
			d.client.Organization,
			d.client.EnvironmentName,
			data.OsVersion.ValueString(),
		)
	} else {
		image, err = references.GetOpenMessagingImageData(
			d.client.Creds,
			data.Distribution.ValueString(),
			d.client.Organization,
			d.client.Environment,
			data.OsVersion.ValueString(),
		)
	}

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
