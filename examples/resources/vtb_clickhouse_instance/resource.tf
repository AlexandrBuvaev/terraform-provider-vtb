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

data "vtb_clickhouse_image_data" "clickhouse_img" {
  distribution = "astra"
  os_version   = "1.8"
}

resource "vtb_clickhouse_instance" "clickhouse_tf" {
  label             = "Terraform Clickhouse Astra"
  lifetime          = 2
  core              = data.vtb_core_data.dev
  flavor            = data.vtb_flavor_data.c2m4
  image             = data.vtb_clickhouse_image_data.clickhouse_img
  financial_project = "VTB.Cloud"
  extra_mounts = {
    "/app/clickhouse" = {
      size = 20
    }
  }
  access = {
    "superuser" = [
      "cloud-soub-developers",
    ]
  }
  ch_version           = "25.3.2.39"
  clickhouse_user      = "caaaaa1"
  clickhouse_password  = "lTLVycs!$dbq4zgZWN7TF$RI(Y*s4jUKCdudkz^J3j0A8v%lR4nFgy~23wWO@$qNm!ef"
  ch_customer_password = "SU3Qd1zd7^g%)7hzWVXhoY!WxQmDQtcgJ0%p8GO5b~eipzfuRBs*CzIbqS9zY7H"
  system_adm_groups = {
    "system_adm_groups" = [
      "cloud-soub-developers1",
      "cloud-soub-developers2"
    ]
  }
  clickhouse_user_ad_groups = {
    "clickhouse_user_ad_groups" = [
      "cloud-soub-developers1",
      "cloud-soub-developers2"
    ]
  }

  clickhouse_app_admin_ad_groups = {
    "clickhouse_app_admin_ad_groups" = [
      "cloud-soub-developers1"
    ]
  }
}
