package entities

func (c AirflowClusterItemConfig) GetProviderType() (string, string) {
	return "airflow", "cluster"
}

func (c AirflowStandaloneItemConfig) GetProviderType() (string, string) {
	return "airflow", "app"
}

type AirflowClusterItemConfig struct {
	Domain                string                     `json:"domain"`
	CoreVersion           string                     `json:"version"`
	Executor              string                     `json:"executor"`
	Environment           string                     `json:"environment"`
	LDAPGroups            []AirflowLDAPGroup         `json:"ldap_groups"`
	OnRubackup            bool                       `json:"on_rubackup"`
	ClusterName           string                     `json:"cluster_name"`
	DeployGroups          AirflowDeployGroups        `json:"deploy_group"`
	CertificateCN         string                     `json:"certificate_cn"`
	ConnectionURL         string                     `json:"connection_url"`
	EnvironmentType       string                     `json:"environment_type"`
	PostgreSQLConfig      AirflowPostgreSQLConfig    `json:"postgresql_config"`
	RabbitMQConfig        AirflowRabbitMQConfig      `json:"rabbitmq_config,omitempty"`
	CertificateExpiration string                     `json:"certificate_expiration"`
	CertificateValidFrom  string                     `json:"certificate_valid_from"`
	ClientCertificates    []AirflowClientCertificate `json:"client_certificates"`
}

type AirflowStandaloneItemConfig struct {
	Domain                string                     `json:"domain"`
	CoreVersion           string                     `json:"version"`
	Executor              string                     `json:"executor"`
	Environment           string                     `json:"environment"`
	LDAPGroups            []AirflowLDAPGroup         `json:"ldap_groups"`
	OnRubackup            bool                       `json:"on_rubackup"`
	ClusterName           string                     `json:"cluster_name"`
	DeployGroups          AirflowDeployGroups        `json:"deploy_group"`
	CertificateCN         string                     `json:"certificate_cn"`
	ConnectionURL         string                     `json:"connection_url"`
	EnvironmentType       string                     `json:"environment_type"`
	PostgreSQLConfig      AirflowPostgreSQLConfig    `json:"postgresql_config"`
	CertificateExpiration string                     `json:"certificate_expiration"`
	CertificateValidFrom  string                     `json:"certificate_valid_from"`
	ClientCertificates    []AirflowClientCertificate `json:"client_certificates"`
}

type AirflowLDAPGroup struct {
	Role   string   `json:"role"`
	Groups []string `json:"groups"`
}

type AirflowDeployGroups struct {
	Role   string   `json:"role"`
	Groups []string `json:"groups"`
}

type AirflowPostgreSQLConfig struct {
	Host     string `json:"db_host"`
	User     string `json:"db_user"`
	Database string `json:"db_database"`
}

type AirflowRabbitMQConfig struct {
	BrokerHost     []string `json:"broker_host"`
	VHost          string   `json:"broker_vhost"`
	RabbitMQCertCN string   `json:"rabbitmq_cert_CN"`
}

type AirflowClientCertificate struct {
	EndDate       string `json:"end_date"`
	StartDate     string `json:"start_date"`
	CertificateCN string `json:"certificate_cn"`
}
