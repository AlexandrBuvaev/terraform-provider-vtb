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

type Redis struct {
	Order
	Attrs RedisAttrs `json:"attrs"`
}

type RedisAttrs struct {
	BasicAttrs
	ExtraNics            []interface{} `json:"extra_nics"`
	AddParameters        bool          `json:"add_parameters"`
	RedisVersion         string        `json:"redis_version"`
	User                 string        `json:"user"`
	UserPassword         string        `json:"user_password"`
	NotifyKeyspaceEvents string        `json:"notify_keyspace_events"`
	LayoutID             string        `json:"layout"`
	GeoDistribution      bool          `json:"geo_distribution"`
	AcceptDocumentation  bool          `json:"accept_documentation"`
	Accept               bool          `json:"accept"`
	Auth                 bool          `json:"auth"`
}

func NewRedis(creds *auth.Credentials, projectName, productID string, attrs RedisAttrs) *Redis {
	return &Redis{
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
		Attrs: attrs,
	}
}

func (o *Redis) Create(p CreateOrderPayload) error {

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

func (o *Redis) GetAttrs() interface{} {
	return o.Attrs
}

func (o *Redis) GetState() (string, error) {

	_, err := o.itemCreated()
	if err != nil {
		return "", err
	}

	err = o.Sync()
	if err != nil {
		return "", err
	}

	item, err := o.GetParentItem()
	if err != nil {
		return "", err
	}
	return item.Data.State, nil
}

func (o *Redis) GetUsers() ([]entities.RedisUser, error) {
	item, err := o.GetParentItem()
	if err != nil {
		return nil, err
	}

	config := item.Data.Config.(entities.RedisItemConfig)

	return config.Users, nil
}

func (o *Redis) GetUser(userName string) (*entities.RedisUser, error) {
	users, err := o.GetUsers()
	if err != nil {
		return nil, err
	}

	for _, user := range users {
		if user.UserName == userName {
			return &user, nil
		}
	}

	return nil, fmt.Errorf("there is no user with name '%s'", userName)
}

func (o *Redis) GetDefaultRedisUser() (*entities.RedisUser, error) {
	redisUsers, err := o.GetUsers()
	if err != nil {
		return nil, err
	}
	if len(redisUsers) == 0 {
		return nil, fmt.Errorf("list of users in Redis config is empty")
	}
	return &redisUsers[0], nil
}

func (o *Redis) ChangeRedisParamNotify(NotifyKeyspaceEvents string) error {

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"notify_keyspace_events": NotifyKeyspaceEvents,
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

	uri := o.generateOrderdActionUri("change_redis_param_notify")

	if item.Type == "cluster" {
		uri = o.generateOrderdActionUri("change_redis_param_notify_cluster")
	}

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

func (o *Redis) ChangeUserPassword(userName, password string, async bool) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"user_name":      userName,
		"redis_password": password,
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

	uri := o.generateOrderdActionUri("reset_redis_user_password")
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

func (o *Redis) ChangeFlavor(flavor entities.Flavor) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"check_agree": true,
		"flavor":      flavor,
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

	uri := o.generateOrderdActionUri("redis_resize_two_layer")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(30)
	if err != nil {
		return err
	}

	return err
}

func (o *Redis) ChangeFlavorProd(flavor entities.Flavor, serverName string) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"check_agree": true,
		"flavor":      flavor,
		"server_name": serverName,
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

	uri := o.generateOrderdActionUri("redis_prod_resize_two_layer")

	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(30)
	if err != nil {
		return err
	}

	return nil
}

func (o *Redis) GetExtraMount(path string) (*entities.ExtraMount, error) {

	items, err := o.GetVMItems()
	if err != nil {
		return nil, err
	}
	appItem, err := o.GetParentItem()
	if err != nil {
		return nil, err
	}

	if len(items) != 1 && appItem.Type == "app" {
		return nil, errors.New("VMs len count must be equal 1")
	} else if len(items) != 2 && appItem.Type == "cluster" {
		return nil, errors.New("VMs len count must be equal 2")
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

func (o *Redis) ExpandMountPoint(mountPoint entities.ExtraMount, async bool) (err error) {

	if err = o.requiredState("on"); err != nil {
		return
	}

	vmItems, err := o.GetVMItems()
	if err != nil {
		return err
	}

	if len(vmItems) == 0 {
		err = errors.New("get vm items have zero length")
		return err
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
	if !async {
		err = o.WaitSuccess(10)
		if err != nil {
			return err
		}
	}
	return err
}

func (o *Redis) Stop(async bool) error {

	if err := o.requiredState("on"); err != nil {
		return err
	}

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

	uri := o.generateOrderdActionUri("stop_two_layer")
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

func (o *Redis) Start(async bool) error {

	if err := o.requiredState("off"); err != nil {
		return err
	}

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

	uri := o.generateOrderdActionUri("start_two_layer")
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

func (o *Redis) ChangeAccessGroupForVm(vmItemId string, changeAD entities.ADLogonGrants) error {

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

func (o *Redis) AddAccessGroupForVm(vmItemId string, members entities.ADLogonGrants, permissions []string) error {

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

func (o *Redis) DeleteAccessGroupForVm(vmitemId string, changeAD entities.ADLogonGrants) error {
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
