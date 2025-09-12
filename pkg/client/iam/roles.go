package iam

import (
	"encoding/json"
	"fmt"
	"io"
	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/requests"
)

type CreateRoleAttrs struct {
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

type UpdateRoleAttrs struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// Получение списка ролей
func GetRoles(creds *auth.Credentials) (response interface{}, err error) {
	uri := "iam/api/v1/roles"
	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, nil)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	return response, nil
}

// Получение списка ролей сервиса
func GetAvailableServiceRoles(creds *auth.Credentials, fullResourceName string) (response interface{}, err error) {
	uri := "iam/api/v1/roles"
	params := map[string]string{
		"ful_resource_name": fullResourceName,
	}

	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, params)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// Получение роли
func GetRoleByName(creds *auth.Credentials, name string) (response interface{}, err error) {
	uri := fmt.Sprintf("iam/api/v1/roles/%s", name)
	params := map[string]string{
		"name": name,
	}

	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, params)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// Получение списка ролей организации
func GetOrganizationRoles(creds *auth.Credentials, organization string) (response interface{}, err error) {
	uri := fmt.Sprintf("iam/api/v1/organizations/%s/roles", organization)
	params := map[string]string{
		"parent_name": organization,
	}

	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, params)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// Создание роли
func CreateRole(creds *auth.Credentials, organization string, attrs CreateRoleAttrs) (err error) {
	uri := fmt.Sprintf("iam/api/v1/organizations/%s/roles", organization)

	data := map[string]interface{}{
		"role": attrs,
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = requests.SendRequest(creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	return nil
}

// Получение роли
func GetOraganizationRoleByName(creds *auth.Credentials, organization, name string) (response interface{}, err error) {
	uri := fmt.Sprintf("iam/api/v1/organizations/%s/roles/%s", organization, name)
	parameters := map[string]string{
		"parent_name": organization,
		"name":        name,
	}

	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, parameters)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// Обновление информации о роли
func UpdateRolePermissions(creds *auth.Credentials, organization, name string, attrs UpdateRoleAttrs) (err error) {
	uri := fmt.Sprintf("iam/api/v1/organizations/%s/roles/%s", organization, name)

	data := map[string]interface{}{
		"role": attrs,
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = requests.SendRequest(creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	return nil
}

// Удаление роли
func DeleteRole(creds *auth.Credentials, organization, name string) (err error) {
	uri := fmt.Sprintf("iam/api/v1/organizations/%s/roles/%s", organization, name)
	params := map[string]string{
		"parent_name": organization,
		"name":        name,
	}

	_, err = requests.SendRequest(creds.AccessToken, uri, "DELETE", nil, params)
	if err != nil {
		return err
	}

	return nil
}
