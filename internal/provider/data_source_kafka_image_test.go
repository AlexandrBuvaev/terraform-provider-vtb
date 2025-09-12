package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccKafkaImageConfig = `
data "vtb_kafka_image_data" "test" {
	distribution    = "astra"
	os_version         = "1.7"
}
`

func TestAccReferenceKafkaImageDataSource(t *testing.T) {

	dataSourceName := "data.vtb_kafka_image_data.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccKafkaImageConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "distribution", "astra"),
					resource.TestCheckResourceAttr(dataSourceName, "os_version", "1.7"),
					resource.TestCheckResourceAttr(dataSourceName, "default_kafka_version", "2.13-2.4.1"),

					resource.TestCheckResourceAttrSet(dataSourceName, "geo_distribution"),
					resource.TestCheckResourceAttrSet(dataSourceName, "product_id"),
					resource.TestCheckResourceAttrSet(dataSourceName, "ad_integration"),
					resource.TestCheckResourceAttrSet(dataSourceName, "on_support"),
				),
			},
		},
	})
}
