package access

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/internal/utils"
	"terraform-provider-vtb/pkg/client/sources"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                 = &AccessGroupResource{}
	_ resource.ResourceWithModifyPlan   = &AccessGroupResource{}
	_ resource.ResourceWithUpgradeState = &AccessGroupResource{}
)

type AccessGroupResource struct {
	client *client.CloudClient
}

func NewAccessGroupResource() resource.Resource {
	return &AccessGroupResource{}
}

func (r AccessGroupResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_access_group_instance"
}

func (r *AccessGroupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.CloudClient)

	if !ok {
		resp.Diagnostics.AddError(
			"unexpected resource configure type",
			fmt.Sprintf("Expected *client.CloudClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = client
}

type AccessGroupModelV0 struct {
	Name        types.String             `tfsdk:"name"`
	FullName    types.String             `tfsdk:"full_name"`
	GroupDN     types.String             `tfsdk:"group_dn"`
	Domain      types.String             `tfsdk:"domain"`
	Description types.String             `tfsdk:"description"`
	Users       []AccessGroupUserModelV0 `tfsdk:"users"`
}

type AccessGroupUserModelV0 struct {
	ID         types.String `tfsdk:"id"`
	Username   types.String `tfsdk:"username"`
	Name       types.String `tfsdk:"name"`
	Email      types.String `tfsdk:"email"`
	UniqueName types.String `tfsdk:"unique_name"`
}

type AccessGroupModelV1 struct {
	Name         types.String             `tfsdk:"name"`
	FullName     types.String             `tfsdk:"full_name"`
	GroupDN      types.String             `tfsdk:"group_dn"`
	Domain       types.String             `tfsdk:"domain"`
	Description  types.String             `tfsdk:"description"`
	Purpose      types.String             `tfsdk:"purpose"`
	AccountsType types.String             `tfsdk:"accounts_type"`
	Users        []AccessGroupUserModelV1 `tfsdk:"users"`
}

type AccessGroupUserModelV1 struct {
	ID         types.String `tfsdk:"id"`
	Username   types.String `tfsdk:"username"`
	Name       types.String `tfsdk:"name"`
	Email      types.String `tfsdk:"email"`
	UniqueName types.String `tfsdk:"unique_name"`
}

func (r AccessGroupResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{

			"name": schema.StringAttribute{
				Required:            true,
				Description:         "Name of group",
				MarkdownDescription: "Name of group",
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						regexp.MustCompile("^[a-z0-9-]+$"),
						"must only contain: lowercase characters, numbers and '-'",
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"full_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Full name of group",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"group_dn": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Name of group",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"domain": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Domain of group",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},

			"description": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Description of group",
			},

			"purpose": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Group of purpose",
				Default:             stringdefault.StaticString("compute"),
				Validators: []validator.String{
					stringvalidator.OneOf("compute", "vlt"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},

			"accounts_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Domain of group",
				Default:             stringdefault.StaticString("personal"),
				Validators: []validator.String{
					stringvalidator.OneOf("personal"),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},

			"users": schema.SetNestedAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "List of users to add to access group",
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"username": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"name": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"email": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
						"unique_name": schema.StringAttribute{
							Optional: true,
							Computed: true,
						},
					},
				},
			},
		},
		Version: 1,
	}
}

func (r AccessGroupResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan AccessGroupModelV1
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groups, err := sources.GetAccessGroups(
		r.client.Creds,
		r.client.ProjectName,
		plan.Domain.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating access group resource error",
			fmt.Sprintf(
				"AccessGroup '%s' can't get info about groups from portal.\nError: %s",
				plan.Name, err.Error(),
			),
		)
		return
	}

	project, err := sources.GetProject(
		r.client.Creds,
		r.client.ProjectName,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating access group resource error",
			fmt.Sprintf(
				"AccessGroup '%s' can't get info about project from portal.\nError: %s",
				plan.Name, err.Error(),
			),
		)
		return
	}

	for _, group := range groups {
		if plan.Name.ValueString() == group.Name {
			resp.Diagnostics.AddError(
				"Creating access group resource error",
				fmt.Sprintf("Group with name '%s' already exists", plan.Name.ValueString()),
			)
			return
		}
	}

	accessGroup, err := sources.NewAccessGroup(
		r.client.Creds,
		project,
		plan.Name.ValueString(),
		plan.Domain.ValueString(),
		plan.Description.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating access group resource error",
			fmt.Sprintf("Group with name '%s' already exists", plan.Name),
		)
		return
	}

	err = accessGroup.Create(
		plan.Purpose.ValueString(),
		plan.AccountsType.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Creating access group resource error",
			fmt.Sprintf(
				"AccessGroup '%s' didn't created.\nError: %s",
				plan.Name, err.Error(),
			),
		)
		return
	}

	if len(plan.Users) > 0 {

		var usersUniqueName []string
		for _, user := range plan.Users {
			usersUniqueName = append(usersUniqueName, user.UniqueName.ValueString())
		}

		err := accessGroup.AddUsers(usersUniqueName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Creating access group resource error",
				fmt.Sprintf(
					"AccessGroup '%s' didn't appened users or appended parted.\nError: %s",
					plan.Name, err.Error(),
				),
			)
			return
		}
	}

	plan.FullName = types.StringValue(accessGroup.Name)
	plan.GroupDN = types.StringValue(accessGroup.GroupDn)

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r AccessGroupResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan AccessGroupModelV1
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	validateDomainDiags := utils.ValidateDomains(
		r.client,
		plan.Domain,
	)
	resp.Diagnostics.Append(validateDomainDiags...)

	var purposePostfix string = ""
	if plan.Purpose.ValueString() != "compute" {
		purposePostfix = plan.Purpose.ValueString() + "-"
	}

	if plan.FullName.IsNull() || plan.FullName.IsUnknown() {
		plan.FullName = types.StringValue(
			fmt.Sprintf(
				"cloud-%s-%s%s",
				strings.ToLower(r.client.RisCode),
				purposePostfix,
				plan.Name.ValueString(),
			),
		)
	}

	if len(plan.FullName.ValueString()) >= 36 {
		resp.Diagnostics.AddError(
			"Validate access group full name",
			fmt.Sprintf(
				"Length of 'full_name' must be less or equal 36. Current: %d",
				len(plan.FullName.ValueString()),
			),
		)
		return
	}

	if req.State.Raw.IsNull() {
		_, err := sources.GetAccessGroupByName(
			r.client.Creds,
			r.client.ProjectName,
			plan.Domain.ValueString(),
			plan.FullName.ValueString(),
		)

		if err == nil || !strings.Contains(err.Error(), "can't find group") {

			if err != nil {
				resp.Diagnostics.AddError(
					"Validate access group values",
					fmt.Sprintf(
						"Request get access group ended with error.\nError:%s",
						err.Error(),
					),
				)
			} else {
				resp.Diagnostics.AddError(
					"Validate access group values",
					fmt.Sprintf(
						"Access Group '%s' already exists.",
						plan.FullName.ValueString(),
					),
				)
			}
		}
	}

	var usersEmails []string
	for _, user := range plan.Users {

		if user.Email.IsNull() || user.Email.IsUnknown() || user.Email.ValueString() == "" {
			continue
		}

		if slices.Contains(usersEmails, user.Email.ValueString()) {
			resp.Diagnostics.AddError(
				"Validate users of access group",
				fmt.Sprintf(
					"User '%s' duplicated in access group. Please remove all duplicates",
					user.Email.ValueString(),
				),
			)
			continue
		}
		usersEmails = append(usersEmails, user.Email.ValueString())
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r AccessGroupResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state AccessGroupModelV1
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	if state.Name.IsNull() {
		if state.FullName.IsNull() {
			resp.Diagnostics.AddError(
				"Import access group resource",
				"You must set full name of group.",
			)
			return
		}

		accessGroup, err := sources.GetAccessGroupByName(
			r.client.Creds,
			r.client.ProjectName,
			state.Domain.ValueString(),
			state.FullName.ValueString(),
		)
		if err != nil {
			resp.Diagnostics.AddError(
				"Import access group resource",
				fmt.Sprintf(
					"Can't find access group by name '%s'.\nError:%s",
					state.FullName.ValueString(), err.Error(),
				),
			)
			return
		}

		state.Name = types.StringValue(accessGroup.Name)
		state.FullName = types.StringValue(accessGroup.Name)
		state.Domain = types.StringValue(accessGroup.Domain)
		state.Description = types.StringValue(accessGroup.Description)
		state.GroupDN = types.StringValue(accessGroup.GroupDn)
		state.Purpose = types.StringValue(accessGroup.Purpose.Code)
		state.AccountsType = types.StringValue(accessGroup.AccountsType)
	}

	accessGroup, err := sources.GetAccessGroupByName(
		r.client.Creds,
		r.client.ProjectName,
		state.Domain.ValueString(),
		state.FullName.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Read access group resource",
			fmt.Sprintf(
				"Can't read group by name '%s'.\nError:%s",
				state.FullName.ValueString(), err.Error(),
			),
		)
		return
	}

	users, err := sources.GetUsersByGroup(
		r.client.Creds,
		state.FullName.ValueString(),
		r.client.ProjectName,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Read access group resource",
			fmt.Sprintf(
				"Reading users of group with name '%s' ended with error.\nError:%s",
				state.FullName.ValueString(),
				err.Error(),
			),
		)
		return
	}

	var usersModels []AccessGroupUserModelV1
	for _, user := range users {
		usersModels = append(usersModels, AccessGroupUserModelV1{
			ID:         types.StringValue(user.ID),
			Username:   types.StringValue(user.Username),
			Name:       types.StringValue(user.Name),
			Email:      types.StringValue(user.Email),
			UniqueName: types.StringValue(user.UniqueName),
		})
	}

	state.Users = usersModels
	state.Description = types.StringValue(accessGroup.Description)
	state.Purpose = types.StringValue(accessGroup.Purpose.Code)
	state.AccountsType = types.StringValue(accessGroup.AccountsType)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r AccessGroupResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan AccessGroupModelV1
	var state AccessGroupModelV1

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	accessGroup, err := sources.GetAccessGroupByName(
		r.client.Creds,
		r.client.ProjectName,
		state.Domain.ValueString(),
		state.FullName.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Update access group resource",
			fmt.Sprintf(
				"Updating access group with name '%s' ended with error.\nError:%s",
				accessGroup.Name, err.Error(),
			),
		)
		return
	}

	if plan.Description != state.Description {
		r.updateGroupDescription(ctx, accessGroup, &plan, resp)
	}

	if isUsersUpdated(&plan, &state) {
		r.updateUsers(ctx, accessGroup, &plan, &state, resp)
	}
}

func (r AccessGroupResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state AccessGroupModelV1
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)

	accessGroup, err := sources.GetAccessGroupByName(
		r.client.Creds,
		r.client.ProjectName,
		state.Domain.ValueString(),
		state.FullName.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Delete access group resource",
			fmt.Sprintf(
				"Deleting access group with name '%s' ended with error.\nError:%s",
				accessGroup.Name, err.Error(),
			),
		)
		return
	}

	err = accessGroup.Delete()
	if err != nil {
		resp.Diagnostics.AddError(
			"Delete access group resource",
			fmt.Sprintf(
				"Deleting access group with name '%s' ended with error.\nError:%s",
				accessGroup.Name, err.Error(),
			),
		)
	}
}

func (r AccessGroupResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("full_name"), req, resp)
}

func (r AccessGroupResource) updateGroupDescription(
	ctx context.Context,
	accessGroup sources.AccessGroup,
	plan *AccessGroupModelV1,
	resp *resource.UpdateResponse,
) {
	err := accessGroup.EditDescription(plan.Description.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Update access group resource",
			fmt.Sprintf(
				"Updating access group with name '%s' ended with error.\nError:%s",
				accessGroup.Name, err.Error(),
			),
		)
		return
	}
	resp.Diagnostics.Append(
		resp.State.SetAttribute(ctx, path.Root("description"), plan.Description)...,
	)
}

func (r AccessGroupResource) updateUsers(
	ctx context.Context,
	accessGroup sources.AccessGroup,
	plan *AccessGroupModelV1,
	state *AccessGroupModelV1,
	resp *resource.UpdateResponse,
) {
	// prepare list of users to remove from group
	var toRemoveUniqueNames []string
	for _, stateUser := range state.Users {

		found := false
		for _, planUser := range plan.Users {
			if stateUser.UniqueName.ValueString() == planUser.UniqueName.ValueString() {
				found = true
			}
		}
		if !found {
			toRemoveUniqueNames = append(toRemoveUniqueNames, stateUser.UniqueName.ValueString())
		}
	}

	// prepare list of users for add to group
	var toAddedUniqueNames []string
	for _, planUser := range plan.Users {

		found := false
		for _, stateUser := range state.Users {
			if stateUser.UniqueName.ValueString() == planUser.UniqueName.ValueString() {
				found = true
			}
		}
		if !found {
			toAddedUniqueNames = append(toAddedUniqueNames, planUser.UniqueName.ValueString())
		}
	}

	// remove users
	if len(toRemoveUniqueNames) > 0 {
		err := accessGroup.RemoveUsers(toRemoveUniqueNames)
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Update access group resource",
				fmt.Sprintf(
					"Updating access group with name '%s' ended with error.\nError:%s",
					accessGroup.Name, err.Error(),
				),
			)
		}
	}

	// added users
	if len(toAddedUniqueNames) > 0 {
		err := accessGroup.AddUsers(toAddedUniqueNames)
		if err != nil {
			resp.Diagnostics.AddWarning(
				"Update access group resource",
				fmt.Sprintf(
					"Updating access group with name '%s' ended with error.\nError:%s",
					accessGroup.Name, err.Error(),
				),
			)
		}
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("users"), plan.Users)...)
}

func isUsersUpdated(plan, state *AccessGroupModelV1) bool {

	// check that not added by side
	for _, state := range state.Users {
		found := false
		for _, plan := range plan.Users {
			if state.UniqueName == plan.UniqueName {
				found = true
			}
		}
		if !found {
			return true
		}
	}

	// check that not removed users by side
	for _, plan := range plan.Users {
		found := false
		for _, state := range state.Users {
			if state.UniqueName == plan.UniqueName {
				found = true
			}
		}
		if !found {
			return true
		}
	}

	return false
}

func (r AccessGroupResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		// State upgrade implementation from 0 (prior state version) to 1 (Schema.Version)
		0: {
			PriorSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "Name of group",
						Required:            true,
						Validators: []validator.String{
							stringvalidator.RegexMatches(
								regexp.MustCompile("^[a-z0-9-]+$"),
								"must only contain: lowercase characters, numbers and '-'",
							),
						},
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"full_name": schema.StringAttribute{
						MarkdownDescription: "Name of group",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"group_dn": schema.StringAttribute{
						MarkdownDescription: "Name of group",
						Computed:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"domain": schema.StringAttribute{
						MarkdownDescription: "Domain of group",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"description": schema.StringAttribute{
						MarkdownDescription: "Description of group",
						Required:            true,
					},
					"users": schema.SetNestedAttribute{
						MarkdownDescription: "Structure of information about ad authentication",
						Optional:            true,
						Computed:            true,
						PlanModifiers: []planmodifier.Set{
							setplanmodifier.UseStateForUnknown(),
						},
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									MarkdownDescription: "Role of authentication",
									Optional:            true,
									Computed:            true,
								},
								"username": schema.StringAttribute{
									MarkdownDescription: "Role of authentication",
									Optional:            true,
									Computed:            true,
								},
								"name": schema.StringAttribute{
									MarkdownDescription: "Role of authentication",
									Optional:            true,
									Computed:            true,
								},
								"email": schema.StringAttribute{
									MarkdownDescription: "Role of authentication",
									Optional:            true,
									Computed:            true,
								},
								"unique_name": schema.StringAttribute{
									MarkdownDescription: "Role of authentication",
									Optional:            true,
									Computed:            true,
								},
							},
						},
					},
				},
			},
			StateUpgrader: func(
				ctx context.Context,
				req resource.UpgradeStateRequest,
				resp *resource.UpgradeStateResponse,
			) {
				var priorState AccessGroupModelV0
				resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)

				if resp.Diagnostics.HasError() {
					return
				}

				var upgradedUsers []AccessGroupUserModelV1
				for _, u := range priorState.Users {
					upgradedUsers = append(upgradedUsers, AccessGroupUserModelV1{
						ID:         u.ID,
						Username:   u.Username,
						Name:       u.Name,
						Email:      u.Email,
						UniqueName: u.UniqueName,
					})
				}
				upgradedStateData := AccessGroupModelV1{
					Name:         priorState.Name,
					FullName:     priorState.FullName,
					GroupDN:      priorState.GroupDN,
					Domain:       priorState.Domain,
					Description:  priorState.Description,
					Users:        upgradedUsers,
					Purpose:      types.StringValue("compute"),
					AccountsType: types.StringValue("service-accounts"),
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, upgradedStateData)...)
			},
		},
	}
}
