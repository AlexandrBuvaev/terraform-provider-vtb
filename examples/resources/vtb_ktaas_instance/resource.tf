resource "vtb_ktaas_instance" "test" {
  label              = "KTaaS tf"
  lifetime           = 2
  financial_project  = "VTB.Cloud"
  topic_name         = "1482_test-buvaev"
  topic_flavor       = 2
  partitions_number  = 4
  net_segment        = "dev-srv-app"
  kafka_cluster_name = "1482-kafka-da-cluster-kafka-ktaas-0095"
  acls = [
    {
      client_cn   = "AP1"
      client_role = "consumer"
    },
    {
      client_cn   = "AP1"
      client_role = "producer"
    }
  ]
  group_acls = [
    {
      group_name = "1482_test-buvaev_consumergroup_test2"
    }
  ]
  lifecycle {
    ignore_changes = [lifetime]
  }
}