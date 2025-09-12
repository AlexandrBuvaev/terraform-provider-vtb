package references

import (
	"log"
	"terraform-provider-vtb/pkg/client/test"
	"testing"
)

func TestGetRabbitMQLatestReleaseVersion(t *testing.T) {
	version := "1.7"
	distribution := "astra"
	org := "vtb"
	environment := "dev"

	imageData, err := GetRabbitMQImageData(
		test.SharedCreds, distribution, org, environment, version,
	)
	if err != nil {
		log.Fatalf("Can't fetch refernce page from reference service: %s", err.Error())
	}
	log.Printf("latest_release_version=%v", imageData.LatestReleaseVersion)
}

func TestGetTarantoolDataGridImageData(t *testing.T) {
	version := "1.7"
	distribution := "astra"
	org := "vtb"
	environment := "dev"

	imageData, err := GetTarantoolDataGridImageData(
		test.SharedCreds, distribution, org, environment, version)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("%#v", imageData)
}
