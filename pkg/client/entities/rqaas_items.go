package entities

func (c RQaaSItemConfig) GetProviderType() (string, string) {
	return "rqaas", "saas"
}

type RQaaSCluster struct {
	Hosts            []string `json:"hosts"`
	Domain           string   `json:"domain"`
	Platform         string   `json:"platform"`
	NetSegment       string   `json:"net_segment"`
	ClusterName      string   `json:"cluster_name"`
	AvailabilityZone string   `json:"availability_zone"`
}

type RQaaSItemConfig struct {
	Fqdn       string      `json:"fqdn"`
	Name       string      `json:"name"`
	Node       string      `json:"node"`
	Type       string      `json:"type"`
	State      string      `json:"state"`
	Vhost      string      `json:"vhost"`
	QueueUsers []QueueUser `json:"queue_users"`
}

type QueueUser struct {
	Read     bool   `json:"read" tfsdk:"read"`
	Write    bool   `json:"write" tfsdk:"write"`
	UserName string `json:"user_name" tfsdk:"username"`
}
