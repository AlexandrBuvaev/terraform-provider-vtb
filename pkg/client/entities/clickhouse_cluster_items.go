package entities

func (c ClickhouseClusterItemConfig) GetProviderType() (string, string) {
	return "clickhouse", "cluster"
}

type ClickhouseClusterItemConfig struct {
	DBUsers         []DBUsers         `json:"db_users"`
	DBOwners        []DBOwners        `json:"db_owners"`
	DBUsersAd       []DBUsersAd       `json:"db_users_ad"`
	DBUserGroup     []DBUserGroup     `json:"db_user_group"`
	DBAdminGroup    []DBAdminGroup    `json:"db_admin_group"`
	DBAppAdminGroup []DBAppAdminGroup `json:"db_app_admin_group"`
	Version         string            `json:"version"`
	ConnectionURL   []string          `json:"connection_url"`
	ClusterName     string            `json:"cluster_name"`
}
