data "vtb_core_data" "dev" {
  platform    = "OpenStack"
  domain      = "corp.dev.vtb"
  net_segment = "dev-srv-app"
  zone        = "msk-north"
}

data "vtb_flavor_data" "clickhouse_flavor" {
  cores  = 2
  memory = 4
}

data "vtb_flavor_data" "zookeeper_flavor" {
  cores  = 2
  memory = 4
}

data "vtb_clickhouse_cluster_image_data" "clickhouse_cluster_img" {
  distribution = "astra"
  os_version   = "1.8"
}

resource "vtb_clickhouse_cluster" "clickhouse_cluster_tf" {
  label     = "Terraform Clickhouse CLuster Astra"
  lifetime  = 2
  core      = data.vtb_core_data.dev
  flavor_ch = data.vtb_flavor_data.clickhouse_flavor
  flavor_zk = data.vtb_flavor_data.zookeeper_flavor
  image     = data.vtb_clickhouse_cluster_image_data.clickhouse_cluster_img
  nodes_count = {
    clickhouse = 2
  }
  financial_project = "VTB.Cloud"
  ch_extra_mounts = {
    "/app/clickhouse" = {
      size = 100
    }
  }
  zk_extra_mounts = {
    "/app/zookeeper" = {
      size = 80
    }
  }
  access = {
    "superuser" = [
      "cloud-soub-developers",
    ]
  }
  ch_version                 = "25.3.2.39"
  cluster_name               = "dev"
  ch_customer_admin          = "caaaaa1"
  ch_customer_admin_password = "lTLVycs!$dbq4zgZWN7TF$RI(Y*s4jUKCdudkz^J3j0A8v%lR4nFgy~23wWO@$qNm!ef"
  ch_customer_password       = "SU3Qd1zd7^g%)7hzWVXhoY!WxQmDQtcgJ0%p8GO5b~eipzfuRBs*CzIbqS9zY7H"
  system_adm_groups = {
    "system_adm_groups" = [
      "cloud-soub-developers1",
      "cloud-soub-developers2"
    ]
  }
  clickhouse_user_ad_groups = {
    "clickhouse_user_ad_groups" = [
      "cloud-soub-developers1",
    ]
  }

  clickhouse_app_admin_ad_groups = {
    "clickhouse_app_admin_ad_groups" = [
      "cloud-soub-developers1",
      "cloud-soub-developers2"
    ]
  }
}
