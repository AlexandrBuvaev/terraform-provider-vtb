package provider

import (
	"context"
	"fmt"
	"regexp"

	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &ArtemisTuzResource{}
	_ resource.ResourceWithImportState = &ArtemisTuzResource{}
	_ resource.ResourceWithModifyPlan  = &ArtemisTuzResource{}
)

type ArtemisTuzResource struct {
	provider *VTBCloudProvider
}

func NewArtemisTuzResource(p *VTBCloudProvider) ArtemisTuzResource {
	return ArtemisTuzResource{
		provider: p,
	}
}

func (r ArtemisTuzResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = "vtb_artemis_tuz"
}

type VTBArtemisTuzListModel struct {
	OrderID types.String         `tfsdk:"vtb_artemis_order_id"`
	Users   []VTBArtemisTuzModel `tfsdk:"users"`
}

type VTBArtemisTuzModel struct {
	UserName      types.String `tfsdk:"user_name"`
	UserOwnerCert types.String `tfsdk:"user_owner_cert"`
}

func (r ArtemisTuzResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description:         "Управление техничискими учетными записями.",
		MarkdownDescription: "Управление техничискими учетными записями.",
		Attributes: map[string]schema.Attribute{
			"vtb_artemis_order_id": schema.StringAttribute{
				Required:            true,
				Description:         "Идентификатор заказа кластера VTB Artemis.",
				MarkdownDescription: "Идентификатор заказа кластера VTB Artemis.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"users": schema.SetNestedAttribute{
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"user_name": schema.StringAttribute{
							Required:            true,
							Description:         "Код пользователя. Должен быть уникальным.",
							MarkdownDescription: "Код пользователя. Должен быть уникальным.",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									regexp.MustCompile(`^[a-z][a-z0-9_@.]{0,48}[a-z0-9]$`),
									`Can contain lowercase Latin letters, numbers, and the punctuation characters '@', '_', '.' `,
								),
								stringvalidator.LengthAtMost(50),
							},
						},
						"user_owner_cert": schema.StringAttribute{
							Required:            true,
							Description:         "subject(owner) сертификата.",
							MarkdownDescription: "subject(owner) сертификата..",
							Validators: []validator.String{
								stringvalidator.RegexMatches(
									regexp.MustCompile(`^CN=[a-zA-Z0-9][a-zA-Z0-9,= .@_-]{0,248}[a-zA-Z0-9]$`),
									`Must start with CN=. Can contain uppercase/lowercase Latin letters, numbers,`+
										`and the following characters: ., @, -, ,, =, _,`+
										`and spaces (cannot start or end with a special character). The maximum length is 250 characters.`,
								),
							},
						},
					},
				},
			},
		},
	}
}

func (r ArtemisTuzResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("vtb_artemis_order_id"), req, resp)
}

func (r ArtemisTuzResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan VTBArtemisTuzListModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	var userNames, userOwnerCerts []string
	for _, user := range plan.Users {
		userNames = append(userNames, user.UserName.ValueString())
	}

	for _, user := range plan.Users {
		userOwnerCerts = append(userOwnerCerts, user.UserOwnerCert.ValueString())
	}

	if containsDuplicate(userNames) {
		resp.Diagnostics.AddError(
			MODIFY_PLAN_FAIL,
			fmt.Sprintf("user_name must be a unique, userNames: %v", userNames),
		)
		return
	}

	if containsDuplicate(userOwnerCerts) {
		resp.Diagnostics.AddError(
			MODIFY_PLAN_FAIL,
			fmt.Sprintf("user_owner_cert must be a unique, userOwnerCerts: %v", userOwnerCerts),
		)
		return
	}

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r ArtemisTuzResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan VTBArtemisTuzListModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	artemis, err := orders.GetArtemisOrder(
		r.provider.Creds,
		r.provider.ProjectName,
		plan.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(CREATE_RES_FAIL, err.Error())
		return
	}

	var usersToCreate []orders.ArtemisTUZ
	for _, user := range plan.Users {
		userToCreate := orders.ArtemisTUZ{
			Username:      user.UserName.ValueString(),
			UserOwnerCert: user.UserOwnerCert.ValueString(),
		}
		usersToCreate = append(usersToCreate, userToCreate)
	}

	err = artemis.CreateTUZ(usersToCreate)
	if err != nil {
		resp.Diagnostics.AddError(CREATE_RES_FAIL, err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r ArtemisTuzResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var orderID types.String
	req.State.GetAttribute(ctx, path.Root("vtb_artemis_order_id"), &orderID)

	artemis, err := orders.GetArtemisOrder(
		r.provider.Creds,
		r.provider.ProjectName,
		orderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(READ_RES_FAIL, err.Error())
		return
	}

	parentItem, err := artemis.GetParentItem()
	if err != nil {
		resp.Diagnostics.AddError(READ_RES_FAIL, err.Error())
		return
	}

	tuzList := parentItem.Data.Config.(entities.VTBArtemisItemConfig).TuzList

	state := &VTBArtemisTuzListModel{
		OrderID: orderID,
		Users:   []VTBArtemisTuzModel{},
	}

	for _, tuz := range tuzList {
		user := VTBArtemisTuzModel{
			UserName:      types.StringValue(tuz.UserName),
			UserOwnerCert: types.StringValue(tuz.UserOwnerCert),
		}
		state.Users = append(state.Users, user)
	}

	if len(state.Users) == 0 {
		resp.Diagnostics.AddWarning(
			READ_RES_FAIL,
			fmt.Sprintf(
				"No TUZ for VTB Artemis oreder (order_id=%s)",
				orderID.ValueString(),
			),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r ArtemisTuzResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state VTBArtemisTuzListModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	planUserMap := make(map[string]string)
	stateUserMap := make(map[string]string)

	for _, user := range plan.Users {
		planUserMap[user.UserName.ValueString()] = user.UserOwnerCert.ValueString()
	}

	for _, user := range state.Users {
		stateUserMap[user.UserName.ValueString()] = user.UserOwnerCert.ValueString()
	}

	var userToAdd, userToUpdate []orders.ArtemisTUZ
	var usersToDelete []string

	for _, user := range plan.Users {
		if cert, exists := stateUserMap[user.UserName.ValueString()]; !exists {
			userToAdd = append(userToAdd, orders.ArtemisTUZ{
				Username:      user.UserName.ValueString(),
				UserOwnerCert: user.UserOwnerCert.ValueString(),
			})
		} else if cert != user.UserOwnerCert.ValueString() {
			userToUpdate = append(userToUpdate, orders.ArtemisTUZ{
				Username:      user.UserName.ValueString(),
				UserOwnerCert: user.UserOwnerCert.ValueString(),
			})
		}
	}

	for _, user := range state.Users {
		if _, exists := planUserMap[user.UserName.ValueString()]; !exists {
			usersToDelete = append(usersToDelete, user.UserName.ValueString())
		}
	}

	artemis, err := orders.GetArtemisOrder(
		r.provider.Creds,
		r.provider.ProjectName,
		plan.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(UPDATE_RES_FAIL, err.Error())
		return
	}

	if len(usersToDelete) > 0 {
		err = artemis.DeleteTUZ(usersToDelete)
		if err != nil {
			resp.Diagnostics.AddError(
				UPDATE_RES_FAIL,
				fmt.Sprintf("error while update TUZ: %s", err.Error()),
			)
		}
	}

	if len(userToAdd) > 0 {
		err = artemis.CreateTUZ(userToAdd)
		if err != nil {
			resp.Diagnostics.AddError(
				UPDATE_RES_FAIL,
				fmt.Sprintf("error while update TUZ: %s", err.Error()),
			)
		}
	}

	if len(userToUpdate) > 0 {
		err = artemis.UpdateTUZ(userToUpdate)
		if err != nil {
			resp.Diagnostics.AddError(
				UPDATE_RES_FAIL,
				fmt.Sprintf("error while update TUZ: %s", err.Error()),
			)
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r ArtemisTuzResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state VTBArtemisTuzListModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	artemis, err := orders.GetArtemisOrder(
		r.provider.Creds,
		r.provider.ProjectName,
		state.OrderID.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(DELETE_RES_FAIL, err.Error())
		return
	}
	var usersToDelete []string
	for _, user := range state.Users {
		usersToDelete = append(usersToDelete, user.UserName.ValueString())
	}

	err = artemis.DeleteTUZ(usersToDelete)
	if err != nil {
		resp.Diagnostics.AddError(DELETE_RES_FAIL, err.Error())
		return
	}
}
