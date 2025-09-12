data "vtb_core_data" "dev" {
  platform    = "OpenStack"
  domain      = "corp.dev.vtb"
  net_segment = "dev-srv-app"
  zone        = "msk-north"
}

data "vtb_flavor_data" "c4m16" {
  cores  = 4
  memory = 16
}

data "vtb_rabbitmq_image_data" "img" {
  distribution = "astra"
  os_version   = "1.7"
}

data "vtb_cluster_layout" "rabbitmq" {
  layout      = "one_dc:rabbitmq-2:quorum-1"
  net_segment = "dev-srv-app"
}

resource "vtb_rabbitmq_cluster" "name" {
  lifetime         = 2
  label            = "TerraformRabbit1"
  rabbitmq_version = "3.11.26"
  cluster_name = "test"
  financial_project = "VTB.Cloud"

  core      = data.vtb_core_data.dev
  flavor    = data.vtb_flavor_data.c4m16
  image     = data.vtb_rabbitmq_image_data.img
  layout_id = data.vtb_cluster_layout.rabbitmq.id
  extra_mounts = {
    "/app" = {
      size        = 50
    }
  }
  web_access = {
    admins = []
    managers = [
      "cloud-soub-rabbitmq",
    ]
  }
}
