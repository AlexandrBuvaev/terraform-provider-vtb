data "vtb_flavor_data" "c2m8" {
  cores  = 4
  memory = 8
}

data "vtb_artemis_image_data" "test" {
  distribution = "astra"
  os_version   = "1.7"
}


data "vtb_cluster_layout" "artemis" {
  layout      = "one_dc:artemis-2:artemis-2"
  net_segment = "dev-srv-app"
}

resource "vtb_artemis_cluster" "test" {
  lifetime        = 2
  label           = "TerraformArtemis1"
  artemis_version = "2.19.1"
  financial_project = "VTB.Cloud"
  core            = data.vtb_core_data.dev
  flavor          = data.vtb_flavor_data.c2m8
  image           = data.vtb_artemis_image_data.test
  layout_id       = data.vtb_cluster_layout.artemis.id
  extra_mounts    = {
    "/app" = {
      size = 248
    },
  }
  access = {
    "superuser" = [
      "cloud-soub-ssh",
    ],
  }
  protocol_amqp = true
}
