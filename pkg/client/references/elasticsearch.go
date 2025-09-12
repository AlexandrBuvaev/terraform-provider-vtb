package references

import (
	"encoding/json"
	"fmt"
	"strings"
	"terraform-provider-vtb/pkg/client/auth"
)

type ElasticSearchNodesReferenceResponse struct {
	ReferenceResponse
	Data struct {
		Data        int64 `json:"data"`
		Master      int64 `json:"master"`
		Coordinator int64 `json:"coordinator"`
	}
}

func GetElasticSearchNodeCounts(creds *auth.Credentials, data, master, coordinator int64) (*ElasticSearchNodesReferenceResponse, error) {

	var elastiSearchNodes []ElasticSearchNodesReferenceResponse
	body, err := getReferenceData(creds.AccessToken, "elastic_nodes", nil)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &elastiSearchNodes)
	if err != nil {
		return nil, err
	}

	var availableNodes []string
	for _, nodesCount := range elastiSearchNodes {
		if nodesCount.Data.Data == data && nodesCount.Data.Master == master && nodesCount.Data.Coordinator == coordinator {
			return &nodesCount, nil
		}
		availableNodes = append(
			availableNodes,
			fmt.Sprintf("{'Data': %v, 'Master': %v, 'Coordinator': %v}", nodesCount.Data.Data, nodesCount.Data.Master, nodesCount.Data.Coordinator),
		)
	}
	return nil, fmt.Errorf(
		"can't find nodes_count with Data=%v and Master=%v and Coordinator=%v. Available nodes_counts are: [%v]",
		data, master, coordinator, strings.Join(availableNodes, ", "),
	)
}
