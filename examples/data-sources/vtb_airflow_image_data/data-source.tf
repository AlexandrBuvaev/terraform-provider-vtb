data "vtb_airflow_image_data" "airflow_img_cluster" {
  distribution = "astra"
  product_type = "cluster"
  os_version = "1.7"
}

data "vtb_airflow_image_data" "airflow_img" {
  distribution = "astra"
  product_type = "stand-alone"
  os_version = "1.7"
}