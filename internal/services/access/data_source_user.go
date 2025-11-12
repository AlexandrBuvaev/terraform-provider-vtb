package access

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/pkg/client/sources"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource = UserDataSource{}
)

type UserDataSource struct {
	client *client.CloudClient
}

func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

func (d UserDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_user_data"
}

func (d *UserDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type VTBDomainUserModel struct {
	QueryString types.String `tfsdk:"query_string"`
	Domain      types.String `tfsdk:"domain"`
	ID          types.String `tfsdk:"id"`
	Username    types.String `tfsdk:"username"`
	Name        types.String `tfsdk:"name"`
	Email       types.String `tfsdk:"email"`
	UniqueName  types.String `tfsdk:"unique_name"`
}

func (t UserDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Core/memory of compute instance",
		Attributes: map[string]schema.Attribute{
			"query_string": schema.StringAttribute{
				Required: true,
			},
			"domain": schema.StringAttribute{
				Required: true,
			},
			"id": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"username": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"name": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"email": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
			"unique_name": schema.StringAttribute{
				Optional: true,
				Computed: true,
			},
		},
	}
}

func (d UserDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data VTBDomainUserModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var domainNames []string
	domains, err := sources.GetDomainsByProjectName(
		d.client.Creds,
		d.client.ProjectName,
	)
	if err != nil {
		resp.Diagnostics.AddError("Get domains from portal", err.Error())
		return
	}

	for _, d := range domains {
		domainNames = append(domainNames, d.Code)
	}

	domainAvaiable := false
	for _, domainName := range domainNames {
		if strings.EqualFold(domainName, data.Domain.ValueString()) {
			domainAvaiable = true
			break
		}
	}

	if !domainAvaiable {
		resp.Diagnostics.AddError(
			"Domain not available for resource",
			fmt.Sprintf(
				"Available domains [%s]. Specified '%s'",
				strings.Join(domainNames, ", "),
				data.Domain.ValueString(),
			),
		)
		return
	}

	// Find users by query string
	foundUsers, err := sources.GetUsersByQuery(
		d.client.Creds,
		data.QueryString.ValueString(),
		d.client.ProjectName,
		data.Domain.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating access group resource error",
			fmt.Sprintf(
				"Can't append user to group with name '%s' by query string '%s'",
				data.Name, err.Error(),
			),
		)
		return
	}

	if len(foundUsers) > 1 {
		resp.Diagnostics.AddError(
			"User data source error",
			fmt.Sprintf(
				"For your query string '%s' found more then 1 user. "+
					"Please specify more details in your query",
				data.QueryString.ValueString(),
			),
		)
		return
	}

	if len(foundUsers) == 0 {
		resp.Diagnostics.AddError(
			"User data source error",
			fmt.Sprintf(
				"Can't find user by your query '%s'. Please specify another querystring",
				data.QueryString.ValueString(),
			),
		)
		return
	}

	foundUser := foundUsers[0]
	data.ID = types.StringValue(foundUser.ID)
	data.Username = types.StringValue(foundUser.Username)
	data.Name = types.StringValue(foundUser.Name)
	data.Email = types.StringValue(foundUser.Email)
	data.UniqueName = types.StringValue(foundUser.UniqueName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
