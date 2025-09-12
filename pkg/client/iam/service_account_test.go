package iam

import (
	"log"
	"testing"

	"terraform-provider-vtb/pkg/client/test"
)

func TestCreateServiceAccount(t *testing.T) {
	serviceAccount := ServiceAccount{
		test.SharedCreds,
		"proj-e73127g7ry3p4t4",
		"buvaev_1",
		[]string{"roles/admin"},
	}
	err := serviceAccount.Create()
	if err != nil {
		log.Fatal(err)
	}
}

func TestGetServiceAccounts(t *testing.T) {
	serviceAccounts, err := GetServiceAccounts(test.SharedCreds, "proj-e73127g7ry3p4t4")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("service_accounts: %v+", serviceAccounts)
}
