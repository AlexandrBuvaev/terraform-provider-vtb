package sources

import (
	"log"
	"testing"

	"terraform-provider-vtb/pkg/client/test"
)

const PROJECT_NAME = "proj-e73127g7ry3p4t4"

func TestGetProcjet(t *testing.T) {

	project, err := GetProject(test.SharedCreds, PROJECT_NAME)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("env_prefix=%v", project.InformationSystem.ShortName)
}

func TestGetFinancialProject(t *testing.T) {
	finProj, err := GetFinancialProjectByName(test.SharedCreds, PROJECT_NAME, "Cервис подключения к BaaS")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("%v", finProj)
}
