package references

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"terraform-provider-vtb/pkg/client/auth"
)

type K8sProjectReferenceResponse struct {
	ReferenceResponse
	Data K8sProjectImageData `json:"data"`
}

type K8sClusterReferenceResponseImage struct {
	ReferenceResponse
	Data K8sClusterImageData `json:"data"`
}

type K8sClusterReferenceResponseRatio struct {
	ReferenceResponse
	Data K8sClusterRatioData `json:"data"`
}

type K8sSpaceReferenceResponse struct {
	ReferenceResponse
	Data K8sSpaceImageData `json:"data"`
}

type K8sProjectImageData struct {
	ProductId      string `json:"product_id"`
	DefaultIngress string `json:"default_ingress"`
}

type K8sClusterImageData struct {
	Products map[string]struct {
		Name            string   `json:"name"`
		ProductID       string   `json:"product_id"`
		AllowedVersions []string `json:"allowed_versions"`
	} `json:"products"`
	RegionIngressAddMax         int64                             `json:"region_ingress_add_max"`
	RegionAddNodesMax           int64                             `json:"region_add_nodes_max"`
	IngressAddNodesMax          int64                             `json:"ingress_add_nodes_max"`
	DefaultRegionSize           int64                             `json:"default_region_size"`
	DefaultIngressSize          int64                             `json:"default_ingress_size"`
	DefaultContainerCPURatio    int64                             `json:"default_container_cpu_ratio"`
	DefaultContainerMemoryRatio int64                             `json:"default_container_memory_ratio"`
	ControlPlane                map[string][]K8sClusterCPDefaults `json:"control_plane"`
}

type K8sClusterRatioData struct {
	ContainerCPURatio    int64 `json:"container_cpu_ratio"`
	ContainerMemoryRatio int64 `json:"container_memory_ratio"`
}

type K8sClusterCPDefaults struct {
	Size   int64  `json:"size"`
	Flavor string `json:"flavor"`
	Role   string `json:"role"`
}

type K8sSpaceImageData struct {
	ProductId           string `json:"product_id"`
	RegionIngressAddMax int64  `json:"region_ingress_add_max"`
	RegionAddNodesMax   int64  `json:"region_add_nodes_max"`
	IngressAddNodesMax  int64  `json:"ingress_add_nodes_max"`
}

func GetK8sProjectImageData(creds *auth.Credentials, organization, enviroment string) (*K8sProjectImageData, error) {

	tags := fmt.Sprintf(
		"k8s,project,%s,%s",
		strings.ToLower(organization),
		strings.ToLower(enviroment),
	)

	parameters := map[string]string{"tags": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	terraformOsData := make([]K8sProjectReferenceResponse, 1)
	err = json.Unmarshal(body, &terraformOsData)
	if err != nil {
		return nil, err
	}

	if len(terraformOsData) == 0 {
		return nil, errors.New("can't find data from reference about image")
	}

	if terraformOsData[0].Data.ProductId == "" {
		return nil, errors.New("can't find product with this parameters")
	}

	image := K8sProjectImageData{
		ProductId:      terraformOsData[0].Data.ProductId,
		DefaultIngress: terraformOsData[0].Data.DefaultIngress,
	}
	return &image, nil
}

func GetK8sClusterImageData(creds *auth.Credentials, organization, enviroment string) (*K8sClusterImageData, error) {

	tags := fmt.Sprintf(
		"k8s,cluster,%s,%s",
		strings.ToLower(organization),
		strings.ToLower(enviroment),
	)

	parameters := map[string]string{"tags": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	terraformOsData := make([]K8sClusterReferenceResponseImage, 1)
	err = json.Unmarshal(body, &terraformOsData)
	if err != nil {
		return nil, err
	}

	if len(terraformOsData) == 0 {
		return nil, errors.New("can't find data from reference about image")
	}

	if len(terraformOsData[0].Data.Products) == 0 {
		return nil, errors.New("can't find product with this parameters")
	}

	image := K8sClusterImageData{
		Products:                    terraformOsData[0].Data.Products,
		ControlPlane:                terraformOsData[0].Data.ControlPlane,
		DefaultRegionSize:           terraformOsData[0].Data.DefaultRegionSize,
		DefaultIngressSize:          terraformOsData[0].Data.DefaultIngressSize,
		DefaultContainerCPURatio:    terraformOsData[0].Data.DefaultContainerCPURatio,
		DefaultContainerMemoryRatio: terraformOsData[0].Data.DefaultContainerMemoryRatio,
		RegionAddNodesMax:           terraformOsData[0].Data.RegionAddNodesMax,
		IngressAddNodesMax:          terraformOsData[0].Data.IngressAddNodesMax,
		RegionIngressAddMax:         terraformOsData[0].Data.RegionIngressAddMax,
	}
	return &image, nil
}

func GetK8sClusterRatioData(creds *auth.Credentials, netSegment, version string) (*K8sClusterRatioData, error) {

	KUBERNETES_V1_PREFIX := "kubernetes_v1"
	directoryName := fmt.Sprintf("%s_%s_env_configs", KUBERNETES_V1_PREFIX, version)

	parameters := map[string]string{
		"name": netSegment,
	}

	body, err := getReferenceData(creds.AccessToken, directoryName, parameters)
	if err != nil {
		return nil, err
	}

	referencedata := make([]K8sClusterReferenceResponseRatio, 1)
	err = json.Unmarshal(body, &referencedata)
	if err != nil {
		return nil, err
	}

	if len(referencedata) == 0 {
		return nil, errors.New("can't find data from reference")
	}

	ratio := K8sClusterRatioData{
		ContainerCPURatio:    referencedata[0].Data.ContainerCPURatio,
		ContainerMemoryRatio: referencedata[0].Data.ContainerMemoryRatio,
	}
	return &ratio, nil
}

func GetK8sSpaceImageData(creds *auth.Credentials, organization, enviroment string) (*K8sSpaceImageData, error) {

	tags := fmt.Sprintf(
		"k8s,space,%s,%s",
		strings.ToLower(organization),
		strings.ToLower(enviroment),
	)

	parameters := map[string]string{"tags": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	terraformOsData := make([]K8sSpaceReferenceResponse, 1)
	err = json.Unmarshal(body, &terraformOsData)
	if err != nil {
		return nil, err
	}

	if len(terraformOsData) == 0 {
		return nil, errors.New("can't find data from reference about image")
	}

	if terraformOsData[0].Data.ProductId == "" {
		return nil, errors.New("can't find product with this parameters")
	}

	image := K8sSpaceImageData{
		ProductId:           terraformOsData[0].Data.ProductId,
		RegionAddNodesMax:   terraformOsData[0].Data.RegionAddNodesMax,
		IngressAddNodesMax:  terraformOsData[0].Data.IngressAddNodesMax,
		RegionIngressAddMax: terraformOsData[0].Data.RegionIngressAddMax,
	}
	return &image, nil
}
