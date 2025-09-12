package entities

func (c TarantoolClusterItemConfig) GetProviderType() (string, string) {
	return "tarantool_v2", "cluster"
}

type TarantoolClusterItemConfig struct {
	StateProvider    string                    `json:"state_provider"`
	TarantoolType    string                    `json:"tarantool_type"`
	TarantoolVersion string                    `json:"tarantool_version"`
	TarantoolAppName string                    `json:"tarantool_app_name"`
	Cluster          map[string][]ZoneInstance `json:"cluster"`
	Certificates     Certificates              `json:"certificates"`
	ClusterName      string                    `json:"cluster_name"`
	DomainRoles      []DomainRoleMembers       `json:"domain_roles"`
}

type Certificates struct {
	Iproto  []IprotoCertificate `json:"iproto"`
	Cluster ClusterCertificate  `json:"cluster"`
}

type IprotoCertificate struct {
	EndDate       string `json:"end_date"`
	StartDate     string `json:"start_date"`
	CertificateCN string `json:"certificate_cn"`
}

type ClusterCertificate struct {
	EndDate       string   `json:"end_date"`
	AltNames      []string `json:"alt_names"`
	StartDate     string   `json:"start_date"`
	CertificateCN string   `json:"certificate_cn"`
}

type DomainRoleMembers struct {
	Role    string   `json:"role"`
	Members []string `json:"members"`
}

type ZoneInstance struct {
	Host         string   `json:"host"`
	Role         []string `json:"role"`
	State        string   `json:"state"`
	Memory       int64    `json:"memory"`
	Instance     string   `json:"instance"`
	MgmtUrl      string   `json:"mgmt_url"`
	Replicaset   string   `json:"replicaset"`
	AdvertiseUri string   `json:"advertise_uri"`
}
