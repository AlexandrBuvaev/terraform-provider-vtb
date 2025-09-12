package entities

func (c RedisItemConfig) GetProviderType() (string, string) {
	return "redis", "app"
}

type RedisItemConfig struct {
	Users                []RedisUser `json:"users"`
	UseAcl               bool        `json:"use_acl"`
	Version              string      `json:"version"`
	AuthMethod           string      `json:"auth_method"`
	ConnectionURL        []string    `json:"connection_url"`
	NotifyKeyspaceEvents string      `json:"notify_keyspace_events"`
}

type RedisUser struct {
	UserName string `json:"user_name"`
}
