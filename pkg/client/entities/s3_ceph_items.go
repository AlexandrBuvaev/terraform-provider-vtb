package entities

func (c S3CephTenantItemConfig) GetProviderType() (string, string) { // Tenant item
	return "s3", "ceph"
}

func (c S3CephBucketItemConfig) GetProviderType() (string, string) { // Bucket item
	return "s3_bucket", "ceph"
}

type S3CephTenantItemConfig struct {
	Name            string                 `json:"name"`
	Users           []S3CephTenantUsers    `json:"users"`
	Polices         []S3CephTenantPolicies `json:"policies"`
	DataCenter      S3CephDataCenter       `json:"data_center"`
	Environment     string                 `json:"environment"`
	NetSegment      string                 `json:"net_segment"`
	StorageType     string                 `json:"storage_type"`
	EnvironmentType string                 `json:"environment_type"`
	MtlsEndpoint    string                 `json:"mtls_endpoint"`
	UserEndpoint    string                 `json:"user_endpoint"`
}

type S3CephBucketItemConfig struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	Versioning bool   `json:"versioning"`
	MaxSizeGb  int64  `json:"max_size_gb"`
	TenantName string `json:"tenant_name"`
}

type S3CephBucketCorsRules struct {
	CorsId        string   `json:"cors_id"`
	MaxAgeSeconds string   `json:"max_age_seconds"`
	Headers       []string `json:"headers"`
	Methods       []string `json:"methods"`
	Origins       []string `json:"origins"`
	ExposeHeaders []string `json:"expose_headers"`
}

type S3CephBucketLifecycleRules struct {
	Name                      string                           `json:"name"`
	Type                      string                           `json:"type"`
	Filter                    S3CephBucketLifecycleRulesFilter `json:"filter"`
	Condition                 string                           `json:"condition"`
	Days                      *int                             `json:"days,omitempty"`
	Date                      *string                          `json:"date,omitempty"`
	ExpiredObjectDeleteMarker *bool                            `json:"expired_object_delete_marker,omitempty"`
	UpdateDate                *string                          `json:"update_date,omitempty"`
}

type S3CephBucketLifecycleRulesFilter struct {
	Type string `json:"type"`
}

type S3CephTenantUsers struct {
	UserName  string `json:"user_name"`
	AccessKey string `json:"access_key"`
}

type S3CephTenantPolicies struct {
	Policy         Policy `json:"policy"`
	Prefix         string `json:"prefix"`
	UserId         string `json:"user_id"`
	PolicyId       string `json:"policy_id"`
	BucketName     string `json:"bucket_name"`
	SelectedRights string `json:"selected_rights"`
}

type S3CephDataCenter struct {
	Code string `json:"user_name"`
	Name string `json:"access_key"`
	Site string `json:"site"`
}

type S3CephResourcePool struct {
	Fsid         string                  `json:"fsid"`
	Name         string                  `json:"name"`
	Uuid         string                  `json:"uuid"`
	Endpoint     string                  `json:"endpoint"`
	Platform     string                  `json:"platform"`
	StorageType  string                  `json:"storage_type"`
	EndpointMtls string                  `json:"endpoint_mtls"`
	DataCenter   []S3CephDataCenter      `json:"data_center"`
	Links        S3CephResourcePoolLinks `json:"links"`
}

type S3CephResourcePoolLinks struct {
	Mtls    map[string]string `json:"mtls"`
	Default map[string]string `json:"default"`
}

type Policy struct {
	Read                 bool `json:"read"`
	Write                bool `json:"write"`
	Delete               bool `json:"delete"`
	PutBucketCors        bool `json:"put_bucket_cors"`
	AbortMultipartUpload bool `json:"abort_multipart_upload"`
}
