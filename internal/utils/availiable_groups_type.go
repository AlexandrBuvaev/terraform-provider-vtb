package utils

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/pkg/client/sources"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func ValidateGroupTypeSet(
	ctx context.Context,
	client *client.CloudClient,
	fieldName string,
	groupType string,
	planGroupsSet types.Set,
) diag.Diagnostics {
	var diags diag.Diagnostics

	planGroups := []string{}
	diags.Append(planGroupsSet.ElementsAs(ctx, &planGroups, false)...)

	err := checkGroupsByAccountType(client, groupType, planGroups)
	if err != nil {
		diags.AddAttributeError(path.Root(fieldName), "Unavailble groups for type resource", err.Error())
	}
	return diags
}

func ValidateGroupTypeMap(
	ctx context.Context,
	client *client.CloudClient,
	fieldName string,
	groupType string,
	tfPlanGroupsMap map[string][]basetypes.StringValue,
) diag.Diagnostics {
	var diags diag.Diagnostics

	for _, planGroups := range tfPlanGroupsMap {
		err := checkGroupsByAccountType(client, groupType, convertStringValues(planGroups))
		if err != nil {
			diags.AddAttributeError(path.Root(fieldName), "Unavailble groups type for resource", err.Error())
		}
	}
	return diags
}

func checkGroupsByAccountType(
	c *client.CloudClient,
	groupType string,
	planGroups []string,
) error {
	groups, err := sources.GetAccessGroupsByAccountsType(c.Creds, c.ProjectName, groupType)
	if err != nil {
		return fmt.Errorf("get groups from portal: %v", err.Error())
	}

	groupsSl := []string{}
	for _, s := range groups {
		groupsSl = append(groupsSl, s.Name)
	}

	for _, gr := range planGroups {
		if !slices.Contains(groupsSl, gr) {
			return fmt.Errorf(
				"available groups with this type `%s` : [%s]. Specified '%s'",
				groupType, strings.Join(groupsSl, ", "), strings.Join(planGroups, ", "),
			)
		}
	}
	return nil
}

func convertStringValues(stringValues []basetypes.StringValue) []string {
	result := make([]string, len(stringValues))
	for i, sv := range stringValues {
		result[i] = sv.ValueString()
	}
	return result
}
