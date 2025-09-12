package sources

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/requests"
	"terraform-provider-vtb/pkg/client/auth"
)

func GetK8sGroups(creds *auth.Credentials, projectName string) ([]entities.AccessGroup, error) {

	params := map[string]string{
		"include":    "total_count",
		"f[purpose]": "compute",
		"page":       "1",
		"per_page":   "100",
	}
	uri := fmt.Sprintf("portal/api/v1/projects/%s/access_groups", projectName)
	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		List []entities.AccessGroup `json:"list"`
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	return response.List, nil

}

func GetK8sRoles(creds *auth.Credentials, projectEnv string) ([]entities.K8sRoles, error) {
	var response []entities.K8sRoles

	params := map[string]string{
		"tags__contains":  projectEnv + ",kubernetes_project",
		"directory__name": "mankube_project_roles",
	}
	uri := "references/api/v1/pages/"
	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func GetK8sProjectEnviroment(creds *auth.Credentials, projectName string) (string, error) {

	params := map[string]string{
		"include":         "project_environment",
		"directory__name": "mankube_project_roles",
	}

	uri := "resource-manager/api/v2/projects/" + projectName
	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, params)
	if err != nil {
		return "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response struct {
		Data entities.Project `json:"data"`
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "None", err
	}
	return response.Data.ProjectEnvironment.Name, err
}

func GetK8sIdCluster(
	AccessToken string,
	projectName string,
	CPU int64,
	memory int64,
	datacenter string,
	netsegment string,
	clusterName string,
) (string, error) {

	params := map[string]string{
		"category":      "container",
		"project_name":  projectName,
		"quota[cpu]":    strconv.FormatInt(CPU, 10),
		"quota[memory]": strconv.FormatInt(memory, 10),
		"data_center":   datacenter,
		"net_segment":   netsegment,
		"product_name":  "kubernetes_project",
		"resource_type": "cluster:kubernetes",
	}
	uri := "order-service/api/v1/products/resource_pools"
	resp, err := requests.SendRequest(AccessToken, uri, "GET", nil, params)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var response struct {
		List []entities.K8sPoolResponse `json:"list"`
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	for i := range response.List {
		if response.List[i].Name == clusterName {
			clusterID := response.List[i].ID
			return clusterID, nil
		}
	}
	return "", nil
}

func GetK8sClusters(
	AccessToken string,
	projectName string,
	CPU int64,
	memory int64,
	datacenter string,
	netsegment string,
) ([]entities.K8sPoolResponse, error) {

	params := map[string]string{
		"category":      "container",
		"project_name":  projectName,
		"quota[cpu]":    strconv.FormatInt(CPU, 10),
		"quota[memory]": strconv.FormatInt(memory, 10),
		"data_center":   datacenter,
		"net_segment":   netsegment,
		"product_name":  "kubernetes_project",
		"resource_type": "cluster:kubernetes",
	}
	uri := "order-service/api/v1/products/resource_pools"

	resp, err := requests.SendRequest(AccessToken, uri, "GET", nil, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var response struct {
		List []entities.K8sPoolResponse `json:"list"`
	}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	return response.List, nil
}
