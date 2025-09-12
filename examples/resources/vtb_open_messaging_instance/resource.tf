data "vtb_core_data" "dev" {
  platform    = "OpenStack"
  domain      = "corp.dev.vtb"
  net_segment = "dev-srv-app"
  zone        = "msk-north"
}

data "vtb_flavor_data" "c2m4" {
  cores  = 2
  memory = 4
}

data "vtb_open_messaging_image_data" "img" {
  distribution = "astra"
  os_version   = "1.7"
}

resource "vtb_open_messaging_instance" "test" {
  lifetime = 2
  label    = "TerraformOM"
  core     = data.vtb_core_data.dev
  flavor   = data.vtb_flavor_data.c2m4
  image    = data.vtb_open_messaging_image_data.img
  financial_project = "VTB.Cloud"
  extra_mounts = {
    "/app" = {
      size        = 30
    }
  }
  user_groups = ["cloud-soub-test1"]
  admin_groups = ["cloud-soub-test2"]
  superuser_groups = ["cloud-soub-test3"]
}
