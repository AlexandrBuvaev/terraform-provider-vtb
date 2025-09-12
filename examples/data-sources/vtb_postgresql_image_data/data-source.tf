data "vtb_postgresql_image_data" "img" {
  product_type                  = "cluster"
  default_transaction_isolation = "READ COMMITTED"
  distribution                  = "astra"
  version                       = "1.7"
}
