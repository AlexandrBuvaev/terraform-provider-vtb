package entities

func (c ClickHouseItemConfig) GetProviderType() (string, string) {
	return "clickhouse", "app"
}

type ClickHouseItemConfig struct {
	DBUsers         []DBUsers         `json:"db_users"`
	DBOwners        []DBOwners        `json:"db_owners"`
	DBUsersAd       []DBUsersAd       `json:"db_users_ad"`
	DBUserGroup     []DBUserGroup     `json:"db_user_group"`
	DBAdminGroup    []DBAdminGroup    `json:"db_admin_group"`
	DBAppAdminGroup []DBAppAdminGroup `json:"db_app_admin_group"`
	Version         string            `json:"version"`
	ConnectionURL   string            `json:"connection_url"`
}

type DBUsers struct {
	Role     string `json:"dbms_role"`
	UserName string `json:"user_name"`
}

type DBOwners struct {
	Role     string `json:"dbms_role"`
	UserName string `json:"user_name"`
}

type DBUsersAd struct {
	Role     string `json:"dbms_role"`
	UserName string `json:"user_name"`
}

type DBUserGroup struct {
	Role     string `json:"dbms_role"`
	UserName string `json:"user_name"`
}

type DBAdminGroup struct {
	Role     string   `json:"dbms_role"`
	UserName []string `json:"user_name"`
}

type DBAppAdminGroup struct {
	Role     string `json:"dbms_role"`
	UserName string `json:"user_name"`
}
