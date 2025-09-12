package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testAccComputeConfig = `
resource "vtb_compute_instance" "test" {
	lifetime = 2
	label    = "TerraformComputeAstra1"
	core     = data.vtb_core_data.dev
	flavor   = data.vtb_flavor_data.c2m4
	image    = data.vtb_compute_image_data.test
	extra_mounts = {
	  "/app" = {
		size        = 10
	  }
	}
	access = {
	  "superuser" = [
		"cloud-soub-ssh",
	  ],
	}
}
` + testAccAstraLinuxImageConfig +
	testAccFlavorConfig +
	testAccCoreDataConfig

func TestAccComputeResource(t *testing.T) {

	resourceName := "vtb_compute_instance.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccComputeConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "order_id"),
					resource.TestCheckResourceAttrSet(resourceName, "item_id"),
					resource.TestCheckResourceAttrSet(resourceName, "hostname"),
					resource.TestCheckResourceAttrSet(resourceName, "fixed_ip"),
					resource.TestCheckResourceAttr(resourceName, "lifetime", "2"),
					resource.TestCheckResourceAttr(resourceName, "label", "TerraformComputeAstra1"),
				),
			},
		},
	})
}
