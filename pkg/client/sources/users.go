package sources

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/requests"
)

type User struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	UniqueName string `json:"unique_name"`
}

type Purpose struct {
	ID          string `json:"id"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	Code        string `json:"code"`
	Postprefix  string `json:"postprefix"`
	Description string `json:"description"`
}

func GetUsersByQuery(creds *auth.Credentials, queryString, projectName, domain string) (users []User, err error) {

	var extendedMode = false

	if strings.Contains(queryString, "@") {
		parts := strings.Split(queryString, "@")
		if len(parts) > 2 {
			return nil, fmt.Errorf(
				"specified query string contains more then one '@' characters. Query string: '%s'",
				queryString,
			)
		}
		extendedMode = true
	}

	params := map[string]string{
		"q":            queryString,
		"domain":       domain,
		"project_name": projectName,
	}
	uri := "portal/api/v1/users"

	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &users)
	if err != nil {
		return nil, err
	}

	if extendedMode {
		for _, u := range users {

			var rawDomainParts []string
			var domainParts []string

			parts := strings.Split(u.UniqueName, ",")
			for _, p := range parts {
				if strings.Contains(p, "DC=") {
					rawDomainParts = append(rawDomainParts, p)
				}
			}

			for _, p := range rawDomainParts {
				splited := strings.Split(strings.TrimSpace(p), "=")
				if len(splited) != 2 {
					return nil, fmt.Errorf("can't getting domain from UniqueName of user")
				}
				domainParts = append(domainParts, splited[1])
			}

			domainFromUnique := strings.Join(domainParts, ".")
			if strings.EqualFold(u.Username, queryString) || strings.EqualFold(u.Username+"@"+domainFromUnique, queryString) {
				return []User{u}, nil
			}

			if strings.EqualFold(u.Email, queryString) {
				return []User{u}, nil
			}
		}
		return nil, fmt.Errorf("can't find user by query string: '%s'", queryString)
	}
	return
}

func GetUsersByGroup(creds *auth.Credentials, groupName, projectName string) (users []User, err error) {

	uri := fmt.Sprintf(
		"portal/api/v1/projects/%s/access_groups/%s/group_users",
		projectName, groupName,
	)
	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &users)
	if err != nil {
		return nil, err
	}
	return
}

func FindUsersByGroupAndQuery(
	creds *auth.Credentials,
	queryString,
	groupName,
	projectName,
	domain string,
) (users []User, err error) {

	group, err := GetAccessGroupByName(creds, projectName, domain, groupName)
	if err != nil {
		return nil, err
	}

	var extendedMode = false
	if strings.Contains(queryString, "@") {
		parts := strings.Split(queryString, "@")
		if len(parts) > 2 {
			return nil, fmt.Errorf(
				"specified query string contains more then one '@' characters. Query string: '%s'",
				queryString,
			)
		}
		extendedMode = true
	}

	params := map[string]string{
		"q":               queryString,
		"domain":          domain,
		"project_name":    projectName,
		"access_group_id": group.ID,
	}
	uri := "portal/api/v1/users"
	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &users)
	if err != nil {
		return nil, err
	}

	if extendedMode {
		for _, u := range users {
			if u.Username == queryString {
				return []User{u}, nil
			}
			if u.Email == queryString {
				return []User{u}, nil
			}
		}
		return nil, fmt.Errorf("can't find user by query string: '%s'", queryString)
	}
	return
}

func GetPurposes(creds *auth.Credentials, queryString, projectName, domain string) ([]Purpose, error) {

	uri := fmt.Sprintf("/portal/api/v1/projects/%s/access_groups/purposes", projectName)
	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var purposes struct {
		List []Purpose `json:"list"`
	}
	err = json.Unmarshal(body, &purposes)
	if err != nil {
		return nil, err
	}
	return purposes.List, nil
}
