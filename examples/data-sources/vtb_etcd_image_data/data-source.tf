data "vtb_etcd_image_data" "etcd_image" {
  distribution = "astra"
  os_version = "1.7"
  nodes_count = 3
}