resource "vtb_artemis_roles" "temporary" {
  vtb_artemis_order_id = vtb_artemis_cluster.test.order_id
  depends_on           = [vtb_artemis_tuz.tuz_test]
  role_list = [
    {
      role = "cluster_manager"
      user_names = ["tuz_2", "tuz_1"]
      security_policy_name = "all"
    },
    {
      role = "address_2_producer"
      user_names = ["tuz_4", "tuz_1"]
      security_policy_name = "DC.service.address_2"
    },
  ]
}