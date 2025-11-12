resource "vtb_sync_xpert_connector" "example" {
  order_id = vtb_sync_xpert_cluster.test.order_id
  name     = "pg-soub-d5-testdb-pg-soub-d5-testdb-testdb"

  database = {
    hostname          = "d5soub-pgc001lk.corp.dev.vtb:5432,d5soub-pgc002lk.corp.dev.vtb:5432"
    name              = "testdb"
    user              = "debezium"
    password          = "Dx6T9SMzcNa33HU-gjMyICSZuK4cKHxM4QWA3MIIpBbO8Yx7GH8hD3E-TjpUPKZytxFzF1V7AENBrU"
    include_list      = "test"
    include_list_type = "schema.include.list"
    publication_name  = "pub_dbzm"
    slot_name         = "syncxpert_test"
    db_topic_prefix = "test"
  }

  ssl = {
    mode    = "disable"
  }

  heartbeat = {
    action_query = "insert into hb_debezium.hb_table (id) values (1)"
    interval_ms  = 60000
    topic_prefix = "debezium-heartbeat"
  }
}