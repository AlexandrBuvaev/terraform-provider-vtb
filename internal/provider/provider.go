package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/services/access"
	agentorchestration "terraform-provider-vtb/internal/services/agent_orchestration"
	"terraform-provider-vtb/internal/services/airflow"
	"terraform-provider-vtb/internal/services/astra"
	balancerv3 "terraform-provider-vtb/internal/services/balancer_v3"
	"terraform-provider-vtb/internal/services/clickhouse"
	clusterlayout "terraform-provider-vtb/internal/services/cluster-layout"
	"terraform-provider-vtb/internal/services/core"
	"terraform-provider-vtb/internal/services/elasticsearch"
	"terraform-provider-vtb/internal/services/etcd"
	"terraform-provider-vtb/internal/services/flavor"
	"terraform-provider-vtb/internal/services/grafana"
	gslbv1 "terraform-provider-vtb/internal/services/gslb_v1"
	k8scluster "terraform-provider-vtb/internal/services/k8s_cluster"
	k8scontainerproject "terraform-provider-vtb/internal/services/k8s_container_project"
	k8scontainerspace "terraform-provider-vtb/internal/services/k8s_container_space"
	k8sproject "terraform-provider-vtb/internal/services/k8s_project"
	"terraform-provider-vtb/internal/services/kafka"
	"terraform-provider-vtb/internal/services/ktaas"
	"terraform-provider-vtb/internal/services/nginx"
	openmessaging "terraform-provider-vtb/internal/services/open_messaging"
	"terraform-provider-vtb/internal/services/postgresql"
	"terraform-provider-vtb/internal/services/rabbitmq"
	rabbitmquser "terraform-provider-vtb/internal/services/rabbitmq_user"
	"terraform-provider-vtb/internal/services/redis"
	redissentinel "terraform-provider-vtb/internal/services/redis_sentinel"
	"terraform-provider-vtb/internal/services/rqaas"
	s3ceph "terraform-provider-vtb/internal/services/s3_ceph"
	scylladb "terraform-provider-vtb/internal/services/scylla-db"
	syncxpert "terraform-provider-vtb/internal/services/sync-xpert"
	"terraform-provider-vtb/internal/services/tarantool"
	vtbartemis "terraform-provider-vtb/internal/services/vtb-artemis"
	"terraform-provider-vtb/internal/services/wildfly"
	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/sources"
)

var _ provider.Provider = &VTBCloudProvider{}

type VTBCloudProvider struct {
	configured bool
	version    string
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

	p.configured = true

	client := client.NewCloudClient(creds, project)

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *VTBCloudProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{

		// GIA Products
		func() resource.Resource { return postgresql.NewPostgresqlResource() },
		func() resource.Resource { return redis.NewRedisResource() },
		func() resource.Resource { return redissentinel.NewRedisSentinelResource() },
		func() resource.Resource { return clickhouse.NewClickHouseResource() },
		func() resource.Resource { return clickhouse.NewClickHouseClusterResource() },
		func() resource.Resource { return etcd.NewEtcdResource() },
		func() resource.Resource { return agentorchestration.NewAgentOrchestrationResource() },
		func() resource.Resource { return grafana.NewGrafanaResource() },
		func() resource.Resource { return elasticsearch.NewElasticSearchResource() },
		func() resource.Resource { return scylladb.NewScyllaDbClusterResource() },
		func() resource.Resource { return s3ceph.NewS3CephResource() },

		// USPA Products
		func() resource.Resource { return access.NewAccessGroupResource() },
		func() resource.Resource { return astra.NewComputeResource() },
		func() resource.Resource { return kafka.NewKafkaResource() },
		func() resource.Resource { return wildfly.NewWildflyResource() },
		func() resource.Resource { return nginx.NewNginxResource() },
		func() resource.Resource { return openmessaging.NewOpenMessagingResource() },
		func() resource.Resource { return airflow.NewAirflowClusterResource() },
		func() resource.Resource { return airflow.NewAirflowStandaloneResource() },
		func() resource.Resource { return tarantool.NewTarantoolClusterResource() },
		func() resource.Resource { return rqaas.NewRQaaSResource() },
		func() resource.Resource { return ktaas.NewKTaaSResource() },
		func() resource.Resource { return gslbv1.NewGSLBV1Resource() },

		// Balancer v3

		func() resource.Resource { return balancerv3.NewBalancerV3Resource() },

		// RabbitMQ
		func() resource.Resource { return rabbitmq.NewRabbitMQResource() },
		func() resource.Resource { return rabbitmq.NewRabbitMQVhostsResource() },
		func() resource.Resource { return rabbitmquser.NewRabbitMQUserResource() },

		// VTB Artemis
		func() resource.Resource { return vtbartemis.NewArtemisResource() },
		func() resource.Resource { return vtbartemis.NewArtemisTuzResource() },
		func() resource.Resource { return vtbartemis.NewArtemisAddresPolicyResource() },
		func() resource.Resource { return vtbartemis.NewArtemisRolesResource() },

		// Debezium
		func() resource.Resource { return syncxpert.NewSyncXpertClusterResource() },
		func() resource.Resource { return syncxpert.NewSyncXpertConnectorResource() },

		// K8s Products
		func() resource.Resource { return k8sproject.NewK8sProjectResource() },
		func() resource.Resource { return k8scluster.NewK8sClusterResource() },
		func() resource.Resource { return k8scontainerspace.NewK8sContainerSpaceResource() },
		func() resource.Resource { return k8scontainerproject.NewK8sSpaceProjectResource() },
	}
}

func (p *VTBCloudProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		func() datasource.DataSource { return clusterlayout.NewClusterProductLayoutDataSource() },
		func() datasource.DataSource { return flavor.NewFlavorDataSource() },
		func() datasource.DataSource { return access.NewUserDataSource() },
		func() datasource.DataSource { return core.NewCoreDataSource() },
		func() datasource.DataSource { return astra.NewComputeImageDataSource() },
		func() datasource.DataSource { return postgresql.NewPostgresqlImageDataSource() },
		func() datasource.DataSource { return wildfly.NewWildflyImageDataSource() },
		func() datasource.DataSource { return nginx.NewNginxImageDataSource() },
		func() datasource.DataSource { return redis.NewRedisImageDataSource() },
		func() datasource.DataSource { return clickhouse.NewClickhouseImageDataSource() },
		func() datasource.DataSource { return clickhouse.NewClickhouseClusterImageDataSource() },
		func() datasource.DataSource { return openmessaging.NewOpenMessagingDataSource() },
		func() datasource.DataSource { return kafka.NewKafkaImageDataSource() },
		func() datasource.DataSource { return rabbitmq.NewRabbitMQImageDataSource() },
		func() datasource.DataSource { return vtbartemis.NewArtemisImageDataSource() },
		func() datasource.DataSource { return syncxpert.NewDebeziumImageDataSource() },
		func() datasource.DataSource { return balancerv3.NewBalancerV3ImageDataSource() },
		func() datasource.DataSource { return airflow.NewAirflowImageDataSource() },
		func() datasource.DataSource { return etcd.NewEtcdImageDataSource() },
		func() datasource.DataSource { return agentorchestration.NewAgentOrchestrationImageDataSource() },
		func() datasource.DataSource { return grafana.NewGrafanaImageDataSource() },
		func() datasource.DataSource { return tarantool.NewTarantoolDataGridImageDataSource() },
		func() datasource.DataSource { return tarantool.NewTarantooEnterpriseImageDataSource() },
		func() datasource.DataSource { return rqaas.NewRQaaSClusterDataSource() },
		func() datasource.DataSource { return redissentinel.NewRedisSentinelImageDataSource() },
		func() datasource.DataSource { return agentorchestration.NewJenkinsAgentSubsystemDataSource() },
		func() datasource.DataSource { return elasticsearch.NewElasticSearchImageDataSource() },
		func() datasource.DataSource { return scylladb.NewScyllaDbClusterImageDataSource() },
		func() datasource.DataSource { return s3ceph.NewS3CephImageDataSource() },
		func() datasource.DataSource { return gslbv1.NewGSLBV1ImageDataSource() },
	}
}
