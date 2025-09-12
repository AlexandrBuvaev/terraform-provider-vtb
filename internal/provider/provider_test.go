package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

const (
	// providerConfig is a shared configuration to combine with the actual
	// test configuration so the HashiCups client is properly configured.
	// It is also possible to use the HASHICUPS_ environment variables instead,
	// such as updating the Makefile and running the testing through that tool.
	// BLUE
	providerConfig = `
	provider "vtb" {
		project_name  = "proj-h1eu89sx40"
		client_id     = "sa_proj-h1eu89sx40-71"
		client_secret = "fc5e2961c45a3698910b054dd8dac21f"
	}
	`

	// PROD
	// providerConfig = `
	// provider "vtb" {
	// 	client_id     = "sa_proj-5ejgs0vfzf-1qf"
	// 	client_secret = "4ae45acda66a73d10f2d4462aaaf0e23"
	// 	project_name  = "proj-5ejgs0vfzf"
	// }
	// `
)

var (
	// testAccProtoV6ProviderFactories are used to instantiate a provider during
	// acceptance testing. The factory function will be invoked for every Terraform
	// CLI command executed to create a provider server to which the CLI can
	// reattach.
	testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
		"vtb": providerserver.NewProtocol6WithError(New("test")()),
	}
)
