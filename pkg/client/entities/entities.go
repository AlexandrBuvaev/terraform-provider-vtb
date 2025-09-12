package entities

type AvailabilityZone struct {
	ID               string   `json:"id"`
	Code             string   `json:"code"`
	Label            string   `json:"label"`
	Weight           any      `json:"weight"`
	Organization     string   `json:"organization"`
	DataCenterIds    []string `json:"data_center_ids"`
	EnvironmentTypes []string `json:"environment_types"`
	IsDeleted        bool     `json:"is_deleted"`
}

type NetSegment struct {
	Id        string `json:"id"`
	Code      string `json:"code"`
	Label     string `json:"label"`
	Weight    int    `json:"weight"`
	IsDeleted bool   `json:"is_deleted"`
}

type ADLogonGrants struct {
	Role   string   `json:"role" tfsdk:"role"`
	Groups []string `json:"groups" tfsdk:"groups"`
}

type Flavor struct {
	Cores  int64  `json:"cpus" tfsdk:"cores"`
	Memory int64  `json:"memory" tfsdk:"memory"`
	Name   string `json:"name" tfsdk:"name"`
	UUID   string `json:"uuid" tfsdk:"uuid"`
}

type ExtraMount struct {
	Path       string `json:"path" tfsdk:"path"`
	Size       int64  `json:"size" tfsdk:"size"`
	FileSystem string `json:"file_system" tfsdk:"file_system"`
}

type BootDisk struct {
	Size int64 `json:"size" tfsdk:"size"`
}

type DefaultNic struct {
	NetSegment string `json:"net_segment"`
}

type Creator struct {
	Id        string `json:"id"`
	Email     string `json:"email"`
	LastName  string `json:"lastname"`
	UserName  string `json:"username"`
	FirstName string `json:"firstname"`
}

type ActionData struct {
	OrderId            string  `json:"order_id"`
	Status             string  `json:"status"`
	HasPrintableOutput bool    `json:"has_printable_output"`
	ActionId           string  `json:"action_id"`
	OperationId        string  `json:"operation_id"`
	Label              string  `json:"label"`
	CreatedAt          string  `json:"created_at"`
	StartTime          string  `json:"start_time"`
	GraphId            string  `json:"graph_id"`
	GraphVersion       string  `json:"graph_version"`
	Duration           int     `json:"duration"`
	Creator            Creator `json:"creator"`
}

type AccessGroup struct {
	IsDeleted   bool   `json:"is_deleted"`
	ID          string `json:"id"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	Name        string `json:"name"`
	ProjectName string `json:"project_name"`
	Description string `json:"description"`
	Creator     string `json:"creator"`
	Domain      string `json:"domain"`
	GroupDn     string `json:"group_dn"`
	Purpose     struct {
		ID          string      `json:"id"`
		CreatedAt   string      `json:"created_at"`
		UpdatedAt   string      `json:"updated_at"`
		Code        string      `json:"code"`
		Postprefix  interface{} `json:"postprefix"`
		Description string      `json:"description"`
	} `json:"purpose"`
}

type Domain struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	Label     string `json:"label"`
	Weight    int    `json:"weight"`
	IsDeleted bool   `json:"is_deleted"`
}

type Platform struct {
	ID                string `json:"id"`
	Code              string `json:"code"`
	Label             string `json:"label"`
	Weight            int    `json:"weight"`
	IsDeleted         bool   `json:"is_deleted"`
	RestrictionReason string `json:"restriction_reason"`
	Status            string `json:"status"`
}

type Project struct {
	Name                 string             `json:"name"`
	Title                string             `json:"title"`
	Description          string             `json:"description"`
	DisableRollback      bool               `json:"disable_rollback"`
	Organization         string             `json:"organization"`
	Folder               string             `json:"folder"`
	InformationSystemID  string             `json:"information_system_id"`
	ProjectEnvironmentID string             `json:"project_environment_id"`
	EnvironmentPrefixID  string             `json:"environment_prefix_id"`
	WasteBucket          bool               `json:"waste_bucket"`
	Availability         bool               `json:"availability"`
	AvailabilityReason   string             `json:"availability_reason"`
	EnvironmentPrefix    EnvironmentPrefix  `json:"environment_prefix"`
	InformationSystem    InformationSystem  `json:"information_system"`
	ProjectEnvironment   ProjectEnvironment `json:"project_environment"`
	CreatedAt            string             `json:"created_at"`
	UpdatedAt            string             `json:"updated_at"`
}

type EnvironmentPrefix struct {
	ID                   string `json:"id"`
	CreatedAt            string `json:"created_at"`
	UpdatedAt            string `json:"updated_at"`
	Name                 string `json:"name"`
	RisName              string `json:"ris_name"`
	Description          string `json:"description"`
	ProjectEnvironmentID string `json:"project_environment_id"`
}

type InformationSystem struct {
	ID               string `json:"id"`
	RisID            string `json:"ris_id"`
	Name             string `json:"name"`
	ShortName        string `json:"short_name"`
	Code             string `json:"code"`
	ShortCode        string `json:"short_code"`
	TypeIs           string `json:"type_is"`
	Owner            string `json:"owner"`
	Status           string `json:"status"`
	CriticalCategory string `json:"critical_category"`
	CodeApd          string `json:"code_apd"`
	Strim            string `json:"strim"`
	Organization     string `json:"organization"`
	Priority         int    `json:"priority"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type ProjectEnvironment struct {
	ID              string `json:"id"`
	EnvironmentType string `json:"environment_type"`
	Name            string `json:"name"`
	Purpose         string `json:"purpose"`
	Organization    string `json:"organization"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

type FinancialProject struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
	Type string `json:"type"`
}

type FinancialSource struct {
	Name       string `json:"name"`
	SourceType string `json:"source_type"`
	Code       string `json:"code"`
	StartDate  string `json:"start_date"`
}

type AirflowConfigHost struct {
	Host string `json:"host"`
}
type DataCenter struct {
	Id        string `json:"id"`
	Site      string `json:"site"`
	Code      string `json:"code"`
	Label     string `json:"label"`
	Name      string `json:"name"`
	Weight    int    `json:"weight"`
	IsDeleted bool   `json:"is_deleted"`
	Status    string `json:"status"`
}

type ServiceAccount struct {
	Name    string `json:"name"`
	Title   string `json:"title"`
	Creator struct {
		Email string `json:"email"`
	} `json:"creator"`
	Roles     []ServiceAccountRole `json:"roles"`
	ApiKey    ApiKey               `json:"api_key"`
	CreatedAt string               `json:"created_at"`
	UpdatedAt string               `json:"updated_at"`
}

type ServiceAccountRole struct {
	Name                    string   `json:"name"`
	Title                   string   `json:"title"`
	Description             string   `json:"description"`
	Type                    string   `json:"type"`
	AvailableForCollections []string `json:"available_for_collections"`
	CreatedAt               string   `json:"created_at"`
	UpdatedAt               string   `json:"updated_at"`
}

type ApiKey struct {
	Name           string `json:"name"`
	ServiceAccount string `json:"service_account"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}
