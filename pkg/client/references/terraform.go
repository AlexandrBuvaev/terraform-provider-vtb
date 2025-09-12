/*
Модуль реализует запросы к справочнику `terraform`
*/
package references

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"terraform-provider-vtb/pkg/client/auth"
)

// TerraformReferencePage определяет базовые поля для страницы продукта в справочнике `terraform`
// Как правило, данные поля используются при составлении attrs для  полезной нагрузки
//
// От среды к среде данные на странице в справочнике могу меняться
type TerraformReferencePage struct {
	OnSupport        bool   `json:"on_support"`
	ProductID        string `json:"product_id"`
	ADIntegration    bool   `json:"ad_integration"`
	DefaultOSVersion string `json:"default_os_version"`
}

// BasicTags определяют поля, которые используются для поиска по тегам на странице в справочнике
type BasicTags struct {
	Distribution string `json:"distribution"` // astra |-
	Organization string `json:"organization"` // vtb   |-- For `terraform` reference
	Environment  string `json:"environment"`  // prod  |-
	Version      string `json:"version"`      // for `image` reference
}

type ComputeReferenceData struct {
	TerraformReferencePage
	BasicTags
	BootDisk struct {
		Size int64 `json:"size"`
	} `json:"boot_disk,omitempty"`
}
type ComputeReferenceResponse struct {
	ReferenceResponse
	Data ComputeReferenceData `json:"data"`
}

func GetComputeImageData(
	creds *auth.Credentials,
	distribution,
	organization,
	environment,
	version string,
) (*ComputeReferenceData, error) {

	tags := fmt.Sprintf(
		"compute,%s,%s,%s",
		strings.ToLower(distribution),
		strings.ToLower(organization),
		strings.ToLower(environment),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	terraformOsData := make([]ComputeReferenceResponse, 1)
	err = json.Unmarshal(body, &terraformOsData)
	if err != nil {
		return nil, err
	}

	if len(terraformOsData) == 0 {
		return nil, errors.New("can't find data from reference about image")
	}

	if terraformOsData[0].Data.ProductID == "" {
		return nil, errors.New("can't find product with this parameters")
	}

	defaultOsVersion, err := GetImageOsVersion(*creds, distribution, version)
	if err != nil {
		return nil, err
	}

	image := ComputeReferenceData{
		TerraformReferencePage: TerraformReferencePage{
			OnSupport:        terraformOsData[0].Data.OnSupport,
			ProductID:        terraformOsData[0].Data.ProductID,
			ADIntegration:    terraformOsData[0].Data.ADIntegration,
			DefaultOSVersion: terraformOsData[0].Data.DefaultOSVersion,
		},
		BasicTags: BasicTags{
			Distribution: distribution,
			Environment:  environment,
			Organization: organization,
			Version:      defaultOsVersion,
		},
	}
	return &image, nil
}

func GetNginxImageData(
	creds *auth.Credentials,
	distribution,
	organization,
	environment,
	version string,
) (*ComputeReferenceData, error) {

	tags := fmt.Sprintf(
		"nginx,%s,%s,%s",
		strings.ToLower(distribution),
		strings.ToLower(organization),
		strings.ToLower(environment),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	terraformOsData := make([]ComputeReferenceResponse, 1)
	err = json.Unmarshal(body, &terraformOsData)
	if err != nil {
		return nil, err
	}

	if len(terraformOsData) == 0 {
		return nil, fmt.Errorf("can't find page in reference `terraform` with tags=%s", tags)
	}
	if terraformOsData[0].Data.ProductID == "" {
		return nil, fmt.Errorf("can't find product_id in reference `terraform` with tags=%s", tags)
	}

	imageOsVersion, err := GetImageOsVersion(*creds, distribution, version)
	if err != nil {
		return nil, err
	}

	return &ComputeReferenceData{
		TerraformReferencePage: TerraformReferencePage{
			OnSupport:     terraformOsData[0].Data.OnSupport,
			ProductID:     terraformOsData[0].Data.ProductID,
			ADIntegration: terraformOsData[0].Data.ADIntegration,
		},
		BasicTags: BasicTags{
			Distribution: distribution,
			Environment:  environment,
			Organization: organization,
			Version:      imageOsVersion,
		},
		BootDisk: terraformOsData[0].Data.BootDisk,
	}, nil
}

type AgentOrchestrationImageData struct {
	BasicTags
	TerraformReferencePage
}
type AgentOrchestrationImageResponse struct {
	ReferenceResponse
	Data AgentOrchestrationImageData `json:"data"`
}

func GetAgentOrchestrationImageData(
	creds *auth.Credentials,
	distribution,
	organization,
	environment,
	version string,
) (*AgentOrchestrationImageData, error) {

	tags := fmt.Sprintf(
		"agent_orchestration,%s,%s,%s",
		strings.ToLower(distribution),
		strings.ToLower(organization),
		strings.ToLower(environment),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	terraformOsData := make([]AgentOrchestrationImageResponse, 1)
	err = json.Unmarshal(body, &terraformOsData)
	if err != nil {
		return nil, err
	}

	if len(terraformOsData) == 0 {
		return nil, fmt.Errorf("can't find page in reference `terraform` with tags=%s", tags)
	}
	if terraformOsData[0].Data.ProductID == "" {
		return nil, fmt.Errorf("can't find product_id in reference `terraform` with tags=%s", tags)
	}

	imageOsVersion, err := GetImageOsVersion(*creds, distribution, version)
	if err != nil {
		return nil, err
	}

	return &AgentOrchestrationImageData{
		TerraformReferencePage: TerraformReferencePage{
			OnSupport:     terraformOsData[0].Data.OnSupport,
			ProductID:     terraformOsData[0].Data.ProductID,
			ADIntegration: terraformOsData[0].Data.ADIntegration,
		},
		BasicTags: BasicTags{
			Distribution: distribution,
			Environment:  environment,
			Organization: organization,
			Version:      imageOsVersion,
		},
	}, nil
}

type RedisImageData struct {
	BasicTags
	TerraformReferencePage

	RedisVersion         string `json:"redis_version"`
	AddParameters        bool   `json:"add_parameters"`
	Auth                 bool   `json:"auth"`
	AuthUser             string `json:"auth_user"`
	User                 string `json:"user"`
	UserPassword         string `json:"user_password"`
	NotifyKeyspaceEvents string `json:"notify_keyspace_events"`
	AcceptDocumentation  bool   `json:"accept_documentation"`
}
type RedisImageResponse struct {
	ReferenceResponse
	Data RedisImageData `json:"data"`
}

func GetRedisImageData(
	creds *auth.Credentials,
	distribution,
	organization,
	environment,
	version string,
) (*RedisImageData, error) {

	tags := fmt.Sprintf(
		"redis,%s,%s,%s",
		strings.ToLower(distribution),
		strings.ToLower(organization),
		strings.ToLower(environment),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	terraformOsData := make([]RedisImageResponse, 1)
	err = json.Unmarshal(body, &terraformOsData)
	if err != nil {
		return nil, err
	}

	if len(terraformOsData) == 0 {
		return nil, fmt.Errorf("can't find page in reference `terraform` with tags=%s", tags)
	}
	if terraformOsData[0].Data.ProductID == "" {
		return nil, fmt.Errorf("can't find product_id in reference `terraform` with tags=%s", tags)
	}

	imageOsVersion, err := GetImageOsVersion(*creds, distribution, version)
	if err != nil {
		return nil, err
	}

	return &RedisImageData{
		TerraformReferencePage: TerraformReferencePage{
			OnSupport:        terraformOsData[0].Data.OnSupport,
			ProductID:        terraformOsData[0].Data.ProductID,
			ADIntegration:    terraformOsData[0].Data.ADIntegration,
			DefaultOSVersion: terraformOsData[0].Data.DefaultOSVersion,
		},
		BasicTags: BasicTags{
			Distribution: distribution,
			Environment:  environment,
			Organization: organization,
			Version:      imageOsVersion,
		},

		RedisVersion:         terraformOsData[0].Data.RedisVersion,
		AddParameters:        terraformOsData[0].Data.AddParameters,
		Auth:                 terraformOsData[0].Data.Auth,
		AuthUser:             terraformOsData[0].Data.AuthUser,
		User:                 terraformOsData[0].Data.User,
		UserPassword:         terraformOsData[0].Data.UserPassword,
		NotifyKeyspaceEvents: terraformOsData[0].Data.NotifyKeyspaceEvents,
		AcceptDocumentation:  terraformOsData[0].Data.AcceptDocumentation,
	}, nil
}

type RedisSentinelImageData struct {
	BasicTags
	TerraformReferencePage

	RedisVersion         string `json:"redis_version"`
	AddParameters        bool   `json:"add_parameters"`
	Auth                 bool   `json:"auth"`
	AuthUser             string `json:"auth_user"`
	User                 string `json:"user"`
	UserPassword         string `json:"user_password"`
	NotifyKeyspaceEvents string `json:"notify_keyspace_events"`
	AcceptDocumentation  bool   `json:"accept_documentation"`
}
type RedisSentinelImageResponse struct {
	ReferenceResponse
	Data RedisSentinelImageData `json:"data"`
}

func GetRedisSentinelImageData(
	creds *auth.Credentials,
	distribution,
	organization,
	environment,
	version string,
) (*RedisSentinelImageData, error) {

	tags := fmt.Sprintf(
		"redis_sentinel,%s,%s,%s",
		strings.ToLower(distribution),
		strings.ToLower(organization),
		strings.ToLower(environment),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	terraformOsData := make([]RedisSentinelImageResponse, 1)
	err = json.Unmarshal(body, &terraformOsData)
	if err != nil {
		return nil, err
	}

	if len(terraformOsData) == 0 {
		return nil, fmt.Errorf("can't find page in reference `terraform` with tags=%s", tags)
	}
	if terraformOsData[0].Data.ProductID == "" {
		return nil, fmt.Errorf("can't find product_id in reference `terraform` with tags=%s", tags)
	}

	imageOsVersion, err := GetImageOsVersion(*creds, distribution, version)
	if err != nil {
		return nil, err
	}

	return &RedisSentinelImageData{
		TerraformReferencePage: TerraformReferencePage{
			OnSupport:        terraformOsData[0].Data.OnSupport,
			ProductID:        terraformOsData[0].Data.ProductID,
			ADIntegration:    terraformOsData[0].Data.ADIntegration,
			DefaultOSVersion: terraformOsData[0].Data.DefaultOSVersion,
		},
		BasicTags: BasicTags{
			Distribution: distribution,
			Environment:  environment,
			Organization: organization,
			Version:      imageOsVersion,
		},

		RedisVersion:         terraformOsData[0].Data.RedisVersion,
		AddParameters:        terraformOsData[0].Data.AddParameters,
		Auth:                 terraformOsData[0].Data.Auth,
		AuthUser:             terraformOsData[0].Data.AuthUser,
		User:                 terraformOsData[0].Data.User,
		UserPassword:         terraformOsData[0].Data.UserPassword,
		NotifyKeyspaceEvents: terraformOsData[0].Data.NotifyKeyspaceEvents,
		AcceptDocumentation:  terraformOsData[0].Data.AcceptDocumentation,
	}, nil
}

type GrafanaImageData struct {
	BasicTags
	TerraformReferencePage
}
type GrafanaImageResponse struct {
	ReferenceResponse
	Data GrafanaImageData `json:"data"`
}

func GetGrafanaImageData(
	creds *auth.Credentials,
	distribution,
	organization,
	environment,
	version string,
) (*GrafanaImageData, error) {
	tags := fmt.Sprintf(
		"grafana,%s,%s,%s",
		strings.ToLower(distribution),
		strings.ToLower(organization),
		strings.ToLower(environment),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	terraformOsData := make([]EtcdImageResponse, 1)
	err = json.Unmarshal(body, &terraformOsData)
	if err != nil {
		return nil, err
	}

	if len(terraformOsData) == 0 {
		return nil, fmt.Errorf("can't find page in reference `terraform` with tags=%s", tags)
	}
	if terraformOsData[0].Data.ProductID == "" {
		return nil, fmt.Errorf("can't find product_id in reference `terraform` with tags=%s", tags)
	}

	imageOsVersion, err := GetImageOsVersion(*creds, distribution, version)
	if err != nil {
		return nil, err
	}

	return &GrafanaImageData{
		TerraformReferencePage: TerraformReferencePage{
			OnSupport:        terraformOsData[0].Data.OnSupport,
			ProductID:        terraformOsData[0].Data.ProductID,
			ADIntegration:    terraformOsData[0].Data.ADIntegration,
			DefaultOSVersion: terraformOsData[0].Data.DefaultOSVersion,
		},
		BasicTags: BasicTags{
			Distribution: distribution,
			Environment:  environment,
			Organization: organization,
			Version:      imageOsVersion,
		},
	}, nil

}

type EtcdImageData struct {
	BasicTags
	TerraformReferencePage

	UseSsl          bool `json:"use_ssl"`
	OnBackup        bool `json:"on_backup"`
	GeoDistribution bool `json:"geo_distribution"`
}
type EtcdImageResponse struct {
	ReferenceResponse
	Data EtcdImageData `json:"data"`
}

func GetEtcdImageData(
	creds *auth.Credentials,
	distribution,
	organization,
	environment,
	version string,
) (*EtcdImageData, error) {
	tags := fmt.Sprintf(
		"etcd,%s,%s,%s",
		strings.ToLower(distribution),
		strings.ToLower(organization),
		strings.ToLower(environment),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	terraformOsData := make([]EtcdImageResponse, 1)
	err = json.Unmarshal(body, &terraformOsData)
	if err != nil {
		return nil, err
	}

	if len(terraformOsData) == 0 {
		return nil, fmt.Errorf("can't find page in reference `terraform` with tags=%s", tags)
	}
	if terraformOsData[0].Data.ProductID == "" {
		return nil, fmt.Errorf("can't find product_id in reference `terraform` with tags=%s", tags)
	}

	imageOsVersion, err := GetImageOsVersion(*creds, distribution, version)
	if err != nil {
		return nil, err
	}

	return &EtcdImageData{
		TerraformReferencePage: TerraformReferencePage{
			OnSupport:        terraformOsData[0].Data.OnSupport,
			ProductID:        terraformOsData[0].Data.ProductID,
			ADIntegration:    terraformOsData[0].Data.ADIntegration,
			DefaultOSVersion: terraformOsData[0].Data.DefaultOSVersion,
		},
		BasicTags: BasicTags{
			Distribution: distribution,
			Environment:  environment,
			Organization: organization,
			Version:      imageOsVersion,
		},

		UseSsl:          terraformOsData[0].Data.UseSsl,
		OnBackup:        terraformOsData[0].Data.OnBackup,
		GeoDistribution: terraformOsData[0].Data.GeoDistribution,
	}, nil

}

type ClickhouseImageData struct {
	BasicTags
	TerraformReferencePage
}
type ClickhouseImageResponse struct {
	ReferenceResponse
	Data ClickhouseImageData `json:"data"`
}

func GetClickhouseImageData(
	creds *auth.Credentials,
	distribution,
	organization,
	environment,
	version string,
) (*ClickhouseImageData, error) {
	tags := fmt.Sprintf(
		"clickhouse,%s,%s,%s",
		strings.ToLower(distribution),
		strings.ToLower(organization),
		strings.ToLower(environment),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	terraformOsData := make([]ClickhouseImageResponse, 1)
	err = json.Unmarshal(body, &terraformOsData)
	if err != nil {
		return nil, err
	}

	if len(terraformOsData) == 0 {
		return nil, fmt.Errorf("can't find page in reference `terraform` with tags=%s", tags)
	}
	if terraformOsData[0].Data.ProductID == "" {
		return nil, fmt.Errorf("can't find product_id in reference `terraform` with tags=%s", tags)
	}

	imageOsVersion, err := GetImageOsVersion(*creds, distribution, version)
	if err != nil {
		return nil, err
	}

	return &ClickhouseImageData{
		TerraformReferencePage: TerraformReferencePage{
			OnSupport:        terraformOsData[0].Data.OnSupport,
			ProductID:        terraformOsData[0].Data.ProductID,
			ADIntegration:    terraformOsData[0].Data.ADIntegration,
			DefaultOSVersion: terraformOsData[0].Data.DefaultOSVersion,
		},
		BasicTags: BasicTags{
			Distribution: distribution,
			Environment:  environment,
			Organization: organization,
			Version:      imageOsVersion,
		},
	}, nil

}

type ClickhouseClusterImageData struct {
	BasicTags
	TerraformReferencePage

	GeoDistribution bool `json:"geo_distribution"`
}
type ClickhouseClusterImageResponse struct {
	ReferenceResponse
	Data ClickhouseClusterImageData `json:"data"`
}

func GetClickhouseClusterImageData(
	creds *auth.Credentials,
	distribution,
	organization,
	environment,
	version string,
) (*ClickhouseClusterImageData, error) {
	tags := fmt.Sprintf(
		"clickhouse_cluster,%s,%s,%s",
		strings.ToLower(distribution),
		strings.ToLower(organization),
		strings.ToLower(environment),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	terraformOsData := make([]ClickhouseClusterImageResponse, 1)
	err = json.Unmarshal(body, &terraformOsData)
	if err != nil {
		return nil, err
	}

	if len(terraformOsData) == 0 {
		return nil, fmt.Errorf("can't find page in reference `terraform` with tags=%s", tags)
	}
	if terraformOsData[0].Data.ProductID == "" {
		return nil, fmt.Errorf("can't find product_id in reference `terraform` with tags=%s", tags)
	}

	imageOsVersion, err := GetImageOsVersion(*creds, distribution, version)
	if err != nil {
		return nil, err
	}

	return &ClickhouseClusterImageData{
		TerraformReferencePage: TerraformReferencePage{
			OnSupport:        terraformOsData[0].Data.OnSupport,
			ProductID:        terraformOsData[0].Data.ProductID,
			ADIntegration:    terraformOsData[0].Data.ADIntegration,
			DefaultOSVersion: terraformOsData[0].Data.DefaultOSVersion,
		},
		BasicTags: BasicTags{
			Distribution: distribution,
			Environment:  environment,
			Organization: organization,
			Version:      imageOsVersion,
		},

		GeoDistribution: terraformOsData[0].Data.GeoDistribution,
	}, nil

}

type WildflyImageData struct {
	TerraformReferencePage
	BasicTags
}
type WildflyReferenceResponse struct {
	ReferenceResponse
	Data WildflyImageData `json:"data"`
}

func GetWildflyImageData(
	creds *auth.Credentials,
	distribution,
	organization,
	environment,
	version string,
) (*WildflyImageData, error) {

	tags := fmt.Sprintf(
		"wildfly,%s,%s,%s",
		strings.ToLower(distribution),
		strings.ToLower(organization),
		strings.ToLower(environment),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	terraformOsData := make([]WildflyReferenceResponse, 1)
	err = json.Unmarshal(body, &terraformOsData)
	if err != nil {
		return nil, err
	}

	if len(terraformOsData) == 0 {
		return nil, fmt.Errorf("can't find page in reference `terraform` with tags=%s", tags)
	}
	if terraformOsData[0].Data.ProductID == "" {
		return nil, fmt.Errorf("can't find product_id in reference `terraform` with tags=%s", tags)
	}

	imageOsVersion, err := GetImageOsVersion(*creds, distribution, version)
	if err != nil {
		return nil, err
	}

	return &WildflyImageData{
		TerraformReferencePage: TerraformReferencePage{
			OnSupport:        terraformOsData[0].Data.OnSupport,
			ProductID:        terraformOsData[0].Data.ProductID,
			ADIntegration:    terraformOsData[0].Data.ADIntegration,
			DefaultOSVersion: terraformOsData[0].Data.DefaultOSVersion,
		},
		BasicTags: BasicTags{
			Distribution: distribution,
			Environment:  environment,
			Organization: organization,
			Version:      imageOsVersion,
		},
	}, nil
}

type RabbitMQImageData struct {
	TerraformReferencePage
	BasicTags

	GeoDistribution        bool   `json:"geo_distribution"`
	DefaultLayout          string `json:"default_layout"`
	DefaultRabbitMQVersion string `json:"default_rabbitmq_version"`
	LatestReleaseVersion   string `json:"latest_release_version"`
}
type RabbitMQReferenceResponse struct {
	ReferenceResponse
	Data RabbitMQImageData `json:"data"`
}

func GetRabbitMQImageData(
	creds *auth.Credentials,
	distribution,
	organization,
	environment,
	version string,
) (*RabbitMQImageData, error) {

	tags := fmt.Sprintf(
		"rabbitmq,%s,%s,%s",
		strings.ToLower(distribution),
		strings.ToLower(organization),
		strings.ToLower(environment),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	terraformOsData := make([]RabbitMQReferenceResponse, 1)
	err = json.Unmarshal(body, &terraformOsData)
	if err != nil {
		return nil, err
	}

	if len(terraformOsData) == 0 {
		return nil, fmt.Errorf("can't find page in reference `terraform` with tags=%s", tags)
	}
	if terraformOsData[0].Data.ProductID == "" {
		return nil, fmt.Errorf("can't find product_id in reference `terraform` with tags=%s", tags)
	}

	imageOsVersion, err := GetImageOsVersion(*creds, distribution, version)
	if err != nil {
		return nil, err
	}

	return &RabbitMQImageData{
		TerraformReferencePage: TerraformReferencePage{
			OnSupport:        terraformOsData[0].Data.OnSupport,
			ProductID:        terraformOsData[0].Data.ProductID,
			ADIntegration:    terraformOsData[0].Data.ADIntegration,
			DefaultOSVersion: terraformOsData[0].Data.DefaultOSVersion,
		},
		BasicTags: BasicTags{
			Distribution: distribution,
			Environment:  environment,
			Organization: organization,
			Version:      imageOsVersion,
		},

		GeoDistribution:        terraformOsData[0].Data.GeoDistribution,
		DefaultLayout:          terraformOsData[0].Data.DefaultLayout,
		DefaultRabbitMQVersion: terraformOsData[0].Data.DefaultRabbitMQVersion,
		LatestReleaseVersion:   terraformOsData[0].Data.LatestReleaseVersion,
	}, nil
}

type PostgresImageData struct {
	TerraformReferencePage
	BasicTags

	LoadProfile                 string `json:"load_profile"`
	PostgresqlVersion           string `json:"postgresql_version"`
	DefaultTransactionIsolation string `json:"default_transaction_isolation"`
	OnBackup                    bool   `json:"on_backup"`
	Replication                 bool   `json:"replication"`
}
type PostgresReferenceResponse struct {
	ReferenceResponse
	Data PostgresImageData `json:"data"`
}

func GetPostgresImageData(
	creds *auth.Credentials,
	distribution,
	productType,
	organization,
	environment,
	version string,
) (*PostgresImageData, error) {

	var productTag string

	switch productType {
	case "cluster":
		productTag = "postgresql_cluster"
	case "stand-alone":
		productTag = "postgresql"
	default:
		return nil, fmt.Errorf("unknow product type %s", productType)
	}

	tags := fmt.Sprintf(
		"%s,%s,%s,%s",
		productTag,
		strings.ToLower(distribution),
		strings.ToLower(organization),
		strings.ToLower(environment),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	terraformOsData := make([]PostgresReferenceResponse, 1)
	err = json.Unmarshal(body, &terraformOsData)
	if err != nil {
		return nil, err
	}

	if len(terraformOsData) == 0 {
		return nil, fmt.Errorf("can't find page in reference `terraform` with tags=%s", tags)
	}
	if terraformOsData[0].Data.ProductID == "" {
		return nil, fmt.Errorf("can't find product_id in reference `terraform` with tags=%s", tags)
	}

	imageOsVersion, err := GetImageOsVersion(*creds, distribution, version)
	if err != nil {
		return nil, err
	}

	return &PostgresImageData{
		TerraformReferencePage: TerraformReferencePage{
			OnSupport:        terraformOsData[0].Data.OnSupport,
			ProductID:        terraformOsData[0].Data.ProductID,
			ADIntegration:    terraformOsData[0].Data.ADIntegration,
			DefaultOSVersion: terraformOsData[0].Data.DefaultOSVersion,
		},
		BasicTags: BasicTags{
			Distribution: distribution,
			Environment:  environment,
			Organization: organization,
			Version:      imageOsVersion,
		},

		LoadProfile:                 terraformOsData[0].Data.LoadProfile,
		PostgresqlVersion:           terraformOsData[0].Data.PostgresqlVersion,
		DefaultTransactionIsolation: terraformOsData[0].Data.DefaultTransactionIsolation,
		OnBackup:                    terraformOsData[0].Data.OnBackup,
		Replication:                 terraformOsData[0].Data.Replication,
	}, nil
}

type KafkaImageData struct {
	TerraformReferencePage
	BasicTags

	GeoDistribution        bool   `json:"geo_distribution"`
	DefaultLayout          string `json:"default_layout"`
	DefaultKafkaVersion    string `json:"default_kafka_version"`
	LatestReleaseVersionV1 string `json:"latest_release_version_v1"`
	LatestReleaseVersionV2 string `json:"latest_release_version_v2"`
}
type KafkaDefaultsReferenceResponse struct {
	ReferenceResponse
	Data KafkaImageData `json:"data"`
}

func GetKafkaImageData(
	creds *auth.Credentials,
	distribution,
	organization,
	environment,
	version string,
) (*KafkaImageData, error) {

	tags := fmt.Sprintf(
		"kafka,%s,%s,%s",
		strings.ToLower(distribution),
		strings.ToLower(organization),
		strings.ToLower(environment),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	referenceKafkaData := make([]KafkaDefaultsReferenceResponse, 1)
	err = json.Unmarshal(body, &referenceKafkaData)
	if err != nil {
		return nil, err
	}

	if len(referenceKafkaData) == 0 {
		return nil, fmt.Errorf("can't find page in reference `terraform` with tags=%s", tags)
	}
	if referenceKafkaData[0].Data.ProductID == "" {
		return nil, fmt.Errorf("can't find product_id in reference `terraform` with tags=%s", tags)
	}

	defaultOsVersion, err := GetImageOsVersion(*creds, distribution, version)
	if err != nil {
		return nil, err
	}

	return &KafkaImageData{
		TerraformReferencePage: TerraformReferencePage{
			OnSupport:        referenceKafkaData[0].Data.OnSupport,
			ADIntegration:    referenceKafkaData[0].Data.ADIntegration,
			ProductID:        referenceKafkaData[0].Data.ProductID,
			DefaultOSVersion: defaultOsVersion,
		},
		BasicTags: BasicTags{
			Distribution: distribution,
			Environment:  environment,
			Organization: organization,
			Version:      defaultOsVersion,
		},

		GeoDistribution:        referenceKafkaData[0].Data.GeoDistribution,
		DefaultLayout:          referenceKafkaData[0].Data.DefaultLayout,
		DefaultKafkaVersion:    referenceKafkaData[0].Data.DefaultKafkaVersion,
		LatestReleaseVersionV1: referenceKafkaData[0].Data.LatestReleaseVersionV1,
		LatestReleaseVersionV2: referenceKafkaData[0].Data.LatestReleaseVersionV2,
	}, nil
}

type OpenMessagingImageData struct {
	BasicTags
	TerraformReferencePage
}
type OpenMessagingReferenceResponse struct {
	ReferenceResponse
	Data OpenMessagingImageData `json:"data"`
}

func GetOpenMessagingImageData(
	creds *auth.Credentials,
	distribution,
	organization,
	environment,
	version string,
) (*OpenMessagingImageData, error) {

	tags := fmt.Sprintf(
		"open-messaging,%s,%s,%s",
		strings.ToLower(distribution),
		strings.ToLower(organization),
		strings.ToLower(environment),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	terraformOsData := make([]OpenMessagingReferenceResponse, 1)
	err = json.Unmarshal(body, &terraformOsData)
	if err != nil {
		return nil, err
	}

	if len(terraformOsData) == 0 {
		return nil, fmt.Errorf("can't find page in reference `terraform` with tags=%s", tags)
	}

	imageOsVersion, err := GetImageOsVersion(*creds, distribution, version)
	if err != nil {
		return nil, err
	}

	return &OpenMessagingImageData{
		TerraformReferencePage: TerraformReferencePage{
			OnSupport:        terraformOsData[0].Data.OnSupport,
			ProductID:        terraformOsData[0].Data.ProductID,
			ADIntegration:    terraformOsData[0].Data.ADIntegration,
			DefaultOSVersion: terraformOsData[0].Data.DefaultOSVersion,
		},
		BasicTags: BasicTags{
			Distribution: distribution,
			Environment:  environment,
			Organization: organization,
			Version:      imageOsVersion,
		},
	}, nil
}

func GetKnownDomains(creds *auth.Credentials) ([]string, error) {

	parameters := map[string]string{"name": "known_domains"}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	var domains []struct {
		ReferenceResponse
		Data []string `json:"data"`
	}
	err = json.Unmarshal(body, &domains)
	if err != nil {
		return nil, err
	}

	if len(domains) == 0 || len(domains) > 1 {
		return nil, fmt.Errorf(
			"`terraform` reference must have exactly one page `known_domains`, " +
				"but seems like it have zero or more than one pages",
		)
	}
	return domains[0].Data, nil
}

type ArtemisImageData struct {
	TerraformReferencePage
	BasicTags

	GeoDistribution      bool   `json:"geo_distribution"`
	LatestReleaseVersion string `json:"latest_release_version"`
}
type ArtemisTerraformReferenceResponse struct {
	ReferenceResponse
	Data ArtemisImageData `json:"data"`
}

func GetArtemisImageData(
	creds *auth.Credentials,
	distribution,
	organization,
	environment,
	version string,
) (*ArtemisImageData, error) {
	tags := fmt.Sprintf(
		"artemis,%s,%s,%s",
		strings.ToLower(distribution),
		strings.ToLower(organization),
		strings.ToLower(environment),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	referenceResponse := make([]ArtemisTerraformReferenceResponse, 1)
	err = json.Unmarshal(body, &referenceResponse)
	if err != nil {
		return nil, err
	}

	if len(referenceResponse) == 0 {
		return nil, fmt.Errorf("can't find page in reference `terraform` with tags=%s", tags)
	}
	if referenceResponse[0].Data.ProductID == "" {
		return nil, fmt.Errorf("can't find product_id in reference `terraform` with tags=%s", tags)
	}

	defaultOsVersion, err := GetImageOsVersion(*creds, distribution, version)
	if err != nil {
		return nil, err
	}

	return &ArtemisImageData{
		TerraformReferencePage: TerraformReferencePage{
			OnSupport:        referenceResponse[0].Data.OnSupport,
			ADIntegration:    referenceResponse[0].Data.ADIntegration,
			ProductID:        referenceResponse[0].Data.ProductID,
			DefaultOSVersion: defaultOsVersion,
		},
		BasicTags: BasicTags{
			Distribution: distribution,
			Environment:  environment,
			Organization: organization,
			Version:      version,
		},
		GeoDistribution:      referenceResponse[0].Data.GeoDistribution,
		LatestReleaseVersion: referenceResponse[0].Data.LatestReleaseVersion,
	}, nil
}

type DebeziumConnectorImageData struct {
	TerraformReferencePage
	BasicTags

	GeoDistribution bool `json:"geo_distribution"`
}
type DebeziumConnectorTerraformReferenceResponse struct {
	ReferenceResponse
	Data ArtemisImageData `json:"data"`
}

func GetDebeziumImageData(
	creds *auth.Credentials,
	distribution,
	organization,
	environment,
	version string,
) (*DebeziumConnectorImageData, error) {

	tags := fmt.Sprintf(
		"debezium,%s,%s,%s",
		strings.ToLower(distribution),
		strings.ToLower(organization),
		strings.ToLower(environment),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	referenceResponse := make([]ArtemisTerraformReferenceResponse, 1)
	err = json.Unmarshal(body, &referenceResponse)
	if err != nil {
		return nil, err
	}

	if len(referenceResponse) == 0 {
		return nil, fmt.Errorf("can't find page in reference `terraform` with tags=%s", tags)
	}
	if referenceResponse[0].Data.ProductID == "" {
		return nil, fmt.Errorf("can't find product_id in reference `terraform` with tags=%s", tags)
	}

	defaultOsVersion, err := GetImageOsVersion(*creds, distribution, version)
	if err != nil {
		return nil, err
	}

	return &DebeziumConnectorImageData{
		TerraformReferencePage: TerraformReferencePage{
			OnSupport:        referenceResponse[0].Data.OnSupport,
			ADIntegration:    referenceResponse[0].Data.ADIntegration,
			ProductID:        referenceResponse[0].Data.ProductID,
			DefaultOSVersion: defaultOsVersion,
		},
		BasicTags: BasicTags{
			Distribution: distribution,
			Environment:  environment,
			Organization: organization,
			Version:      defaultOsVersion,
		},

		GeoDistribution: referenceResponse[0].Data.GeoDistribution,
	}, nil
}

type AirflowImageData struct {
	TerraformReferencePage
	BasicTags

	AirflowVersion       string `json:"airflow_version"`
	OnBackup             bool   `json:"on_backup"`
	GeoDistribution      bool   `json:"geo_distribution"`
	LatestReleaseVersion string `json:"latest_release_version"`
}

type AirflowReferenceResponse struct {
	ReferenceResponse
	Data AirflowImageData `json:"data"`
}

func GetAirflowImageData(
	creds *auth.Credentials,
	distribution,
	productType,
	organization,
	environment,
	version string,
) (*AirflowImageData, error) {

	var productTag string

	switch productType {
	case "cluster":
		productTag = "airflow_cluster"
	case "stand-alone":
		productTag = "airflow"
	default:
		return nil, fmt.Errorf("unknow product type %s", productType)
	}

	tags := fmt.Sprintf(
		"%s,%s,%s,%s",
		productTag,
		strings.ToLower(distribution),
		strings.ToLower(organization),
		strings.ToLower(environment),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	terraformOsData := make([]AirflowReferenceResponse, 1)
	err = json.Unmarshal(body, &terraformOsData)
	if err != nil {
		return nil, err
	}

	if len(terraformOsData) == 0 {
		return nil, fmt.Errorf("can't find page in reference `terraform` with tags=%s", tags)
	}
	if terraformOsData[0].Data.ProductID == "" {
		return nil, fmt.Errorf("can't find product_id in reference `terraform` with tags=%s", tags)
	}

	imageOsVersion, err := GetImageOsVersion(*creds, distribution, version)
	if err != nil {
		return nil, err
	}

	return &AirflowImageData{
		TerraformReferencePage: TerraformReferencePage{
			OnSupport:        terraformOsData[0].Data.OnSupport,
			ProductID:        terraformOsData[0].Data.ProductID,
			ADIntegration:    terraformOsData[0].Data.ADIntegration,
			DefaultOSVersion: terraformOsData[0].Data.DefaultOSVersion,
		},
		BasicTags: BasicTags{
			Distribution: distribution,
			Environment:  environment,
			Organization: organization,
			Version:      imageOsVersion,
		},
		AirflowVersion:       terraformOsData[0].Data.AirflowVersion,
		OnBackup:             terraformOsData[0].Data.OnBackup,
		GeoDistribution:      terraformOsData[0].Data.GeoDistribution,
		LatestReleaseVersion: terraformOsData[0].Data.LatestReleaseVersion,
	}, nil
}

type TarantoolDataGridImageData struct {
	TerraformReferencePage
	BasicTags

	TarantoolVersion     string `json:"tarantool_version"`
	GeoDistribution      bool   `json:"geo_distribution"`
	LatestReleaseVersion string `json:"latest_release_version"`
	TarantoolType        string `json:"tarantool_type"`
}

type TarantoolDataGridReferenceResponse struct {
	ReferenceResponse
	Data TarantoolDataGridImageData `json:"data"`
}

func GetTarantoolDataGridImageData(
	creds *auth.Credentials,
	distribution,
	organization,
	environment,
	version string,
) (*TarantoolDataGridImageData, error) {
	tags := fmt.Sprintf(
		"tarantool_tdg,%s,%s,%s",
		strings.ToLower(distribution),
		strings.ToLower(organization),
		strings.ToLower(environment),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	referenceResponse := make([]TarantoolDataGridReferenceResponse, 1)
	err = json.Unmarshal(body, &referenceResponse)
	if err != nil {
		return nil, err
	}

	if len(referenceResponse) == 0 {
		return nil, fmt.Errorf("can't find page in reference `terraform` with tags=%s", tags)
	}
	if referenceResponse[0].Data.ProductID == "" {
		return nil, fmt.Errorf("can't find product_id in reference `terraform` with tags=%s", tags)
	}

	defaultOsVersion, err := GetImageOsVersion(*creds, distribution, version)
	if err != nil {
		return nil, err
	}

	return &TarantoolDataGridImageData{
		TerraformReferencePage: TerraformReferencePage{
			OnSupport:        referenceResponse[0].Data.OnSupport,
			ADIntegration:    referenceResponse[0].Data.ADIntegration,
			ProductID:        referenceResponse[0].Data.ProductID,
			DefaultOSVersion: defaultOsVersion,
		},
		BasicTags: BasicTags{
			Distribution: distribution,
			Environment:  environment,
			Organization: organization,
			Version:      version,
		},
		GeoDistribution:      referenceResponse[0].Data.GeoDistribution,
		LatestReleaseVersion: referenceResponse[0].Data.LatestReleaseVersion,
		TarantoolVersion:     referenceResponse[0].Data.TarantoolVersion,
		TarantoolType:        referenceResponse[0].Data.TarantoolType,
	}, nil
}

type TarantoolEnterpriseImageData struct {
	TerraformReferencePage
	BasicTags

	TarantoolVersion     string `json:"tarantool_version"`
	GeoDistribution      bool   `json:"geo_distribution"`
	LatestReleaseVersion string `json:"latest_release_version"`
	TarantoolType        string `json:"tarantool_type"`
}

type TarantoolEnterpriseReferenceResponse struct {
	ReferenceResponse
	Data TarantoolEnterpriseImageData `json:"data"`
}

func GetTarantoolEnterpriseImageData(
	creds *auth.Credentials,
	distribution,
	organization,
	environment,
	version string,
) (*TarantoolEnterpriseImageData, error) {
	tags := fmt.Sprintf(
		"tarantool_te,%s,%s,%s",
		strings.ToLower(distribution),
		strings.ToLower(organization),
		strings.ToLower(environment),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	referenceResponse := make([]TarantoolEnterpriseReferenceResponse, 1)
	err = json.Unmarshal(body, &referenceResponse)
	if err != nil {
		return nil, err
	}

	if len(referenceResponse) == 0 {
		return nil, fmt.Errorf("can't find page in reference `terraform` with tags=%s", tags)
	}
	if referenceResponse[0].Data.ProductID == "" {
		return nil, fmt.Errorf("can't find product_id in reference `terraform` with tags=%s", tags)
	}

	defaultOsVersion, err := GetImageOsVersion(*creds, distribution, version)
	if err != nil {
		return nil, err
	}

	return &TarantoolEnterpriseImageData{
		TerraformReferencePage: TerraformReferencePage{
			OnSupport:        referenceResponse[0].Data.OnSupport,
			ADIntegration:    referenceResponse[0].Data.ADIntegration,
			ProductID:        referenceResponse[0].Data.ProductID,
			DefaultOSVersion: defaultOsVersion,
		},
		BasicTags: BasicTags{
			Distribution: distribution,
			Environment:  environment,
			Organization: organization,
			Version:      version,
		},
		GeoDistribution:      referenceResponse[0].Data.GeoDistribution,
		LatestReleaseVersion: referenceResponse[0].Data.LatestReleaseVersion,
		TarantoolVersion:     referenceResponse[0].Data.TarantoolVersion,
		TarantoolType:        referenceResponse[0].Data.TarantoolType,
	}, nil
}

type BalancerV3ImageData struct {
	TerraformReferencePage
	BasicTags

	GeoDistribution      bool     `json:"geo_distribution"`
	DefaultLayout        string   `json:"default_layout"`
	LatestReleaseVersion string   `json:"latest_release_version"`
	AllowedVersions      []string `json:"allowed_versions"`
	Products             map[string]struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"products"`
}
type BalancerV3ReferenceResponse struct {
	ReferenceResponse
	Data BalancerV3ImageData `json:"data"`
}

func GetBalancerV3ImageData(
	creds *auth.Credentials,
	distribution,
	organization,
	environment,
	version string,
) (*BalancerV3ImageData, error) {

	tags := fmt.Sprintf(
		"balancer_v3,%s,%s,%s",
		strings.ToLower(distribution),
		strings.ToLower(organization),
		strings.ToLower(environment),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}
	terraformOsData := make([]BalancerV3ReferenceResponse, 1)
	err = json.Unmarshal(body, &terraformOsData)
	if err != nil {
		return nil, err
	}

	if len(terraformOsData) == 0 {
		return nil, fmt.Errorf("can't find page in reference `terraform` with tags=%s", tags)
	}
	if terraformOsData[0].Data.ProductID == "" {
		return nil, fmt.Errorf("can't find product_id in reference `terraform` with tags=%s", tags)
	}

	imageOsVersion, err := GetImageOsVersion(*creds, distribution, version)
	if err != nil {
		return nil, err
	}
	return &BalancerV3ImageData{
		TerraformReferencePage: TerraformReferencePage{
			OnSupport:        terraformOsData[0].Data.OnSupport,
			ProductID:        terraformOsData[0].Data.ProductID,
			ADIntegration:    terraformOsData[0].Data.ADIntegration,
			DefaultOSVersion: terraformOsData[0].Data.DefaultOSVersion,
		},
		BasicTags: BasicTags{
			Distribution: distribution,
			Environment:  environment,
			Organization: organization,
			Version:      imageOsVersion,
		},
		GeoDistribution:      terraformOsData[0].Data.GeoDistribution,
		DefaultLayout:        terraformOsData[0].Data.DefaultLayout,
		LatestReleaseVersion: terraformOsData[0].Data.LatestReleaseVersion,
		AllowedVersions:      terraformOsData[0].Data.AllowedVersions,
		Products:             terraformOsData[0].Data.Products,
	}, nil
}

type ElasticSearchImageData struct {
	TerraformReferencePage
	BasicTags

	GeoDistribution bool `json:"geo_distribution"`
}

type ElasticSearchReferenceResponse struct {
	ReferenceResponse
	Data ElasticSearchImageData `json:"data"`
}

func GetElasticSearchImageData(
	creds *auth.Credentials,
	distribution,
	organization,
	environment,
	version string,
) (*ElasticSearchImageData, error) {

	tags := fmt.Sprintf(
		"elasticsearch_os,%s,%s,%s",
		strings.ToLower(distribution),
		strings.ToLower(organization),
		strings.ToLower(environment),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	terraformOsData := make([]ElasticSearchReferenceResponse, 1)
	err = json.Unmarshal(body, &terraformOsData)
	if err != nil {
		return nil, err
	}

	if len(terraformOsData) == 0 {
		return nil, fmt.Errorf("can't find page in reference `terraform` with tags=%s", tags)
	}

	imageOsVersion, err := GetImageOsVersion(*creds, distribution, version)
	if err != nil {
		return nil, err
	}

	return &ElasticSearchImageData{
		TerraformReferencePage: TerraformReferencePage{
			OnSupport:        terraformOsData[0].Data.OnSupport,
			ProductID:        terraformOsData[0].Data.ProductID,
			ADIntegration:    terraformOsData[0].Data.ADIntegration,
			DefaultOSVersion: terraformOsData[0].Data.DefaultOSVersion,
		},
		BasicTags: BasicTags{
			Distribution: distribution,
			Environment:  environment,
			Organization: organization,
			Version:      imageOsVersion,
		},
		GeoDistribution: terraformOsData[0].Data.GeoDistribution,
	}, nil
}
