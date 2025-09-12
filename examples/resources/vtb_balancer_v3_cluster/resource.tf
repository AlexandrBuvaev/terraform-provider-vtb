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

data "vtb_balancer_v3_image_data" "img" {
  distribution = "astra"
  os_version   = "1.7"
}

data "vtb_cluster_layout" "balancer" {
  layout      = "balancer-1"
  net_segment = "dev-srv-app"
}

resource "vtb_balancer_v3_cluster" "test" {
  label             = "Load Balancer v3.4 Terraform"
  core              = data.vtb_core_data.dev
  flavor            = data.vtb_flavor_data.c2m4
  image             = data.vtb_balancer_v3_image_data.img
  financial_project = "VTB.Cloud"
  extra_mounts = {
    "/app" = {
      size = 50
    },
  }
  access = {
    "superuser" = [
      "cloud-soub-test1",
      "cloud-soub-test2",
    ],
  }
  layout_id = data.vtb_cluster_layout.balancer.id

  setup_version = "3.4.0"
  cluster_name  = "terra2"
  dns_zone      = "oslb-dev-dev.corp.dev.vtb"
  config = {
    backends = [
      {
        backend_name        = "back3"
        balancing_algorithm = "leastconn"
        cookie = {
          enable = false
        }
        globalname = "back3.terra2.soub.d5.oslb-dev-dev.corp.dev.vtb"
        healthcheck = {
          check_strings = [
            {
              send_proxy = "disabled"
            }
          ]
          fall_count = 3
          interval   = 5
          mode       = "tcp"
          rise_count = 3
        }
        keep_alive = {
          mode = "default"
        }
        mode = "tcp"
        retries = {
          count      = 3
          conditions = ["conn-failure"]
          enabled    = true
          redispatch = "disabled"
        }
        servers = [
          {
            address    = "10.170.10.26"
            maxconn    = 0
            name       = "mock_server"
            send_proxy = "disabled"
            state      = "active"
          }
        ]
        servers_settings = {
          port        = 12001
          slow_start  = 10
        }
      }
    ]

    defaults = {
      client_timeout  = 40
      connect_timeout = 10
      server_timeout  = 30
    }

    globals = {
      h2_workaround_bogus_websocket_clients = false
      maxconn                               = 10000
      tune_options = "h2_fe_initial_window_size: 666\nh2_header_table_size: 5000\nh2_max_frame_size: 16384"
    }

    ports = [
      {
        frontend_name = "frontend_12001_tcp"
        keep_alive = {
          tcp = {
            mode = "default"
          }
        }
        maxconn = 0
        mode    = "tcp"
        port    = 12001
      }
    ]

    publications = [
      {
        alive_serv_count = 1
        default_routing = true
        globalname      = "back3.terra2.soub.d5.oslb-dev-dev.corp.dev.vtb"
        main_backend    = "back3"
        mode            = "tcp"
        port            = 12001
      }
    ]
  }
}