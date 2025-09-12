package entities

type KTaaSKafkaCluster struct {
	Data KTaaSKafkaClusterData `json:"data"`
}

type KTaaSKafkaClusterData struct {
	DC               []string `json:"dc"`
	CIHost           []string `json:"ci_host"`
	Platform         string   `json:"platform"`
	QuotaLabel       string   `json:"quota_label"`
	ResourcePoolID   string   `json:"resource_pool_id"`
	AvailabilityZone []string `json:"availability_zone"`
	ResourcePoolName string   `json:"resource_pool_name"`
}

func (c KTaaSConfig) GetProviderType() (string, string) {
	return "paas_ktaas", "ktaas"
}

type KTaaSConfig struct {
	ACLS             []KTaaSAcls           `json:"acls"`
	GroupAcls        []KtaaSGroupAcls      `json:"group_acls"`
	TopicName        string                `json:"topic_name"`
	Environment      string                `json:"environment"`
	NetSegment       string                `json:"net_segment"`
	ClusterName      string                `json:"cluster_name"`
	TopicFlavor      int64                 `json:"topic_flavor"`
	KafkaBrokers     []string              `json:"kafka_brokers"`
	ResourcePool     KTaaSKafkaClusterData `json:"resource_pool"`
	BillingStorage   int64                 `json:"billing_storage"`
	PartitionsNumber int64                 `json:"partitions_number"`
}

type KTaaSAcls struct {
	ClientCN   string `json:"client_cn" tfsdk:"client_cn"`
	ClientRole string `json:"client_role" tfsdk:"client_role"`
}

type KtaaSGroupAcls struct {
	ClientCN  string `json:"client_cn" tfsdk:"client_cn"`
	GroupName string `json:"group_name" tfsdk:"group_name"`
}
