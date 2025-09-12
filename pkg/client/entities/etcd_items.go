package entities

func (c EtcdlItemConfig) GetProviderType() (string, string) {
	return "etcd", "cluster"
}

type EtcdlItemConfig struct {
	UseSsl           bool       `json:"use_ssl"`
	Version          string     `json:"version"`
	EtcdUser         []EtcdUser `json:"etcd_user"`
	ClusterName      string     `json:"cluster_name"`
	ActionWithoutSsl bool       `json:"action_without_ssl"`
}

type EtcdUser struct {
	Username string `json:"etcd_user"`
}
