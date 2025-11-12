package entities

func (c GSLBV1ItemData) GetProviderType() (string, string) {
	return "gslb_cluster_v1", "cluster"
}

func (c GSLBAppItemConfig) GetProviderType() (string, string) {
	return "app_info", "gslb"
}

// Сущность GSLB App
type GSLBAppItemConfig struct {
	Name                string `json:"name"`
	IPv4                string `json:"ip_v4"`
	Region              string `json:"region"`
	DataCenter          string `json:"data_center"`
	Description         string `json:"description"`
	SetupVersion        string `json:"setup_version"`
	MaintenanceMode     string `json:"maintenance_mode"`
	InstallationVersion int64  `json:"installation_version"`
}

// Сущность кластера
type GSLBV1Item struct {
	CreatedRowDT string         `json:"created_row_dt"`
	UpdateDT     string         `json:"update_dt"`
	ID           string         `json:"item_id"`
	OrderID      string         `json:"order_id"`
	GraphID      string         `json:"graph_id"`
	Data         GSLBV1ItemData `json:"data"`
}

type GSLBV1ItemData struct {
	Build  GSLBV1Build  `json:"build"`
	Config GSLBV1Config `json:"config"`
}

type GSLBV1Build struct {
	SetupVersion string `json:"setup_version"`
}

type GSLBV1Config struct {
	Domain         string     `json:"domain"`
	DNSZone        string     `json:"dns_zone"`
	NetSegment     string     `json:"net_segment"`
	LastUpdate     string     `json:"last_update"`
	AnycastEnabled bool       `json:"anycast_enabled"`
	LBConfigs      []LBConfig `json:"lb_configs"`
}

type LBConfig struct {
	Pool       string `json:"pool"`
	GlobalName string `json:"global_name"`
}
