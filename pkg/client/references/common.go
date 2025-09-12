package references

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/requests"
)

// ReferenceResponse базовые поля в ответе от reference service
type ReferenceResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Directory string `json:"directory"`
}

type FlavorReferenceResponse struct {
	ReferenceResponse
	Data struct {
		Cores  int64 `json:"cpus"`
		Memory int64 `json:"memory"`
	} `json:"data"`
}

func GetFlavor(creds *auth.Credentials, memory, cores int64) (*FlavorReferenceResponse, error) {

	var flavors []FlavorReferenceResponse
	body, err := getReferenceData(creds.AccessToken, "flavors", nil)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &flavors)
	if err != nil {
		return nil, err
	}

	var availableFlavors []string
	for _, flavor := range flavors {
		if flavor.Data.Cores == cores && flavor.Data.Memory == memory {
			return &flavor, nil
		}
		availableFlavors = append(
			availableFlavors,
			fmt.Sprintf("{'Memory': %v, 'Cores': %v}", flavor.Data.Memory, flavor.Data.Cores),
		)
	}
	return nil, fmt.Errorf(
		"can't find flavor with memory=%v and cores=%v. Available flavors are: [%v]",
		memory, cores, strings.Join(availableFlavors, ", "),
	)
}

func GetFlavorByFilter(creds *auth.Credentials, filters string, memory, cores int64) (*FlavorReferenceResponse, error) {

	var flavors []FlavorReferenceResponse
	body, err := getReferenceDataByFilter(creds.AccessToken, filters)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(body, &flavors)
	if err != nil {
		return nil, err
	}

	var availableFlavors []string
	for _, flavor := range flavors {
		if flavor.Data.Cores == cores && flavor.Data.Memory == memory {
			return &flavor, nil
		}
		availableFlavors = append(
			availableFlavors,
			fmt.Sprintf("{'Memory': %v, 'Cores': %v}", flavor.Data.Memory, flavor.Data.Cores),
		)
	}
	return nil, fmt.Errorf(
		"can't find flavor with memory=%v and cores=%v. Available flavors are: [%v]",
		memory, cores, strings.Join(availableFlavors, ", "),
	)
}

type RolesReferenceResponse struct {
	ReferenceResponse
	Data struct {
		GroupName   string   `json:"group_name"`
		Permissions []string `json:"permissions"`
	} `json:"data"`
}

func GetRoles(creds *auth.Credentials, environment, filter string) ([]RolesReferenceResponse, error) {

	tags := fmt.Sprintf("%s:%s", filter, strings.ToUpper(environment))
	parameters := map[string]string{"tags__contains": tags}

	body, err := getReferenceData(creds.AccessToken, "ldap_linux_acl", parameters)
	if err != nil {
		return nil, err
	}

	var roles []RolesReferenceResponse
	err = json.Unmarshal(body, &roles)
	if err != nil {
		return nil, err
	}
	return roles, err
}

// Получить справочник со всеми страницами по названию его директории
func getReferenceData(AccessToken string, dirName string, params map[string]string) ([]byte, error) {

	if params == nil {
		params = make(map[string]string)
	}
	params["directory__name"] = dirName

	uri := "references/api/v1/pages/"
	resp, err := requests.SendRequest(AccessToken, uri, "GET", nil, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf(
			"request to reference with directory name = '%s', was returned with code %s",
			dirName, fmt.Sprint(resp.StatusCode),
		)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, err
}

// Получить справочник по фильтру
func getReferenceDataByFilter(AccessToken string, filters string) ([]byte, error) {

	params := map[string]string{"page_filter_chain": filters}
	uri := "references/api/v1/pages/"
	resp, err := requests.SendRequest(AccessToken, uri, "GET", nil, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, err
}

func GetImageOsVersion(creds auth.Credentials, distribution, version string) (string, error) {

	parameters := map[string]string{
		"tags__contains":         "general",
		"data__os__distribution": distribution,
		"data__os__version":      version,
	}
	body, err := getReferenceData(creds.AccessToken, "images", parameters)
	if err != nil {
		return "", err
	}

	type imageReferenceResponse struct {
		ReferenceResponse
		Data struct {
			Os struct {
				Version string `json:"version"`
			} `json:"os"`
			Size int `json:"size"`
		} `json:"data"`
	}

	response := make([]imageReferenceResponse, 1)
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}
	if len(response) == 0 {
		return "", fmt.Errorf("can't find page in reference `images` with params=%s", parameters)
	}
	if response[0].Directory == "" {
		return "", fmt.Errorf("dir is empty string for reference `images` with params=%s", parameters)
	}

	return response[0].Data.Os.Version, nil
}

func GetImageID(creds auth.Credentials, imgName string) (string, error) {

	body, err := getReferenceData(creds.AccessToken, "images", nil)
	if err != nil {
		return "", err
	}

	type imageReferenceResponse struct {
		ReferenceResponse
		Data struct {
			Os struct {
				Version string `json:"version"`
			} `json:"os"`
			Size int `json:"size"`
		} `json:"data"`
	}

	response := []imageReferenceResponse{}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", err
	}

	if len(response) == 0 {
		return "", fmt.Errorf("can't find page in reference `images`")
	}

	for _, page := range response {
		if page.Name == imgName {
			return page.ID, nil
		}
	}
	return "", fmt.Errorf("can't find page with name '%s' in reference 'images'", imgName)

}
