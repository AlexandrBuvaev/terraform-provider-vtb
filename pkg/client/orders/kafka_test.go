package orders

import (
	"log"
	"testing"

	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/test"
)

const kafkaClusterOrderID = "76747987-fe7b-4e3e-9fee-dfc621afcf1c"

// test should running with increased timeout
// go test -run TestKafkaQuotasOrder -timeout 9999s
func TestKafkaQuotasOrder(t *testing.T) {

	order, err := GetKafkaOrder(test.SharedCreds, "proj-h1eu89sx40", kafkaClusterOrderID)
	if err != nil {
		log.Fatalln(err)
	}
	log.Printf("Kafka cluster was created (order_id=`%v`, status=`%v`)", order.ID, order.Status)

	quotas := []KafkaQuotasBulkAction{
		{
			QuotaType:        "personal",
			ClientCNs:        []string{"mypersonalquotas"},
			ProducerByteRate: 131072,
		},
	}

	log.Printf("Start creating quotas")
	err = order.CreateOrUpdateQuotas(quotas, false)
	if err != nil {
		log.Fatalln(err)
	}
	log.Print("Quotas created was updated")

}

func TestKafkaChangeFlavor(t *testing.T) {
	order, err1 := GetKafkaOrder(test.SharedCreds, "proj-e73127g7ry3p4t4", kafkaClusterOrderID)
	if err1 != nil {
		log.Fatalln(err1)
	}
	log.Printf("Kafka flavor was changed (order_id=`%v`, status=`%v`)", order.ID, order.Status)
	test_flavor := entities.Flavor{
		Cores:  4,
		Memory: 8,
		Name:   "c4m8",
		UUID:   "1498c9c9-5f20-4c1e-8d53-683bb8c2c205",
	}
	err2 := order.ChangeFlavor(test_flavor)
	if err2 != nil {
		log.Fatal(err2)
	}
	log.Println("ChangeFlavor completed!")

}
