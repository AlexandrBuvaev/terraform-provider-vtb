package sources

import (
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/requests"
)

func GetKTaaSCluster(
	creds *auth.Credentials,
	projectName, envPrefix, environment, netSegment, clusterName string,
	storageMemory int64,
) (*entities.KTaaSKafkaCluster, error) {

	params := map[string]string{
		"project_name":          projectName,
		"category":              "storage",
		"platform":              "ktaas",
		"env_prefix":            envPrefix,
		"environment":           strings.ToLower(environment),
		"net_segment":           netSegment,
		"quota[storage_memory]": strconv.FormatInt(storageMemory, 10),
	}

	uri := "order-service/api/v1/products/resource_pools"
	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, params)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed with status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var response struct {
		List []entities.KTaaSKafkaCluster `json:"list"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(response.List) == 0 {
		return nil, fmt.Errorf("no clusters found matching the criteria")
	}

	for _, cluster := range response.List {
		if cluster.Data.ResourcePoolName == clusterName {
			return &cluster, nil
		}
	}

	var availableClusters []string
	for _, cluster := range response.List {
		availableClusters = append(availableClusters, cluster.Data.ResourcePoolName)
	}

	return nil, fmt.Errorf(
		"cluster '%s' not found. Available clusters: [%s]",
		clusterName,
		strings.Join(availableClusters, ", "),
	)
}
