package iam

import (
	"encoding/json"
	"fmt"
	"io"
	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/requests"
)

type ServiceAccount struct {
	creds       *auth.Credentials `json:"-"`
	ProjectName string            `json:"project_name"`
	Title       string            `json:"title"`
	Roles       []string          `json:"roles"`
}

type Binding struct {
	Role string `json:"role"`
}

type Policy struct {
	Bindings []Binding `json:"bindings"`
}

type ServiceAccountAttrs struct {
	Title  string `json:"title"`
	Policy Policy `json:"policy"`
}

func NewServiceAccount(
	creds *auth.Credentials, projectName, title string,
) (serviceAccount *ServiceAccount, err error) {
	serviceAccount = &ServiceAccount{
		creds:       creds,
		ProjectName: projectName,
		Title:       title,
	}
	return
}

// Создание сервисного аккаунта
func (s *ServiceAccount) Create() (err error) {
	uri := fmt.Sprintf("iam/api/v1/projects/%s/service_accounts", s.ProjectName)

	data := s.preparePayloadData()

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = requests.SendRequest(s.creds.AccessToken, uri, "POST", payload, nil)
	if err != nil {
		return err
	}

	return nil
}

// Обновление сервисного аккаунта
func (s *ServiceAccount) Update(name string) (err error) {
	uri := fmt.Sprintf("iam/api/v1/projects/%s/service_accounts/%s", s.ProjectName, name)

	data := s.preparePayloadData()

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, err = requests.SendRequest(s.creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	return nil
}

// Удаление сервисного аккаунта
func (s *ServiceAccount) Delete(name string) (err error) {
	uri := fmt.Sprintf("iam/api/v1/projects/%s/service_accounts/%s", s.ProjectName, name)

	_, err = requests.SendRequest(s.creds.AccessToken, uri, "DELETE", nil, nil)
	if err != nil {
		return err
	}

	return nil
}

func (s *ServiceAccount) preparePayloadData() (data map[string]interface{}) {
	bindings := make([]Binding, 0, len(s.Roles))
	for _, role := range s.Roles {
		bindings = append(bindings, Binding{Role: role})
	}

	attrs := ServiceAccountAttrs{
		Title: s.Title,
		Policy: Policy{
			Bindings: bindings,
		},
	}

	data = map[string]interface{}{
		"service_account": attrs,
	}

	return data
}

// Получение сервисного аккаунта по имени
func GetServiceAccountByName(creds *auth.Credentials, project_name, name string) (serviceAccount *entities.ServiceAccount, err error) {
	uri := fmt.Sprintf("iam/api/v1/projects/%s/service_accounts/%s", project_name, name)
	params := map[string]string{"project_name": project_name, "name": name}

	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	type Response struct {
		Data entities.ServiceAccount `json:"data"`
	}

	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}
	return &response.Data, nil
}

// Получение сервисных аккаунтов проекта
func GetServiceAccounts(creds *auth.Credentials, project_name string) ([]entities.ServiceAccount, error) {
	uri := fmt.Sprintf("iam/api/v1/projects/%s/service_accounts", project_name)
	params := map[string]string{"project_name": project_name, "include": "total_count"}

	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, params)
	if err != nil {
		return nil, err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	type Response struct {
		Data []entities.ServiceAccount `json:"data"`
	}
	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}
