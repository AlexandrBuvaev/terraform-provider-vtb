package sources

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/requests"
)

func GetAvailAbilityZones(creds *auth.Credentials, netSegmentCode, organization string) ([]entities.AvailabilityZone, error) {

	params := map[string]string{
		"organization":     organization,
		"net_segment_code": netSegmentCode,
		"with_deleted":     "true",
		"page":             "1",
		"per_page":         "100",
	}
	uri := "order-service/api/v1/availability_zones"
	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var AZResponse struct {
		List []entities.AvailabilityZone `json:"list"`
	}
	err = json.Unmarshal(body, &AZResponse)
	if err != nil {
		return nil, err
	}
	return AZResponse.List, nil
}

func GetDomains(creds *auth.Credentials, netSegmentCode, organization string) ([]entities.Domain, error) {

	params := map[string]string{
		"net_segment_code": netSegmentCode,
		"organization":     organization,
		"page":             "1",
		"per_page":         "100",
	}
	uri := "order-service/api/v1/domains"
	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var DomainsResponse struct {
		List []entities.Domain `json:"list"`
	}
	err = json.Unmarshal(body, &DomainsResponse)
	if err != nil {
		return nil, err
	}
	return DomainsResponse.List, nil
}

func GetDomainsByProjectName(creds *auth.Credentials, projectName string) ([]entities.Domain, error) {

	project, err := GetProject(creds, projectName)
	if err != nil {
		return nil, err
	}

	params := map[string]string{
		"project_name":             projectName,
		"organiztion":              project.Organization,
		"with_deleted":             "false",
		"information_system_check": "false",
		"include":                  "total_count",
		"page":                     "1",
		"per_page":                 "100",
	}
	uri := "order-service/api/v1/domains"
	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var DomainsResponse struct {
		List []entities.Domain `json:"list"`
	}
	err = json.Unmarshal(body, &DomainsResponse)
	if err != nil {
		return nil, err
	}
	return DomainsResponse.List, nil
}

func GetNetSegments(creds *auth.Credentials, projectName string) ([]entities.NetSegment, error) {

	params := map[string]string{
		"project_name": projectName,
		"include":      "total_count",
		"page":         "1",
		"per_page":     "100",
	}
	uri := "order-service/api/v1/net_segments"
	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var NetSegmentsResponse struct {
		List []entities.NetSegment `json:"list"`
	}
	err = json.Unmarshal(body, &NetSegmentsResponse)
	if err != nil {
		return nil, err
	}
	return NetSegmentsResponse.List, nil
}

func GetPlatforms(
	creds *auth.Credentials,
	netSegmentCode,
	organization,
	availabilityZoneCode string,
) ([]entities.Platform, error) {

	params := map[string]string{
		"net_segment_code":       netSegmentCode,
		"availability_zone_code": availabilityZoneCode,
		"organization":           organization,
		"include":                "total_count",
		"page":                   "1",
		"per_page":               "100",
		// "with_restrictions":      "true",
	}
	uri := "order-service/api/v1/platforms"
	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var PlatformsResponse struct {
		List []entities.Platform `json:"list"`
	}
	err = json.Unmarshal(body, &PlatformsResponse)
	if err != nil {
		return nil, err
	}
	return PlatformsResponse.List, nil
}

func GetProject(creds *auth.Credentials, projectName string) (*entities.Project, error) {

	params := map[string]string{
		"include": "information_system,environment_prefix,availability,project_environment",
	}
	uri := "resource-manager/api/v2/projects/" + projectName
	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var ProjectResponse struct {
		Data entities.Project `json:"data"`
	}
	err = json.Unmarshal(body, &ProjectResponse)
	if err != nil {
		return nil, err
	}
	return &ProjectResponse.Data, nil
}

func GetFinancialProjectByName(creds *auth.Credentials, projectName, name string) (*entities.FinancialProject, error) {
	finProjects, err := GetFinancialProjects(creds, projectName, name)
	if err != nil {
		return nil, err
	}

	available := make([]string, 0)
	for _, fp := range finProjects {
		if fp.Name == name {
			return &fp, nil
		}
		available = append(available, fp.Name)
	}
	return nil, fmt.Errorf(
		"can't find financial_project=%s; available=[%s]",
		name,
		strings.Join(available, ", "),
	)
}

func GetFinancialProjectByID(creds *auth.Credentials, projectName, id string) (*entities.FinancialProject, error) {
	finProjects, err := GetFinancialProjects(creds, projectName, "")
	if err != nil {
		return nil, err
	}

	available := make([]string, 0)
	for _, fp := range finProjects {
		if fp.ID == id {
			return &fp, nil
		}
		available = append(available, fp.Code)
	}
	return nil, fmt.Errorf(
		"can't find financial_project=%s in project=%s; available=[%s]",
		id,
		projectName,
		strings.Join(available, ", "),
	)
}

func GetFinancialProjects(creds *auth.Credentials, projectName, finProjectName string) ([]entities.FinancialProject, error) {

	params := map[string]string{
		"include":  "total_count",
		"page":     "1",
		"per_page": "100",
		"status":   "active",
	}

	if finProjectName != "" {
		params["query"] = finProjectName
	}

	uri := fmt.Sprintf("portal/api/v1/projects/%s/financial_projects", projectName)
	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var FinProjResponse struct {
		List []entities.FinancialProject `json:"list"`
	}
	err = json.Unmarshal(body, &FinProjResponse)
	if err != nil {
		return nil, err
	}
	return FinProjResponse.List, nil
}

func GetDataCenters(
	creds *auth.Credentials,
	projectName,
	organization,
	environment,
	netSegmentCode string,
) ([]entities.DataCenter, error) {

	params := map[string]string{
		"project_name":        projectName,
		"net_segment_code":    netSegmentCode,
		"organization":        organization,
		"environment":         environment,
		"available_for_order": "true",
	}
	uri := "order-service/api/v1/data_centers"
	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var DataCentersResponse struct {
		List []entities.DataCenter `json:"list"`
	}
	err = json.Unmarshal(body, &DataCentersResponse)
	if err != nil {
		return nil, err
	}
	return DataCentersResponse.List, nil
}

func GetOrchestrationAgents(creds *auth.Credentials, risID, netSegment string) ([]entities.JenkinsAgentSubsystem, error) {

	params := map[string]string{
		"net_segment": netSegment,
		"ris_id":      risID,
		"platform":    "sfera",
		"item_type":   "node_group",
	}

	uri := "order-service/api/v1/orders/orchestration_agents"
	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var SferaAgents struct {
		Results []entities.JenkinsAgentSubsystem `json:"results"`
	}

	err = json.Unmarshal(body, &SferaAgents)
	if err != nil {
		return nil, err
	}

	return SferaAgents.Results, nil
}
