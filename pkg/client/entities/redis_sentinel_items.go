package entities

func (c RedisSentinelItemConfig) GetProviderType() (string, string) {
	return "redis_sentinel", "app"
}

type RedisSentinelItemConfig struct {
	Users                []RedisSentinelUser `json:"users"`
	UseAcl               bool                `json:"use_acl"`
	Version              string              `json:"version"`
	AuthMethod           string              `json:"auth_method"`
	PoolName             string              `json:"pool_name"`
	ConnectionURL        []string            `json:"connection_url"`
	NotifyKeyspaceEvents string              `json:"notify_keyspace_events"`
	AcceptDocumentation  bool                `json:"accept_documentation"`
}

type RedisSentinelUser struct {
	UserName string `json:"user_name"`
}
