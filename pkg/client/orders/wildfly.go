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

type Wildfly struct {
	Order
	Attrs WildflyAttrs `json:"attrs"`
}

type WildflyAttrs struct {
	BasicAttrs
	ExtraNics       []any                  `json:"extra_nics"`
	AccessGroups    []string               `json:"access_group"`
	DevopsGroup     string                 `json:"devops_group"`
	StandaloneType  string                 `json:"standalone_type"`
	JavaVersion     string                 `json:"java_version"`
	WildflyVersion  string                 `json:"wildfly_version"`
	InfoAd          map[string]interface{} `json:"info_ad"`
	InfoVm          map[string]interface{} `json:"info_vm"`
	InfoOs          map[string]interface{} `json:"info_os"`
	InfoDevOps      map[string]interface{} `json:"info_devops"`
	InfoAccessGroup map[string]interface{} `json:"info_access_group"`
}

func NewWildfly(creds *auth.Credentials, projectName, productID string, attrs WildflyAttrs) *Wildfly {
	return &Wildfly{
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
		Attrs: attrs,
	}
}

func (o *Wildfly) AddAccessGroup(members entities.ADLogonGrants, permissions []string) error {

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

func (o *Wildfly) AddManagmentGroup(role, group string) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"new_wildfly_user": map[string]interface{}{
			"user_role":  role,
			"group_name": []string{group},
		},
	}

	data := map[string]interface{}{
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
		"item_id": itemID,
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("wildfly_release_add_group")
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

func (o *Wildfly) ChangeAccessGroup(changeAD entities.ADLogonGrants) error {

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

func (o *Wildfly) ChangeJavaVersion(wildflyVersion, javaVersion string) error {

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"accept":          true,
		"wildfly_version": wildflyVersion,
		"java_version":    javaVersion,
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

	uri := o.generateOrderdActionUri("wildfly_release_change_java")
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

func (o *Wildfly) Create(p CreateOrderPayload) error {

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

	err = o.Sync()
	if err != nil {
		return err
	}
	return nil
}

func (o *Wildfly) DeleteAccessGroup(changeAD entities.ADLogonGrants) error {

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

func (o *Wildfly) DeleteManagmentGroup(role, group string) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	groupTag := "wildfly_" + strings.ToLower(role)

	attrs := map[string]interface{}{
		"user_role": role,
		groupTag:    group,
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

	uri := o.generateOrderdActionUri("wildfly_release_del_group")
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

func (o *Wildfly) ExpandMountPoint(mountPoint entities.ExtraMount) (err error) {

	items, err := o.GetVMItems()
	if err != nil {
		return
	}

	stateOff := "off"
	stateOn := "on"

	if items[0].Data.State != stateOn {
		return fmt.Errorf(
			"can't do action, vm.state is not equal %s, current state is %s",
			stateOn, stateOff,
		)
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
		return fmt.Errorf(
			"can't change mount point less then 10 GB. Current is %d, new is %d",
			extraMount.Size, mountPoint.Size,
		)
	}

	attrs := map[string]interface{}{
		"mount": mountPoint.Path,
		"size":  int(increaseSize),
	}

	data := map[string]interface{}{
		"item_id": items[0].ID,
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

func (o *Wildfly) GetAttrs() interface{} {
	return o.Attrs
}

func (o *Wildfly) GetBasicAttrs() *BasicAttrs {
	return &o.Attrs.BasicAttrs
}

func (o *Wildfly) GetExtraMount(path string) (*entities.ExtraMount, error) {

	items, err := o.GetVMItems()
	if err != nil {
		return nil, err
	}

	if len(items) != 1 {
		return nil, errors.New("length of elements not equal 1")
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

func (o *Wildfly) StartService() error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"accept_cert_updating": true,
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

	uri := o.generateOrderdActionUri("wildfly_release_start_wf")
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

func (o *Wildfly) StopService() error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"accept": true,
	}

	data := map[string]interface{}{
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
		"item_id": itemID,
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("wildfly_release_stop_wf")
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

func (o *Wildfly) UpdateCert(altNames []string) error {

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"accept_cert_updating": true,
		"alt_names":            altNames,
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

	uri := o.generateOrderdActionUri("wildfly_release_update_certs")
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

func (o *Wildfly) VerticalScaling(flavor entities.Flavor) error {

	parentItemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"accept": true,
		"flavor": flavor,
	}

	data := map[string]interface{}{
		"item_id": parentItemID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("wildfly_release_vertical_scaling")
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

func (o *Wildfly) SetMaintanceModeOn(dateEnd string) error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"accept": true,
		"zni_value": map[string]interface{}{
			"date_end": dateEnd,
		},
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

	uri := o.generateOrderdActionUri("wildfly_release_action_set_mm_on")

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

func (o *Wildfly) SetMaintanceModeOff() error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"accept": true,
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

	uri := o.generateOrderdActionUri("wildfly_release_action_set_mm_off")

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

func (o *Wildfly) AddClientCert() error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"accept_cert_updating": true,
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

	uri := o.generateOrderdActionUri("wildfly_release_add_client_cert")

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

func (o *Wildfly) DeleteClientCert() error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"accept_cert_updating": true,
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

	uri := o.generateOrderdActionUri("wildfly_release_delete_client_cert")

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
