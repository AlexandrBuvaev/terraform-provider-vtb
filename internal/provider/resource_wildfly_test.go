package provider

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testWildflyResourceConfigRender() string {

	today := time.Now()
	todayFormatted := today.Format("2006/01/02")

	testWildflyResourceConfig := fmt.Sprintf(`
	resource "vtb_wildfly_instance" "test" {
		lifetime         = 2
		label            = "Terraform"
		wildfly_version  = "23.0.2.Final"
		java_version     = "11"
		service_status   = "on"
		update_cert_date = "%s"
		standalone_type  = "full"
	  
		image  = data.vtb_wildfly_image_data.wildfly_img
		core   = data.vtb_core_data.dev
		flavor = data.vtb_flavor_data.c2m4
	
		extra_mounts = {
		  "/app/app" = {
			size        = 60
			file_system = "xfs"
		  },
		  "/app/logs" = {
			size        = 20
			file_system = "xfs"
		  }
		}
		
		access = {
		  "superuser" = [
			"cloud-soub-kafka",
			"cloud-soub-kafkatest"
		  ],
		}
	
		wildfly_access = {
		  "Operator" = [
			"cloud-soub-kafka"
		  ],
		}
	
		balancer = {
		  type = "global"
		  dns  = "oslb-dev-dev.corp.dev.vtb"
		  alt_names = [
			"spidermanaisjsadsa.oslb-dev-dev.corp.dev.vtb",
		  ]
		}
	  }
	`, todayFormatted,
	)

	return testWildflyResourceConfig +
		testAccWildflyImageConfig +
		testAccCoreDataConfig +
		testAccFlavorConfig
}

func TestAccWildflyResource(t *testing.T) {

	resourceName := "vtb_wildfly_instance.test"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testWildflyResourceConfigRender(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "lifetime", "2"),
					resource.TestCheckResourceAttr(resourceName, "label", "Terraform"),
					resource.TestCheckResourceAttr(resourceName, "wildfly_version", "23.0.2.Final"),
					resource.TestCheckResourceAttr(resourceName, "java_version", "11"),
					resource.TestCheckResourceAttr(resourceName, "service_status", "on"),
					resource.TestCheckResourceAttr(resourceName, "standalone_type", "full"),

					resource.TestCheckResourceAttr(resourceName, "balancer.type", "global"),
					resource.TestCheckResourceAttr(resourceName, "balancer.dns", "oslb-dev-dev.corp.dev.vtb"),
					resource.TestCheckResourceAttr(resourceName, "balancer.alt_names.#", "1"),
					resource.TestCheckResourceAttr(
						resourceName,
						"balancer.alt_names.0",
						"spidermanaisjsadsa.oslb-dev-dev.corp.dev.vtb",
					),
				),
			},
		},
	})
}
