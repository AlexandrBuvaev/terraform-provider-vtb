package references

import (
	"encoding/json"
	"fmt"
	"strings"
	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/entities"
)

type RQaasClusterReferenceResponse struct {
	ReferenceResponse
	Data struct {
		Hosts            []string `json:"hosts"`
		Domain           string   `json:"domain"`
		Platform         string   `json:"platform"`
		NetSegment       string   `json:"net_segment"`
		AvailabilityZone string   `json:"availability_zone"`
	}
}

func GetRQaasCluster(creds *auth.Credentials, env, clusterName string) (*entities.RQaaSCluster, error) {
	parameters := map[string]string{
		"tags_contains": env,
	}

	body, err := getReferenceData(creds.AccessToken, "RQAAS_cluster", parameters)
	if err != nil {
		return nil, err
	}

	var RQaaSClusters []RQaasClusterReferenceResponse

	err = json.Unmarshal(body, &RQaaSClusters)
	if err != nil {
		return nil, err
	}

	availiableClusters := []string{}
	for _, cluster := range RQaaSClusters {
		if cluster.Name == clusterName {
			return &entities.RQaaSCluster{
				ClusterName:      cluster.Name,
				Hosts:            cluster.Data.Hosts,
				Domain:           cluster.Data.Domain,
				NetSegment:       cluster.Data.NetSegment,
				AvailabilityZone: cluster.Data.AvailabilityZone,
				Platform:         cluster.Data.Platform,
			}, nil
		}
		availiableClusters = append(
			availiableClusters,
			fmt.Sprintf("{ClusterName: %v}", cluster.Name),
		)
	}

	return nil, fmt.Errorf(
		"can't find cluster with this name: %v. Avaliable clusters are: [%v]",
		clusterName, strings.Join(availiableClusters, ", "),
	)
}
