package entities

func (c WildflyItemConfig) GetProviderType() (string, string) {
	return "wildfly", "app"
}

type WildflyItemConfig struct {
	User           []any                `json:"user"`
	Group          []WildflyGroupConfig `json:"group"`
	StandaloneType string               `json:"standalone_type,omitempty"`
	JavaVersion    string               `json:"java_version"`
	WildflyVersion string               `json:"wildfly_version"`
	Certificate    struct {
		EndDate       string   `json:"end_date"`
		AltNames      []string `json:"alt_names"`
		StartDate     string   `json:"start_date"`
		CertificateCn string   `json:"certificate_cn"`
		ClientCert    bool     `json:"client_cert"`
	} `json:"certificate"`
}

type WildflyGroupConfig struct {
	Name string `json:"name"`
	Role string `json:"role"`
}
