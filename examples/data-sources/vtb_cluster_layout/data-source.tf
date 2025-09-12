data "vtb_cluster_layout" "rabbitmq" {
	layout      = "one_dc:rabbitmq-2:quorum-1"
	net_segment = "dev-srv-app"
}

data "vtb_cluster_layout" "debezium" {
	layout      = "one_dc:debezium-1"
	net_segment = "b2b-hce-ts-dev-srv-app"
}

data "vtb_cluster_layout" "artemis" {
	layout      = "geo:artemis-2:artemis-2"
	net_segment = "prod-srv-app"
}

data "vtb_cluster_layout" "kafka" {
	layout      = "one_dc:kafka-6:zookeeper-3"
	net_segment = "test-srv-synt"
}