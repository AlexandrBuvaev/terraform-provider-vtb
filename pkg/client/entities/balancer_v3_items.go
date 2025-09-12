package entities

func (c BalancerV3ItemData) GetProviderType() (string, string) {
	return "balancer_v3", "cluster"
}

type BalancerV3Item struct {
	CreatedRowDT string             `json:"created_row_dt"`
	UpdateDT     string             `json:"update_dt"`
	ID           string             `json:"item_id"`
	OrderId      string             `json:"order_id"`
	GraphId      string             `json:"graph_id"`
	Type         string             `json:"type"`
	Data         BalancerV3ItemData `json:"data"`
}

type BalancerV3ItemData struct {
	Build          BalancerV3Build            `json:"build"`
	Config         Config                     `json:"config"`
	ClusterConfig  BalancerV3ClusterConfig    `json:"cluster_config"`
	ClusterMembers []BalancerV3ClusterMembers `json:"cluster_members"`
}

type BalancerV3Build struct {
	SetupVersion string `json:"setup_version"`
}

type BalancerV3ClusterConfig struct {
	RAM             int    `json:"ram"`
	Cpus            int    `json:"cpus"`
	Domain          string `json:"domain"`
	DNSZone         string `json:"dns_zone"`
	EnvPrefix       string `json:"env_prefix"`
	Environment     string `json:"environment"`
	NetSegment      string `json:"net_segment"`
	ClusterName     string `json:"cluster_name"`
	EnvironmentType string `json:"environment_type"`
}

type BalancerV3ClusterMembers struct {
	Ip         string `json:"ip"`
	Name       string `json:"name"`
	MainStatus string `json:"main_status"`
}

type Config struct {
	Ports        []Port        `json:"ports" tfsdk:"ports"`
	Globals      Globals       `json:"globals" tfsdk:"globals"`
	Backends     []Backend     `json:"backends" tfsdk:"backends"`
	Defaults     Defaults      `json:"defaults" tfsdk:"defaults"`
	Publications []Publication `json:"publications" tfsdk:"publications"`
}

// -----------globals---------------
type Globals struct {
	Maxconn                           *int64  `json:"maxconn,omitempty" tfsdk:"maxconn"`
	TuneOptions                       *string `json:"tune_options,omitempty" tfsdk:"tune_options"`
	H2WorkaroundBogusWebsocketClients bool    `json:"h2_workaround_bogus_websocket_clients" tfsdk:"h2_workaround_bogus_websocket_clients"`
}

// -----------defaults---------------
type Defaults struct {
	ClientTimeout  int64 `json:"client_timeout" tfsdk:"client_timeout"`
	ServerTimeout  int64 `json:"server_timeout" tfsdk:"server_timeout"`
	ConnectTimeout int64 `json:"connect_timeout" tfsdk:"connect_timeout"`
}

// -----------common---------------
type HttpSettings struct {
	GrpcOverHttp2             bool                `json:"grpc_over_http2" tfsdk:"grpc_over_http2"`
	VersionActivationPriority []VersionActivation `json:"version_activation_priority" tfsdk:"version_activation_priority"`
}

type VersionActivation struct {
	Version string `json:"version" tfsdk:"version"`
}

type Backend struct {
	Mode               string            `json:"mode" tfsdk:"mode"` // tcp, http, https
	Retries            *Retries          `json:"retries,omitempty" tfsdk:"retries"`
	Servers            []*Server         `json:"servers" tfsdk:"servers"`
	Globalname         string            `json:"globalname" tfsdk:"globalname"`
	Healthcheck        *Healthcheck      `json:"healthcheck" tfsdk:"healthcheck"`
	BackendName        string            `json:"backend_name" tfsdk:"backend_name"`
	BalancingAlgorithm string            `json:"balancing_algorithm" tfsdk:"balancing_algorithm"` // leastconn, roundrobin, source
	Cookie             *Cookie           `json:"cookie" tfsdk:"cookie"`
	Forwardfor         *Forwardfor       `json:"forwardfor,omitempty" tfsdk:"forwardfor"`
	HttpReuse          *HttpReuse        `json:"http_reuse,omitempty" tfsdk:"http_reuse"`
	KeepAlive          *BackendKeepAlive `json:"keep_alive,omitempty" tfsdk:"keep_alive"`
	ServersSettings    *ServerSettings   `json:"servers_settings,omitempty" tfsdk:"servers_settings"`
}

type Retries struct {
	Enabled    bool     `json:"enabled" tfsdk:"enabled"`
	Count      int64    `json:"count" tfsdk:"count"`
	Conditions []string `json:"conditions,omitempty" tfsdk:"conditions"`
	Redispatch *string  `json:"redispatch,omitempty" tfsdk:"redispatch"` //disabled, every_try
}

type Server struct {
	Name      string  `json:"name" tfsdk:"name"`
	State     string  `json:"state" tfsdk:"state"` // active, backup, maintenance, drain
	Address   string  `json:"address" tfsdk:"address"`
	SendProxy *string `json:"send_proxy,omitempty" tfsdk:"send_proxy"` // enabled-v1, enabled-v2, disabled
	Maxconn   int64   `json:"maxconn" tfsdk:"maxconn"`
	Weight    int64   `json:"weight,omitempty" tfsdk:"weight"`
}

type CheckString struct {
	Uri       *string `json:"uri,omitempty" tfsdk:"uri"`
	Port      *int64  `json:"port,omitempty" tfsdk:"port"`
	Match     *string `json:"match,omitempty" tfsdk:"match"` //status, string, rstring, rstatus
	Value     *string `json:"value,omitempty" tfsdk:"value"`
	Method    *string `json:"method,omitempty" tfsdk:"method"` //GET, PUT
	Address   *string `json:"address,omitempty" tfsdk:"address"`
	UseSsl    *string `json:"use_ssl,omitempty" tfsdk:"use_ssl"` //enabled, disabled
	Version   *string `json:"version,omitempty" tfsdk:"version"`
	HostHdr   *string `json:"host_hdr,omitempty" tfsdk:"host_hdr"`
	SendProxy *string `json:"send_proxy" tfsdk:"send_proxy"` //enabled, disabled
	Body      *string `json:"body,omitempty" tfsdk:"body"`
}

type Healthcheck struct {
	Mode         string         `json:"mode" tfsdk:"mode"` // tcp, http
	Interval     *int64         `json:"interval,omitempty" tfsdk:"interval"`
	FallCount    *int64         `json:"fall_count,omitempty" tfsdk:"fall_count"`
	RiseCount    *int64         `json:"rise_count,omitempty" tfsdk:"rise_count"`
	CheckStrings []*CheckString `json:"check_strings,omitempty" tfsdk:"check_strings"`
}

type Cookie struct {
	Enable bool  `json:"enable" tfsdk:"enable"`
	Secure *bool `json:"secure,omitempty" tfsdk:"secure"`
}

type Forwardfor struct {
	Xff  bool `json:"xff" tfsdk:"xff"`
	Xcip bool `json:"xcip" tfsdk:"xcip"`
	Xrip bool `json:"xrip" tfsdk:"xrip"`
}

type HttpReuse struct {
	Mode           *string `json:"mode,omitempty" tfsdk:"mode"` // never, safe, aggressive, always
	PoolLowConn    *int64  `json:"pool_low_conn,omitempty" tfsdk:"pool_low_conn"`
	PoolMaxConn    *int64  `json:"pool_max_conn,omitempty" tfsdk:"pool_max_conn"`
	PoolPurgeDelay *int64  `json:"pool_purge_delay,omitempty" tfsdk:"pool_purge_delay"`
}

type BackendKeepAlive struct {
	Mode  *string `json:"mode,omitempty" tfsdk:"mode"` // default, advanced, disabled
	Timer *int64  `json:"timer,omitempty" tfsdk:"timer"`
}

type ServerSettings struct {
	Port         int64         `json:"port" tfsdk:"port"`
	SlowStart    *int64        `json:"slow_start,omitempty" tfsdk:"slow_start"`
	TlsProfile   *string       `json:"tls_profile,omitempty" tfsdk:"tls_profile"`
	HttpSettings *HttpSettings `json:"http_settings,omitempty" tfsdk:"http_settings"`
	UseSni       *bool         `json:"use_sni,omitempty" tfsdk:"use_sni"`
}

type Port struct {
	Mode         string         `json:"mode" tfsdk:"mode"` // tcp, http, https
	Port         int64          `json:"port" tfsdk:"port"`
	Maxconn      int64          `json:"maxconn" tfsdk:"maxconn"`
	KeepAlive    *PortKeepAlive `json:"keep_alive,omitempty" tfsdk:"keep_alive"`
	Redirect     *Redirect      `json:"redirect,omitempty" tfsdk:"redirect"`
	TlsProfile   *string        `json:"tls_profile,omitempty" tfsdk:"tls_profile"`
	Prescription *Prescription  `json:"prescription,omitempty" tfsdk:"prescription"`
	HttpSettings *HttpSettings  `json:"http_settings,omitempty" tfsdk:"http_settings"`
}

type PortKeepAlive struct {
	Tcp  *TCPKeepAlive  `json:"tcp,omitempty" tfsdk:"tcp"`
	Http *HTTPKeepAlive `json:"http,omitempty" tfsdk:"http"`
}

type TCPKeepAlive struct {
	Mode  *string `json:"mode,omitempty" tfsdk:"mode"` // default, advanced
	Count *int64  `json:"count,omitempty" tfsdk:"count"`
	Timer *int64  `json:"timer,omitempty" tfsdk:"timer"`
}

type HTTPKeepAlive struct {
	Mode  *string `json:"mode,omitempty" tfsdk:"mode"` // default, advanced
	Timer *int64  `json:"timer,omitempty" tfsdk:"timer"`
}

type Redirect struct {
	Enabled bool `json:"enabled" tfsdk:"enabled"`
}

type Prescription struct {
	Enabled           bool   `json:"enabled" tfsdk:"enabled"`
	LiveTime          *int64 `json:"live_time,omitempty" tfsdk:"live_time"`
	IncludeSubdomains *bool  `json:"include_subdomains,omitempty" tfsdk:"include_subdomains"`
}

type Publication struct {
	Mode            string   `json:"mode" tfsdk:"mode"` // tcp, http, https
	Port            int64    `json:"port" tfsdk:"port"`
	Globalname      string   `json:"globalname" tfsdk:"globalname"`
	MainBackend     string   `json:"main_backend" tfsdk:"main_backend"`
	AliveServCount  int64    `json:"alive_serv_count" tfsdk:"alive_serv_count"`
	StandinBackends []string `json:"standin_backends" tfsdk:"standin_backends"`
	DefaultRouting  bool     `json:"default_routing" tfsdk:"default_routing"`
	Cnames          []string `json:"cnames,omitempty" tfsdk:"cnames"`
	Routes          []Route  `json:"routes" tfsdk:"routes"`
}

type Route struct {
	Uri     []Endpoint `json:"uri,omitempty" tfsdk:"uri"`
	Headers []Header   `json:"headers,omitempty" tfsdk:"headers"`
	Ips     []string   `json:"ips,omitempty" tfsdk:"ips"`
}

type Endpoint struct {
	Condition string   `json:"condition" tfsdk:"condition"`
	Endpoints []string `json:"endpoints" tfsdk:"endpoints"`
}

type Header struct {
	Name      string  `json:"name" tfsdk:"name"`
	CheckType string  `json:"check_type" tfsdk:"check_type"` // existing, value
	Value     *string `json:"value,omitempty" tfsdk:"value"`
}
