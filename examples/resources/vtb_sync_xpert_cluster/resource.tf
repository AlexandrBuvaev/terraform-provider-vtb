data "vtb_core_data" "dev" {
  platform    = "OpenStack"
  domain      = "corp.dev.vtb"
  net_segment = "dev-srv-app"
  zone        = "msk-north"
}

data "vtb_debezium_image_data" "test" {
  distribution = "astra"
  os_version   = "1.7"
}

data "vtb_cluster_layout" "debezium" {
  layout      = "one_dc:debezium-1"
  net_segment = "dev-srv-app"
}

resource "vtb_sync_xpert_cluster" "test" {
  lifetime  = 2
  label     = "TerraformDebezium1"
  core      = data.vtb_core_data.dev
  flavor    = data.vtb_flavor_data.fv
  image     = data.vtb_debezium_image_data.test
  layout_id = data.vtb_cluster_layout.debezium.id
  financial_project = "VTB.Cloud"
  extra_mounts = {
    "/app" = {
      size        = 30
    },
  }
  access = {
    "superuser" = [
      "cloud-soub-dbzm",
    ],
  }

  api_user             = "VTB4096014"
  api_password         = "{Sq4-[5zP&7pk~Y_B?c<mV=o,.#G-/|r]B/x+M"
  cluster_group_id     = "dbzm-test"
  debezium_version     = "1.1.0"
  kafka_server         = "dasoub-kfc179lk.corp.dev.vtb:9092"
  kafka_cert_cname     = "APD09.26-1482-kafka-da-cluster-a-kafka-astra-0341"
  config_storage_topic = "dbzm-config-test"
  offset_storage_topic = "dbzm-offset-test"
  status_storage_topic = "dbzm-status-test"
}