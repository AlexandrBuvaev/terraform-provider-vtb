package references

import (
	"log"
	"terraform-provider-vtb/pkg/client/test"
	"testing"
)

func TestGetGeoDistibutionLayoutNameByID(t *testing.T) {
	product := "artemis"
	layout_id := "ec8dacb8-4750-41ec-8d51-46137b2b518d"
	org := "vtb"
	net_segment := "dev-srv-app"

	layoutName, err := GetGeoDistributionLayoutNameByID(
		test.SharedCreds, product, layout_id, org, net_segment,
	)
	if err != nil {
		log.Fatalf("Can't fetch layout name from reference service: %s", err.Error())
	}
	log.Printf("layout name=%s", layoutName)
}
