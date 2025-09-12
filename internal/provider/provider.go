package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/sources"
)

var _ provider.Provider = &VTBCloudProvider{}

type VTBCloudProvider struct {
	configured      bool
	version         string
	ClientId        string
	ClientSecret    string
	ProjectName     string
	Organization    string
	Environment     string
	EnvironmentName string
	EnvID           string
	RisCode         string
	RisID           string
	EnvPrefix       string
	RisShortName    string
	Creds           *auth.Credentials
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &VTBCloudProvider{
			version: version,
		}
	}
}

func (p *VTBCloudProvider) Metadata(
	_ context.Context,
	_ provider.MetadataRequest,
	resp *provider.MetadataResponse,
) {
	resp.TypeName = "vtb"
}

type VTBProviderModel struct {
	ClientId     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	ProjectName  types.String `tfsdk:"project_name"`
}

func (p *VTBCloudProvider) Schema(
	ctx context.Context,
	req provider.SchemaRequest,
	resp *provider.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description:         "Manages authentication for VTB Cloud service account.",
		MarkdownDescription: "Manages authentication for VTB Cloud service account.",
		Attributes: map[string]schema.Attribute{
			"client_id": schema.StringAttribute{
				MarkdownDescription: "Identification of service client id",
				Required:            true,
			},
			"client_secret": schema.StringAttribute{
				MarkdownDescription: "Private secret of service client",
				Required:            true,
				Sensitive:           true,
			},
			"project_name": schema.StringAttribute{
				MarkdownDescription: "Name of project where will placed orders",
				Required:            true,
			},
		},
	}
}

func (p *VTBCloudProvider) Configure(
	ctx context.Context,
	req provider.ConfigureRequest,
	resp *provider.ConfigureResponse,
) {
	var config VTBProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	creds, err := auth.NewCredentials(
		config.ClientId.ValueString(),
		config.ClientSecret.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Can't get access token for authorization", err.Error())
		return
	}

	project, err := sources.GetProject(
		creds,
		config.ProjectName.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Get project data from portal", err.Error())
		return
	}

	p.ProjectName = project.Name
	p.Organization = project.Organization
	p.Environment = project.ProjectEnvironment.EnvironmentType
	p.EnvironmentName = project.ProjectEnvironment.Name
	p.EnvPrefix = project.EnvironmentPrefix.Name
	p.EnvID = project.ProjectEnvironment.ID
	p.RisCode = project.InformationSystem.Code
	p.EnvPrefix = project.EnvironmentPrefix.Name
	p.RisShortName = project.InformationSystem.ShortName
	p.RisID = project.InformationSystem.RisID
	p.ClientId = config.ClientId.ValueString()
	p.ClientSecret = config.ClientSecret.ValueString()
	p.Creds = creds

	p.configured = true
}

func (p *VTBCloudProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{

		// GIA Products
		func() resource.Resource { return NewPostgresqlResource(p) },
		func() resource.Resource { return NewRedisResource(p) },
		func() resource.Resource { return NewRedisSentinelResource(p) },
		func() resource.Resource { return NewClickHouseResource(p) },
		func() resource.Resource { return NewClickHouseClusterResource(p) },
		func() resource.Resource { return NewEtcdResource(p) },
		func() resource.Resource { return NewAgentOrchestrationResource(p) },
		func() resource.Resource { return NewGrafanaResource(p) },
		func() resource.Resource { return NewElasticSearchResource(p) },
		func() resource.Resource { return NewScyllaDbClusterResource(p) },

		// USPA Products
		func() resource.Resource { return NewAccessGroupResource(p) },
		func() resource.Resource { return NewComputeResource(p) },
		func() resource.Resource { return NewKafkaResource(p) },
		func() resource.Resource { return NewWildflyResource(p) },
		func() resource.Resource { return NewNginxResource(p) },
		func() resource.Resource { return NewOpenMessagingResource(p) },
		func() resource.Resource { return NewAirflowClusterResource(p) },
		func() resource.Resource { return NewAirflowStandaloneResource(p) },
		func() resource.Resource { return NewTarantoolClusterResource(p) },
		func() resource.Resource { return NewRQaaSResource(p) },
		func() resource.Resource { return NewKTaaSResource(p) },

		// Balancer v3

		func() resource.Resource { return NewBalancerV3Resource(p) },

		// RabbitMQ
		func() resource.Resource { return NewRabbitMQResource(p) },
		func() resource.Resource { return NewRabbitMQVhostsResource(p) },
		func() resource.Resource { return NewRabbitMQUserResource(p) },

		// VTB Artemis
		func() resource.Resource { return NewArtemisResource(p) },
		func() resource.Resource { return NewArtemisTuzResource(p) },
		func() resource.Resource { return NewArtemisAddresPolicyResource(p) },
		func() resource.Resource { return NewArtemisRolesResource(p) },

		// Debezium
		func() resource.Resource { return NewSyncXpertClusterResource(p) },
		func() resource.Resource { return NewSyncXpertConnectorResource(p) },

		// K8s Products
		func() resource.Resource { return NewK8sProjectResource(p) },
		func() resource.Resource { return NewK8sClusterResource(p) },
	}
}

func (p *VTBCloudProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		func() datasource.DataSource { return NewClusterProductLayoutDataSource(p) },
		func() datasource.DataSource { return NewFlavorDataSource(p) },
		func() datasource.DataSource { return NewUserDataSource(p) },
		func() datasource.DataSource { return NewCoreDataSource(p) },
		func() datasource.DataSource { return NewComputeImageDataSource(p) },
		func() datasource.DataSource { return NewPostgresqlImageDataSource(p) },
		func() datasource.DataSource { return NewWildflyImageDataSource(p) },
		func() datasource.DataSource { return NewNginxImageDataSource(p) },
		func() datasource.DataSource { return NewRedisImageDataSource(p) },
		func() datasource.DataSource { return NewClickhouseImageDataSource(p) },
		func() datasource.DataSource { return NewClickhouseClusterImageDataSource(p) },
		func() datasource.DataSource { return NewOpenMessagingDataSource(p) },
		func() datasource.DataSource { return NewKafkaImageDataSource(p) },
		func() datasource.DataSource { return NewRabbitMQImageDataSource(p) },
		func() datasource.DataSource { return NewArtemisImageDataSource(p) },
		func() datasource.DataSource { return NewDebeziumImageDataSource(p) },
		func() datasource.DataSource { return NewBalancerV3ImageDataSource(p) },
		func() datasource.DataSource { return NewAirflowImageDataSource(p) },
		func() datasource.DataSource { return NewEtcdImageDataSource(p) },
		func() datasource.DataSource { return NewAgentOrchestrationImageDataSource(p) },
		func() datasource.DataSource { return NewGrafanaImageDataSource(p) },
		func() datasource.DataSource { return NewTarantoolDataGridImageDataSource(p) },
		func() datasource.DataSource { return NewTarantooEnterpriseImageDataSource(p) },
		func() datasource.DataSource { return NewRQaaSClusterDataSource(p) },
		func() datasource.DataSource { return NewRedisSentinelImageDataSource(p) },
		func() datasource.DataSource { return NewJenkinsAgentSubsystemDataSource(p) },
		func() datasource.DataSource { return NewElasticSearchImageDataSource(p) },
		func() datasource.DataSource { return NewScyllaDbClusterImageDataSource(p) },
	}
}
