package references

import (
	"encoding/json"
	"fmt"
	"strings"

	"terraform-provider-vtb/pkg/client/auth"
)

type WildflyVersionWithSupportedJavaVersions struct {
	WildflyVersion string   `json:"wildfly_version"`
	JavaVersions   []string `json:"java_version"`
}

type WildflyVersionsResponse struct {
	ReferenceResponse
	Data []WildflyVersionWithSupportedJavaVersions `json:"data"`
}

func GetWildflyAndJavaVersions(creds *auth.Credentials) ([]WildflyVersionWithSupportedJavaVersions, error) {

	parameters := map[string]string{"tags__contains": "versions"}
	body, err := getReferenceData(creds.AccessToken, "wildfly", parameters)
	if err != nil {
		return nil, err
	}

	var versionsResponse []WildflyVersionsResponse
	err = json.Unmarshal(body, &versionsResponse)
	if err != nil {
		return nil, err
	}
	return versionsResponse[0].Data, err
}

func GetWildflyManagmentRoles(creds *auth.Credentials, environment string) ([]string, error) {

	tags := fmt.Sprintf("management_roles,%s", strings.ToLower(environment))
	parameters := map[string]string{"tags__contains": tags}

	body, err := getReferenceData(creds.AccessToken, "wildfly", parameters)
	if err != nil {
		return nil, err
	}

	var rolesResponse []struct {
		ReferenceResponse
		Data []string `json:"data"`
	}
	err = json.Unmarshal(body, &rolesResponse)
	if err != nil {
		return nil, err
	}
	return rolesResponse[0].Data, err
}

func GetDnsZonesForGlobalBalanacer(creds *auth.Credentials, environment string) ([]string, error) {

	parameters := map[string]string{"tags": environment}
	body, err := getReferenceData(creds.AccessToken, "gslb_servers", parameters)
	if err != nil {
		return nil, err
	}

	var DNSzones []struct {
		ReferenceResponse
		Data struct {
			Name  string `json:"name"`
			Hosts []struct {
				Hostname    string `json:"hostname"`
				AnsibleHost string `json:"ansible_host"`
			} `json:"hosts"`
			Domain    string `json:"domain"`
			Available bool   `json:"available"`
			Creatable bool   `json:"creatable"`
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
