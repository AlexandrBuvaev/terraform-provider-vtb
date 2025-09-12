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

type RedisSentinel struct {
	Order
	Attrs RedisSentinelAttrs `json:"attrs"`
}

type RedisSentinelAttrs struct {
	BasicAttrs
	ExtraNics            []interface{} `json:"extra_nics"`
	AuthUser             string        `json:"auth_user"`
	AddParameters        bool          `json:"add_parameters"`
	Auth                 bool          `json:"auth"`
	RedisVersion         string        `json:"redis_version"`
	User                 string        `json:"user"`
	UserPassword         string        `json:"user_password"`
	NotifyKeyspaceEvents string        `json:"notify_keyspace_events"`
	LayoutID             string        `json:"layout"`
	GeoDistribution      bool          `json:"geo_distribution"`
	AcceptDocumentation  bool          `json:"accept_documentation"`
	Accept               bool          `json:"accept"`
}

func NewRedisSentinel(creds *auth.Credentials, projectName, productID string, attrs RedisSentinelAttrs) *RedisSentinel {
	return &RedisSentinel{
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
		Attrs: attrs,
	}
}

func (o *RedisSentinel) Create(p CreateOrderPayload) error {

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

func (o *RedisSentinel) GetAttrs() interface{} {
	return o.Attrs
}

func (o *RedisSentinel) GetState() (string, error) {

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

func (o *RedisSentinel) GetRedisSentinelVmItems() ([]entities.Item, error) {

	item, err := o.GetParentItem()
	if err != nil {
		return nil, err
	}

	if item.Type != "app" {
		return nil, errors.New("this is not cluster. Can't get VM with node role 'redis'")
	}

	vmItems, err := o.GetVMItems()
	if err != nil {
		return nil, err
	}

	if len(vmItems) == 0 {
		return nil, errors.New("get vm items have zero length")
	}

	var RedisSentinelVmItems []entities.Item
	for _, vm := range vmItems {
		nodeRoles := vm.Data.Config.(entities.VMItemConfig).NodeRoles
		for _, role := range nodeRoles {
			if role == "redis" {
				RedisSentinelVmItems = append(RedisSentinelVmItems, vm)
			}
		}
	}

	if len(RedisSentinelVmItems) != 2 {
		return nil, errors.New("vm items with node role 'redis' are not equal 2")
	}

	return RedisSentinelVmItems, nil
}

func (o *RedisSentinel) GetUsers() ([]entities.RedisSentinelUser, error) {
	item, err := o.GetParentItem()
	if err != nil {
		return nil, err
	}

	config := item.Data.Config.(entities.RedisSentinelItemConfig)

	return config.Users, nil
}

func (o *RedisSentinel) GetUser(userName string) (*entities.RedisSentinelUser, error) {
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

func (o *RedisSentinel) GetDefaultRedisUser() (*entities.RedisSentinelUser, error) {
	redisUsers, err := o.GetUsers()
	if err != nil {
		return nil, err
	}
	if len(redisUsers) == 0 {
		return nil, fmt.Errorf("list of users in Redis config is empty")
	}
	return &redisUsers[0], nil
}

func (o *RedisSentinel) ChangeUserPassword(username string, password string) (err error) {
	if err := o.requiredState("on"); err != nil {
		return err
	}
	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	user, err := o.GetDefaultRedisUser()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"user_name":      user.UserName,
		"redis_password": password,
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
	_, err = requests.SendRequest(o.Creds.AccessToken, o.generateOrderdActionUri("reset_sentinel_redis_user_password"), "PATCH", payload, nil)
	if err != nil {
		return
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return
	}
	return
}

func (o *RedisSentinel) ChangeNotifyKeyspaceEvents(notifyKeyspaceEvents string) (err error) {

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"notify_keyspace_events": notifyKeyspaceEvents,
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

	uri := o.generateOrderdActionUri("change_redis_sentinel_param_notify")

	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return
	}
	return
}

func (o *RedisSentinel) ChangeFlavor(flavor entities.Flavor, async bool) error {

	if err := o.requiredState("on"); err != nil {
		return err
	}

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"flavor":      flavor,
		"check_agree": true,
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

	uri := o.generateOrderdActionUri("redis_sentinel_resize_two_layer")

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

func (o *RedisSentinel) GetExtraMounts() (extraMounts []entities.ExtraMount, err error) {
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

func (o *RedisSentinel) GetExtraMount(path string) (*entities.ExtraMount, error) {
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

func (o *RedisSentinel) ChangeAccessGroupForVm(vmItemId string, changeAD entities.ADLogonGrants) error {

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

func (o *RedisSentinel) AddAccessGroupForVm(vmItemId string, members entities.ADLogonGrants, permissions []string) error {

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

func (o *RedisSentinel) DeleteAccessGroupForVm(vmitemId string, changeAD entities.ADLogonGrants) error {

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
