package entities

func (c VTBArtemisItemConfig) GetProviderType() (string, string) {
	return "vtb-artemis", "cluster"
}

type ArtemisProtocols struct {
	AMQP bool `json:"AMQP"`
	CORE bool `json:"CORE"`
}

type ArtemisUnquieIdPlugin struct {
	Status bool `json:"status"`
}

type ArtemisLimitsPlugin struct {
	Status      bool  `json:"status"`
	BlockSize   int64 `json:"blockSize,omitempty"`
	PacketLimit int64 `json:"packetLimit,omitempty"`
	RateLimit   int64 `json:"rateLimit,omitempty"`
	SizeLimit   int64 `json:"sizeLimit,omitempty"`
}

type ArtemisPlugins struct {
	Limits   ArtemisLimitsPlugin   `json:"Limits"`
	UniqueID ArtemisUnquieIdPlugin `json:"UniqueID"`
}

type AddressPolicy struct {
	AddressName             string `json:"address_name"`
	MaxSizeBytes            string `json:"max_size_bytes"`
	MaxExpiryDelay          int64  `json:"max_expiry_delay"`
	MinExpiryDelay          int64  `json:"min_expiry_delay"`
	AutoCreateQueues        bool   `json:"auto_create_queues"`
	AutoDeleteQueues        bool   `json:"auto_delete_queues"`
	AddressFullPolicy       string `json:"address_full_policy"`
	AddressPolicyName       string `json:"address_policy_name"`
	CascadeAddressName      string `json:"cascade_address_name"`
	SecurityPolicyName      string `json:"security_policy_name"`
	SlowConsumerPolicy      string `json:"slow_consumer_policy"`
	AutoCreateAddresses     bool   `json:"auto_create_addresses"`
	AutoDeleteAddresses     bool   `json:"auto_delete_addresses"`
	MaxDeliveryAttempts     int64  `json:"max_delivery_attempts"`
	SlowConsumerThreshold   int64  `json:"slow_consumer_threshold"`
	SlowConsumerCheckPeriod int64  `json:"slow_consumer_check_period"`
}

type TUZ struct {
	UserName      string `json:"user_name"`
	UserOwnerCert string `json:"user_owner_cert"`
}

type Role struct {
	Role               string   `json:"role"`
	SecurityPolicyName string   `json:"security_policy_name"`
	UserNames          []string `json:"user_names"`
}

type ArtemisHostsInfo struct {
	Count int64 `json:"count"`
}

type VTBArtemisItemConfig struct {
	CertCn            string           `json:"cert_cn"`
	GrafanaURL        string           `json:"grafana_url"`
	ClusterName       string           `json:"cluster_name"`
	ArtemisVersion    string           `json:"artemis_version"`
	Protocols         ArtemisProtocols `json:"protocols"`
	AddressPolicyList []AddressPolicy  `json:"addresses_and_policies_list"`
	TuzList           []TUZ            `json:"tuz_list"`
	RoleList          []Role           `json:"roles_list"`
	HostsInfo         ArtemisHostsInfo `json:"hosts_info"`
	Plugins           ArtemisPlugins   `json:"plugins"`
}

func (c OpenMessagingItemConfig) GetProviderType() (string, string) {
	return "app", "artemis"
}

type OpenMessagingItemConfig struct {
	ArtemisVersion string `json:"artemis_version"`
}

func (c OpenMessagingLtItemConfig) GetProviderType() (string, string) {
	return "app", "artemis_lt"
}

type OpenMessagingLtItemConfig struct {
	ArtemisVersion string `json:"artemis_version"`
}
