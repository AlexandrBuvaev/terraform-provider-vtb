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

type ClickHouseClusterAttrs struct {
	ADIntegration    bool                     `json:"ad_integration"`
	OnSupport        bool                     `json:"on_support"`
	OsVersion        string                   `json:"os_version"`
	ADLogonGrants    []entities.ADLogonGrants `json:"ad_logon_grants,omitempty"`
	AvailabilityZone string                   `json:"availability_zone"`
	Domain           string                   `json:"domain"`
	Platform         string                   `json:"platform"`
	DefaultNic       entities.DefaultNic      `json:"default_nic"`

	ClickHouseVersion               string                            `json:"ch_version"`
	ZookeeperVersion                string                            `json:"zk_version"`
	ClusterName                     string                            `json:"cluster_name"`
	ClickHouseUser                  *string                           `json:"ch_customer_admin,omitempty"`
	ClickHousePassword              *string                           `json:"ch_customer_admin_password,omitempty"`
	Accept                          bool                              `json:"accept"`
	ChCustomerPassword              string                            `json:"ch_customer_password"`
	SystemAdmGroupsAttrs            []SystemAdmGroupsModel            `json:"system_adm_groups,omitempty"`
	ClickHouseAppAdminAdGroupsAttrs []ClickHouseAppAdminAdGroupsModel `json:"clickhouse_app_admin_ad_groups"`
	ClickHouseUserAdGroupsAttrs     []ClickHouseUserAdGroupsModel     `json:"clickhouse_user_ad_groups"`
	NodesCount                      NodesCount                        `json:"nodes_count"`
	FlavorCH                        entities.Flavor                   `json:"flavor_ch"`
	FlavorZK                        entities.Flavor                   `json:"flavor_zk"`
	CHExtraMounts                   []entities.ExtraMount             `json:"ch_extra_mounts"`
	ZKExtraMounts                   []entities.ExtraMount             `json:"zk_extra_mounts"`
	GeoDistribution                 bool                              `json:"geo_distribution,omitempty"`
	Layout                          string                            `json:"layout"`
	CreatedWithOpenTofu             bool                              `json:"created_with_opentofu"`
}

type ClickHouseCluster struct {
	Order
	Attrs ClickHouseClusterAttrs `json:"attrs"`
}

type NodesCount struct {
	ClickHouse int `tfsdk:"clickhouse" json:"clickhouse"`
	Zookeeper  int `tfsdk:"zookeeper" json:"zookeeper"`
}

func NewClickHouseCluster(creds *auth.Credentials, projectName, productID string, attrs ClickHouseClusterAttrs) *ClickHouseCluster {
	return &ClickHouseCluster{
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
		Attrs: attrs,
	}
}

func (o *ClickHouseCluster) Create(p CreateOrderPayload) error {

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

func (o *ClickHouseCluster) GetAttrs() interface{} {
	return o.Attrs
}

func (o *ClickHouseCluster) GetState() (string, error) {

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

func (o *ClickHouseCluster) GetClickHouseClusterVmItems() ([]entities.Item, error) {

	item, err := o.GetParentItem()
	if err != nil {
		return nil, err
	}

	if item.Type != "cluster" {
		return nil, errors.New("this is not cluster. Can't get VM with node role 'clickhouse'")
	}

	vmItems, err := o.GetVMItems()
	if err != nil {
		return nil, err
	}

	if len(vmItems) == 0 {
		return nil, errors.New("get vm items have zero length")
	}

	var ClickHouseClusterVmItems []entities.Item
	for _, vm := range vmItems {
		nodeRoles := vm.Data.Config.(entities.VMItemConfig).NodeRoles
		for _, role := range nodeRoles {
			if role == "clickhouse" {
				ClickHouseClusterVmItems = append(ClickHouseClusterVmItems, vm)
			}
		}
	}

	if len(ClickHouseClusterVmItems) < 2 {
		return nil, errors.New("vm items with node role 'clickhouse' are less 2")
	}

	return ClickHouseClusterVmItems, nil
}

func (o *ClickHouseCluster) GetClickhouseClusterZookeeperVmItems() ([]entities.Item, error) {

	item, err := o.GetParentItem()
	if err != nil {
		return nil, err
	}

	if item.Type != "cluster" {
		return nil, errors.New("this is not cluster. Can't get VM with node role 'clickhouse'")
	}

	vmItems, err := o.GetVMItems()
	if err != nil {
		return nil, err
	}

	if len(vmItems) == 0 {
		return nil, errors.New("get vm items have zero length")
	}

	var ClickHouseClusterZookeeperVmItems []entities.Item
	for _, vm := range vmItems {
		nodeRoles := vm.Data.Config.(entities.VMItemConfig).NodeRoles
		for _, role := range nodeRoles {
			if role == "zookeeper" {
				ClickHouseClusterZookeeperVmItems = append(ClickHouseClusterZookeeperVmItems, vm)
			}
		}
	}

	// if len(ClickHouseClusterZookeeperVmItems) < 3 {
	// 	return nil, errors.New("vm items with node role 'zookeeper' are less 3")
	// }

	return ClickHouseClusterZookeeperVmItems, nil
}

func (o *ClickHouseCluster) GetUsers() ([]entities.DBOwners, error) {
	item, err := o.GetParentItem()
	if err != nil {
		return nil, err
	}

	config := item.Data.Config.(entities.ClickhouseClusterItemConfig)

	return config.DBOwners, nil
}

func (o *ClickHouseCluster) GetUser(userName string) (*entities.DBOwners, error) {
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

func (o *ClickHouseCluster) GetDefaultClickHouseClusterUser() (*entities.DBOwners, error) {
	clickhouseUsers, err := o.GetUsers()
	if err != nil {
		return nil, err
	}
	if len(clickhouseUsers) == 0 {
		return nil, fmt.Errorf("list of users in Clickhouse Cluster config is empty")
	}
	return &clickhouseUsers[0], nil
}

func (o *ClickHouseCluster) ChangeCHFlavor(flavor entities.Flavor, async bool) error {

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
		"checking_the_reading_of_the_instruction": true,
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

	uri := o.generateOrderdActionUri("vertical_resize_clickhouse_cluster")

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

func (o *ClickHouseCluster) ChangeZKFlavor(flavor entities.Flavor, async bool) error {

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
		"checking_the_reading_of_the_instruction": true,
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

	uri := o.generateOrderdActionUri("vertical_resize_clickhouse_cluster_zookeeper")

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

func (o *ClickHouseCluster) GetExtraMounts() (extraMounts []entities.ExtraMount, err error) {
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

func (o *ClickHouseCluster) GetExtraMount(path string) (*entities.ExtraMount, error) {
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

func (o *ClickHouseCluster) ChangeUserPassword(username string, password string) (err error) {
	if err := o.requiredState("on"); err != nil {

		return err
	}
	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"user_name":     username,
		"user_password": password,
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
	_, err = requests.SendRequest(o.Creds.AccessToken, o.generateOrderdActionUri("clickhouse_cluster_reset_db_user_password"), "PATCH", payload, nil)
	if err != nil {
		return
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return
	}
	return
}

func (o *ClickHouseCluster) ChangeCHUserPassword(username string, password string) (err error) {
	if err := o.requiredState("on"); err != nil {

		return err
	}
	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	user, err := o.GetDefaultClickHouseClusterUser()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"user_name":     user.UserName,
		"user_password": password,
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
	_, err = requests.SendRequest(o.Creds.AccessToken, o.generateOrderdActionUri("clickhouse_cluster_reset_db_user_password"), "PATCH", payload, nil)
	if err != nil {
		return
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return
	}
	return
}

func (o *ClickHouseCluster) ChangeAccessGroupForVm(vmItemId string, changeAD entities.ADLogonGrants) error {
	attrs := map[string]interface{}{
		"members": changeAD.Groups,
		"role":    changeAD.Role,
	}

	data := map[string]interface{}{
		"item_id": vmItemId,
		"order": map[string]interface{}{
			"attrs": attrs,
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

func (o *ClickHouseCluster) AddAccessGroupForVm(vmItemId string, members entities.ADLogonGrants, permissions []string) error {

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

func (o *ClickHouseCluster) DeleteAccessGroupForVm(vmitemId string, changeAD entities.ADLogonGrants) error {

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

func (o *ClickHouseCluster) GetAppAdGroups(roleName string) (map[string][]string, error) {

	item, err := o.GetParentItem()
	if err != nil {
		return nil, err
	}

	clickHouseConfig, ok := item.Data.Config.(entities.ClickhouseClusterItemConfig)
	if !ok {
		return nil, fmt.Errorf("invalid clickHouse config type")
	}

	result := make(map[string][]string)

	switch roleName {
	case "system_adm_groups":
		if clickHouseConfig.DBAdminGroup[0].UserName[0] == "" {
			return result, nil
		}
		result["system_adm_groups"] = clickHouseConfig.DBAdminGroup[0].UserName

	case "clickhouse_app_admin_ad_groups":
		groups := make([]string, 0, len(clickHouseConfig.DBAppAdminGroup))
		for _, group := range clickHouseConfig.DBAppAdminGroup {
			groups = append(groups, group.UserName)
		}
		result["clickhouse_app_admin_ad_groups"] = groups

	case "clickhouse_user_ad_groups":
		groups := make([]string, 0, len(clickHouseConfig.DBUserGroup))
		for _, group := range clickHouseConfig.DBUserGroup {
			groups = append(groups, group.UserName)
		}
		result["clickhouse_user_ad_groups"] = groups
	}

	return result, nil
}

func (o *ClickHouseCluster) CreateNewAppAdminGroupAd(groupName string) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"ad_integration": true,
		"clickhouse_app_admin_ad_groups": []map[string]interface{}{
			{
				"groups": []string{groupName},
			},
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

	uri := o.generateOrderdActionUri("clickhouse_cluster_create_new_app_admin_group_ad")
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

func (o *ClickHouseCluster) RemoveNewAppAdminGroupAd(groupName string) error {

	itemId, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"user_name": groupName,
	}

	data := map[string]interface{}{
		"item_id": itemId,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("clickhouse_cluster_remove_new_app_admin_group_ad")
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

func (o *ClickHouseCluster) CreateNewAppUserGroupAd(groupName string) error {

	itemId, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"ad_integration": true,
		"clickhouse_user_ad_groups": []map[string]interface{}{
			{
				"groups": []string{groupName},
			},
		},
	}

	data := map[string]interface{}{
		"item_id": itemId,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("clickhouse_cluster_create_new_app_user_group_ad")
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

func (o *ClickHouseCluster) RemoveNewAppUserGroupAd(groupName string) error {

	itemId, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"user_name": groupName,
	}

	data := map[string]interface{}{
		"item_id": itemId,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return nil
	}

	uri := o.generateOrderdActionUri("clickhouse_cluster_remove_new_app_user_group_ad")
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

func (o *ClickHouseCluster) DeleteClickHouseCluster(env string) error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{}
	var actionName string
	if strings.EqualFold(env, "prod") {
		actionName = "delete_two_layer_clickhouse_cluster_prod"
		attrs = map[string]interface{}{
			"accept": true,
		}
	} else {
		actionName = "delete_two_layer"
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
