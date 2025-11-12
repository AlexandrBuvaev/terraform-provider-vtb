package utils

import (
	"fmt"
	"strings"
	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/services/flavor"
	"terraform-provider-vtb/pkg/client/references"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func ValidateFlavor(
	client *client.CloudClient,
	planFlavor flavor.FlavorModel,
	filter string,
) diag.Diagnostics {

	var diags diag.Diagnostics
	wishedCores := planFlavor.Cores.ValueInt64()
	wishedMemory := planFlavor.Memory.ValueInt64()

	fullFilter := fmt.Sprintf("flavor:%s:%s", filter, strings.ToLower(client.Environment))

	_, err := references.GetFlavorByFilter(
		client.Creds,
		fullFilter,
		wishedMemory,
		wishedCores,
	)
	if err != nil {
		diags.AddError("Flavor not available for resource", err.Error())
	}
	return diags
}
