package entities

func (c NginxItemConfig) GetProviderType() (string, string) {
	return "nginx", "app"
}

type NginxItemConfig struct {
	Certificate string   `json:"certificate_expiration"`
	Distrib     string   `json:"distrib"`
	AltNames    []string `json:"alt_names"`
	Build       struct {
		SetupVersion string `json:"setup_version"`
	} `json:"build"`
}
