data "vtb_jenkins_agent_subsystem_data" "sfera_subsystem" {
  net_segment = data.vtb_core_data.prod.net_segment
  ris_id = "1482"
}