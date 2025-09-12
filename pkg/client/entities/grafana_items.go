package entities

func (c GrafanaItemConfig) GetProviderType() (string, string) {
	return "grafana", "app"
}

type GrafanaItemConfig struct {
	GrafanaVersion    string        `json:"grafana_version"`
	PrometheusVersion string        `json:"version_prometheus"`
	GrafanaUsers      []GrafanaUser `json:"all_users"`
	ConnectionUrl     string        `json:"connection_url"`
}

type GrafanaUser struct {
	Username string `json:"user_name"`
}
