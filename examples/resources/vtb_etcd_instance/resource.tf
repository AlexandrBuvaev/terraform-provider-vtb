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

data "vtb_etcd_image_data" "etcd_image" {
  distribution = "astra"
  os_version = "1.7"
  nodes_count = 3
}

resource "vtb_etcd_instance" "etcd" {
  etcd_user_name = "demo"
  etcd_user_password = "utcXltlPxEYzS17HFwLVXEWwi8IGVD"
  label = "terraform etcd develop"
  flavor = data.vtb_flavor_data.c4m16
  image = data.vtb_etcd_image_data.etcd_image
  cluster_name = "demo"
  extra_mounts = {
    "/app/etcd/" = {
      size = 50
    }
    "/app/logs/" = {
      size = 30
    }
    "/app/backup/" = {
      size = 50
    }
  }
  financial_project = "VTB.Cloud"
  core = data.vtb_core_data.dev
}