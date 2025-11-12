package entities

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ProviderTyper interface {
	GetProviderType() (string, string)
}

type Item struct {
	CreatedRowDT string   `json:"created_row_dt"`
	UpdateDT     string   `json:"update_dt"`
	ID           string   `json:"item_id"`
	OrderId      string   `json:"order_id"`
	GraphId      string   `json:"graph_id"`
	Type         string   `json:"type"`
	Data         ItemData `json:"data"`
}

type Build struct {
	SetupVersion string `json:"setup_version"`
	// Optional fields
	ArtemisVersion  interface{} `json:"artemis_version,omitempty"`
	RabbitMQVersion *string     `json:"rabbitmq_version,omitempty"`
}

type MmMode struct {
	DateEnd string `json:"date_end"`
}

type ItemData struct {
	State    string        `json:"state"`
	ACLs     []AccessACL   `json:"acls"`
	Parent   string        `json:"parent"`
	Provider string        `json:"provider"`
	Build    Build         `json:"build"`
	MmMode   MmMode        `json:"mm_mode,omitempty"`
	Config   ProviderTyper `json:"config"`
}

type AccessACL struct {
	Role    string   `json:"role"`
	Members []string `json:"members"`
}

func (i *Item) UnmarshalJSON(data []byte) (err error) {

	var rawData struct {
		CreatedRowDT string `json:"created_row_dt"`
		UpdateDT     string `json:"update_dt"`
		ID           string `json:"item_id"`
		OrderId      string `json:"order_id"`
		GraphId      string `json:"graph_id"`
		Type         string `json:"type"`
		Data         struct {
			State    string          `json:"state"`
			ACLs     []AccessACL     `json:"acls"`
			Parent   string          `json:"parent"`
			Provider string          `json:"provider"`
			Build    Build           `json:"build"`
			MmMode   MmMode          `json:"mm_mode,omitempty"`
			Config   json.RawMessage `json:"config"`
		} `json:"data"`
	}
	err = json.Unmarshal(data, &rawData)
	if err != nil {
		return
	}

	*i = Item{
		CreatedRowDT: rawData.CreatedRowDT,
		UpdateDT:     rawData.UpdateDT,
		ID:           rawData.ID,
		OrderId:      rawData.OrderId,
		GraphId:      rawData.GraphId,
		Type:         rawData.Type,
		Data: ItemData{
			State:    rawData.Data.State,
			ACLs:     rawData.Data.ACLs,
			Parent:   rawData.Data.Parent,
			Provider: rawData.Data.Provider,
			Build:    rawData.Data.Build,
			MmMode:   rawData.Data.MmMode,
		},
	}

	if i.Type == "vm" {
		var vmConfig VMItemConfig
		err = json.Unmarshal(rawData.Data.Config, &vmConfig)
		if err != nil {
			return err
		}
		i.Data.Config = vmConfig
		return
	}

	if i.Type == "snapshot" {
		var snapshotConfig SnapshotItemConfig
		err = json.Unmarshal(rawData.Data.Config, &snapshotConfig)
		if err != nil {
			return err
		}
		i.Data.Config = snapshotConfig
		return
	}

	if (i.Type == "cluster") && (i.Data.Provider == "kafka") {
		var kafkaConfig KafkaItemConfig
		err = json.Unmarshal(rawData.Data.Config, &kafkaConfig)
		if err != nil {
			return err
		}
		i.Data.Config = kafkaConfig
		return
	}

	if (i.Type == "cluster") && (strings.Contains(i.Data.Provider, "kubernetes_v1")) {
		var k8sClusterConfig K8sClusterConfig
		err = json.Unmarshal(rawData.Data.Config, &k8sClusterConfig)
		if err != nil {
			return err
		}
		i.Data.Config = k8sClusterConfig
		return
	}

	if (i.Type == "project") && (i.Data.Provider == "kubernetes") {
		var k8sProjectConfig K8sProjectItemConfig
		err = json.Unmarshal(rawData.Data.Config, &k8sProjectConfig)
		if err != nil {
			return err
		}
		i.Data.Config = k8sProjectConfig
		return
	}

	if (i.Type == "container_space") && (i.Data.Provider == "kubernetes") {
		var k8sContainerSpaceConfig K8sContainerSpaceConfig
		err = json.Unmarshal(rawData.Data.Config, &k8sContainerSpaceConfig)
		if err != nil {
			return err
		}
		i.Data.Config = k8sContainerSpaceConfig
		return
	}

	if (i.Type == "app") && i.Data.Provider == "wildfly" {
		var wildFlyConfig WildflyItemConfig
		err = json.Unmarshal(rawData.Data.Config, &wildFlyConfig)
		if err != nil {
			return err
		}
		i.Data.Config = wildFlyConfig
		return
	}

	if (i.Type == "app" || i.Type == "cluster") && (i.Data.Provider == "postgresql_v001") {
		var postgreConfig PostgresqlItemConfig
		err = json.Unmarshal(rawData.Data.Config, &postgreConfig)
		if err != nil {
			return err
		}
		i.Data.Config = postgreConfig
		return
	}

	if (i.Type == "db") && (i.Data.Provider == "postgresql_v001") {
		var dbConfig PostgresqlDbItemConfig
		err = json.Unmarshal(rawData.Data.Config, &dbConfig)
		if err != nil {
			return err
		}
		i.Data.Config = dbConfig
		return
	}

	if (i.Type == "slot") && (i.Data.Provider == "postgresql_v001") {
		var slotConfig PostgresqlItemSlot
		err = json.Unmarshal(rawData.Data.Config, &slotConfig)
		if err != nil {
			return err
		}
		i.Data.Config = slotConfig
		return
	}

	if (i.Type == "publication") && (i.Data.Provider == "postgresql_v001") {
		var publicationConfig PostgresqlItemPublication
		err = json.Unmarshal(rawData.Data.Config, &publicationConfig)
		if err != nil {
			return err
		}
		i.Data.Config = publicationConfig
		return
	}

	if (i.Type == "app" || i.Type == "cluster") && (i.Data.Provider == "redis") {
		var redisConfig RedisItemConfig
		err = json.Unmarshal(rawData.Data.Config, &redisConfig)
		if err != nil {
			return err
		}
		i.Data.Config = redisConfig
		return
	}

	if (i.Type == "app") && (i.Data.Provider == "redis_sentinel") {
		var redisConfig RedisSentinelItemConfig
		err = json.Unmarshal(rawData.Data.Config, &redisConfig)
		if err != nil {
			return err
		}
		i.Data.Config = redisConfig
		return
	}

	if (i.Type == "app") && (i.Data.Provider == "clickhouse") {
		var clickhouseConfig ClickHouseItemConfig
		err = json.Unmarshal(rawData.Data.Config, &clickhouseConfig)
		if err != nil {
			return err
		}
		i.Data.Config = clickhouseConfig
		return
	}

	if (i.Type == "cluster") && (i.Data.Provider == "clickhouse") {
		var clickhouseClusterConfig ClickhouseClusterItemConfig
		err = json.Unmarshal(rawData.Data.Config, &clickhouseClusterConfig)
		if err != nil {
			return err
		}
		i.Data.Config = clickhouseClusterConfig
		return
	}

	if (i.Type == "app") && (i.Data.Provider == "nginx_develop" || i.Data.Provider == "nginx") {
		var nginxConfig NginxItemConfig
		err = json.Unmarshal(rawData.Data.Config, &nginxConfig)
		if err != nil {
			return err
		}
		i.Data.Config = nginxConfig
		return
	}

	if (i.Type == "app") && (i.Data.Provider == "agent_orchestration") {
		var agent_orchestrationConfig AgentOrchestrationItemConfig
		err = json.Unmarshal(rawData.Data.Config, &agent_orchestrationConfig)
		if err != nil {
			return err
		}
		i.Data.Config = agent_orchestrationConfig
		return
	}

	if (i.Type == "cluster") && (i.Data.Provider == "rabbitmq" || i.Data.Provider == "rabbitmq_develop") {
		var rabbitConfig RabbitMQItemConfig
		err = json.Unmarshal(rawData.Data.Config, &rabbitConfig)
		if err != nil {
			return err
		}
		i.Data.Config = rabbitConfig
		return
	}

	if (i.Type == "cluster") && (i.Data.Provider == "vtb-artemis") {
		var artemisConfig VTBArtemisItemConfig
		err = json.Unmarshal(rawData.Data.Config, &artemisConfig)
		if err != nil {
			return err
		}
		i.Data.Config = artemisConfig
		return
	}

	// OpenMessaging Astra
	if (i.Type == "app") && (i.Data.Provider == "artemis") {
		var omConfig OpenMessagingItemConfig
		err = json.Unmarshal(rawData.Data.Config, &omConfig)
		if err != nil {
			return err
		}
		i.Data.Config = omConfig
		return
	}
	// OpenMessaging Astra LT
	if (i.Type == "app") && (i.Data.Provider == "artemis_lt") {
		var omConfig OpenMessagingLtItemConfig
		err = json.Unmarshal(rawData.Data.Config, &omConfig)
		if err != nil {
			return err
		}
		i.Data.Config = omConfig
		return
	}

	if (i.Type == "cluster") && (i.Data.Provider == "debezium") {
		var debeziumConfig SyncXpertItemConfig
		err = json.Unmarshal(rawData.Data.Config, &debeziumConfig)
		if err != nil {
			return err
		}
		i.Data.Config = debeziumConfig
		return
	}

	// TODO BALANCER сделать собственный анмаршалер айтема
	if (i.Type == "cluster") && (strings.Contains(i.Data.Provider, "balancer_v3")) {
		var balancerV3 BalancerV3Item
		err = json.Unmarshal(data, &balancerV3)
		if err != nil {
			return err
		}
		i.Data.Config = balancerV3.Data
		return
	}

	if i.Type == "graph" {
		return
	}

	if (i.Type == "cluster") && (i.Data.Provider == "airflow") {
		var airflowConfig AirflowClusterItemConfig
		err = json.Unmarshal(rawData.Data.Config, &airflowConfig)
		if err != nil {
			return err
		}
		i.Data.Config = airflowConfig
		return
	}

	if (i.Type == "app") && (i.Data.Provider == "airflow") {
		var airflowConfig AirflowStandaloneItemConfig
		err = json.Unmarshal(rawData.Data.Config, &airflowConfig)
		if err != nil {
			return err
		}
		i.Data.Config = airflowConfig
		return

	}

	if (i.Type == "cluster") && (i.Data.Provider == "etcd") {
		var etcdConfig EtcdlItemConfig
		err = json.Unmarshal(rawData.Data.Config, &etcdConfig)
		if err != nil {
			return err
		}
		i.Data.Config = etcdConfig
		return
	}

	if (i.Type == "app") && (i.Data.Provider == "grafana") {
		var grafanaConfig GrafanaItemConfig
		err = json.Unmarshal(rawData.Data.Config, &grafanaConfig)
		if err != nil {
			return err
		}
		i.Data.Config = grafanaConfig
		return
	}

	if (i.Type == "cluster") && (i.Data.Provider == "tarantool_v2") {
		var tarantoolDataGridConfig TarantoolClusterItemConfig
		err = json.Unmarshal(rawData.Data.Config, &tarantoolDataGridConfig)
		if err != nil {
			return err
		}
		i.Data.Config = tarantoolDataGridConfig
		return
	}

	if (i.Type == "saas") && (i.Data.Provider == "rqaas") {
		var rqaasConfig RQaaSItemConfig
		err = json.Unmarshal(rawData.Data.Config, &rqaasConfig)
		if err != nil {
			return err
		}
		i.Data.Config = rqaasConfig
		return
	}

	if (i.Type == "paas_ktaas") && (i.Data.Provider == "ktaas") {
		var ktaaSConfig KTaaSConfig
		err = json.Unmarshal(rawData.Data.Config, &ktaaSConfig)
		if err != nil {
			return err
		}
		i.Data.Config = ktaaSConfig
		return
	}

	if (i.Type == "cluster") && (i.Data.Provider == "elasticsearch_os") {
		var elasticSearchConfig ElasticSearchConfig
		err = json.Unmarshal(rawData.Data.Config, &elasticSearchConfig)
		if err != nil {
			return err
		}
		i.Data.Config = elasticSearchConfig
		return
	}

	if (i.Type == "cluster") && (i.Data.Provider == "scylladb") {
		var scylladbclusterConfig ScyllaDbClusterItemConfig
		err = json.Unmarshal(rawData.Data.Config, &scylladbclusterConfig)
		if err != nil {
			return err
		}
		i.Data.Config = scylladbclusterConfig
		return
	}

	if i.Type == "cluster" && (i.Data.Provider == "balancer") {
		return
	}

	if i.Type == "gslb_record" && (i.Data.Provider == "v1_1") {
		return
	}

	if (i.Type == "s3") && (i.Data.Provider == "ceph") {
		var tenantConfig S3CephTenantItemConfig
		err = json.Unmarshal(rawData.Data.Config, &tenantConfig)
		if err != nil {
			return err
		}
		i.Data.Config = tenantConfig
		return
	}

	if (i.Type == "s3_bucket") && (i.Data.Provider == "ceph") {
		var bucketConfig S3CephBucketItemConfig
		err = json.Unmarshal(rawData.Data.Config, &bucketConfig)
		if err != nil {
			return err
		}
		i.Data.Config = bucketConfig
		return
	}

	if (i.Type == "cluster") && (strings.Contains(i.Data.Provider, "gslb_cluster_v1")) {
		var GSLBV1 GSLBV1Item
		err = json.Unmarshal(data, &GSLBV1)
		if err != nil {
			return err
		}
		i.Data.Config = GSLBV1.Data
		return
	}

	if (i.Type == "gslb") && (i.Data.Provider == "app_info") {
		var GSLBAppInfoConfig GSLBAppItemConfig
		err = json.Unmarshal(rawData.Data.Config, &GSLBAppInfoConfig)
		if err != nil {
			return err
		}
		i.Data.Config = GSLBAppInfoConfig
		return
	}

	if i.Type == "gslb" && (i.Data.Provider == "anycast") {
		return
	}

	if i.Type == "gslb" && (i.Data.Provider == "bgpaas") {
		return
	}

	err = fmt.Errorf("can't unmarshal item, type=%s, provider=%s", i.Type, i.Data.Provider)
	return
}
