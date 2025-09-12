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

data "vtb_redis_image_data" "redis_img" {
  distribution = "astra"
  os_version      = "1.7"
}

resource "vtb_redis_instance" "new_redis_demo_tf" {
  label = "TerraformRedis"
  core   = data.vtb_core_data.dev
  flavor = data.vtb_flavor_data.c2m4
  image  = data.vtb_redis_image_data.redis_img
  access = {
    "superuser" = [
      "cloud-soub-redis",
    ]
  }
  extra_mounts = {
    "/app/redis/logs" = {
      size = 10
    }
    "/app/redis/data" = {
      size = 10
    },
  }

  redis_version          = "7.2.4"
  user                   = "demo"
  user_password          = "T3j9r8u4W9k6HfJ7q2V5lJd3R8B0pQ1M6X8ZpC5A1K9NwT2Xs5gV2L3Yt7sdawawawWvfv"
  notify_keyspace_events = "AKE"
}
