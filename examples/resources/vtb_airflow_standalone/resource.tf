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

data "vtb_airflow_image_data" "airflow_img" {
  distribution = "astra"
  product_type = "stand-alone"
  os_version   = "1.7"
}

resource "vtb_airflow_standalone" "airflow_stalone" {
  lifetime          = 30
  label             = "Terraform Airflow Standalone"
  financial_project = "VTB.Cloud"
  core              = data.vtb_core_data.dev
  flavor            = data.vtb_flavor_data.c4m16
  image             = data.vtb_airflow_image_data.airflow_img
  extra_mounts = {
    "/app" = {
      size = 30
    },
    "/app_data" = {
      size = 30
    }
  }
  access = {
    "superuser" = [
      "cloud-soub-airflow-dev"
    ]
  }
  deploy_grants = {
    airflow_deploy = [
      "cloud-soub-deploy"
    ]
  }
  web_console_grants = {
    Operator = [
      "cloud-soub-airflow-dev"
    ]
  }
  postgresql_config = {
    db_order_id = "6d1d1165-5b60-4df4-956a-657a0b9229f7"
    db_user     = "airflow_tf_admin"
    db_database = "airflow_tf"
    db_password = "Ly8NcLUH7FK_WNYOi38.cYYnHNX7mA9Qm7leM3.WILi_HiYk6AfJFtdhxxerfd1ucZ.39ZIB8S"
  }
  update_product_mode = "latest"
}
