package entities

func (c SyncXpertItemConfig) GetProviderType() (string, string) {
	return "debezium", "cluster"
}

type SyncXpertItemConfig struct {
	Environment      string `json:"environment"`
	BootstrapServers string `json:"bootstrap_servers"`
	Certificates     []struct {
		RestCn          string `json:"rest_cn"`
		KafkaCn         string `json:"kafka_cn"`
		RestURI         string `json:"rest_uri"`
		RestUser        string `json:"rest_user"`
		RestCreated     string `json:"rest_created"`
		KafkaCreated    string `json:"kafka_created"`
		RestExpiration  string `json:"rest_expiration"`
		KafkaExpiration string `json:"kafka_expiration"`
	} `json:"certificates"`
}
