package references

import (
	"log"
	"testing"

	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/test"
)

func TestGetImageID(t *testing.T) {
	img := "tpl_linux_astra_1.7_x86_64_en_20230916"
	imageID, err := GetImageID(*test.SharedCreds, img)
	if err != nil {
		log.Fatalf("Can't fetch image ID from reference serivce: %s", err.Error())
	}
	log.Printf("image=%s; id=%s", img, imageID)
}

func TestGetFlavor(t *testing.T) {
	cores := 4
	memory := 4
	// filter := "flavor:vm:linux:dev"
	flavor, err := GetFlavor(test.SharedCreds, int64(memory), int64(cores))
	if err != nil {
		log.Fatalf("Can't fetch flavor from reference service: %s", err.Error())
	}
	resultFlavor := entities.Flavor{
		Cores:  flavor.Data.Cores,
		Memory: flavor.Data.Memory,
		Name:   flavor.Name,
		UUID:   flavor.ID,
	}
	log.Printf("flavor_name=%v; flavor_id=%v, cpus=%v, memory=%v", resultFlavor.Name, resultFlavor.UUID, resultFlavor.Cores, resultFlavor.Memory)
}

func TestGetImageOsVersion(t *testing.T) {
	osVersion, err := GetImageOsVersion(*test.SharedCreds, "astra", "1.7")
	if err != nil {
		log.Fatalf("Can't fetch imageOsVersion from reference service: %s", err.Error())
	}
	log.Printf("version=%v", osVersion)
}
