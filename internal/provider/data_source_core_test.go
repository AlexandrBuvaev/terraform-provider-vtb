package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccCoreDataConfig = `
data "vtb_core_data" "dev" {
	net_segment = "dev-srv-app"
	platform    = "OpenStack"
	domain      = "corp.dev.vtb"
	zone        = "msk-north"
}
`

func TestAccCoreDataSource(t *testing.T) {

	dataSourceName := "data.vtb_core_data.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccCoreDataConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "platform", "OpenStack"),
					resource.TestCheckResourceAttr(dataSourceName, "domain", "corp.dev.vtb"),
					resource.TestCheckResourceAttr(dataSourceName, "net_segment", "dev-srv-app"),
					resource.TestCheckResourceAttr(dataSourceName, "zone", "msk-north"),
				),
			},
		},
	})
}
