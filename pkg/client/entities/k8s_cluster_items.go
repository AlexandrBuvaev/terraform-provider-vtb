package entities

func (c K8sClusterConfig) GetProviderType() (string, string) {
	return "kubernetes", "cluster"
}

type K8sClusterConfig struct {
	Name                 string                         `json:"name"`
	Label                string                         `json:"label"`
	Domain               string                         `json:"domain"`
	Version              string                         `json:"version"`
	Platform             string                         `json:"platform"`
	DataCenter           string                         `json:"data_center"`
	NetSegment           string                         `json:"net_segment"`
	ProductVersion       string                         `json:"product_version"`
	AvailabilityZone     string                         `json:"availability_zone"`
	ControlPanelSize     string                         `json:"control_panel_size"`
	ContainerCPURatio    int64                          `json:"container_cpu_ratio"`
	ContainerMemoryRatio int64                          `json:"container_memory_ratio"`
	Flavor               K8sClusterConfigFlavor         `json:"flavor"`
	Regions              []K8sConfigRegion              `json:"regions"`
	Features             K8sClusterConfigFeatures       `json:"features"`
	Components           K8sClusterConfigComponents     `json:"components"`
	ControlPlane         []K8sClusterConfigControlPlane `json:"control_plane"`
	Ingress              []K8sConfigIngress             `json:"ingress_shards"`
	Products             []string                       `json:"products"`
	GslbOnly             *bool                          `json:"gslb_only,omitempty" tfsdk:"gslb_only"`
}

type K8sClusterConfigFlavor struct {
	Lb         K8sConfigNodeFlavor `json:"lb"`
	Infra      K8sConfigNodeFlavor `json:"infra"`
	Master     K8sConfigNodeFlavor `json:"master"`
	Monitoring K8sConfigNodeFlavor `json:"monitoring"`
}

type K8sConfigRegion struct {
	// VMs                  []K8sClusterConfigVM               `json:"vms"`
	Name                 string                     `json:"name"`
	Size                 int64                      `json:"size"`
	Flavor               K8sConfigNodeFlavor        `json:"flavor"`
	IsCodes              []string                   `json:"iscodes"`
	PodCPUMax            int64                      `json:"pod_cpu_max"`
	PodMemoryMax         int64                      `json:"pod_memory_max"`
	ContainerCPURatio    int64                      `json:"container_cpu_ratio"`
	ContainerMemoryRatio int64                      `json:"container_memory_ratio"`
	Components           *K8sConfigRegionComponents `json:"components"`
}

// type K8sClusterConfigVM struct {
// 	Hostname string `json:"hostname"`
// }

type K8sConfigNodeFlavor struct {
	Cpus   int64  `json:"cpus"`
	Name   string `json:"name"`
	UUID   string `json:"uuid"`
	Memory int64  `json:"memory"`
}

type K8sConfigRegionComponents struct {
	Astrom       bool `json:"astrom"`
	TsamOperator bool `json:"tsam_operator"`
	ChaosMesh    bool `json:"chaos_mesh"`
	TsdsOperator bool `json:"tsds_operator"`
	TslgOperator bool `json:"tslg_operator"`
	Tyk          bool `json:"tyk"`
}

type K8sClusterConfigFeatures struct {
	Istio        bool                          `json:"istio"`
	Console      bool                          `json:"console"`
	Monitoring   bool                          `json:"monitoring"`
	CniPlugin    K8sClusterConfigCniPlugin     `json:"cni_plugin"`
	IstioOptions *K8sClusterConfigIstioOptions `json:"istio_options"`
}

type K8sClusterConfigCniPlugin struct {
	Name string `json:"name"`
}

type K8sClusterConfigComponents struct {
	Istio           *K8sClusterConfigIstio `json:"istio"`
	Astrom          *K8sClusterConfigFlag  `json:"astrom"`
	GpuOperator     *K8sClusterConfigFlag  `json:"gpu_operator"`
	TridentOperator *K8sClusterConfigFlag  `json:"trident_operator"`
	TsamOperator    *K8sClusterConfigFlag  `json:"tsam_operator"`
	ChaosMesh       *K8sClusterConfigFlag  `json:"chaos_mesh"`
	TsdsOperator    *K8sClusterConfigFlag  `json:"tsds_operator"`
	TslgOperator    *K8sClusterConfigFlag  `json:"tslg_operator"`
	Tyk             *K8sClusterConfigFlag  `json:"tyk_gw"`
}

type K8sClusterConfigIstio struct {
	Options       *K8sClusterConfigIstioOptions       `json:"options"`
	Installed     bool                                `json:"installed"`
	ControlPlanes []K8sClusterConfigIstioControlPlane `json:"control_planes"`
	OptionsFlavor string                              `json:"options_flavor"`
}

type K8sClusterConfigIstioOptions struct {
	ExtauthOperator     *bool `json:"extauth_operator,omitempty" tfsdk:"extauth_operator"`
	GatewayOperator     *bool `json:"gateway_operator,omitempty" tfsdk:"gateway_operator"`
	AccesslogsOperator  *bool `json:"accesslogs_operator,omitempty" tfsdk:"accesslogs_operator"`
	RatelimiterOperator *bool `json:"ratelimiter_operator,omitempty" tfsdk:"ratelimiter_operator"`
	SmOperator          *bool `json:"sm_operator,omitempty" tfsdk:"sm_operator"`
	SpanOperator        *bool `json:"span_operator,omitempty" tfsdk:"span_operator"`
}

type K8sClusterConfigFlag struct {
	Installed bool `json:"installed"`
}

type K8sClusterConfigIstioControlPlane struct {
	Name    string                          `json:"name"`
	Flavor  string                          `json:"flavor"`
	Options *K8sClusterConfigIstioCPOptions `json:"options"`
}

type K8sClusterConfigIstioCPOptions struct {
	Eventrouter  bool `json:"eventrouter"`
	Mesherizator bool `json:"mesherizator"`
}

type K8sClusterConfigControlPlane struct {
	Size     int64               `json:"size"`
	Flavor   K8sConfigNodeFlavor `json:"flavor"`
	RoleName string              `json:"role_name"`
}

type K8sConfigIngress struct {
	Name           string                    `json:"name"`
	Size           int64                     `json:"size"`
	Flavor         K8sConfigNodeFlavor       `json:"flavor"`
	IsCodes        []string                  `json:"iscodes"`
	Features       *K8sConfigIngressFeatures `json:"features"`
	TcpUdpSettings []K8sConfigIngressTcpUdp  `json:"tcp_udp_settings"`
}

type K8sConfigIngressFeatures struct {
	Http2Protocol bool `json:"http2_protocol"`
	ProxyProtocol bool `json:"proxy_protocol"`
}

type K8sConfigIngressTcpUdp struct {
	Mode    string `json:"mode"`
	Port    int64  `json:"port"`
	Service string `json:"service"`
}

type WorkerIngressK8sCluster struct {
	Size    int64               `json:"size"`
	Flavor  K8sConfigNodeFlavor `json:"flavor"`
	IsCodes []string            `json:"iscodes"`
}

type VersionK8sCluster struct {
	K8sVersion     string `json:"k8s_version"`
	ProductVersion string `json:"product_version"`
}

type Limits struct {
	ContainerCPURatio    int64 `json:"container_cpu_ratio"`
	ContainerMemoryRatio int64 `json:"container_memory_ratio"`
}
