package references

import (
	"encoding/json"
	"fmt"
	"strings"
	"terraform-provider-vtb/pkg/client/auth"
)

type GSLBV1ImageData struct {
	TerraformReferencePage
	BasicTags

	GeoDistribution bool     `json:"geo_distribution"`
	AllowedVersions []string `json:"allowed_versions"`
	Products        map[string]struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"products"`
}

type GSLBV1ReferenceResponse struct {
	ReferenceResponse
	Data GSLBV1ImageData `json:"data"`
}

func GetGSLBV1ImageData(
	creds *auth.Credentials,
	distribution,
	organization,
	environment,
	version string,
) (*GSLBV1ImageData, error) {

	tags := fmt.Sprintf(
		"gslb_v1,%s,%s,%s",
		strings.ToLower(distribution),
		strings.ToLower(organization),
		strings.ToLower(environment),
	)

	parameters := map[string]string{"tags__contains": tags}
	body, err := getReferenceData(creds.AccessToken, "terraform", parameters)
	if err != nil {
		return nil, err
	}

	terraformOsData := make([]GSLBV1ReferenceResponse, 1)
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

	return &GSLBV1ImageData{
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
		AllowedVersions: terraformOsData[0].Data.AllowedVersions,
		Products:        terraformOsData[0].Data.Products,
	}, nil
}
