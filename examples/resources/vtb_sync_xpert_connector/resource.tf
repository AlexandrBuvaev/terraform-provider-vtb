resource "vtb_sync_xpert_connector" "test" {
  order_id = vtb_sync_xpert_cluster.testd.order_id
  name              = "pg-soub-da-baza-baza2"
  database = {
    hostname          = "dasoub-pgc159lk.corp.dev.vtb:5432,dasoub-pgc158lk.corp.dev.vtb:5432"
    name              = "baza"
    user              = "debezium"
    password          = "your_super_secret_password_for_technical_user"
    include_list      = "dbzm"
    include_list_type = "schema.include.list"
    publication_name  = "pub_dbzm"
    slot_name         = "slot"
  }
  ssl = {
    mode = "disable"
  }
  heartbeat = {
    action_query = "insert into hb_debezium.hb_table (id) values (1)"
    interval_ms  = 60000
    topic_prefix = "debezium-heartbeat"
  }
}