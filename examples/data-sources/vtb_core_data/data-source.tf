data "vtb_core_data" "dev" {
	net_segment = "dev-srv-app"
	platform    = "OpenStack"
	domain      = "corp.dev.vtb"
	zone        = "msk-north"
}
