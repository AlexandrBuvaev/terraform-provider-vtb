package utils

import (
	"fmt"
	"slices"
	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/consts"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func ValidateActiveDirectoryAccessForEnvironmentsV1(
	client *client.CloudClient,
	planAccess map[string][]types.String,
	environments ...string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	currentEnv := client.Environment
	isTargetEnv := slices.Contains(environments, currentEnv)
	hasAccess := len(planAccess) != 0

	switch {
	case !isTargetEnv && hasAccess:
		diags.AddError(
			consts.MODIFY_PLAN_FAIL,
			fmt.Sprintf(
				"`access = {}` (ad_logon_grants) not available for environment: %s\n"+
					"Available environments: %v",
				currentEnv,
				environments,
			),
		)
	}

	return diags
}

func ValidateActiveDirectoryAccessForEnvironmentsV2(
	client *client.CloudClient,
	planAccess map[string][]string,
	environments ...string,
) diag.Diagnostics {
	var diags diag.Diagnostics

	currentEnv := client.Environment
	isTargetEnv := slices.Contains(environments, currentEnv)
	hasAccess := len(planAccess) != 0

	switch {
	case !isTargetEnv && hasAccess:
		diags.AddError(
			consts.MODIFY_PLAN_FAIL,
			fmt.Sprintf(
				"`access = {}` (ad_logon_grants) not available for environment: %s\n"+
					"Available environments: %v",
				currentEnv,
				environments,
			),
		)
	}

	return diags
}
