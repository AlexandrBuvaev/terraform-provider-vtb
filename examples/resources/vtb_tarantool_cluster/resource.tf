data "vtb_core_data" "test_lt" {
  platform    = "OpenStack"
  domain      = "test.vtb.ru"
  net_segment = "test-srv-perf"
  zone        = "msk-north"
}

data "vtb_tdg_image_data" "img" { // Указываем для заказа Tarantool Data Grid
  distribution = "astra"
  os_version   = "1.7"
}

data "vtb_te_image_data" "img_te" { // Указываем для заказа Tarantool Enterprise
  distribution = "astra"
  os_version   = "1.7"
}


data "vtb_cluster_layout" "tarantool_test" {
  net_segment = "test-srv-perf"
  layout      = "one_dc:tarantool:r-1:storage-24GB:x2:etcd"
}


resource "vtb_tarantool_cluster" "test" {
  lifetime          = 7
  label             = "Tarantool Data Grid v2"
  core              = data.vtb_core_data.test_lt
  image             = data.vtb_tdg_image_data.img
  financial_project = "1095 Импортозамещение инфраструктурных сервисов"
  access = {
    "superuser" = [
      "cloud-soub-tarantool-lt",
    ],
  }
  tarantool_access  = ["cloud-soub-tarantool-lt"]
  layout            = data.vtb_cluster_layout.tarantool_test.id
  tarantool_version = "2.12.1"
}
