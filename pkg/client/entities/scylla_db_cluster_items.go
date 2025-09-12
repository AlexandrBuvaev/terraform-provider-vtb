package entities

func (c ScyllaDbClusterItemConfig) GetProviderType() (string, string) {
	return "scylladb", "cluster"
}

type ScyllaDbClusterItemConfig struct {
	Version              string                 `json:"version"`
	OnBackup             bool                   `json:"on_backup"`
	BackupPath           string                 `json:"backup_path"`
	Environment          string                 `json:"environment"`
	ClusterName          string                 `json:"cluster_name"`
	ConnectionURL        string                 `json:"connection_url"`
	EnvironmentType      string                 `json:"environment_type"`
	Dbs                  []ScyllaDbs            `json:"dbs"`
	DbUsers              []ScyllaDbusers        `json:"db_users"`
	Permissions          []ScyllaDbPermissions  `json:"permissions"`
	ClusterConfiguration []ClusterConfiguration `json:"cluster_configuration"`
}

type ScyllaDbs struct {
	DbName string `json:"db_name"`
}
type ScyllaDbusers struct {
	DbmsRole     string `json:"dbms_role"`
	UserName     string `json:"user_name"`
	UserPassword string `json:"user_password"`
}
type ScyllaDbPermissions struct {
	ID       string `json:"id"`
	DbName   string `json:"db_name"`
	UserName string `json:"user_name"`
}

type ClusterConfiguration struct {
	Hosts  []string `json:"hosts"`
	DcName string   `json:"dc_name"`
}
