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

data "vtb_airflow_image_data" "airflow_img_cluster" {
  distribution = "astra"
  product_type = "cluster"
  os_version   = "1.7"
}

data "vtb_cluster_layout" "airflow" {
  layout      = "one_dc:webserver-2:scheduler-2:worker-4"
  net_segment = "dev-srv-app"
}

resource "vtb_airflow_cluster" "airflow_cluster" {
  lifetime          = 30
  label             = "Terraform Airflow Cluster"
  financial_project = "VTB.Cloud"
  core              = data.vtb_core_data.dev
  flavor_worker     = data.vtb_flavor_data.c4m16
  flavor_scheduler  = data.vtb_flavor_data.c4m16
  flavor_webserver  = data.vtb_flavor_data.c4m16
  image             = data.vtb_airflow_image_data.airflow_img_cluster
  layout_id         = data.vtb_cluster_layout.airflow.id
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
    "Operator" = [
      "cloud-soub-airflow-dev"
    ]
  }
  postgresql_config = {
    db_order_id = "6d1d1165-5b60-4df4-956a-657a0b9229f7"
    db_user     = "airflow_cluster_tf_admin"
    db_database = "airflow_cluster_tf"
    db_password = "_NFlkqNzTjvll9_J_XyH4zo41asjw6M67RIEadETMCv_V6U53cbIiD163j-EKOFSpqGamXnKNxSJd7S0CbPFOCecnQ.5yKTcjwCxcHs"
  }
  rabbitmq_config = {
    broker_order_id = "d08ab921-9666-4803-b00d-dac247a19448"
    broker_vhost    = "airflow-clu-005"
  }
  update_product_mode = "latest"
}
