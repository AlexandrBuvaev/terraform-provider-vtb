package test

import (
	"log"

	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/env"
)

// SharedCreds is used only for testing purposes
var SharedCreds *auth.Credentials

func init() {

	serviceAcc := env.Load()

	if serviceAcc.ProjectName == "" || serviceAcc.ClientID == "" || serviceAcc.ClientSecret == "" {
		log.Fatalf("Service Account Data has empty value: %v", serviceAcc)
		return
	}

	log.Printf("Service Account initializtion for project: %s", serviceAcc.ProjectName)
	newCreds, err := auth.NewCredentials(
		serviceAcc.ClientID,
		serviceAcc.ClientSecret,
	)
	if err != nil {
		log.Fatalf("Service Account initializtion error: %s", err.Error())
	}
	SharedCreds = newCreds
}
