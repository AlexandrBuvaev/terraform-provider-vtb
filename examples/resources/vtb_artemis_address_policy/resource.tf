resource "vtb_artemis_address_policy" "test1" {
  vtb_artemis_order_id = vtb_artemis_cluster.test.order_id
  address_policy_list = [
    {
      address_prefix             = "DC.service." # с ограничением времени на брокере
      address_name               = "test12"
      address_full_policy        = "BLOCK"
      slow_consumer_check_period = 12
      slow_consumer_threshold    = 21
      slow_consumer_policy       = "NOTIFY"
      max_size                   = "200Mb"
      min_expiry_delay           = 10000
      max_expiry_delay           = 60000
    },
    {
      address_prefix             = "DC.client." # без ограничения времени на брокере
      address_name               = "test3"
      address_full_policy        = "FAIL"
      slow_consumer_check_period = 10
      slow_consumer_threshold    = 12
      slow_consumer_policy       = "NOTIFY"
      max_size                   = "200Mb"
      # min_expiry_delay           = 10000
      # max_expiry_delay           = 60000
    },
    {
      address_prefix             = "DC.service." # без ограничения времени на брокере
      address_name               = "test2"
      address_full_policy        = "FAIL"
      slow_consumer_check_period = 10
      slow_consumer_threshold    = 10
      slow_consumer_policy       = "NOTIFY"
      max_size                   = "200Mb"
      min_expiry_delay           = -1
      max_expiry_delay           = -1
    },
    
  ]

}
