package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccDebeziumImageConfig = `
data "vtb_debezium_image_data" "test" {
	distribution       = "astra"
	os_version         = "1.7"
}
`

func TestAccDebeziumImageDataSource(t *testing.T) {

	dataSourceName := "data.vtb_debezium_image_data.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccDebeziumImageConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "distribution", "astradsds"),
					resource.TestCheckResourceAttr(dataSourceName, "os_version", "1.7"),

					resource.TestCheckResourceAttrSet(dataSourceName, "geo_distribution"),
					resource.TestCheckResourceAttrSet(dataSourceName, "product_id"),
					resource.TestCheckResourceAttrSet(dataSourceName, "ad_integration"),
					resource.TestCheckResourceAttrSet(dataSourceName, "on_support"),
				),
			},
		},
	})
}
