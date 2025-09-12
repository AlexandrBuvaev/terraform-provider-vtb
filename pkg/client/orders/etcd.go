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

type EtcdAttrs struct {
	BasicAttrs
	UseSsl          bool   `json:"use_ssl"`
	EtcdUser        string `json:"etcd_user"`
	OnBackup        bool   `json:"on_backup"`
	NodesCount      int    `json:"nodes_count"`
	ClusterName     string `json:"cluster_name"`
	EtcdVersion     string `json:"etcd_version"`
	EtcdPassword    string `json:"etcd_password"`
	GeoDistribution bool   `json:"geo_distribution"`
	Layout          string `json:"layout"`
}

type EtcdOrder struct {
	Order
	Attrs EtcdAttrs `json:"attrs"`
}

func NewEtcdOrder(
	creds *auth.Credentials,
	projectName,
	productID string,
	attrs EtcdAttrs,
) *EtcdOrder {
	return &EtcdOrder{
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
		Attrs: attrs,
	}
}

func (o *EtcdOrder) GetAttrs() interface{} {
	return o.Attrs
}

func (o *EtcdOrder) Create(p CreateOrderPayload) error {
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

func (o *EtcdOrder) Delete() error {
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
	_, err = requests.SendRequest(o.Creds.AccessToken, o.generateOrderdActionUri("delete_etcd_cluster"), "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}

	return err
}

func (o *EtcdOrder) GetUsers() ([]entities.EtcdUser, error) {
	item, err := o.GetParentItem()
	if err != nil {
		return nil, err
	}

	config := item.Data.Config.(entities.EtcdlItemConfig)
	return config.EtcdUser, nil
}

func (o *EtcdOrder) GetUser(username string) (*entities.EtcdUser, error) {
	etcdUsers, err := o.GetUsers()
	if err != nil {
		return nil, err
	}
	for _, etcdUser := range etcdUsers {
		if etcdUser.Username == username {
			return &etcdUser, nil
		}
	}
	return nil, fmt.Errorf("no user with name %s", username)
}

func (o *EtcdOrder) GetDefaultUser() (*entities.EtcdUser, error) {
	etcdUsers, err := o.GetUsers()
	if err != nil {
		return nil, err
	}
	if len(etcdUsers) == 0 {
		return nil, fmt.Errorf("list of users in Etcd config is empty")
	}
	return &etcdUsers[0], nil
}

func (o *EtcdOrder) ChangeUserPassword(username string, password string) (err error) {
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
		"etcd_user":     user.Username,
		"etcd_password": password,
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
	_, err = requests.SendRequest(o.Creds.AccessToken, o.generateOrderdActionUri("etcd_reset_user_pass_without_ssl"), "PATCH", payload, nil)
	if err != nil {
		return
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return
	}
	return
}

func (o *EtcdOrder) GetExtraMounts() (extraMounts []entities.ExtraMount, err error) {
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

func (o *EtcdOrder) GetExtraMount(path string) (*entities.ExtraMount, error) {
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

func (o *EtcdOrder) ExpandMountPointForVm(vmItemId string, path string, size int64) (err error) {

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

func (o *EtcdOrder) ChangeFlavorForVm(vmItemId string, flavor entities.Flavor) (err error) {

	attrs := map[string]interface{}{
		"flavor": flavor,
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

func (o *EtcdOrder) ChangeAccessGroupForVm(vmItemId string, changeAD entities.ADLogonGrants) error {

	attrs := map[string]interface{}{
		"members": changeAD.Groups,
		"role":    changeAD.Role,
	}

	data := map[string]interface{}{
		"item_id": vmItemId,
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

func (o *EtcdOrder) AddAccessGroupForVm(vmItemId string, members entities.ADLogonGrants, permissions []string) error {

	attrs := map[string]interface{}{
		"members": members.Groups,
		"role": map[string]interface{}{
			"group_name":  members.Role,
			"permissions": permissions,
		},
	}

	data := map[string]interface{}{
		"item_id": vmItemId,
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

func (o *EtcdOrder) DeleteAccessGroupForVm(vmitemId string, changeAD entities.ADLogonGrants) error {

	attrs := map[string]interface{}{
		"members": changeAD.Groups,
		"role":    changeAD.Role,
	}

	data := map[string]interface{}{
		"item_id": vmitemId,
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
