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

data "vtb_postgresql_image_data" "img" {
  product_type                  = "cluster"
  default_transaction_isolation = "READ COMMITTED"
  distribution                  = "astra"
  version                       = "1.7"
}

resource "vtb_postgresql_instance" "name" {
  lifetime = 2
  label    = "TerraformPostgreSQL"
  core     = data.vtb_core_data.dev
  flavor   = data.vtb_flavor_data.c4m16
  image    = data.vtb_postgresql_image_data.img
  extra_mounts = {
    "/pg_data" = {
      size        = 50
    }
  }
  access = {
    "superuser" = [
      "cloud-soub-posgres",
    ],
  }
  dbs = {
    "terraform" = {
      conn_limit    = 10
      db_admin_pass = "-A1b2C3d4E5f6G7h8.I9J0kL1m2N3o4P5q6R7S8-t9U0vW1x2Y3Z4a5B6C7d8A1b2C3d4E5f6G7h8I9J0kL1m2N3o4P5q6R7S8t9U0vW1x2Y3Z4a5B6C7d8"
    }
  }
  db_users = {
    "terraform_user1" = {
      db_name       = "terraform"
      dbms_role     = "user"
      user_password = "-A1b2C3d4E5f6G7h8.I9J0kL1m2N3o4P5q6R7S8-t9U0vW1x2Y3Z4a5B6C7d8A1b2C3d4E5f6G7h8I9J0kL1m2N3o4P5q6R7S8t9U0vW1x2Y3Z4a5B6C7d8"
    }
  }
}