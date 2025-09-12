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

type ScyllaDbCluster struct {
	Order
	Attrs ScyllaDbClusterAttrs `json:"attrs"`
}

type ScyllaDbClusterAttrs struct {
	Domain                     string                         `json:"domain"`
	Flavor                     entities.Flavor                `json:"flavor"`
	Platform                   string                         `json:"platform"`
	OnBackup                   bool                           `json:"on_backup"`
	OnSupport                  bool                           `json:"on_support"`
	OsVersion                  string                         `json:"os_version"`
	DefaultNic                 entities.DefaultNic            `json:"default_nic"`
	AdIntegration              bool                           `json:"ad_integration"`
	ADLogonGrants              []entities.ADLogonGrants       `json:"ad_logon_grants"`
	ExtraMountsLog             []entities.ExtraMount          `json:"extra_mounts_log"`
	ScylladbVersion            string                         `json:"scylladb_version"`
	AvailabilityZone           string                         `json:"availability_zone"`
	ExtraMountsData            []entities.ExtraMount          `json:"extra_mounts_data"`
	Dbs                        []entities.ScyllaDbs           `json:"dbs"`
	DbUsers                    []entities.ScyllaDbusers       `json:"db_users"`
	Permissions                []entities.ScyllaDbPermissions `json:"permissions"`
	ScyllaClusterConfiguration ScyllaClusterConfiguration     `json:"scylla_cluster_configuration"`
	CreatedWithOpenTofu        bool                           `json:"created_with_opentofu"`
}

type ScyllaClusterConfiguration struct {
	DC1 int `json:"dc1"`
	DC2 int `json:"dc2"`
	DC3 int `json:"dc3"`
}

func NewScyllaDbCluster(creds *auth.Credentials, projectName, productID string, attrs ScyllaDbClusterAttrs) *ScyllaDbCluster {
	return &ScyllaDbCluster{
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
		Attrs: attrs,
	}
}

func (o *ScyllaDbCluster) Create(p CreateOrderPayload) error {

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

func (o *ScyllaDbCluster) GetAttrs() interface{} {
	return o.Attrs
}

func (o *ScyllaDbCluster) GetState() (string, error) {

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
		return "Error to get Parent Item", err
	}
	return item.Data.State, nil
}

func (o *ScyllaDbCluster) GetScyllaDbClusterVmItems() ([]entities.Item, error) {

	item, err := o.GetParentItem()
	if err != nil {
		return nil, err
	}

	if item.Type != "cluster" {
		return nil, errors.New("this is not cluster. Can't get VM with node role 'scylladb'")
	}

	vmItems, err := o.GetVMItems()
	if err != nil {
		return nil, err
	}

	if len(vmItems) == 0 {
		return nil, errors.New("get vm items have zero length")
	}

	var ScyllaDbClusterVmItems []entities.Item
	for _, vm := range vmItems {
		nodeRoles := vm.Data.Config.(entities.VMItemConfig).NodeRoles
		for _, role := range nodeRoles {
			if role == "scylladb" {
				ScyllaDbClusterVmItems = append(ScyllaDbClusterVmItems, vm)
			}
		}
	}

	if len(ScyllaDbClusterVmItems) != 2 {
		return nil, errors.New("vm items with node role 'scylladb' are not equal 2")
	}

	return ScyllaDbClusterVmItems, nil
}

func (o *ScyllaDbCluster) ChangeFlavor(flavor entities.Flavor, async bool) error {

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

	uri := o.generateOrderdActionUri("scylladb_cluster_vertical_scaling")

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

func (o *ScyllaDbCluster) GetExtraMounts() (extraMounts []entities.ExtraMount, err error) {
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

func (o *ScyllaDbCluster) GetExtraMount(path string) (*entities.ExtraMount, error) {
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

func (o *ScyllaDbCluster) ChangeAccessGroupForVm(vmItemId string, changeAD entities.ADLogonGrants) error {
	data := map[string]interface{}{
		"item_id": vmItemId,
		"order": map[string]interface{}{
			"attrs": map[string]interface{}{
				"members":               changeAD.Groups,
				"role":                  changeAD.Role,
				"created_with_opentofu": true,
			},
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

func (o *ScyllaDbCluster) AddAccessGroupForVm(vmItemId string, members entities.ADLogonGrants, permissions []string) error {
	data := map[string]interface{}{
		"item_id": vmItemId,
		"order": map[string]interface{}{
			"attrs": map[string]interface{}{
				"members":               members.Groups,
				"created_with_opentofu": true,
				"role": map[string]interface{}{
					"group_name":  members.Role,
					"permissions": permissions,
				},
			},
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

func (o *ScyllaDbCluster) DeleteAccessGroupForVm(vmitemId string, changeAD entities.ADLogonGrants) error {
	data := map[string]interface{}{
		"item_id": vmitemId,
		"order": map[string]interface{}{
			"attrs": map[string]interface{}{
				"members":               changeAD.Groups,
				"role":                  changeAD.Role,
				"created_with_opentofu": true,
			},
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

func (o *ScyllaDbCluster) Delete() error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"item_id": itemID,
		"order": map[string]interface{}{
			"attrs": map[string]interface{}{
				"created_with_opentofu": true,
			},
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("delete_scylladb_cluster")
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

func (o *ScyllaDbCluster) CreateDb(dbname string) error {

	var uri string
	var data map[string]interface{}
	if err := o.requiredState("on"); err != nil {
		return err
	}

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	data = map[string]interface{}{
		"item_id": item.ID,
		"order": map[string]interface{}{
			"attrs": map[string]interface{}{
				"db_name":               dbname,
				"created_with_opentofu": true,
			},
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri = fmt.Sprintf(
		"order-service/api/v1/projects/%s/orders/%s/actions/scylladb_create_db",
		o.ProjectName, o.ID,
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

func (o *ScyllaDbCluster) DeleteDb(dbname string) error {

	var uri string
	var data map[string]interface{}
	if err := o.requiredState("on"); err != nil {
		return err
	}

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	data = map[string]interface{}{
		"item_id": item.ID,
		"order": map[string]interface{}{
			"attrs": map[string]interface{}{
				"db_name":               dbname,
				"created_with_opentofu": true,
			},
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri = fmt.Sprintf(
		"order-service/api/v1/projects/%s/orders/%s/actions/scylladb_remove_db",
		o.ProjectName, o.ID,
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

func (o *ScyllaDbCluster) CreateDbUser(dbuser string, dbmsrole string, userpassword string) error {

	var uri string
	var data map[string]interface{}
	if err := o.requiredState("on"); err != nil {
		return err
	}

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	data = map[string]interface{}{
		"item_id": item.ID,
		"order": map[string]interface{}{
			"attrs": map[string]interface{}{
				"user_name":             dbuser,
				"dbms_role":             dbmsrole,
				"user_password":         userpassword,
				"created_with_opentofu": true,
			},
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri = fmt.Sprintf(
		"order-service/api/v1/projects/%s/orders/%s/actions/scylladb_create_dbms_user",
		o.ProjectName, o.ID,
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

func (o *ScyllaDbCluster) DeleteDbUser(dbuser string) error {

	var uri string
	var data map[string]interface{}
	if err := o.requiredState("on"); err != nil {
		return err
	}

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	data = map[string]interface{}{
		"item_id": item.ID,
		"order": map[string]interface{}{
			"attrs": map[string]interface{}{
				"user_name":             dbuser,
				"created_with_opentofu": true,
			},
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri = fmt.Sprintf(
		"order-service/api/v1/projects/%s/orders/%s/actions/scylladb_remove_dbms_user",
		o.ProjectName, o.ID,
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

func (o *ScyllaDbCluster) GetUsers() ([]entities.ScyllaDbusers, error) {

	item, err := o.GetParentItem()
	if err != nil {
		return nil, err
	}

	config := item.Data.Config.(entities.ScyllaDbClusterItemConfig)
	return config.DbUsers, nil
}

func (o *ScyllaDbCluster) DeleteDbPermission(dbname, dbuser, id string) error {

	var uri string
	var data map[string]interface{}
	if err := o.requiredState("on"); err != nil {
		return err
	}

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	data = map[string]interface{}{
		"item_id": item.ID,
		"order": map[string]interface{}{
			"attrs": map[string]interface{}{
				"db_name":               dbname,
				"user_name":             dbuser,
				"id":                    id,
				"created_with_opentofu": true,
			},
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri = fmt.Sprintf(
		"order-service/api/v1/projects/%s/orders/%s/actions/scylladb_remove_dbms_permissions",
		o.ProjectName, o.ID,
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

func (o *ScyllaDbCluster) CreateDbPermission(dbname, dbuser string) error {

	var uri string
	var data map[string]interface{}
	if err := o.requiredState("on"); err != nil {
		return err
	}

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	data = map[string]interface{}{
		"item_id": item.ID,
		"order": map[string]interface{}{
			"attrs": map[string]interface{}{
				"db_name":               dbname,
				"user_name":             dbuser,
				"created_with_opentofu": true,
			},
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri = fmt.Sprintf(
		"order-service/api/v1/projects/%s/orders/%s/actions/scylladb_dbms_permissions",
		o.ProjectName, o.ID,
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

func (o *ScyllaDbCluster) GetDbPermissions() (dbPermissions []string, err error) {
	parentItem, err := o.GetParentItem()
	if err != nil {
		return
	}

	dbPerms := parentItem.Data.Config.(entities.ScyllaDbClusterItemConfig).Permissions

	for _, dbPerm := range dbPerms {
		dbPermissions = append(dbPermissions, dbPerm.UserName+":"+dbPerm.DbName)
	}

	return
}

func (o *ScyllaDbCluster) ChangeDbUserPassword(userName, password string, async bool) error {

	if err := o.requiredState("on"); err != nil {
		return err
	}

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"item_id": item.ID,
		"order": map[string]interface{}{
			"attrs": map[string]interface{}{
				"user_name":             userName,
				"user_password":         password,
				"created_with_opentofu": true,
			},
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	var action = "scylladb_reset_user_password"

	uri := fmt.Sprintf(
		"order-service/api/v1/projects/%s/orders/%s/actions/%s",
		o.ProjectName, o.ID, action,
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
