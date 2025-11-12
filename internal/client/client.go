package client

import (
	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/entities"
)

type CloudClient struct {
	ProjectName     string
	Organization    string
	Environment     string
	EnvironmentName string
	EnvID           string
	RisCode         string
	RisID           string
	EnvPrefix       string
	RisShortName    string
	Creds           *auth.Credentials
}

func NewCloudClient(creds *auth.Credentials, project *entities.Project) *CloudClient {
	return &CloudClient{
		Creds:           creds,
		Organization:    project.Organization,
		Environment:     project.ProjectEnvironment.EnvironmentType,
		EnvironmentName: project.ProjectEnvironment.Name,
		EnvPrefix:       project.EnvironmentPrefix.Name,
		EnvID:           project.ProjectEnvironment.ID,
		RisCode:         project.InformationSystem.Code,
		RisShortName:    project.InformationSystem.ShortName,
		RisID:           project.InformationSystem.RisID,
		ProjectName:     project.Name,
	}

}
