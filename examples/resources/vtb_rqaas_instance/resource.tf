data "vtb_rqaas_cluster_data" "test" {
  name = "288542b2-test"
}



resource "vtb_rqaas_instance" "test" {
  cluster = {
    domain      = data.vtb_rqaas_cluster_data.test.domain
    hosts       = data.vtb_rqaas_cluster_data.test.hosts
    name        = data.vtb_rqaas_cluster_data.test.name
    net_segment = data.vtb_rqaas_cluster_data.test.net_segment
    platform    = data.vtb_rqaas_cluster_data.test.platform
    zone        = data.vtb_rqaas_cluster_data.test.zone
  }
  financial_project = "VTB.Cloud"
  label             = "RabbitMQ Очередь как сервис"
  name              = "test-queue-1"
  queue_users = [
    {
      read     = true
      username = "1482-rbmq-d5-client-test1"
      write    = false
    },
    {
      read     = true
      username = "APD12345-1482-rbmq-d5-client-test1"
      write    = true
    },
    {
      read     = false
      username = "APD12345-1482-rbmq-d5-client-test"
      write    = true
    },
    {
      read     = false
      username = "APD12345-1482-rbmq-d5-client-test2"
      write    = true
    },
  ]
}
