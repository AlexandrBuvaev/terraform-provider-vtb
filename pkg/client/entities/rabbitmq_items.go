package entities

import "encoding/json"

func (c RabbitMQItemConfig) GetProviderType() (string, string) {
	return "rabbitmq", "cluster"
}

type RabbitMQItemConfig struct {
	ClusterName           string                `json:"cluster_name"`
	FullClusterName       string                `json:"full_cluster_name"`
	RabbitMQVersion       string                `json:"rabbitmq_version"`
	ErlangVersion         string                `json:"erlang_version"`
	Domain                string                `json:"domain"`
	NetSegment            string                `json:"net_segment"`
	QuorumHost            string                `json:"quorum_host"`
	GrafanaDashboardURL   string                `json:"grafana_dashboard_url"`
	ConnectionURL         string                `json:"connection_url"`
	CertificateCn         string                `json:"certificate_cn"`
	CertificateExpiration string                `json:"certificate_expiration"`
	CertificateValidFrom  string                `json:"certificate_valid_from"`
	Flavor                RabbitMQFlavor        `json:"flavor"`
	ExtraMounts           RabbitMQExtraMounts   `json:"extra_mounts"`
	Users                 []RabbitMQUsers       `json:"users"`
	Vhosts                []RabbitMQVhost       `json:"vhosts"`
	VhostAccess           []RabbitMQVhostAccess `json:"vhost_access"`
	WebAccessGroups       WebAccessGroups       `json:"web_access_groups,omitempty"`
	HostsInfo             HostsInfo             `json:"hosts_info"`
}

type RabbitMQUsers struct {
	Name string `json:"name"`
}

type RabbitMQFlavor struct {
	Cpus   int64 `json:"cpus"`
	Memory int64 `json:"memory"`
}

type RabbitMQVhost struct {
	Name string `json:"name"`
}

type HostsInfo struct {
	QuantityQuorum   int64 `json:"quantity_quorum"`
	QuantityRabbitMQ int64 `json:"quantity_rabbitmq"`
}

// fixme
//
//	в айтемах кластера фигурирует file_system
//
// в действии вертикального масштабирования fstype
type RabbitMQExtraMounts struct {
	Size       int64  `json:"size"`
	Mount      string `json:"mount,omitempty"`
	FileSystem string `json:"file_system,omitempty"`
	FSType     string `json:"fstype,omitempty"`
	Device     string `json:"device,omitempty"`
	Options    string `json:"options,omitempty"`
}

type QuorumExtraMounts struct {
	Size   int64  `json:"size"`
	Mount  string `json:"mount"`
	FStype string `json:"fstype"`
}

type RabbitMQVhostAccess struct {
	UserName    string   `json:"user_name,omitempty"`
	VhostName   string   `json:"vhost_name,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

type RabbitMQUserInputPermissions struct {
	UserName       string   `json:"user_name,omitempty"`
	VhostRead      []string `json:"vhost_read,omitempty"`
	VHostWrite     []string `json:"vhost_write,omitempty"`
	VhostConfigure []string `json:"vhost_configure,omitempty"`
}

type WebAccessGroups struct {
	Managers []string `json:"manager,omitempty"`
	Admins   []string `json:"administrator,omitempty"`
}

// Если групп нет вообще, то приходит не массив с объектами, а пустой массив
func (w *WebAccessGroups) UnmarshalJSON(data []byte) error {
	if string(data) == "[]" {
		return nil
	}

	type alias WebAccessGroups
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}

	w.Managers = a.Managers
	w.Admins = a.Admins

	return nil
}

func (w *WebAccessGroups) IsEmpty() bool {
	return len(w.Managers) == 0 && len(w.Admins) == 0
}
