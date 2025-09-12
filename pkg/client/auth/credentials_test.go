package auth

import (
	"log"
	"testing"

	"terraform-provider-vtb/pkg/client/env"
)

func TestNewCredentials(t *testing.T) {

	serviceAcc := env.Load()

	creds, err := NewCredentials(
		serviceAcc.ClientID,
		serviceAcc.ClientSecret,
	)
	if err != nil {
		log.Fatalln(err)
	}

	if creds.AccessToken == "" {
		t.Errorf("Access token is empty")
	}
	log.Println(creds.AccessToken)

}

func TestUpdateToken(t *testing.T) {

	serviceAcc := env.Load()

	creds, err := NewCredentials(
		serviceAcc.ClientID,
		serviceAcc.ClientSecret,
	)
	if err != nil {
		log.Fatalln(err)
	}

	oldToken := creds.AccessToken

	err = creds.UpdateToken()
	if err != nil {
		t.Errorf("Error while updating token: %v", err)
	}

	if creds.AccessToken == oldToken {
		t.Errorf("Access token was not updated")
	}
}
