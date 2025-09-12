data "vtb_scylla_db_cluster_image_data" "img" {
  distribution = "astra"
  os_version   = "1.7"
}

data "vtb_flavor_data" "c2m8" {
  cores  = 2
  memory = 8
}

data "vtb_core_data" "test" {
  platform    = "OpenStack"
  domain      = "test.vtb.ru"
  net_segment = "test-srv-synt"
  zone        = "msk-north"
}

resource "vtb_scylla_db_cluster_instance" "scylla_test" {
  core              = data.vtb_core_data.test-srv-synt
  flavor            = data.vtb_flavor_data.c2m8
  image             = data.vtb_scylla_db_cluster_image_data.img
  financial_project = "VTB.Cloud"
  label             = "ScyllaDB-TF-import2"
  scylladb_version = "5.4.4"
  extra_mounts_log = {
    "/app/scylla/logs" = {
      size = 30
    },
  }
  extra_mounts_data = {
    "/app/scylla/data" = {
      size = 100
    },
  }
  db_names = ["db1","db2"]
  db_users = {
    "user3" = {
      dbms_role = "user"
      user_password = "Q4ovbsrUudsMXfsmQ8.yhpzLpDnS0lZ91IFXn6bXCzS.02mQY16Ix4"
    }
    "admin1" = {
      dbms_role = "admin"
      user_password = "Q4ovbsrUudsMXfsmQ8.yhpzLpDnS0lZ91IFXn6bXCzS.02mQY16Ix5"
    }
  }
  db_permissions = [
    "user3:db1",
    "admin1:db2"
  ]
  scylla_cluster_configuration = {
    dc1 = 1
    dc2 = 0
    dc3 = 0
  }
}
