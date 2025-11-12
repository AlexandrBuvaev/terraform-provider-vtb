package utils

import (
	"fmt"
	"strings"
	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/pkg/client/references"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func ValidateDomains(
	client *client.CloudClient,
	planDomain types.String,
) diag.Diagnostics {
	var diags diag.Diagnostics

	domains, err := references.GetKnownDomains(client.Creds)
	if err != nil {
		diags.AddError(
			"Getting known domains",
			fmt.Sprintf(
				"reference has [%s] in page or failed with error '%s'",
				domains, err.Error(),
			),
		)
	}

	for _, domainName := range domains {
		if strings.EqualFold(domainName, planDomain.ValueString()) {
			return diags
		}
	}
	diags.AddError(
		"Domain not available for resource",
		fmt.Sprintf(
			"Available domains [%s]. Specified '%s'",
			strings.Join(domains, ", "), planDomain.ValueString(),
		),
	)

	return diags
}
