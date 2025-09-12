package orders

import (
	"testing"
)

func TestArtemisOrder(t *testing.T) {

	// var extraMounts []entities.ExtraMount
	// extraMounts = append(extraMounts, entities.ExtraMount{
	// 	Path:       "/app",
	// 	Size:       30,
	// 	FileSystem: "xfs",
	// })

	// // FIXME
	// attrs := ArtemisClusterAttrs{
	// BasicAttrs: BasicAttrs{
	// 	ADIntegration: true,
	// 	ADLogonGrants: []entities.ADLogonGrants{
	// 		{
	// 			Role:   "superuser",
	// 			Groups: []string{"cloud-soub-kafkatest"},
	// 		},
	// 	},
	// 	AvailabilityZone: "msk-north",
	// 	Domain:           "corp.dev.vtb",
	// 	DefaultNic:       entities.DefaultNic{NetSegment: "dev-srv-app"},
	// 	Flavor: entities.VMFlavor{
	// 		Cores:  4,
	// 		Memory: 8,
	// 		Name:   "c4m8",
	// 		UUID:   "1498c9c9-5f20-4c1e-8d53-683bb8c2c205",
	// 	},
	// 	ExtraMounts: extraMounts,
	// },
	// ArtemisVersion:   "2.19.1",
	// ClusterName:      "2.19.1",
	// LayoutID:         "8b0594ca-e239-45f9-bb0d-60233b1dc3f4",
	// GeoDistribution:  true,
	// TypeLoadBalancer: "ON_DEMAND",
}

// order := NewArtemisOrder(
// 	test.SharedCreds,
// 	"proj-h1eu89sx40",
// 	"066230da-3cbc-48e7-83c0-e3a48b4b5df9",
// 	attrs,
// )
// err := order.Create("testing", 2, false)
// if err != nil {
// 	log.Fatal(err)
// }
// }
