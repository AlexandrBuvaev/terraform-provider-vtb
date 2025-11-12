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

data "vtb_debezium_image_data" "test" {
  distribution = "astra"
  os_version   = "1.7"
}

data "vtb_cluster_layout" "debezium" {
  layout      = "one_dc:debezium-2"
  net_segment = "dev-srv-app"
}

resource "vtb_sync_xpert_cluster" "test" {
  label     = "SyncExpert Astra"
  financial_project = "VTB.Cloud"
  core      = data.vtb_core_data.dev
  flavor    = data.vtb_flavor_data.c2m4
  image     = data.vtb_debezium_image_data.test
  layout_id = data.vtb_cluster_layout.debezium.id
  extra_mounts = {
    "/app" = {
      size = 30
    },
  }
  access = {
    "superuser" = [
      "cloud-soub-buvaev",
      "cloud-soub-chumakovas",
    ],
    "user" = [
      "cloud-soub-buvaev",
    ],
  }

  cluster_name = "test-cluster"
  api_user             = "VTB4115884"
  api_password         = "btC9ox.B3Ai87EqA02ZXBjbUiWBaswNU4HSTNyySw.zmtCpj"
  cluster_group_id     = "test"
  debezium_version     = "1.1.0"
  kafka_server         = "d5soub-kfc005lk.corp.dev.vtb:9092,d5soub-kfc010lk.corp.dev.vtb:9092,d5soub-kfc001lk.corp.dev.vtb:9092,d5soub-kfc009lk.corp.dev.vtb:9092"
  kafka_cert_cname     = "APD09.26.01.01-1482-kafka-client-syncxpert-d5-test"
}