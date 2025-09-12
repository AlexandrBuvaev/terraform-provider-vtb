package provider

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/orders"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &ArtemisAddressPolicyResource{}
	_ resource.ResourceWithImportState = &ArtemisAddressPolicyResource{}
	_ resource.ResourceWithModifyPlan  = &ArtemisAddressPolicyResource{}
)

type ArtemisAddressPolicyResource struct {
	provider *VTBCloudProvider
}

func NewArtemisAddresPolicyResource(p *VTBCloudProvider) ArtemisAddressPolicyResource {
	return ArtemisAddressPolicyResource{
		provider: p,
	}
}

func (r ArtemisAddressPolicyResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = "vtb_artemis_address_policy"
}

type ArtemisAddressPolicyListModel struct {
	OrderID           types.String                `tfsdk:"vtb_artemis_order_id"`
	AddressPolicyList []ArtemisAddressPolicyModel `tfsdk:"address_policy_list"`
}

type ArtemisAddressPolicyModel struct {
	AddrPrefix              types.String `tfsdk:"address_prefix"`
	AddrName                types.String `tfsdk:"address_name"`
	AddrFullPolicy          types.String `tfsdk:"address_full_policy"`
	MaxSize                 types.String `tfsdk:"max_size"`
	SlowConsumerCheckPeriod types.Int64  `tfsdk:"slow_consumer_check_period"`
	SlowConsumerPolicy      types.String `tfsdk:"slow_consumer_policy"`
	SlowConsumerThreshold   types.Int64  `tfsdk:"slow_consumer_threshold"`

	MinExpiryDelay types.Int64 `tfsdk:"min_expiry_delay"`
	MaxExpiryDelay types.Int64 `tfsdk:"max_expiry_delay"`
}

type SecurityPolicyRolesModel struct {
	Consumers types.List `tfsdk:"consumers"`
	Producers types.List `tfsdk:"producers"`
}

func (r ArtemisAddressPolicyResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description:         "Управление адресами и политиками в кластере VTB Artemis.",
		MarkdownDescription: "Управление адресами и политиками в кластере VTB Artemis.",
		Attributes: map[string]schema.Attribute{
			"vtb_artemis_order_id": schema.StringAttribute{
				Required:            true,
				Description:         "Идентификатор заказа существующего кластера VTB Artemis.",
				MarkdownDescription: "Идентификатор заказа существующего кластера VTB Artemis.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"address_policy_list": schema.SetNestedAttribute{
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"address_prefix": schema.StringAttribute{
							Required:            true,
							Description:         `Адресный префикс, должен совпадать с одним из: "DC.", "DC.service.", "DC.client.".`,
							MarkdownDescription: `Адресный префикс, должен совпадать с одним из: "DC.", "DC.service.", "DC.client.".`,
							Validators: []validator.String{
								stringvalidator.OneOf("DC.", "DC.client.", "DC.service."),
							},
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"address_name": schema.StringAttribute{
							Required:            true,
							Description:         "Название адреса.",
							MarkdownDescription: "Название адреса. ",
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(2),
								stringvalidator.LengthAtMost(100),
							},
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"address_full_policy": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								stringvalidator.OneOf("FAIL", "BLOCK"),
							},
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"max_size": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								stringvalidator.OneOf("100Mb", "150Mb", "200Mb"),
							},
						},
						"slow_consumer_check_period": schema.Int64Attribute{
							Required: true,
							Validators: []validator.Int64{
								int64validator.AtLeast(1),
								int64validator.AtMost(30),
							},
						},
						"slow_consumer_policy": schema.StringAttribute{
							Required: true,
							Validators: []validator.String{
								stringvalidator.OneOf("KILL", "NOTIFY"),
							},
						},
						"slow_consumer_threshold": schema.Int64Attribute{
							Required: true,
							Validators: []validator.Int64{
								int64validator.AtLeast(-1),
								int64validator.AtMost(30),
							},
						},
						"min_expiry_delay": schema.Int64Attribute{
							Optional: true,
							Computed: true,
							Default:  int64default.StaticInt64(-1),
						},
						"max_expiry_delay": schema.Int64Attribute{
							Optional: true,
							Computed: true,
							Default:  int64default.StaticInt64(-1),
						},
					},
				},
			},
		},
	}
}

func (r ArtemisAddressPolicyResource) ModifyPlan(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan ArtemisAddressPolicyListModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	var addressNames []string
	for _, addressPolicy := range plan.AddressPolicyList {
		addressNames = append(addressNames, addressPolicy.AddrName.ValueString())
	}

	if containsDuplicate(addressNames) {
		resp.Diagnostics.AddError(
			MODIFY_PLAN_FAIL,
			fmt.Sprintf("AddrName must be a unique, addressNames: %v", addressNames),
		)
	}

	for _, addressPolicy := range plan.AddressPolicyList {
		if addressPolicy.MaxExpiryDelay.ValueInt64() == -1 && addressPolicy.MinExpiryDelay.ValueInt64() != -1 {
			resp.Diagnostics.AddError(
				MODIFY_PLAN_FAIL,
				fmt.Sprintf("max_expiry_delay and min_expiry_delay must"+
					"both be -1 or both are not -1,"+
					"max_expiry_delay: %v, min_expiry_delay: %v", addressPolicy.MaxExpiryDelay.ValueInt64(), addressPolicy.MinExpiryDelay.ValueInt64()),
			)
			return
		}
		if addressPolicy.MinExpiryDelay.ValueInt64() == -1 && addressPolicy.MaxExpiryDelay.ValueInt64() != -1 {
			resp.Diagnostics.AddError(
				MODIFY_PLAN_FAIL,
				fmt.Sprintf("max_expiry_delay and min_expiry_delay must"+
					"both be -1 or both are not -1,"+
					"max_expiry_delay: %v, min_expiry_delay: %v", addressPolicy.MaxExpiryDelay.ValueInt64(), addressPolicy.MinExpiryDelay.ValueInt64()),
			)
			return
		}
		if addressPolicy.MinExpiryDelay.ValueInt64() > addressPolicy.MaxExpiryDelay.ValueInt64() {
			resp.Diagnostics.AddError(
				MODIFY_PLAN_FAIL,
				fmt.Sprintf("max_expiry_delay not less than min_expiry_delay"+
					"max_expiry_delay: %v, min_expiry_delay: %v", addressPolicy.MaxExpiryDelay.ValueInt64(), addressPolicy.MinExpiryDelay.ValueInt64(),
				),
			)
			return
		}

	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)

}

func (r ArtemisAddressPolicyResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("vtb_artemis_order_id"), req, resp)
}

func (r ArtemisAddressPolicyResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan ArtemisAddressPolicyListModel
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

	var addressPolicyToCreate []orders.CascadeAddressCreateAttrs
	for _, address := range plan.AddressPolicyList {
		addressToCreate := orders.CascadeAddressCreateAttrs{
			CascadeAddressPrefix:    address.AddrPrefix.ValueString(),
			CascadeAddressName:      address.AddrName.ValueString(),
			AddressFullPolicy:       address.AddrFullPolicy.ValueString(),
			MaxSizeBytes:            address.MaxSize.ValueString(),
			SlowConsumerPolicy:      address.SlowConsumerPolicy.ValueString(),
			SlowConsumerCheckPeriod: address.SlowConsumerCheckPeriod.ValueInt64(),
			SlowConsumerThreshold:   address.SlowConsumerThreshold.ValueInt64(),
			MaxExpiryDelay:          address.MaxExpiryDelay.ValueInt64(),
			MinExpiryDelay:          address.MinExpiryDelay.ValueInt64(),
		}

		// Check max_expiry_delay and min_expiry_delay
		maxDelayDefault := address.MaxExpiryDelay.ValueInt64() == -1
		minDelayDefault := address.MinExpiryDelay.ValueInt64() == -1
		if minDelayDefault && maxDelayDefault {
			addressToCreate.NoExpiryDelay = true
		} else {
			addressToCreate.NoExpiryDelay = false
		}
		addressPolicyToCreate = append(addressPolicyToCreate, addressToCreate)
	}

	err = artemis.CascadeAdressCreate(addressPolicyToCreate)
	if err != nil {
		resp.Diagnostics.AddError(CREATE_RES_FAIL, err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r ArtemisAddressPolicyResource) Read(
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

	addrPolicies := parentItem.Data.Config.(entities.VTBArtemisItemConfig).AddressPolicyList

	state := &ArtemisAddressPolicyListModel{
		OrderID:           orderID,
		AddressPolicyList: []ArtemisAddressPolicyModel{},
	}
	for _, addressPolicy := range addrPolicies {
		addressPolicyToState := ArtemisAddressPolicyModel{
			AddrPrefix:              types.StringValue(getAddressPrefix(addressPolicy.AddressName)),
			AddrName:                types.StringValue(getAddressName(addressPolicy.AddressName)),
			AddrFullPolicy:          types.StringValue(addressPolicy.AddressFullPolicy),
			MaxSize:                 types.StringValue(addressPolicy.MaxSizeBytes),
			SlowConsumerPolicy:      types.StringValue(addressPolicy.SlowConsumerPolicy),
			SlowConsumerCheckPeriod: types.Int64Value(addressPolicy.SlowConsumerCheckPeriod),
			SlowConsumerThreshold:   types.Int64Value(addressPolicy.SlowConsumerThreshold),
			MinExpiryDelay:          types.Int64Value(addressPolicy.MinExpiryDelay),
			MaxExpiryDelay:          types.Int64Value(addressPolicy.MaxExpiryDelay),
		}
		state.AddressPolicyList = append(state.AddressPolicyList, addressPolicyToState)
	}

	if len(state.AddressPolicyList) == 0 {
		resp.Diagnostics.AddWarning(
			READ_RES_FAIL,
			fmt.Sprintf(
				"No address policy for VTB Artemis (order_id=%s)",
				orderID.ValueString(),
			),
		)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r ArtemisAddressPolicyResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan, state ArtemisAddressPolicyListModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	planAddressPolicyMap := make(map[string]ArtemisAddressPolicyModel)
	stateAddressPolicyMap := make(map[string]ArtemisAddressPolicyModel)

	for _, addressPolicy := range plan.AddressPolicyList {
		addressName := fmt.Sprintf("%s%s", addressPolicy.AddrPrefix.ValueString(), addressPolicy.AddrName.ValueString())
		planAddressPolicyMap[addressName] = addressPolicy
	}

	for _, addressPolicy := range state.AddressPolicyList {
		addressName := fmt.Sprintf("%s%s", addressPolicy.AddrPrefix.ValueString(), addressPolicy.AddrName.ValueString())
		stateAddressPolicyMap[addressName] = addressPolicy
	}
	var addressPolicyToAdd []orders.CascadeAddressCreateAttrs
	var addressPolicyToUpdate []orders.CascadeAddressUpdateAttrs
	var addressPolicyToDelete []string

	for addressName, addressPolicy := range planAddressPolicyMap {
		if existingPolicy, exists := stateAddressPolicyMap[addressName]; !exists {
			addressToCreate := orders.CascadeAddressCreateAttrs{
				CascadeAddressPrefix:    addressPolicy.AddrPrefix.ValueString(),
				CascadeAddressName:      addressPolicy.AddrName.ValueString(),
				AddressFullPolicy:       addressPolicy.AddrFullPolicy.ValueString(),
				MaxSizeBytes:            addressPolicy.MaxSize.ValueString(),
				SlowConsumerPolicy:      addressPolicy.SlowConsumerPolicy.ValueString(),
				SlowConsumerCheckPeriod: addressPolicy.SlowConsumerCheckPeriod.ValueInt64(),
				SlowConsumerThreshold:   addressPolicy.SlowConsumerThreshold.ValueInt64(),
				MaxExpiryDelay:          addressPolicy.MaxExpiryDelay.ValueInt64(),
				MinExpiryDelay:          addressPolicy.MinExpiryDelay.ValueInt64(),
			}

			// Check max_expiry_delay and min_expiry_delay
			maxDelayDefault := addressPolicy.MaxExpiryDelay.ValueInt64() == -1
			minDelayDefault := addressPolicy.MinExpiryDelay.ValueInt64() == -1
			if minDelayDefault && maxDelayDefault {
				addressToCreate.NoExpiryDelay = true
				addressToCreate.MinExpiryDelay = -1
				addressToCreate.MaxExpiryDelay = -1
			} else {
				addressToCreate.NoExpiryDelay = false
				addressToCreate.MinExpiryDelay = addressPolicy.MinExpiryDelay.ValueInt64()
				addressToCreate.MaxExpiryDelay = addressPolicy.MaxExpiryDelay.ValueInt64()
			}
			addressPolicyToAdd = append(addressPolicyToAdd, addressToCreate)
		} else if existingPolicy != addressPolicy {
			addressToUpdate := orders.CascadeAddressUpdateAttrs{
				CascadeAddressName:      addressName,
				AddressFullPolicy:       addressPolicy.AddrFullPolicy.ValueString(),
				SlowConsumerPolicy:      addressPolicy.SlowConsumerPolicy.ValueString(),
				SlowConsumerCheckPeriod: addressPolicy.SlowConsumerCheckPeriod.ValueInt64(),
				SlowConsumerThreshold:   addressPolicy.SlowConsumerThreshold.ValueInt64(),
				MaxSize:                 addressPolicy.MaxSize.ValueString(),
				MaxSizeBytes:            addressPolicy.MaxSize.ValueString(),
			}
			addressFulPolicyChanged := !existingPolicy.AddrFullPolicy.Equal(addressPolicy.AddrFullPolicy)

			if addressFulPolicyChanged {
				resp.Diagnostics.AddError(
					UPDATE_RES_FAIL,
					fmt.Sprintf("Address_full_policy is immutable field. "+
						"plan_address_full_policy: %s, state_address_full_policy: %s",
						addressPolicy.AddrFullPolicy.ValueString(), existingPolicy.AddrFullPolicy.ValueString(),
					),
				)
				return
			}

			minDelayChanged := !existingPolicy.MinExpiryDelay.Equal(addressPolicy.MinExpiryDelay)
			maxDelayChanged := !existingPolicy.MaxExpiryDelay.Equal(addressPolicy.MaxExpiryDelay)

			if minDelayChanged && maxDelayChanged {
				addressToUpdate.ChangeExpiryDelay = true

				if (addressPolicy.MinExpiryDelay.ValueInt64() == -1) && (addressPolicy.MaxExpiryDelay.ValueInt64() == -1) {
					addressToUpdate.NewExpiryDelay = &orders.NewExpiryDelay{
						NoExpiryDelay:     true,
						NewMinExpiryDelay: 1000,
						NewMaxExpiryDelay: 60000,
					}
				} else {
					addressToUpdate.MinExpiryDelay = existingPolicy.MinExpiryDelay.ValueInt64()
					addressToUpdate.MaxExpiryDelay = existingPolicy.MaxExpiryDelay.ValueInt64()
					addressToUpdate.NewExpiryDelay = &orders.NewExpiryDelay{
						NoExpiryDelay:     false,
						NewMinExpiryDelay: addressPolicy.MinExpiryDelay.ValueInt64(),
						NewMaxExpiryDelay: addressPolicy.MaxExpiryDelay.ValueInt64(),
					}
				}
			} else {
				addressToUpdate.NewExpiryDelay = nil
				addressToUpdate.ChangeExpiryDelay = false
				addressToUpdate.MinExpiryDelay = addressPolicy.MinExpiryDelay.ValueInt64()
				addressToUpdate.MaxExpiryDelay = addressPolicy.MaxExpiryDelay.ValueInt64()
			}

			addressPolicyToUpdate = append(addressPolicyToUpdate, addressToUpdate)
		}
	}

	for addressName := range stateAddressPolicyMap {
		if _, exists := planAddressPolicyMap[addressName]; !exists {
			addressPolicyToDelete = append(addressPolicyToDelete, addressName)
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

	if len(addressPolicyToAdd) > 0 {
		err := artemis.CascadeAdressCreate(addressPolicyToAdd)
		if err != nil {
			resp.Diagnostics.AddError(
				UPDATE_RES_FAIL,
				fmt.Sprintf("error while create new address_policy: %s", err.Error()),
			)
		}
	}

	if len(addressPolicyToUpdate) > 0 {
		for _, addressPolicy := range addressPolicyToUpdate {
			err := artemis.CascadeAdressUpdate(addressPolicy)
			if err != nil {
				resp.Diagnostics.AddError(
					UPDATE_RES_FAIL,
					fmt.Sprintf("error while update address_policy: %s", err.Error()),
				)
			}
		}
	}

	if len(addressPolicyToDelete) > 0 {
		err = artemis.CascadeAdressDelete(addressPolicyToDelete)
		if err != nil {
			resp.Diagnostics.AddError(
				UPDATE_RES_FAIL,
				fmt.Sprintf("error while delete address_policy: %s", err.Error()),
			)
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r ArtemisAddressPolicyResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state ArtemisAddressPolicyListModel
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

	if len(state.AddressPolicyList) == 0 {
		resp.Diagnostics.AddWarning(
			READ_RES_FAIL,
			fmt.Sprintf(
				"No address policy for VTB Artemis (order_id=%s)",
				state.OrderID.ValueString(),
			),
		)
		return
	}
	var AddressPolicyToDelete []string
	for _, addressPolicy := range state.AddressPolicyList {
		addressName := fmt.Sprintf("%s%s", addressPolicy.AddrPrefix.ValueString(), addressPolicy.AddrName.ValueString())
		AddressPolicyToDelete = append(AddressPolicyToDelete, addressName)
	}

	err = artemis.CascadeAdressDelete(AddressPolicyToDelete)
	if err != nil {
		resp.Diagnostics.AddError(DELETE_RES_FAIL, err.Error())
		return
	}
}

func getAddressPrefix(addressName string) string {
	prefixes := []string{"DC.service.", "DC.client.", "DC."}

	for _, prefix := range prefixes {
		if strings.HasPrefix(addressName, prefix) {
			return prefix
		}
	}
	return ""
}

func getAddressName(fullAddressName string) string {
	prefixes := []string{"DC.service.", "DC.client.", "DC."}

	for _, prefix := range prefixes {
		if strings.HasPrefix(fullAddressName, prefix) {
			return strings.TrimPrefix(fullAddressName, prefix)
		}
	}
	return fullAddressName
}
