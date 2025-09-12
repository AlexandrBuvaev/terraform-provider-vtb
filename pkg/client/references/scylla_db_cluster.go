package references

import (
	"encoding/json"
	"fmt"
	"strings"

	"terraform-provider-vtb/pkg/client/auth"
)

type ScyllaDbClusterImageData struct {
	BasicTags
	TerraformReferencePage
}
type ScyllaDbClusterImageResponse struct {
	ReferenceResponse
	Data ScyllaDbClusterImageData `json:"data"`
}

func GetScyllaDbClusterImageData(
	creds *auth.Credentials,
	distribution,
	organization,
	environment,
	version string,
) (*ScyllaDbClusterImageData, error) {

	tags := fmt.Sprintf(
		"scylladb,%s,%s,%s",
		strings.ToLower(distribution),
		strings.ToLower(organization),
		strings.ToLower(environment),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	terraformOsData := make([]ScyllaDbClusterImageResponse, 1)
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

	return &ScyllaDbClusterImageData{
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
