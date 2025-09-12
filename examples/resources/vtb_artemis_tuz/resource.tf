resource "vtb_artemis_tuz" "test" {
  vtb_artemis_order_id = vtb_artemis_cluster.test.order_id
  users = [
    {
      user_name       = "test4"
      user_owner_cert = "CN=test24"
    },
    {
      user_name       = "test2"
      user_owner_cert = "CN=test23"
    },
    {
      user_name       = "test3"
      user_owner_cert = "CN=test33"
    },
    {
      user_name       = "test333"
      user_owner_cert = "CN=test334"
    },
  ]
}