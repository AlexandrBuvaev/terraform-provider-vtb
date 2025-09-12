data "vtb_core_data" "dev" {
  platform    = "OpenStack"
  net_segment = "dev-srv-app"
  zone        = "msk-north"
  domain      = "corp.dev.vtb"
}

data "vtb_elasticsearch_image_data" "elastic_image" {
  os_version   = "1.8"
  distribution = "astra"
}

data "vtb_flavor_data" "master_flavor" {
  cores  = 4
  memory = 8
}

data "vtb_flavor_data" "data_flavor" {
  cores  = 4
  memory = 8
}

data "vtb_flavor_data" "coordinator_flavor" {
  cores  = 4
  memory = 8
}

// required if 'install_kibana' = true
data "vtb_flavor_data" "kibana_flavor" {
  cores  = 4
  memory = 8
}

resource "vtb_elasticsearch_cluster" "cluster_with_install_kibana" {
  label                 = "Elasticsearch Astra tf"
  financial_project     = "VTB.Cloud"
  core                  = data.vtb_core_data.dev
  image                 = data.vtb_elasticsearch_image_data.elastic_image
  flavor_master         = data.vtb_flavor_data.master_flavor
  flavor_data           = data.vtb_flavor_data.data_flavor
  flavor_coordinator    = data.vtb_flavor_data.coordinator_flavor
  elasticsearch_version = "2.15.0 + Exporter 1.1.0"
  cluster_name          = "test-cluster"
  fluentd_password      = "AonlELAyj8roIzT%NrKYd4gLW2mSc@tW5TEq7^Miw~q%2t#823^50sG5@E"
  kibana_password       = "hZBo3dYhU#pvhuuQsFN0LaTz!ycKjCzp%Q#9uLvLH"
  nodes_count = {
    data        = 2
    master      = 1
    coordinator = 0
  }
  adm_app_groups  = ["cloud-soub-developers"]
  user_app_groups = ["cloud-soub-developers"]
  data_extra_mounts = {
    "/app/" = {
      size = 100
    }
  }
  install_kibana = true

  // dev only
  system_adm_groups = ["cloud-soub-developers"]
  access = {
    "superuser" = [
      "cloud-soub-developers",
    ]
  }

  // required if 'install_kibana' = true
  kibana_extra_mounts = {
    "/app/kibana" = {
      size = 100
    }
  }
  kibana_location = "separate host"
  flavor_kibana   = data.vtb_flavor_data.kibana_flavor
}


resource "vtb_elasticsearch_cluster" "cluster_without_install_kibana" {
  label                 = "Elasticsearch Astra tf"
  financial_project     = "VTB.Cloud"
  core                  = data.vtb_core_data.dev
  image                 = data.vtb_elasticsearch_image_data.elastic_image
  flavor_master         = data.vtb_flavor_data.master_flavor
  flavor_data           = data.vtb_flavor_data.data_flavor
  flavor_coordinator    = data.vtb_flavor_data.coordinator_flavor
  elasticsearch_version = "2.15.0 + Exporter 1.1.0"
  cluster_name          = "test-cluster"
  fluentd_password      = "AonlELAyj8roIzT%NrKYd4gLW2mSc@tW5TEq7^Miw~q%2t#823^50sG5@E"
  kibana_password       = "hZBo3dYhU#pvhuuQsFN0LaTz!ycKjCzp%Q#9uLvLH"
  nodes_count = {
    data        = 2
    master      = 1
    coordinator = 0
  }
  data_extra_mounts = {
    "/app/" = {
      size = 100
    }
  }
  adm_app_groups  = ["cloud-soub-buvaev"]
  user_app_groups = ["cloud-soub-buvaev"]
  install_kibana  = false

  // dev only
  system_adm_groups = ["cloud-soub-buvaev"]
  access = {
    "superuser" = [
      "cloud-soub-developers",
    ]
  }
}
