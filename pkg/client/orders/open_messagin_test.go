package orders

// test should running with increased timeout
// go test -run TestRabbitMQOrder -timeout 9999s
// func TestOpenMessaginOrder(t *testing.T) {

// 	var extraMounts []entities.ExtraMount
// 	extraMounts = append(extraMounts, entities.ExtraMount{
// 		Path:       "/app",
// 		Size:       30,
// 		FileSystem: "xfs",
// 	})

// 	basicAttrs := OpenMessagingAttrs{
// 		ExtraNics: []interface{}{},
// 		BasicAttrs: BasicAttrs{
// 			ADIntegration: true,
// 			ADLogonGrants: []entities.ADLogonGrants{
// 				{
// 					Role:   "superuser",
// 					Groups: []string{"cloud-soub-kafkatest"},
// 				},
// 			},
// 			Flavor: entities.Flavor{
// 				Cores:  2,
// 				Memory: 4,
// 				Name:   "c2m4",
// 				UUID:   "3dfab004-286c-4f39-9932-baf865756a23",
// 			},
// 			DefaultNic:       entities.DefaultNic{NetSegment: "dev-srv-app"},
// 			Platform:         "OpenStack",
// 			Domain:           "corp.dev.vtb",
// 			ExtraMounts:      extraMounts,
// 			OnSupport:        true,
// 			OsVersion:        "1.7",
// 			AvailabilityZone: "msk-north",
// 		},
// 	}

// 	order := NewOpenMessaging(
// 		test.SharedCreds,
// 		"proj-h1eu89sx40",
// 		"fdf648ca-c92f-410f-a791-61aeffee2ce4",
// 		basicAttrs,
// 	)
// 	err := order.Create("testing", 2, false)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	log.Printf("RabbitMQ cluster was created (order_id=`%v`, status=`%v`)", order.ID, order.Status)
// }
