package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccKafkaLayoutConfig = `
data "vtb_cluster_layout" "kafka" {
	layout      = "one_dc:kafka:zookeeper"
	net_segment = "dev-srv-app"
}`

const testAccKafkaInstanceConfig = `
resource "vtb_kafka_instance" "test" {
  lifetime      = 2
  label         = "TerraformKafka"
  kafka_version = "2.13-2.4.1"
  core          = data.vtb_core_data.dev
  flavor        = data.vtb_flavor_data.c2m4
  image         = data.vtb_kafka_image_data.test
  layout_id     = data.vtb_cluster_layout.kafka.id
  
  extra_mounts = {
    "/app" = {
      size        = 25
    }
  }

  access = {
    "kafka_admin" = [
      "cloud-soub-ssh",
    ]
  }

   topics = {
    "test-1" = {
      cleanup_policy  = "delete,compact"
      partitions      = 1
      retention_ms    = 7776000000
    }
    "test-4" = {
      cleanup_policy  = "delete,compact"
      partitions      = 5
      retention_ms    = 1900000
    }
    "test-3" = {
      cleanup_policy = "compact"
      partitions     = 10
    }
  }

  acls = {
	"APD1" = {
		allow_idempotent      = false 
		transactional_by_name = ["test-xxx"]
	},
    "APD2" = {
      allow_idempotent      = false 
      consumer_by_name      = ["consumer-1"]
      producer_by_name      = []
      consumer_by_mask      = []
      producer_by_mask      = []
      transactional_by_name = []
      transactional_by_mask = []
    },
    "APD5" = {
      allow_idempotent      = false
      consumer_by_name      = []
      producer_by_name      = ["consumer-2"]
      consumer_by_mask      = []
      producer_by_mask      = ["test-5"]
      transactional_by_name = []
      transactional_by_mask = []
    }
  }
}

` + testAccKafkaImageConfig +
	testAccFlavorConfig +
	testAccCoreDataConfig +
	testAccKafkaLayoutConfig

func TestAccKafkaInstanceResource(t *testing.T) {

	resourceName := "vtb_compute_instance.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccKafkaInstanceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "order_id"),
					resource.TestCheckResourceAttrSet(resourceName, "item_id"),
					resource.TestCheckResourceAttr(resourceName, "lifetime", "2"),
					resource.TestCheckResourceAttr(resourceName, "label", "TerraformKafka"),

					resource.TestCheckResourceAttr(resourceName, "acls.#", "3"),
					resource.TestCheckResourceAttr(resourceName, "topics.#", "3"),
				),
			},
		},
	})
}
