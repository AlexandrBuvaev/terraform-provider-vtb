package sources

import (
	"log"
	"testing"

	"terraform-provider-vtb/pkg/client/test"
)

func TestGetAvailabilityZones(t *testing.T) {
	zones, err := GetAvailAbilityZones(test.SharedCreds, "dev-srv-app", "vtb")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(zones)
}
