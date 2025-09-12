resource "vtb_rabbitmq_vhosts" "vhost_list" {
  rabbitmq_order_id = vtb_rabbitmq_cluster.name.order_id
  hostnames = [
    "vhost1",
    "vhost2",
    "vhost3"
  ]
}
