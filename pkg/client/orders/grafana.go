package orders

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/requests"
)

type GrafanaAttrs struct {
	BasicAttrs
	GrafanaUser         string `json:"users"`
	GrafanaUserPassword string `json:"users_password"`
	GrafanaVersion      string `json:"grafana_version"`
}

type GrafanaOrder struct {
	Order
	Attrs GrafanaAttrs `json:"attrs"`
}

func NewGrafanaOrder(
	creds *auth.Credentials,
	projectName,
	productID string,
	attrs GrafanaAttrs,
) *GrafanaOrder {
	return &GrafanaOrder{
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
		Attrs: attrs,
	}
}

func (o *GrafanaOrder) GetAttrs() interface{} {
	return o.Attrs
}

func (o *GrafanaOrder) Create(p CreateOrderPayload) error {
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

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}
	return nil
}

func (o *GrafanaOrder) Delete() error {
	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{}

	data := map[string]interface{}{
		"item_id": item.ID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = requests.SendRequest(o.Creds.AccessToken, o.generateOrderdActionUri("delete_two_layer"), "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}

	return err
}

func (o *GrafanaOrder) GetUsers() ([]entities.GrafanaUser, error) {
	item, err := o.GetParentItem()
	if err != nil {
		return nil, err
	}

	config := item.Data.Config.(entities.GrafanaItemConfig)
	return config.GrafanaUsers, nil
}

func (o *GrafanaOrder) GetUser(username string) (*entities.GrafanaUser, error) {
	grafanaUsers, err := o.GetUsers()
	if err != nil {
		return nil, err
	}
	for _, grafanaUser := range grafanaUsers {
		if grafanaUser.Username == username {
			return &grafanaUser, nil
		}
	}
	return nil, fmt.Errorf("no user with name %s", username)
}

func (o *GrafanaOrder) GetDefaultUser() (*entities.GrafanaUser, error) {
	grafanaUsers, err := o.GetUsers()
	if err != nil {
		return nil, err
	}
	if len(grafanaUsers) == 0 {
		return nil, fmt.Errorf("list of users in grafana config is empty")
	}
	return &grafanaUsers[0], nil
}

func (o *GrafanaOrder) ChangeUserPassword(username string, password string) (err error) {
	if err := o.requiredState("on"); err != nil {
		return err
	}
	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	user, err := o.GetUser(username)
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"user_name":      user.Username,
		"users_password": password,
	}

	data := map[string]interface{}{
		"item_id": item.ID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = requests.SendRequest(o.Creds.AccessToken, o.generateOrderdActionUri("reset_grafana_user_password"), "PATCH", payload, nil)
	if err != nil {
		return
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return
	}
	return
}

func (o *GrafanaOrder) GetExtraMount(path string) (*entities.ExtraMount, error) {
	vmItems, err := o.GetVMItems()
	if err != nil {
		return nil, err
	}
	var size int64
	var fileSystem string
	for _, vmItem := range vmItems {
		extraMounts := vmItem.Data.Config.(entities.VMItemConfig).ExtraMounts
		if len(extraMounts) == 0 {
			return nil, errors.New("list of extra mounts is empty")
		}
		for _, em := range extraMounts {
			if em.Mount == path {
				if size == 0 || size > int64(em.Size) {
					size = int64(em.Size)
					fileSystem = em.Fstype
				}
				break
			}
		}
	}
	return &entities.ExtraMount{
		Path:       path,
		Size:       size,
		FileSystem: fileSystem,
	}, nil
}

func (o *GrafanaOrder) ExpandMountPointForVm(vmItemId string, path string, size int64) (err error) {

	attrs := map[string]interface{}{
		"mount": path,
		"size":  size,
	}

	data := map[string]interface{}{
		"item_id": vmItemId,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	_, err = requests.SendRequest(o.Creds.AccessToken, o.generateOrderdActionUri("expand_mount_point_new"), "PATCH", payload, nil)
	if err != nil {
		return
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return
	}
	return
}

func (o *GrafanaOrder) GetExtraMounts() (extraMounts []entities.ExtraMount, err error) {
	vmItems, err := o.GetVMItems()

	if err != nil {
		return nil, err
	}

	vmExtraMount := vmItems[0].Data.Config.(entities.VMItemConfig).ExtraMounts

	for _, extraMount := range vmExtraMount {
		formatedExtraMount, err := o.GetExtraMount(extraMount.Mount)
		if err != nil {
			return nil, err
		}
		extraMounts = append(extraMounts, *formatedExtraMount)
	}

	return
}

func (o *GrafanaOrder) ChangeFlavor(flavor entities.Flavor) (err error) {
	vmItems, err := o.GetVMItems()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"check_agree": true,
		"flavor":      flavor,
	}

	data := map[string]interface{}{
		"item_id": vmItems[0].ID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	_, err = requests.SendRequest(o.Creds.AccessToken, o.generateOrderdActionUri("resize_vm"), "PATCH", payload, nil)
	if err != nil {
		return
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return
	}
	return
}

func (o *GrafanaOrder) ChangeAccessGroup(changeAD entities.ADLogonGrants) error {
	vmItems, err := o.GetVMItems()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"members": changeAD.Groups,
		"role":    changeAD.Role,
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

	_, err = requests.SendRequest(o.Creds.AccessToken, o.generateOrderdActionUri("vm_acls_set_linux"), "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}
	return err
}

func (o *GrafanaOrder) AddAccessGroup(members entities.ADLogonGrants, permissions []string) error {
	vmItems, err := o.GetVMItems()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"members": members.Groups,
		"role": map[string]interface{}{
			"group_name":  members.Role,
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

	_, err = requests.SendRequest(o.Creds.AccessToken, o.generateOrderdActionUri("vm_acls_add_with_parent"), "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}
	return err
}

func (o *GrafanaOrder) DeleteAccessGroup(changeAD entities.ADLogonGrants) error {
	vmItems, err := o.GetVMItems()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"members": changeAD.Groups,
		"role":    changeAD.Role,
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

	_, err = requests.SendRequest(o.Creds.AccessToken, o.generateOrderdActionUri("vm_acls_remove"), "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}
	return err
}
