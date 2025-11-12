package entities

func (c K8sProjectItemConfig) GetProviderType() (string, string) {
	return "kubernetes", "project"
}

type K8sProjectItemConfig struct {
	UUID             string                      `json:"uuid"`
	Router           string                      `json:"router"`
	UIDRange         string                      `json:"uid_range"`
	Environment      string                      `json:"environment"`
	ProjectURL       string                      `json:"project_url"`
	ProjectName      string                      `json:"project_name"`
	EnvironmentType  string                      `json:"environment_type"`
	Quota            K8sProjectQuotaData         `json:"quota"`
	Cluster          K8sProjectClusterData       `json:"cluster"`
	Roles            []RolesK8sProject           `json:"roles"`
	Region           string                      `json:"region"`
	Istio            K8sProjectIstioData         `json:"istio"`
	Tyk              K8sProjectFullComponentData `json:"tyk"`
	TslgOperator     K8sProjectFullComponentData `json:"tslg_operator"`
	TsamOperator     K8sProjectFullComponentData `json:"tsam_operator"`
	TsdsOperator     K8sProjectComponentData     `json:"tsds_operator"`
	OmniCertificates []K8sProjectOmniData        `json:"omni_certificates"`
	ChaosMesh        K8sProjectComponentData     `json:"chaos_mesh"`
}

type K8sProjectQuotaData struct {
	CPU    float64 `json:"cpu"`
	Memory float64 `json:"memory"`
}

type K8sProjectClusterData struct {
	Name    string `json:"name"`
	UUID    string `json:"uuid"`
	Domain  string `json:"domain"`
	APIURL  string `json:"api_url"`
	Segment string `json:"segment"`
	Version string `json:"version"`
}

type K8sProjectIstioData struct {
	ControlPlane string            `json:"control_plane"`
	Roles        []RolesK8sProject `json:"roles"`
}

type K8sProjectFullComponentData struct {
	Namespace string            `json:"namespace"`
	Roles     []RolesK8sProject `json:"roles"`
}

type K8sProjectComponentData struct {
	Namespace string `json:"namespace"`
}

type K8sProjectOmniData struct {
	AppName    string `json:"app_name"`
	ClientName string `json:"cn"`
	Name       string `json:"name"`
}

type QuotaK8sProject struct {
	CPU    int64 `json:"cpu" tfsdk:"cpu"`
	Memory int64 `json:"memory" tfsdk:"memory"`
}

type RolesK8sProject struct {
	Role   string   `json:"role" tfsdk:"role"`
	Groups []string `json:"groups" tfsdk:"groups"`
}

type ResourcePoolK8sProject struct {
	ID     string `json:"id" tfsdk:"id"`
	Name   string `json:"name" tfsdk:"name_cluster"`
	Region string `json:"region" tfsdk:"region"`
}

type K8sRoles struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Data struct {
		Name  string `json:"name"`
		Title string `json:"title"`
	} `json:"data"`
	Directory string `json:"directory"`
}

type K8sPoolResponse struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Name   string `json:"name"`
	Region string `json:"region"`
}
