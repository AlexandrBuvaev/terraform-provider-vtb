package references

import (
	"encoding/json"
	"fmt"
	"terraform-provider-vtb/pkg/client/auth"
)

func GetBalancerDnsZones(creds *auth.Credentials, net_segment string) ([]string, error) {

	parameters := map[string]string{
		"tags__contains": fmt.Sprintf(
			"%s,available,RELEASE", net_segment,
		),
	}
	body, err := getReferenceData(creds.AccessToken, "gslb_servers", parameters)
	if err != nil {
		return nil, err
	}

	var DNSzones []struct {
		ReferenceResponse
		Data struct {
			Name  string `json:"name"`
		} `json:"data"`
	}
	err = json.Unmarshal(body, &DNSzones)
	if err != nil {
		return nil, err
	}

	var zoneNames []string
	for _, zone := range DNSzones {
		zoneNames = append(zoneNames, zone.Data.Name)
	}
	return zoneNames, err
}
