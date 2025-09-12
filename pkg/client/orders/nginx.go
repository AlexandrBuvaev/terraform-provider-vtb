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

type Nginx struct {
	Order
	Attrs NginxAttrs `json:"attrs"`
}

type NginxAttrs struct {
	BasicAttrs
	ExtraNics    []interface{} `json:"extra_nics"`
	NginxVersion string        `json:"nginx_version"`
	DevOpsGroup  string        `json:"devops_group"`
	InfoDevOps   interface{}   `json:"info_devops"`
}

func NewNginx(creds *auth.Credentials, projectName, productID string, attrs NginxAttrs) *Nginx {
	return &Nginx{
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
		Attrs: attrs,
	}
}

func (o *Nginx) AddAccessGroup(members entities.ADLogonGrants, premissions []string) error {

	vmItems, err := o.GetVMItems()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"members": members.Groups,
		"role": map[string]interface{}{
			"group_name":  members.Role,
			"permissions": premissions,
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

	uri := o.generateOrderdActionUri("vm_acls_add_with_parent")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}
	return err
}

func (o *Nginx) ChangeAccessGroup(changeAD entities.ADLogonGrants) error {

	vmItems, err := o.GetVMItems()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"members": changeAD.Groups,
		"role":    changeAD.Role,
	}

	data := map[string]interface{}{
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
		"item_id": vmItems[0].ID,
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("vm_acls_set_linux")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}
	return err
}

func (o *Nginx) ChangeFlavor(flavor entities.Flavor) error {

	vmItem, err := o.GetVMItems()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"flavor": flavor,
	}

	data := map[string]interface{}{
		"item_id": vmItem[0].ID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("resize_vm")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}
	return err
}

func (o *Nginx) Create(p CreateOrderPayload) error {

	payload, err := o.createPayload(p, o)
	if err != nil {
		return err
	}

	resp, err := requests.SendRequest(o.Creds.AccessToken, generateCreateOrderUri(o.ProjectName), "POST", payload, nil)
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

func (o *Nginx) DeleteAccessGroup(changeAD entities.ADLogonGrants) error {

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

	uri := o.generateOrderdActionUri("vm_acls_remove")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}
	return err
}

func (o *Nginx) ExpandMountPoint(mountPoint entities.ExtraMount) (err error) {

	if err = o.requiredState("on"); err != nil {
		return
	}

	vmItems, err := o.GetVMItems()
	if err != nil {
		return
	}

	stateOff := "off"
	stateOn := "on"

	if vmItems[0].Data.State != stateOn {
		err = fmt.Errorf(
			"can't do action, because vm if vm.state is not equal %s, current state is %s",
			stateOn, stateOff,
		)
		return
	}

	extraMount, err := o.GetExtraMount(mountPoint.Path)
	if err != nil {
		return err
	}

	if extraMount == nil {
		err = fmt.Errorf("can't find mount point with path = %s", mountPoint.Path)
		return
	}

	if mountPoint == *extraMount {
		return
	}

	increaseSize := mountPoint.Size - extraMount.Size
	if increaseSize < 10 {
		err = fmt.Errorf(
			"can't change mount point less then 10 GB. Current is %d, new is %d",
			extraMount.Size, mountPoint.Size,
		)
		return
	}

	attrs := map[string]interface{}{
		"mount": mountPoint.Path,
		"size":  int(increaseSize),
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

	uri := o.generateOrderdActionUri("expand_mount_point_new")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}
	return err
}

func (o *Nginx) GetBasicAttrs() *BasicAttrs {
	return &o.Attrs.BasicAttrs
}

func (o *Nginx) GetAttrs() interface{} {
	return o.Attrs
}

func (o *Nginx) GetExtraMount(path string) (*entities.ExtraMount, error) {

	items, err := o.GetVMItems()
	if err != nil {
		return nil, err
	}

	if len(items) != 1 {
		return nil, errors.New("VMs len count must be equal 1")
	}

	extraMounts := items[0].Data.Config.(entities.VMItemConfig).ExtraMounts
	if len(extraMounts) == 0 {
		return nil, errors.New("length should be more then 0")
	}

	for _, em := range extraMounts {
		if em.Mount == path {
			return &entities.ExtraMount{
				Path:       em.Mount,
				Size:       int64(em.Size),
				FileSystem: em.Fstype,
			}, nil
		}
	}
	return nil, nil
}

func (o *Nginx) MigrateToAngie() error {

	var nginxItemID string
	for _, item := range o.Data {
		if item.Type == "app" {
			nginxItemID = item.ID
		}
	}

	attrs := map[string]interface{}{
		"dumb": "empty",
	}

	data := map[string]interface{}{
		"item_id": nginxItemID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("nginx_switch_to_angie")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}
	return err
}

func (o *Nginx) UpdateCert(altNames []string) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"accept_cert_updating": true,
		"alt_names":            altNames,
	}

	data := map[string]interface{}{
		"item_id": itemID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("nginx_update_certs")
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
