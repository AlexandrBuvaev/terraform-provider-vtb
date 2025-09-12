data "vtb_core_data" "prod" {
  domain      = "region.vtb.ru"
  net_segment = "prod-srv-ss"
  platform    = "OpenStack"
  zone        = "msk-east"
}

data "vtb_flavor_data" "c2m4" {
  cores  = 2
  memory = 4
}

data "vtb_agent_orchestration_image_data" "img" {
  distribution = "astra"
  os_version   = "1.7"
}

data "vtb_jenkins_agent_subsystem_data" "sfera_subsystem" {
  net_segment = data.vtb_core_data.prod.net_segment
  ris_id = "1482"
}

resource "vtb_agent_orchestration_instance" "test" {
  core              = data.vtb_core_data.prod
  flavor            = data.vtb_flavor_data.c2m4
  image             = data.vtb_agent_orchestration_image_data.img
  financial_project = "VTB.Cloud"
  label             = "Агент Оркестрации"
  sfera_agent = {
    jenkins_agent_executors = 1
    jenkins_agent_subsystem = data.vtb_jenkins_agent_subsystem_data.sfera_subsystem
  }
  extra_mounts = {
    "/app" = {
      size = 100
    },
  }
}
