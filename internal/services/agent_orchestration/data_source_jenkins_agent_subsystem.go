package agentorchestration

import (
	"context"
	"fmt"
	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/consts"
	"terraform-provider-vtb/pkg/client/sources"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource = &JenkinsAgentSubsystemDataSource{}
)

type JenkinsAgentSubsystemDataSource struct {
	client *client.CloudClient
}

func NewJenkinsAgentSubsystemDataSource() datasource.DataSource {
	return &JenkinsAgentSubsystemDataSource{}
}

func (d JenkinsAgentSubsystemDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_jenkins_agent_subsystem_data"
}

func (d *JenkinsAgentSubsystemDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

type JenkinsAgentSubsystemModel struct {
	RisID         types.String `tfsdk:"ris_id"`
	IsCode        types.String `tfsdk:"is_code"`
	HeadName      types.String `tfsdk:"head_name"`
	NetSegment    types.String `tfsdk:"net_segment"`
	DisplayName   types.String `tfsdk:"display_name"`
	SferaHeadURL  types.String `tfsdk:"sfera_head_url"`
	NodeGroupName types.String `tfsdk:"node_group_name"`
}

var dataJenkinsAgentSubsystemSchema = map[string]schema.Attribute{
	"ris_id": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Код информационной системы",
		Description:         "Код информационной системы",
	},
	"is_code": schema.StringAttribute{
		Computed:            true,
		MarkdownDescription: "Код информационной системы (IS code)",
		Description:         "Код информационной системы (IS code)",
	},
	"head_name": schema.StringAttribute{
		Computed:            true,
		MarkdownDescription: "Название головной системы",
		Description:         "Название головной системы",
	},
	"net_segment": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "Сетевой сегмент",
		Description:         "Сетевой сегмент",
	},
	"display_name": schema.StringAttribute{
		Computed:            true,
		MarkdownDescription: "Отображаемое имя подсистемы",
		Description:         "Отображаемое имя подсистемы",
	},
	"sfera_head_url": schema.StringAttribute{
		Computed:            true,
		MarkdownDescription: "URL головной системы в сфере",
		Description:         "URL головной системы в сфере",
	},
	"node_group_name": schema.StringAttribute{
		Computed:            true,
		MarkdownDescription: "Имя группы нод (агентов)",
		Description:         "Имя группы нод (агентов)",
	},
}

func (d JenkinsAgentSubsystemDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes:          dataJenkinsAgentSubsystemSchema,
		Description:         "Схема данных для подсистемы агента Jenkins",
		MarkdownDescription: "Схема данных для подсистемы агента Jenkins",
	}
}

func (d JenkinsAgentSubsystemDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data JenkinsAgentSubsystemModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	jenkinsSubsystems, err := sources.GetOrchestrationAgents(
		d.client.Creds,
		data.RisID.ValueString(),
		data.NetSegment.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Get orchestration agents ended with error", err.Error())
		return
	}

	availiableSubsystemNames := []string{}
	for _, subsystem := range jenkinsSubsystems {
		availiableSubsystemNames = append(availiableSubsystemNames, subsystem.DisplayName.Value)
	}

	found := false
	for _, subsystem := range jenkinsSubsystems {
		if subsystem.RisID.Value == data.RisID.ValueString() {
			data.DisplayName = types.StringValue(subsystem.DisplayName.Value)
			data.HeadName = types.StringValue(subsystem.HeadName.Value)
			data.IsCode = types.StringValue(subsystem.IsCode.Value)
			data.SferaHeadURL = types.StringValue(subsystem.SferaHeadURL.Value)
			data.NodeGroupName = types.StringValue(subsystem.NodeGroupName.Value)
			found = true
			break
		}
	}

	if !found {
		resp.Diagnostics.AddError(
			consts.READ_RES_FAIL,
			fmt.Sprintf("Not found Orchestration Agents for ris_id %s, Available: %v",
				data.RisID.ValueString(), availiableSubsystemNames),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
