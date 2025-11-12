data "vtb_core_data" "dev" {
  platform    = "OpenStack"
  domain      = "corp.dev.vtb"
  zone        = "msk-north"
  net_segment = "dev-srv-app"
}

data "vtb_flavor_data" "gslb_v1_flavor" {
  cores  = 2
  memory = 4
}

data "vtb_gslb_v1_cluster_image_data" "gslb_v1_image" {
  os_version      = "1.7"
  distribution    = "astra"
  product_version = "gslb_cluster_v1_5"
}

data "vtb_cluster_layout" "gslb_v1_layout" {
  layout = "gslb-4"
}

resource "vtb_gslb_v1_cluster" "gslb_cluster_blue_dev" {
  label             = "GSLB v1.5 tf"
  core              = data.vtb_core_data.dev
  layout            = data.vtb_cluster_layout.gslb_v1_layout.id
  flavor            = data.vtb_flavor_data.gslb_v1_flavor
  image             = data.vtb_gslb_v1_cluster_image_data.gslb_v1_image
  financial_project = "VTB.Cloud"
  extra_mounts = {
    "/app" = {
      size = 60
    }
  }
  access = {
    "user" = ["cloud-soub-xxx"]
  }

  desired_version = "latest"
  dns_zone        = "test-tofu"
  api_password    = "xxxxxxxxxxx"
  nginx_password  = "xxxxxxxxxxx"
}

