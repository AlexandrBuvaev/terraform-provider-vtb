package entities

func (c ElasticSearchConfig) GetProviderType() (string, string) {
	return "elasticsearch_os", "cluster"
}

type ElasticSearchConfig struct {
	ApiUrl                   string         `json:"api_url"`
	KibanaUser               KibanaUser     `json:"kibana_user"`
	ClusterName              string         `json:"cluster_name"`
	FluentdUser              FluentdUser    `json:"fluentd_user"`
	AdmAppGroups             []string       `json:"adm_app_groups"`
	AdditionalURLs           AdditionalURLs `json:"additional_urls"`
	UserAppGroups            []string       `json:"user_app_groups"`
	SystemAdmGroups          []string       `json:"system_adm_groups"`
	ElasticSearchVersion     string         `json:"elasticsearch_version"`
	ElasticSearchClusterName string         `json:"elasticsearch_cluster_name"`
}

type KibanaUser struct {
	UserName string `json:"user_name"`
}

type FluentdUser struct {
	UserName string `json:"user_name"`
}

type AdditionalURLs struct {
	Kibana                string `json:"kibana"`
	ElasticSearchExporter string `json:"elasticsearch-exporter"`
}
