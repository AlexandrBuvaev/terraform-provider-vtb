package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccFlavorConfig = `
data "vtb_flavor_data" "c2m4" {
	cores  = 2
	memory = 4
}
`

func TestAccFlavorDataSource(t *testing.T) {

	dataSourceName := "data.vtb_flavor_data.c2m4"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccFlavorConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "cores", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "memory", "4"),

					resource.TestCheckResourceAttrSet(dataSourceName, "uuid"),
					resource.TestCheckResourceAttrSet(dataSourceName, "name"),
				),
			},
		},
	})
}
