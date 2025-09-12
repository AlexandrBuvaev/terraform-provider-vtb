package entities

func (c PostgresqlItemConfig) GetProviderType() (string, string) {
	return "postgresql", "app"
}

func (c PostgresqlDbItemConfig) GetProviderType() (string, string) {
	return "postgresql", "db"
}

type PostgresqlItemConfig struct {
	Version         string                      `json:"version"`
	DBUsers         []PostgresqlItemDbUsers     `json:"db_users"`
	DBOwners        []PostgresqlItemDbUsers     `json:"db_owners"`
	Environment     string                      `json:"environment"`
	LoadProfile     string                      `json:"load_profile"`
	Configuration   PostgresqlItemConfiguration `json:"configuration"`
	ConnectionURL   string                      `json:"connection_url"`
	EnvironmentType string                      `json:"environment_type"`
}

type PostgresqlDbItemConfig struct {
	Owner           string        `json:"owner"`
	DbName          string        `json:"db_name"`
	Encoding        string        `json:"encoding"`
	LcCtype         string        `json:"lc_ctype"`
	Extensions      []interface{} `json:"extensions"`
	LcCollate       string        `json:"lc_collate"`
	ConnLimit       int64         `json:"conn_limit"`
	Environment     string        `json:"environment"`
	EnvironmentType string        `json:"environment_type"`
}

type PostgresqlItemDbUsers struct {
	Comment  string `json:"comment"`
	DbName   string `json:"db_name"`
	DbmsRole string `json:"dbms_role"`
	UserName string `json:"user_name"`
}

type PostgresqlItemPublication struct {
	Data            []any  `json:"data"`
	Name            string `json:"name"`
	Type            string `json:"type"`
	Owner           string `json:"owner"`
	Environment     string `json:"environment"`
	EnvironmentType string `json:"environment_type"`
}

func (c PostgresqlItemPublication) GetProviderType() (string, string) {
	return "postgresql_v001", "publication"
}

type PostgresqlItemSlot struct {
	Name            string `json:"name"`
	Type            string `json:"type"`
	Plugin          string `json:"plugin"`
	Environment     string `json:"environment"`
	EnvironmentType string `json:"environment_type"`
}

func (c PostgresqlItemSlot) GetProviderType() (string, string) {
	return "postgresql_v001", "slot"
}

type PostgresqlItemConfiguration struct {
	WorkMem                     string `json:"work_mem"`
	MaxWalSize                  string `json:"max_wal_size"`
	MinWalSize                  string `json:"min_wal_size"`
	SharedBuffers               string `json:"shared_buffers"`
	MaxConnections              string `json:"max_connections"`
	TempFileLimit               string `json:"temp_file_limit"`
	RandomPageCost              string `json:"random_page_cost"`
	CheckpointTimeout           string `json:"checkpoint_timeout"`
	EffectiveCacheSize          string `json:"effective_cache_size"`
	MaintenanceWorkMem          string `json:"maintenance_work_mem"`
	MaxParallelWorkers          string `json:"max_parallel_workers"`
	MaxWorkerProcesses          string `json:"max_worker_processes"`
	DefaultTransactionIsolation string `json:"default_transaction_isolation"`
	MaxParallelWorkersPerGather string `json:"max_parallel_workers_per_gather"`
}
