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

data "vtb_wildfly_image_data" "img" {
  distribution = "astra"
  os_version   = "1.7"
}

resource "vtb_wildfly_instance" "test" {
  lifetime = 2
  label    = "TerraformWildfly"
  core     = data.vtb_core_data.dev
  flavor   = data.vtb_flavor_data.c2m4
  image    = data.vtb_wildfly_image_data.img
  financial_project = "VTB.Cloud"
  extra_mounts = {
    "/app/app" = {
      size = 60
    },
    "/app/logs" = {
      size = 20
    }
  }
  access = {
    "superuser" = [
      "cloud-soub-chumakovas",
    ],
  }
  wildfly_access = {
    "Operator" = [
      "cloud-soub-chumakovas",
    ],
  }
  wildfly_version = "26.1.3.Final"
  java_version    = "11"
  service_status  = "on"
  standalone_type = "standard"
  cert_alt_names = [
    "d5soub-wfc007lk.corp.dev.vtb",
    "d5soub-wfc009lk.corp.dev.vtb",
  ]
  client_cert = true
  mm_mode_end_date = "2025-03-18 21:00"
}