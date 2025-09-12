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

const (
	BALANCER_V3_PREFIX  = "balancer_v3"
	EXPAND_MOUNT_POINT  = "expand_mount_point"
	VERTICAL_SCALING    = "resize_cluster_vms"
	HORIZONTAL_SCALING  = "horizontal_scale"
	COMPLEX_APPLY       = "complex_apply"
	DELETE_ORDER        = "delete"
	DELETE_ALL_ENTITIES = "delete_all_entities"
)

type BalancerV3 struct {
	Order
	Attrs BalancerV3Attrs `json:"attrs"`
}

type BalancerV3Attrs struct {
	BasicAttrs
	ClusterName     string             `json:"cluster_name"`
	SetupVersion    string             `json:"setup_version"`
	DNSZone         string             `json:"dns_zone"`
	Password        string             `json:"password"`
	Layout          string             `json:"layout"`
	GeoDistribution bool               `json:"geo_distribution"`
	BootDisk        *entities.BootDisk `json:"boot_disk"`
}

func NewBalancerV3(creds *auth.Credentials, attrs BalancerV3Attrs, productId, projectName string) *BalancerV3 {
	return &BalancerV3{
		Order: Order{
			ProductID:   productId,
			ProjectName: projectName,
			Creds:       creds,
		},
		Attrs: attrs,
	}
}

func (o *BalancerV3) formatActionName(actionName string) (string, error) {
	balancerItem, err := o.GetParentItem()
	if err != nil {
		return "", err
	}
	bI := balancerItem.Data.Config.(entities.BalancerV3ItemData)

	versions := strings.Split(bI.Build.SetupVersion, ".")
	if len(versions) < 2 {
		return "", err
	}

	return fmt.Sprintf(
		"%s_%s_%s",
		BALANCER_V3_PREFIX, versions[1], actionName,
	), nil
}

func (o *BalancerV3) Create(p CreateOrderPayload) error {

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

func (o *BalancerV3) DeleteTwoLayer(async bool) error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{}

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

	actionName, err := o.formatActionName(DELETE_ORDER)
	if err != nil {
		return err
	}

	uri := fmt.Sprintf(
		"order-service/api/v1/projects/%s/orders/%s/actions/%s",
		o.ProjectName, o.ID, actionName,
	)
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	if !async {
		err = o.WaitSuccess(10)
		if err != nil {
			return err
		}
	}
	return err
}

func (o *BalancerV3) GetExtraMount(path string) (*entities.ExtraMount, error) {

	items, err := o.GetVMItems()
	if err != nil {
		return nil, err
	}

	if len(items) == 0 {
		return nil, errors.New("length of elements less then 1")
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

func (o *BalancerV3) GetAttrs() interface{} {
	return o.Attrs
}

func (o *BalancerV3) ExpandMountPoint(mountPoint entities.ExtraMount) (err error) {

	orderItem, err := o.GetParentItem()
	if err != nil {
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
		"item_id": orderItem.ID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	actionName, err := o.formatActionName(EXPAND_MOUNT_POINT)
	if err != nil {
		return err
	}

	uri := fmt.Sprintf(
		"order-service/api/v1/projects/%s/orders/%s/actions/%s",
		o.ProjectName, o.ID, actionName,
	)
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

func (o *BalancerV3) AddAccessGroup(members entities.ADLogonGrants, permissions []string) error {

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

	for _, vm := range vmItems {
		data := map[string]interface{}{
			"item_id": vm.ID,
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
	}
	return err
}

func (o *BalancerV3) ChangeAccessGroup(changeAD entities.ADLogonGrants) error {

	vmItems, err := o.GetVMItems()
	if err != nil {
		return err
	}
	for _, vm := range vmItems {

		attrs := map[string]interface{}{
			"members": changeAD.Groups,
			"role":    changeAD.Role,
		}

		data := map[string]interface{}{
			"item_id": vm.ID,
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
	}
	return err
}

func (o *BalancerV3) DeleteAccessGroup(changeAD entities.ADLogonGrants) error {

	vmItems, err := o.GetVMItems()
	if err != nil {
		return err
	}

	for _, vm := range vmItems {

		attrs := map[string]interface{}{
			"members": changeAD.Groups,
			"role":    changeAD.Role,
		}

		data := map[string]interface{}{
			"item_id": vm.ID,
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
	}
	return err
}

func (o *BalancerV3) VerticalScaling(flavor entities.Flavor) error {

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

	actionName, err := o.formatActionName(VERTICAL_SCALING)
	if err != nil {
		return err
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
	return err
}

func (o *BalancerV3) HorizontalScaling(new_haproxy_count int) error {

	parentItemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"check_agree":       true,
		"new_haproxy_count": new_haproxy_count,
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

	actionName, err := o.formatActionName(HORIZONTAL_SCALING)
	if err != nil {
		return err
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
	return err
}

func (o *BalancerV3) ComplexApply(new_config entities.Config) error {

	parentItemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"backends":     new_config.Backends,
		"ports":        new_config.Ports,
		"defaults":     new_config.Defaults,
		"globals":      new_config.Globals,
		"publications": new_config.Publications,
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
	fmt.Printf("payload:%#v", string(payload))

	actionName, err := o.formatActionName(COMPLEX_APPLY)
	if err != nil {
		return err
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
	return err
}

func (o *BalancerV3) Migrate(version string) error {

	parentItemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"accept":            false,
		"version_to_update": version,
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

	uri := o.generateOrderdActionUri("balancer_v3_all_migrate_to_new_version")
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

func (o *BalancerV3) DeleteAllEntities() error {

	parentItemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"accept": true,
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

	actionName, err := o.formatActionName(DELETE_ALL_ENTITIES)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri(actionName)

	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return nil
	}

	return nil
}

func (o *BalancerV3) CheckClusterItemStatusOn() (bool, error) {

	item, err := o.GetParentItem()
	if err != nil {
		return false, err
	}
	config, ok := item.Data.Config.(entities.BalancerV3ItemData)
	if !ok {
		return false, fmt.Errorf("invalid type for Item.Data.Config")
	}

	for _, member := range config.ClusterMembers {
		if member.MainStatus == "off" {
			return false, nil
		}
	}
	return true, nil
}
