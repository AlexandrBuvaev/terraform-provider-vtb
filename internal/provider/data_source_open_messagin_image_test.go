package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccOpenMessagingImgConfig = `
data "vtb_open_messaging_image_data" "img" {
	distribution    = "astra"
	os_version      = "1.7"
}
`

func TestAccReferenceOpenMessagingImgDataSource(t *testing.T) {

	dataSourceName := "data.vtb_open_messaging_image_data.img"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccOpenMessagingImgConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "distribution", "astra"),
					resource.TestCheckResourceAttr(dataSourceName, "os_version", "1.7"),

					resource.TestCheckResourceAttrSet(dataSourceName, "product_id"),
					resource.TestCheckResourceAttrSet(dataSourceName, "ad_integration"),
					resource.TestCheckResourceAttrSet(dataSourceName, "on_support"),
				),
			},
		},
	})
}
