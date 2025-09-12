data "vtb_core_data" "dev" {
  platform    = "OpenStack"
  domain      = "corp.dev.vtb"
  net_segment = "dev-srv-app"
  zone        = "msk-north"
}

data "vtb_flavor_data" "c4m8" {
  cores  = 4
  memory = 8
}

data "vtb_kafka_image_data" "kafka_img" {
  distribution = "astra"
  os_version   = "1.7"
}

data "vtb_cluster_layout" "kafka" {
  layout      = "one_dc:kafka:zookeeper"
  net_segment = "dev-srv-app"
}

resource "vtb_kafka_instance" "kafka_powere_test2" {
  label         = "TerraformKafka"
  kafka_version = "2.13-2.4.1"
  core          = data.vtb_core_data.dev
  flavor        = data.vtb_flavor_data.c4m8
  image         = data.vtb_kafka_image_data.kafka_img
  layout_id     = data.vtb_cluster_layout.kafka.id
  financial_project = "VTB.Cloud"
  extra_mounts = {
    "/app" = {
      size        = 25
    }
  }
  access = {
    "kafka_admin" = [
      "cloud-soub-kafka",
    ]
  }
   
   topics = {
    "test-1" = {
      cleanup_policy  = "delete"
      partitions      = 1
      retention_ms    = 1800009
    }
    "test-2" = {
      cleanup_policy  = "delete,compact"
      partitions      = 1
      retention_ms    = 1800009
    }
    "test-3" = {
      cleanup_policy  = "compact"
      partitions      = 1
    }
  }

  acls = {
    "APD2" = {
      allow_idempotent      = false 
      consumer_by_name      = ["consumer-1"]
      producer_by_name      = []
      consumer_by_mask      = []
      producer_by_mask      = []
      transactional_by_name = []
      transactional_by_mask = []
    },
    "APD1" = {
      allow_idempotent      = false 
      transactional_by_name = ["test-xxx"]
    }
  }
}
