package orders

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/requests"
)

type OpenMessagingOrder struct {
	Order
	Attrs OpenMessagingAttrs `json:"attrs"`
}

type OpenMessagingAttrs struct {
	Path       string `json:"path"`
	Size       int    `json:"size"`
	FileSystem string `json:"file_system"`

	AdIntegration    bool                `json:"ad_integration"`
	AvailabilityZone string              `json:"availability_zone"`
	DefaultNic       entities.DefaultNic `json:"default_nic"`
	Domain           string              `json:"domain"`
	Flavor           entities.Flavor     `json:"flavor"`
	Platform         string              `json:"platform"`
	OnSupport        bool                `json:"on_support"`
	OsVersion        string              `json:"os_version"`

	AdminGroups     []string `json:"admin_groups"`
	SuperuserGroups []string `json:"superuser_groups"`
	UserGroups      []string `json:"user_groups"`

	CreatedWithOpenTofu bool `json:"created_with_opentofu"`
}

type OpenMessagingVerticalScalingAttrs struct {
	ReadIT                 bool            `json:"read_it"`
	Flavor                 entities.Flavor `json:"flavor"`
	CurrentFlavor          string          `json:"current_flavor"`
	StateServiceRAM        int64           `json:"state_service_ram"`
	StateServiceCPU        int64           `json:"state_service_cpu"`
	StateServiceFlavorName string          `json:"state_service_flavor_name"`
	ChangeRequest          bool            `json:"change_request,omitempty"` // Согласованное ЗНИ (для PROD)
}

func NewOpenMessaging(
	creds *auth.Credentials,
	projectName,
	productID string,
	attrs OpenMessagingAttrs,
) *OpenMessagingOrder {
	return &OpenMessagingOrder{
		Attrs: attrs,
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
	}
}

func (o *OpenMessagingOrder) GetAttrs() interface{} {
	return o.Attrs
}

func (o *OpenMessagingOrder) Create(p CreateOrderPayload) error {

	payload, err := o.createPayload(p, o)
	if err != nil {
		return err
	}

	uri := fmt.Sprintf("order-service/api/v1/projects/%s/orders", o.ProjectName)
	resp, err := requests.SendRequest(o.Creds.AccessToken, uri, "POST", payload, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	trimmedBody := strings.Trim(string(body), "[]")
	err = json.Unmarshal([]byte(trimmedBody), o)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(30)
	if err != nil {
		return err
	}
	return nil
}

func (o *OpenMessagingOrder) VerticalScaling(attrs OpenMessagingVerticalScalingAttrs) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrsData := map[string]interface{}{
		"read_it":                   attrs.ReadIT,
		"flavor":                    attrs.Flavor,
		"current_flavor":            attrs.CurrentFlavor,
		"state_service_ram":         attrs.StateServiceRAM,
		"state_service_cpu":         attrs.StateServiceCPU,
		"state_service_flavor_name": attrs.StateServiceFlavorName,
		"change_request":            attrs.ChangeRequest,
	}

	data := map[string]interface{}{
		"item_id": itemID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrsData),
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("openmessaging_vertical_scaling_release")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}
	return nil
}

func (o *OpenMessagingOrder) UpdateAccessGroup(toUpdate entities.ADLogonGrants, env string) error {

	vmItems, err := o.GetVMItems()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"members": toUpdate.Groups,
		"role":    toUpdate.Role,
	}

	data := map[string]interface{}{
		"item_id": vmItems[0].ID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	actionName := "openmessaging_vm_acls_set"
	if strings.EqualFold(env, "dev") {
		actionName = "vm_acls_set_linux"
	}

	uri := o.generateOrderdActionUri(actionName)

	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}

	return nil
}

func (o *OpenMessagingOrder) AddAccessGroup(toAdd entities.ADLogonGrants, permissions []string, env string) error {

	vmItems, err := o.GetVMItems()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"members": toAdd.Groups,
		"role": map[string]interface{}{
			"group_name":  toAdd.Role,
			"permissions": permissions,
		},
	}

	data := map[string]interface{}{
		"item_id": vmItems[0].ID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	actionName := "openmessaging_vm_acls_add_with_parent"
	if strings.EqualFold(env, "dev") {
		actionName = "vm_acls_add_with_parent"
	}

	uri := o.generateOrderdActionUri(actionName)

	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}

	return nil
}

func (o *OpenMessagingOrder) DeleteAccessGroup(toDelete entities.ADLogonGrants, env string) error {

	vmItems, err := o.GetVMItems()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"members": toDelete.Groups,
		"role":    toDelete.Role,
	}

	data := map[string]interface{}{
		"item_id": vmItems[0].ID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	actionName := "openmessaging_vm_acls_remove"
	if strings.EqualFold(env, "dev") {
		actionName = "vm_acls_remove"
	}

	uri := o.generateOrderdActionUri(actionName)

	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}

	return nil
}

func (o *OpenMessagingOrder) GetRoleGroups(role string) ([]string, error) {
	vmItems, err := o.GetVMItems()
	if err != nil {
		return nil, fmt.Errorf("failed to get VM items: %w", err)
	}

	if len(vmItems) == 0 {
		return nil, fmt.Errorf("no VM items available")
	}

	acls := vmItems[0].Data.ACLs

	found := false
	groups := []string{}
	for _, acl := range acls {
		if acl.Role == role {
			found = true
			groups = append(groups, acl.Members...)
		}
	}

	if !found {
		return nil, nil
	}

	return groups, nil
}
