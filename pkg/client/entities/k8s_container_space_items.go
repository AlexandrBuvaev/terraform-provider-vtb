package entities

func (c K8sContainerSpaceConfig) GetProviderType() (string, string) {
	return "kubernetes", "container_space"
}

type K8sContainerSpaceConfig struct {
	Name             string                              `json:"name"`
	Domain           string                              `json:"domain"`
	Platform         string                              `json:"platform"`
	DataCenter       string                              `json:"data_center"`
	NetSegment       string                              `json:"net_segment"`
	AvailabilityZone string                              `json:"availability_zone"`
	ControlPlane     K8sContainerSpaceConfigControlPlane `json:"control_plane"`
	Region           K8sConfigRegion                     `json:"region"`
	Ingress          K8sConfigIngress                    `json:"ingress_shard"`
}

// type K8sContainerSpaceConfigRegion struct {
// 	// VMs                  []K8sContainerSpaceConfigVM               `json:"vms"`
// 	Name                 string                                   `json:"name"`
// 	Size                 int64                                    `json:"size"`
// 	Flavor               K8sContainerSpaceConfigNodeFlavor        `json:"flavor"`
// 	IsCodes              []string                                 `json:"iscodes"`
// 	PodCPUMax            int64                                    `json:"pod_cpu_max"`
// 	PodMemoryMax         int64                                    `json:"pod_memory_max"`
// 	ContainerCPURatio    int64                                    `json:"container_cpu_ratio"`
// 	ContainerMemoryRatio int64                                    `json:"container_memory_ratio"`
// 	Components           *K8sContainerSpaceConfigRegionComponents `json:"components"`
// }

type K8sContainerSpaceConfigControlPlane struct {
	ID             string `json:"uuid"`
	Name           string `json:"name"`
	Version        string `json:"version"`
	ProductVersion string `json:"product_version"`
}

// type K8sContainerSpaceConfigIngress struct {
// 	Name           string                                  `json:"name"`
// 	Size           int64                                   `json:"size"`
// 	Flavor         K8sContainerSpaceConfigNodeFlavor       `json:"flavor"`
// 	IsCodes        []string                                `json:"iscodes"`
// 	Features       *K8sContainerSpaceConfigIngressFeatures `json:"features"`
// 	TcpUdpSettings []K8sContainerSpaceConfigIngressTcpUdp  `json:"tcp_udp_settings"`
// }

// type K8sContainerSpaceConfigIngressFeatures struct {
// 	Http2Protocol bool `json:"http2_protocol"`
// 	ProxyProtocol bool `json:"proxy_protocol"`
// }

// type K8sContainerSpaceConfigIngressTcpUdp struct {
// 	Mode    string `json:"mode"`
// 	Port    int64  `json:"port"`
// 	Service string `json:"service"`
// }

type K8sContainerSpaceConfigNodeFlavor struct {
	Cpus   int64  `json:"cpus"`
	Name   string `json:"name"`
	UUID   string `json:"uuid"`
	Memory int64  `json:"memory"`
}

type K8sContainerSpaceConfigRegionComponents struct {
	Astrom       bool `json:"astrom"`
	TsamOperator bool `json:"tsam_operator"`
	ChaosMesh    bool `json:"chaos_mesh"`
	TsdsOperator bool `json:"tsds_operator"`
	TslgOperator bool `json:"tslg_operator"`
	Tyk          bool `json:"tyk"`
}

// type K8sContainerSpaceConfigVM struct {
// 	Hostname string `json:"hostname"`
// }

type RatioK8sContainerSpace struct {
	ID                   string `json:"id"`
	ContainerCPURatio    int64  `json:"container_cpu_ratio"`
	ContainerMemoryRatio int64  `json:"container_memory_ratio"`
}

type WorkerK8sContainerSpace struct {
	Size                 int64               `json:"size"`
	Flavor               K8sConfigNodeFlavor `json:"flavor"`
	PodCPUMax            int64               `json:"pod_cpu_max"`
	PodMemoryMax         int64               `json:"pod_memory_max"`
	ContainerCPURatio    int64               `json:"container_cpu_ratio"`
	ContainerMemoryRatio int64               `json:"container_memory_ratio"`
}

type ControlPlaneK8sContainerSpace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type IngressK8sContainerSpace struct {
	Size            int64               `json:"size"`
	Flavor          K8sConfigNodeFlavor `json:"flavor"`
	IngressInRegion bool                `json:"ingress_in_region"`
}

type K8sRatio struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Data struct {
		ContainerCPURatio    int64 `json:"container_cpu_ratio"`
		ContainerMemoryRatio int64 `json:"container_memory_ratio"`
	} `json:"data"`
	ObjectInfo string `json:"object_info"`
	Directory  string `json:"directory"`
}

type QuotaK8sSpaceProject struct {
	CPU    float64 `json:"cpu" tfsdk:"cpu"`
	Memory float64 `json:"memory" tfsdk:"memory"`
}
