package sources

import (
	"encoding/json"
	"fmt"
	"io"

	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/requests"
)

type AccessGroup struct {
	creds        *auth.Credentials `json:"-"`
	ID           string            `json:"id"`
	CreatedAt    string            `json:"created_at"`
	UpdatedAt    string            `json:"updated_at"`
	Name         string            `json:"name"`
	ProjectName  string            `json:"project_name"`
	Description  string            `json:"description"`
	Creator      string            `json:"creator"`
	IsDeleted    bool              `json:"is_deleted"`
	Domain       string            `json:"domain"`
	GroupDn      string            `json:"group_dn"`
	AccountsType string            `json:"accounts_type"`
	Purpose      Purpose           `json:"purpose"`
	Users        []User            `json:"-"`
}

type AccessGroupsResponse struct {
	List []AccessGroup `json:"list"`
}

func NewAccessGroup(
	creds *auth.Credentials,
	project *entities.Project,
	name, domain, description string,
) (group *AccessGroup, err error) {
	group = &AccessGroup{
		creds:       creds,
		Name:        name,
		ProjectName: project.Name,
		Domain:      domain,
		Description: description,
	}
	return
}

func GetAccessGroups(creds *auth.Credentials, projectName, domain string) (group []AccessGroup, err error) {

	params := map[string]string{
		"include":  "total_count",
		"page":     "1",
		"per_page": "100",
		"domain":   domain,
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

	var response AccessGroupsResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	for index := range response.List {
		response.List[index].creds = creds
		users, err := response.List[index].getUsers()
		if err != nil {
			return nil, err
		}
		response.List[index].Users = users
	}
	return response.List, nil
}

func GetAccessGroupsByAccountsType(creds *auth.Credentials, projectName, accountsType string) (group []AccessGroup, err error) {

	if accountsType != "personal" && accountsType != "service-accounts" {
		return nil, fmt.Errorf("wrong acoounts type: %s", accountsType)
	}
	params := map[string]string{
		"include":          "total_count",
		"f[purpose]":       "compute",
		"f[accounts_type]": accountsType,
		"page":             "1",
		"per_page":         "100",
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

	var response AccessGroupsResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	for index := range response.List {
		response.List[index].creds = creds
		users, err := response.List[index].getUsers()
		if err != nil {
			return nil, err
		}
		response.List[index].Users = users
	}
	return response.List, nil
}

func GetAccessGroupByName(creds *auth.Credentials, projectName, domain, groupName string) (AccessGroup, error) {

	groups, err := GetAccessGroups(creds, projectName, domain)
	if err != nil {
		return AccessGroup{}, err
	}
	for _, g := range groups {
		if g.Name == groupName {
			return g, nil
		}
	}
	return AccessGroup{}, fmt.Errorf(
		"can't find group with name '%s' at project '%s'", groupName, projectName,
	)
}

func (g *AccessGroup) Create(purpose, accountsType string) (err error) {

	data := map[string]interface{}{
		"access_group": map[string]interface{}{
			"code_purpose":  purpose,
			"accounts_type": accountsType,
			"name":          g.Name,
			"description":   g.Description,
			"domain":        g.Domain,
			"project_name":  g.ProjectName,
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := fmt.Sprintf("portal/api/v1/projects/%s/access_groups", g.ProjectName)
	resp, err := requests.SendRequest(g.creds.AccessToken, uri, "POST", payload, nil)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &g)
	return
}

func (g *AccessGroup) Delete() (err error) {
	uri := fmt.Sprintf("portal/api/v1/projects/%s/access_groups/%s", g.ProjectName, g.Name)
	resp, err := requests.SendRequest(g.creds.AccessToken, uri, "DELETE", nil, nil)
	if err != nil {
		return
	}
	resp.Body.Close()
	return
}

func (g *AccessGroup) AddUsers(usersUniqueName []string) (err error) {

	data := map[string]interface{}{"users": usersUniqueName}
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}

	uri := fmt.Sprintf("portal/api/v1/projects/%s/access_groups/%s/group_users", g.ProjectName, g.Name)
	resp, err := requests.SendRequest(g.creds.AccessToken, uri, "POST", payload, nil)
	if err != nil {
		return
	}
	resp.Body.Close()
	return
}

func (g *AccessGroup) AddUserByQuery(queryString string) (foundUser User, err error) {

	users, err := GetUsersByQuery(g.creds, queryString, g.ProjectName, g.Domain)
	if err != nil {
		return
	}

	if len(users) > 1 {
		err = fmt.Errorf(
			"for your request found more then 1 user. Please specify more details in your query",
		)
		return
	}

	if len(users) == 0 {
		err = fmt.Errorf(
			"can't find user by your query '%s'. Please specify another querystring", queryString,
		)
		return
	}

	foundUser = users[0]
	data := map[string]interface{}{
		"users": []string{foundUser.UniqueName},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	uri := fmt.Sprintf("portal/api/v1/projects/%s/access_groups/%s/group_users", g.ProjectName, g.Name)
	resp, err := requests.SendRequest(g.creds.AccessToken, uri, "POST", payload, nil)
	resp.Body.Close()
	return
}

func (g *AccessGroup) RemoveUsers(usersUniqueNames []string) (err error) {

	for _, uniqueName := range usersUniqueNames {

		data := map[string]interface{}{
			"unique_name": uniqueName,
		}
		payload, err := json.Marshal(data)
		if err != nil {
			return err
		}

		uri := fmt.Sprintf("portal/api/v1/projects/%s/access_groups/%s/group_users", g.ProjectName, g.Name)
		resp, err := requests.SendRequest(g.creds.AccessToken, uri, "DELETE", payload, nil)
		resp.Body.Close()
		if err != nil {
			return err
		}
	}
	return
}

func (g *AccessGroup) RemoveUserByQuery(queryString string) (err error) {

	users, err := GetUsersByQuery(g.creds, queryString, g.ProjectName, g.Domain)

	if err != nil {
		return
	}
	if len(users) > 1 {
		err = fmt.Errorf(
			"for your request found more then 1 user. Please specify more details in your query",
		)
		return
	}
	if len(users) == 0 {
		err = fmt.Errorf(
			"can't find user by your query '%s'. Please specify another querystring", queryString,
		)
		return
	}

	data := map[string]interface{}{
		"unique_name": users[0].UniqueName,
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := fmt.Sprintf("portal/api/v1/projects/%s/access_groups/%s/group_users", g.ProjectName, g.Name)
	resp, err := requests.SendRequest(g.creds.AccessToken, uri, "DELETE", payload, nil)
	resp.Body.Close()
	return
}

func (g *AccessGroup) EditDescription(description string) (err error) {

	data := map[string]interface{}{
		"access_group": map[string]interface{}{
			"description": description,
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := fmt.Sprintf("portal/api/v1/projects/%s/access_groups/%s", g.ProjectName, g.Name)
	resp, err := requests.SendRequest(g.creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var response AccessGroupsResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return
	}
	return
}

func (g *AccessGroup) Sync() (err error) {

	uri := fmt.Sprintf("portal/api/v1/projects/%s/access_groups/%s", g.ProjectName, g.Name)
	resp, err := requests.SendRequest(g.creds.AccessToken, uri, "GET", nil, nil)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &g)
	if err != nil {
		return
	}

	users, err := g.getUsers()
	if err != nil {
		return
	}
	g.Users = users
	return
}

func (g *AccessGroup) getUsers() (users []User, err error) {

	uri := fmt.Sprintf("portal/api/v1/projects/%s/access_groups/%s/group_users", g.ProjectName, g.Name)
	resp, err := requests.SendRequest(g.creds.AccessToken, uri, "GET", nil, nil)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(body, &users)
	if err != nil {
		return
	}
	return
}
