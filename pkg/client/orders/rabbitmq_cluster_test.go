package orders

import (
	"log"
	"strconv"
	"testing"

	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/test"
)

const rabbitClusterOrderID = "8044a08e-7d26-496b-87c9-ba39975676a1"

// test should running with increased timeout
// go test -run TestRabbitMQOrder -timeout 9999s -short
func TestRabbitMQOrder(t *testing.T) {

	// layout, err := references.GetGeoDistributionLayoutID(
	// 	test.SharedCreds,
	// 	"rabbitmq",
	// 	"one_dc:rabbitmq-2:quorum-1",
	// 	"vtb",
	// 	"dev-srv-app",
	// )
	// if err != nil {
	// 	log.Fatalln(err)
	// }

	// attrs := RabbitMQAttrs{
	// 	ADIntegration: true,
	// 	ADLogonGrants: []entities.ADLogonGrants{
	// 		{
	// 			Role:   "superuser",
	// 			Groups: []string{"cloud-soub-ssh"},
	// 		},
	// 	},
	// 	Flavor: entities.Flavor{
	// 		Cores:  2,
	// 		Memory: 4,
	// 		Name:   "c2m4",
	// 		UUID:   "3dfab004-286c-4f39-9932-baf865756a23",
	// 	},
	// 	DefaultNic: entities.DefaultNic{NetSegment: "dev-srv-app"},
	// 	Platform:   "OpenStack",
	// 	Domain:     "corp.dev.vtb",
	// 	OnSupport:  true,
	// 	OsVersion:  "1.7",

	// 	AvailabilityZone: "msk-north",
	// 	Path:             "/app",
	// 	Size:             50,
	// 	FileSystem:       "xfs",

	// 	ClusterName:      "test-terraform1231",
	// 	LayoutID:         layout,
	// 	RabbitMQVersion:  "3.8.14",
	// 	GeoDistribution:  true,
	// 	ExtraNics:        []interface{}{},
	// 	WebAdmingGroups:  []string{},
	// 	WebManagerGroups: []string{"cloud-soub-ssh"},
	// }

	// order := NewRabbitMQ(
	// 	test.SharedCreds,
	// 	"FIXME",
	// 	"9f40de87-e690-4c38-935d-bb1cd39c8a71",
	// 	attrs,
	// )

	// err = order.Create("test-go-cloud-api-rabbitmq", 2, false)
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	// log.Printf("RabbitMQ cluster was created (order_id=`%v`, status=`%v`)", order.ID, order.Status)

	// if !testing.Short() {
	// 	updateWebAttrs := UpdateWebAccessGroupsAttrs{
	// 		Accept:   true,
	// 		Admins:   []string{"cloud-soub-kafka"},
	// 		Managers: []string{"cloud-soub-kafka"},
	// 	}

	// 	// WebAcessGroups
	// 	log.Printf("Start updating WebAccessGroups")
	// 	err = order.UpdateWebAccessGroups(updateWebAttrs)
	// 	if err != nil {
	// 		log.Fatalln(err)
	// 	}
	// 	log.Print("WebAccessGroups was updated")

	// 	// VHosts
	// 	log.Printf("Start creating VHosts")
	// 	vhosts := []entities.RabbitMQVhost{{Name: "vhost"}}
	// 	err = order.CreateVHosts(vhosts)
	// 	if err != nil {
	// 		log.Fatalln(err)
	// 	}
	// 	log.Print("VHosts was created")

	// 	log.Print("Start deleting VHosts")
	// 	err = order.DeleteVHosts([]string{"vhost"})
	// 	if err != nil {
	// 		log.Fatalln(err)
	// 	}
	// 	log.Printf("VHosts was deleted")

	// 	// users
	// 	users := []RabbitMQUsers{
	// 		{
	// 			RisCode: "1234123",
	// 			Name:    "uniquenes",
	// 		},
	// 		{
	// 			RisCode: "1234123",
	// 			Name:    "uniquenes123",
	// 		},
	// 	}

	// 	// FIXME
	// 	err = order.CreateUsers(&RabbitqMQCreateUserAttrs{})
	// 	if err != nil {
	// 		log.Fatalln(err)
	// 	}

	// 	userToDelete := fmt.Sprintf(
	// 		"%s-rabbitmq-da-client-%s",
	// 		users[0].RisCode, users[0].Name,
	// 	)

	// 	err = order.DeleteUser(userToDelete)
	// 	if err != nil {
	// 		log.Fatalln(err)
	// 	}
	// }

	// // Finish
	// err = order.DeleteTwoLayer(true)
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	// log.Print("RabbitMQ cluster would be deleted async in a while")
}

// func TestGetRabbitMQOrder(t *testing.T) {

// 	order, err := GetRabbitMQOrder(test.SharedCreds, "proj-h1eu89sx40", rabbitClusterOrderID)
// 	if err != nil {
// 		log.Fatalln(err)
// 	}
// 	log.Printf("AZ=%s, DC=%s", order.Attrs.AvailabilityZone, order.Attrs.DataCenter)
// }

func TestAddVhost(t *testing.T) {

	order, err := GetRabbitMQOrder(test.SharedCreds, "proj-h1eu89sx40", rabbitClusterOrderID)
	if err != nil {
		log.Fatalln(err)
	}

	log.Printf("Start creating VHosts")
	vhosts := []entities.RabbitMQVhost{{Name: "vhostbandit"}}
	err = order.CreateVHosts(vhosts)
	if err != nil {
		log.Fatalln(err)
	}
	log.Print("VHosts was created")
}

func TestUpdateVhostAccess(t *testing.T) {

	order, err := GetRabbitMQOrder(test.SharedCreds, "proj-h1eu89sx40", rabbitClusterOrderID)
	if err != nil {
		log.Fatalln(err)
	}

	user := "APD123455-1234-rabbitmq-da-client-test"
	access := []entities.RabbitMQVhostAccess{
		{
			VhostName:   "batman",
			UserName:    user,
			Permissions: []string{"READ"},
		},
	}
	err = order.UpdateVhostAccess(user, access)
	if err != nil {
		log.Fatalln(err)
	}
}

func TestDeleteVhostAccess(t *testing.T) {
	order, err := GetRabbitMQOrder(test.SharedCreds, "proj-e73127g7ry3p4t4", rabbitClusterOrderID)
	if err != nil {
		log.Fatalln(err)
	}

	user := "1234-rbmq-d5-client-buvaev"
	access := entities.RabbitMQVhostAccess{
		UserName:  user,
		VhostName: "vhost1",
	}
	err = order.DeleteVhostAccess(access)
	if err != nil {
		log.Fatalln(err)
	}
}

func TestUpdateVhostAccessMultiply(t *testing.T) {

	order, err := GetRabbitMQOrder(test.SharedCreds, "proj-e73127g7ry3p4t4", rabbitClusterOrderID)
	if err != nil {
		log.Fatal(err)
	}
	user := "1234-rbmq-d5-client-baraev"
	input_permissions := []entities.RabbitMQUserInputPermissions{
		{
			UserName:       user,
			VhostRead:      []string{"vhost1", "vhost2"},
			VhostConfigure: []string{"vhost1", "vhost2"},
			VHostWrite:     []string{"vhost1", "vhost2"},
		},
	}
	err = order.UpdateVhostAccessMultiply(input_permissions)
	if err != nil {
		log.Fatal(err)
	}
}

func TestVerticalScale(t *testing.T) {

	order, err := GetRabbitMQOrder(test.SharedCreds, "proj-e73127g7ry3p4t4", rabbitClusterOrderID)
	if err != nil {
		log.Fatalln(err)
	}

	attrs := RabbitMQVerticalScalingAttrs{
		CheckAgree:    true,
		SelectMode:    "one_vm",
		Fstype:        "xfs",
		Mount:         order.Attrs.Path,
		CurrentFlavor: strconv.Itoa(int(order.Attrs.Size)),
		Size:          50,
		Flavor: entities.Flavor{
			Cores:  2,
			Memory: 8,
			Name:   "c2m8",
			UUID:   "930a8be8-1c71-4730-b881-0633a56b7f2b",
		},
		QuorumExtraMounts: entities.QuorumExtraMounts{
			Mount:  "/app",
			Size:   10,
			FStype: "xfs",
		},
		QuorumFlavor: entities.Flavor{
			Cores:  2,
			Memory: 4,
			Name:   "c2m4",
			UUID:   "3dfab004-286c-4f39-9932-baf865756a23",
		},
	}

	err = order.VerticalScaling(attrs)
	if err != nil {
		log.Fatalln(err)
	}
}

func TestHorizontalScale(t *testing.T) {
	order, err := GetRabbitMQOrder(test.SharedCreds, "proj-e73127g7ry3p4t4", rabbitClusterOrderID)
	if err != nil {
		log.Fatalln(err)
	}

	attrs := RabbitMQGorizontalScalingAttrs{
		CurrentLayout: "rabbitmq-2:quorum-1",
		NetSegment:    "dev-srv-app",
		Layout:        "one_dc:rabbitmq-4:quorum-1",
	}
	err = order.HorizontalScaling(attrs)
	if err != nil {
		log.Fatal(err)
	}
}

func TestUpdateRabbitMQVersion(t *testing.T) {
	order, err := GetRabbitMQOrder(test.SharedCreds, "proj-e73127g7ry3p4t4", rabbitClusterOrderID)
	if err != nil {
		log.Fatalln(err)
	}

	attrs := RabbitMQUpgradeVersionAttrs{
		Maintance:             true,
		TargetRabbitMQVersion: "3.11.26",
	}

	err = order.UpgradeRabbitMQVersion(attrs)
	if err != nil {
		log.Fatalln(err)
	}

}

func TestGetRabbitMQOrder(t *testing.T) {
	orderID := "5b8a153d-d45d-44d7-8d12-c2b107550875"
	projecName := "proj-e73127g7ry3p4t4"
	order, err := GetRabbitMQOrder(test.SharedCreds, projecName, orderID)
	if err != nil {
		log.Fatal(err)
	}
	cluster, err := order.GetParentItem()
	if err != nil {
		log.Fatal(err)
	}
	clusterConfig := cluster.Data.Config.(entities.RabbitMQItemConfig)
	quantity_rabbitmq := clusterConfig.HostsInfo.QuantityRabbitMQ

	log.Printf("quantity_rabbitmq: %+v\n", quantity_rabbitmq)
}

func TestChangeFinancialProject(t *testing.T) {
	order, err := GetRabbitMQOrder(test.SharedCreds, "proj-e73127g7ry3p4t4", rabbitClusterOrderID)
	if err != nil {
		log.Fatal(err)
	}
	finprojId := "ent-3wctgiel4w"
	err = order.ChangeFinancialProject(finprojId)
	if err != nil {
		log.Fatal(err)
	}
}
