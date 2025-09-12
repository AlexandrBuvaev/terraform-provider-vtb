package entities

func (c VMItemConfig) GetProviderType() (string, string) {
	return "openstack", "vm"
}

type VMItemConfig struct {
	Domain           string              `json:"domain"`
	Hostname         string              `json:"hostname"`
	SwapSize         int                 `json:"swap_size"`
	OnSupport        bool                `json:"on_support"`
	OsVersion        string              `json:"os_version"`
	Environment      string              `json:"environment"`
	AdIntegration    bool                `json:"ad_integration"`
	EnvironmentType  string              `json:"environment_type"`
	DefaultV4Address string              `json:"default_v4_address"`
	DefaultV6Address string              `json:"default_v6_address"`
	Flavor           Flavor              `json:"flavor"`
	DefaultNic       VMItemDefaultNic    `json:"default_nic"`
	BootDisk         VMItemBootDisk      `json:"boot_disk"`
	Image            VMItemImage         `json:"image"`
	ResourcePool     VMItemResourcePool  `json:"resource_pool"`
	Tenant           VMItemTenant        `json:"tenant"`
	Mounts           []VMItemMount       `json:"mounts"`
	ExtraNics        []VMItemExtraNics   `json:"extra_nics"`
	ExtraDisks       []VMItemExtraDisks  `json:"extra_disks"`
	ExtraMounts      []VMItemExtraMounts `json:"extra_mounts"`
	NodeRoles        []string            `json:"node_roles"`
}

type VMItemImage struct {
	Os struct {
		Type         string `json:"type"`
		Vendor       string `json:"vendor"`
		Version      string `json:"version"`
		Architecture string `json:"architecture"`
		Distribution string `json:"distribution"`
		Localization string `json:"localization"`
	} `json:"os"`
	Name string `json:"name"`
	Size int    `json:"size"`
	UUID string `json:"uuid"`
}

type VMItemMount struct {
	Size    float64 `json:"size"`
	Mount   string  `json:"mount"`
	Device  string  `json:"device"`
	Fstype  string  `json:"fstype"`
	Options string  `json:"options"`
}

type VMItemTenant struct {
	Name string `json:"name"`
	UUID string `json:"uuid"`
}

type VMItemBootDisk struct {
	Path   string `json:"path"`
	Size   int    `json:"size"`
	UUID   string `json:"uuid"`
	Serial string `json:"serial"`
}

type VMItemExtraNics struct{}

type VMItemDefaultNic struct {
	Mtu    int    `json:"mtu"`
	Name   string `json:"name"`
	UUID   string `json:"uuid"`
	Subnet struct {
		Name string `json:"name"`
		UUID string `json:"uuid"`
	} `json:"subnet"`
	Addresses []struct {
		Type    string `json:"type"`
		Address string `json:"address"`
	} `json:"addresses"`
	MacAddress        string `json:"mac_address"`
	NetSegment        string `json:"net_segment"`
	AddressAssignment string `json:"address_assignment"`
}

type VMItemExtraDisks struct {
	Path   string `json:"path"`
	Size   int    `json:"size"`
	UUID   string `json:"uuid"`
	Serial string `json:"serial"`
}

type VMItemExtraMounts struct {
	Mount   string  `json:"mount"`
	Size    float64 `json:"size"`
	Device  string  `json:"device"`
	Fstype  string  `json:"fstype"`
	Options string  `json:"options"`
}

type VMItemResourcePool struct {
	Name         string `json:"name"`
	UUID         string `json:"uuid"`
	Domain       string `json:"domain"`
	UILink       string `json:"ui_link"`
	Endpoint     string `json:"endpoint"`
	Platform     string `json:"platform"`
	TenantPrefix string `json:"tenant_prefix"`
}
