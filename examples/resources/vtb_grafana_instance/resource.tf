data "vtb_core_data" "dev" {
  platform    = "OpenStack"
  domain      = "corp.dev.vtb"
  net_segment = "dev-srv-app"
  zone        = "msk-north"
}

data "vtb_flavor_data" "c2m8" {
  cores  = 2
  memory = 8
}

data "vtb_grafana_image_data" "grafana_image" {
  distribution = "astra"
  os_version = "1.7"
}

resource "vtb_grafana_instance" "grafana" {
  grafana_user_name = "terraform_user"
  grafana_user_password = "wxjGFCrkTDpIyN0xtOcmwSQGeLttXYd6FPdKPalwiQ7KV9cTlahx0k1yDH"
  label = "terraform develop grafana"
  flavor = data.vtb_flavor_data.c2m8
  image = data.vtb_grafana_image_data.grafana_image
  extra_mounts = {
    "/app" = {
      size = 45
    }
  }
  financial_project = "VTB.Cloud"
  core = data.vtb_core_data.dev
  access = {
    "user" = [
      "cloud-soub-develop"
    ]
  }
}