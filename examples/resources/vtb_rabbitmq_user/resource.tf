resource "vtb_rabbitmq_vhosts" "vhost_list" {
  rabbitmq_order_id = vtb_rabbitmq_cluster.name.order_id
  hostnames = [
    "vhost1",
    "vhost2",
  ]
}

resource "vtb_rabbitmq_user" "user" {
  rabbitmq_order_id = vtb_rabbitmq_cluster.name.order_id
  username          = "1234-rbmq-d5-client-buvaev"
  vhosts_access = {
    vhost_read      = ["vhost1"]
    vhost_write     = ["vhost1"]
    vhost_configure = ["vhost1", "vhost2"]
  }
}
