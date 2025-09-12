package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccClusterLayoutConfig = `
data "vtb_cluster_layout" "rabbitmq" {
	layout      = "one_dc:rabbitmq-2:quorum-1"
	net_segment = "dev-srv-app"
}

data "vtb_cluster_layout" "debezium" {
	layout      = "ts:debezium-1"
	net_segment = "b2b-hce-ts-dev-srv-app"
}

data "vtb_cluster_layout" "artemis" {
	layout      = "geo:artemis-2:artemis-2"
	net_segment = "prod-srv-app"
}

data "vtb_cluster_layout" "kafka" {
	layout      = "one_dc:kafka-6:zookeeper-3"
	net_segment = "test-srv-synt"
}
`

func TestAccReferenceClusterLayoutConfigDataSource(t *testing.T) {

	rabbitmqLayout := "data.vtb_cluster_layout.rabbitmq"
	debeziumLayout := "data.vtb_cluster_layout.debezium"
	artemisLayout := "data.vtb_cluster_layout.artemis"
	kafkaLayout := "data.vtb_cluster_layout.kafka"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccClusterLayoutConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(rabbitmqLayout, "id"),
					resource.TestCheckResourceAttrSet(debeziumLayout, "id"),
					resource.TestCheckResourceAttrSet(artemisLayout, "id"),
					resource.TestCheckResourceAttrSet(kafkaLayout, "id"),
				),
			},
		},
	})
}
