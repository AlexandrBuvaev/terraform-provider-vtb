package utils

import (
	"fmt"
	"strings"
	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/pkg/client/references"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func ValidateAccessRolesV1(
	client *client.CloudClient,
	filter string,
	plannedAccess map[string][]string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	roles, err := references.GetRoles(client.Creds, client.Environment, filter)
	if err != nil {
		diags.AddError("Get roles from portal", err.Error())
	}

	var rolesNames []string
	for _, r := range roles {
		rolesNames = append(rolesNames, r.Data.GroupName)
	}

	for key := range plannedAccess {
		exists := false
		for _, rname := range rolesNames {
			if key == rname {
				exists = true
				break
			}
		}
		if !exists {
			diags.AddError(
				"Access Roles not available for resource",
				fmt.Sprintf(
					"Available roles [%s]. Specified '%s'",
					strings.Join(rolesNames, ", "), key,
				),
			)
		}
	}

	return diags

}

func ValidateAccessRolesV2(
	client *client.CloudClient,
	filter string,
	plannedAccess map[string][]types.String,
) diag.Diagnostics {
	var diags diag.Diagnostics

	roles, err := references.GetRoles(client.Creds, client.Environment, filter)
	if err != nil {
		diags.AddError("Get roles from portal", err.Error())
	}

	var rolesNames []string
	for _, r := range roles {
		rolesNames = append(rolesNames, r.Data.GroupName)
	}

	for key := range plannedAccess {
		exists := false
		for _, rname := range rolesNames {
			if key == rname {
				exists = true
				break
			}
		}
		if !exists {
			diags.AddError(
				"Access Roles not available for resource",
				fmt.Sprintf(
					"Available roles [%s]. Specified '%s'",
					strings.Join(rolesNames, ", "), key,
				),
			)
		}
	}

	return diags

}
