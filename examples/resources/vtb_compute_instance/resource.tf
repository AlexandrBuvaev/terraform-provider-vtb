data "vtb_core_data" "dev" {
  platform    = "OpenStack"
  domain      = "corp.dev.vtb"
  net_segment = "dev-srv-app"
  zone        = "msk-north"
}

data "vtb_compute_image_data" "name" {
  distribution = "astra"
  os_version   = "1.7"
}

data "vtb_flavor_data" "c2m4" {
  cores  = 2
  memory = 4
}

resource "vtb_compute_instance" "name" {
  lifetime = 2
  label    = "TerraformComputeAstra1"
  core     = data.vtb_core_data.dev
  flavor   = data.vtb_flavor_data.c2m4
  image    = data.vtb_compute_image_data.name
  financial_project = "VTB.Cloud"
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