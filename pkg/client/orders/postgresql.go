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

type PostgresqlConfig struct {
	DefaultTransactionIsolation string `json:"default_transaction_isolation"`
}

type PostgresqlAttrs struct {
	BasicAttrs

	ExtraNics         []interface{}    `json:"extra_nics"`
	GeoDistribution   bool             `json:"geo_distribution"`
	LayoutID          string           `json:"layout"`
	LoadProfile       string           `json:"load_profile"`
	PostgresqlConfig  PostgresqlConfig `json:"postgresql_config"`
	PostgresqlVersion string           `json:"postgresql_version"`
	Replication       bool             `json:"replication"`
	OnBackup          bool             `json:"on_backup"`
	EnableSSL         bool             `json:"enable_ssl"`
	CorparchCheck     bool             `json:"corparch_check"`
}

type PostgresqlOrder struct {
	Order
	Attrs PostgresqlAttrs `json:"attrs"`
}

func NewPostgresqlOrder(
	creds *auth.Credentials,
	projectName,
	productID string,
	attrs PostgresqlAttrs,
) *PostgresqlOrder {
	return &PostgresqlOrder{
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
		Attrs: attrs,
	}
}

func (o *PostgresqlOrder) GetAttrs() interface{} {
	return o.Attrs
}

func (o *PostgresqlOrder) Create(p CreateOrderPayload) error {

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

func (o *PostgresqlOrder) Delete(async bool) error {

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

	var uri string
	switch item.Type {
	case "cluster":
		uri = o.generateOrderdActionUri("delete_postgresql_cluster")
	case "app":
		uri = o.generateOrderdActionUri("delete_postgresql")
	}

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

func (o *PostgresqlOrder) GetDBs() ([]entities.Item, error) {

	created, err := o.itemCreated()
	if err != nil {
		return nil, err
	}

	var items []entities.Item
	if created {
		for _, item := range o.Data {
			if item.Type == "db" && item.Data.State == "on" {
				items = append(items, item)
			}
		}
	}
	return items, nil
}

func (o *PostgresqlOrder) GetDB(dbName string) (*entities.Item, error) {

	dbItems, err := o.GetDBs()
	if err != nil {
		return nil, err
	}

	for _, item := range dbItems {
		if item.Data.Config.(entities.PostgresqlDbItemConfig).DbName == dbName {
			return &item, nil
		}
	}
	return nil, fmt.Errorf("there is no database with name '%s'", dbName)
}

func (o *PostgresqlOrder) GetUsers() ([]entities.PostgresqlItemDbUsers, error) {

	item, err := o.GetParentItem()
	if err != nil {
		return nil, err
	}

	config := item.Data.Config.(entities.PostgresqlItemConfig)
	return config.DBUsers, nil
}

func (o *PostgresqlOrder) GetPostgresqlVmItems() ([]entities.Item, error) {

	item, err := o.GetParentItem()
	if err != nil {
		return nil, err
	}

	if item.Type != "cluster" {
		return nil, errors.New("this is not cluster. Can't get VM with node role 'postgresql'")
	}

	vmItems, err := o.GetVMItems()
	if err != nil {
		return nil, err
	}

	if len(vmItems) == 0 {
		return nil, errors.New("get vm items have zero length")
	}

	var postgresqlVmItems []entities.Item
	for _, vm := range vmItems {
		nodeRoles := vm.Data.Config.(entities.VMItemConfig).NodeRoles
		for _, role := range nodeRoles {
			if role == "postgresql" {
				postgresqlVmItems = append(postgresqlVmItems, vm)
			}
		}
	}

	if len(postgresqlVmItems) > 2 {
		return nil, errors.New("vm items with node role 'postgresql' are more than 2")
	}
	if len(postgresqlVmItems) < 2 {
		return nil, errors.New("m items with node role 'postgresql' are less than 2")
	}
	return postgresqlVmItems, nil
}

func (o *PostgresqlOrder) GetExtraMount(path string) (*entities.ExtraMount, error) {

	item, err := o.GetParentItem()
	if err != nil {
		return nil, err
	}

	var vmItem entities.Item
	switch item.Type {
	case "cluster":
		vmItems, err := o.GetPostgresqlVmItems()
		if err != nil {
			return nil, err
		}
		vmItem = vmItems[0]
	case "app":
		vmItems, err := o.GetVMItems()
		if err != nil {
			return nil, err
		}

		if len(vmItems) != 1 {
			return nil, errors.New("number of VM is not equal to 1")
		}
		vmItem = vmItems[0]
	}

	extraMounts := vmItem.Data.Config.(entities.VMItemConfig).ExtraMounts
	if len(extraMounts) == 0 {
		return nil, errors.New("list of extra mount is empty")
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

func (o *PostgresqlOrder) CreateDb(name, adminPass string, dbEncoding *string, connectionLimit int64, async bool) error {

	var uri string
	var data map[string]interface{}
	if err := o.requiredState("on"); err != nil {
		return err
	}

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	itemConfig := item.Data.Config.(entities.PostgresqlItemConfig)
	envType := strings.ToLower(itemConfig.EnvironmentType)
	env := strings.ToLower(itemConfig.Environment)

	attrs := map[string]interface{}{
		"db_name":       name,
		"db_admin_pass": adminPass,
		"db_encoding":   dbEncoding,
		//"db_custom_encoding": dbCustomEncoding,
		"conn_limit": connectionLimit,
	}

	if dbEncoding == nil {
		attrs["db_custom_encoding"] = false
	} else {
		attrs["db_custom_encoding"] = true
	}

	data = map[string]interface{}{
		"item_id": item.ID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	uri = o.generateOrderdActionUri("postgresql_create_db")

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

	if (envType == "dev" || envType == "test") && env != "lt" && connectionLimit != 0 {
		err := o.SetConnectionLimit(connectionLimit, name, false)
		if err != nil {
			return err
		}
	}
	return err
}

func (o *PostgresqlOrder) CreateUser(
	user *entities.PostgresqlItemDbUsers,
	password string,
	async bool,
) error {

	if err := o.requiredState("on"); err != nil {
		return err
	}

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"db_name":       user.DbName,
		"user_name":     user.UserName,
		"user_password": password,
		"comment":       user.Comment,
		"dbms_role":     user.DbmsRole,
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

	uri := o.generateOrderdActionUri("create_dbms_user")
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

func (o *PostgresqlOrder) DeleteUser(name string, async bool) error {

	if err := o.requiredState("on"); err != nil {
		return err
	}

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"user_name": name,
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

	var action string
	switch item.Type {
	case "cluster":
		action = "postgresql_cluster_remove_dbms_user"
	case "app":
		action = "remove_dbms_user"
	}

	uri := o.generateOrderdActionUri(action)

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

func (o *PostgresqlOrder) DeleteDB(name string, async bool) error {

	if err := o.requiredState("on"); err != nil {
		return err
	}

	db, err := o.GetDB(name)
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"db_name": name,
	}

	data := map[string]interface{}{
		"item_id": db.ID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("postgresql_remove_db")

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

func (o *PostgresqlOrder) SetConnectionLimit(connectionLimit int64, dbName string, async bool) error {

	if err := o.requiredState("on"); err != nil {
		return err
	}

	db, err := o.GetDB(dbName)
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"db_name":    dbName,
		"conn_limit": connectionLimit,
	}

	data := map[string]interface{}{
		"item_id": db.ID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("postgresql_db_set_conn_limit")

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

func (o *PostgresqlOrder) RemoveConnectionLimit(dbName string, async bool) error {

	if err := o.requiredState("on"); err != nil {
		return err
	}

	db, err := o.GetDB(dbName)
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"db_name":    dbName,
		"conn_limit": -1,
	}

	data := map[string]interface{}{
		"item_id": db.ID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("postgresql_db_remove_conn_limit")

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

func (o *PostgresqlOrder) ChangeUserPassword(userName, password string, async bool) error {

	if err := o.requiredState("on"); err != nil {
		return err
	}

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"user_name":     userName,
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

	var action string
	switch item.Type {
	case "cluster":
		action = "postgresql_cluster_reset_db_user_password"
	case "app":
		action = "reset_db_user_password"
	}

	uri := o.generateOrderdActionUri(action)

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

func (o *PostgresqlOrder) ChangeOwnerPassword(userName, password string, async bool) error {

	if err := o.requiredState("on"); err != nil {
		return err
	}

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"user_name":     userName,
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

	var action string
	switch item.Type {
	case "cluster":
		action = "postgresql_cluster_reset_db_owner_password"
	case "app":
		action = "reset_db_owner_password"
	}

	uri := o.generateOrderdActionUri(action)

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

func (o *PostgresqlOrder) ChangeFlavor(flavor entities.Flavor, async bool) error {

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

	var action string
	switch item.Type {
	case "cluster":
		action = "resize_postgresql_cluster"
	case "app":
		action = "resize_two_layer"
	}

	uri := o.generateOrderdActionUri(action)

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

func (o *PostgresqlOrder) ExpandMountPoint(mountPoint entities.ExtraMount, async bool) (err error) {

	if err = o.requiredState("on"); err != nil {
		return
	}

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	extraMount, err := o.GetExtraMount(mountPoint.Path)
	if err != nil {
		return err
	}

	if extraMount == nil {
		return fmt.Errorf("can't find mount point with path = %s", mountPoint.Path)
	}

	if mountPoint == *extraMount {
		return fmt.Errorf("for expand mount point '%s' new size equal current size", mountPoint.Path)
	}

	increaseSize := mountPoint.Size - extraMount.Size
	if increaseSize < 10 {
		return fmt.Errorf(
			"can't change mount point less then 10 GB. Current is %d, new is %d",
			extraMount.Size, mountPoint.Size,
		)
	}

	var action string
	switch item.Type {
	case "cluster":
		action = "expand_mount_point_postgresql_pgdata"
	case "app":
		action = "postgresql_expand_mount_point_pg_data"
	}

	uri := o.generateOrderdActionUri(action)

	attrs := map[string]interface{}{
		"mount": mountPoint.Path,
		"size":  int(increaseSize),
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

func (o *PostgresqlOrder) AddMountPoint(mountPointPath string) (err error) {
	if mountPointPath == "/pg_backup" {
		err = o.AddMountPointPgBackup()
	}
	if mountPointPath == "/pg_walarchive" {
		err = o.AddMountPointPgWalarchive()
	}
	if mountPointPath == "/pg_audit" {
		err = o.AddMountPointPgAudit()
	}
	if mountPointPath == "/app/logs" {
		err = o.AddMountPointAppLogs()
	}
	if mountPointPath == "/app/backup" {
		err = o.AddMountPointAppBackup()
	}
	return err
}

func (o *PostgresqlOrder) AddMountPointPgBackup() (err error) {
	if err = o.requiredState("on"); err != nil {
		return
	}

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	var action string
	switch item.Type {
	case "cluster":
		action = "postgresql_cluster_add_mount_point_pg_backup"
	case "app":
		action = "postgresql_add_mount_point_pg_backup"
	}

	uri := o.generateOrderdActionUri(action)

	attrs := map[string]interface{}{
		"mount": "/pg_backup",
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

func (o *PostgresqlOrder) AddMountPointPgWalarchive() (err error) {
	if err = o.requiredState("on"); err != nil {
		return
	}

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	var action string
	switch item.Type {
	case "cluster":
		action = "postgresql_cluster_add_mount_point_pg_walarchive"
	case "app":
		action = "postgresql_add_mount_point_pg_walarchive"
	}

	uri := o.generateOrderdActionUri(action)

	attrs := map[string]interface{}{
		"mount": "/pg_walarchive",
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

func (o *PostgresqlOrder) AddMountPointPgAudit() (err error) {
	if err = o.requiredState("on"); err != nil {
		return
	}

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	var action string
	switch item.Type {
	case "cluster":
		action = "postgresql_cluster_add_mount_point_pg_audit"
	case "app":
		action = "postgresql_add_mount_point_pg_audit"
	}

	uri := o.generateOrderdActionUri(action)

	attrs := map[string]interface{}{
		"mount": "/pg_audit",
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

func (o *PostgresqlOrder) AddMountPointAppLogs() (err error) {
	if err = o.requiredState("on"); err != nil {
		return
	}

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	var action string
	switch item.Type {
	case "cluster":
		action = "postgresql_cluster_add_mount_point_app_logs"
	default:
		return fmt.Errorf("product_type = %s - does not support this action", item.Type)
	}

	uri := o.generateOrderdActionUri(action)

	attrs := map[string]interface{}{
		"mount": "/app/logs",
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

func (o *PostgresqlOrder) AddMountPointAppBackup() (err error) {
	if err = o.requiredState("on"); err != nil {
		return
	}

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	var action string
	switch item.Type {
	case "cluster":
		action = "postgresql_cluster_etcd_add_mount_point_app_backup"
	default:
		return fmt.Errorf("product_type = %s - does not support this action", item.Type)
	}

	uri := o.generateOrderdActionUri(action)

	attrs := map[string]interface{}{
		"mount": "/app/backup",
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
