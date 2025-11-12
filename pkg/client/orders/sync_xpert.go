package orders

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/requests"
)

type SyncXpertCluster struct {
	Order
	Attrs SyncXpertAttrs `json:"attrs"`
}

type SyncXpertAttrs struct {
	BasicAttrs
	ExtraNics       []any                  `json:"extra_nics"`
	Layout          string                 `json:"layout"`
	GeoDistribution bool                   `json:"geo_distribution"`
	DebeziumConfig  SyncXpertClusterConfig `json:"debezium_config"`
	ClusterName     string                 `json:"cluster_name"`
	AcceptVM        bool                   `json:"accept_vm"`
	NewInput1       map[string]any         `json:"newInput1"`
}

type SyncXpertClusterConfig struct {
	BasicAuthUser     string `json:"basic_auth_user"`
	ClusterGroupID    string `json:"cluster_group_id"`
	DebeziumVersion   string `json:"debezium_version"`
	KafkaCertCname    string `json:"kafka_cert_cname"`
	BootstrapServers  string `json:"bootstrap_servers"`
	BasicAuthPassword string `json:"basic_auth_password"`
}

type SyncXpertDbConnectroConfig struct {
	ConnectorName     string `json:"connector_name"`
	DbHostname        string `json:"db_address"`
	DbName            string `json:"db_name"`
	DbUser            string `json:"db_user"`
	DbPassword        string `json:"db_password"`
	DbIncludeListType string `json:"db_include_list_type"`
	DbIncludeList     string `json:"db_include_list"`
	DbPublicationName string `json:"db_publication_name"`
	DbSlotName        string `json:"db_slot_name"`
	DbTopicPref       string `json:"db_topic_pref"`

	SSLMode     string `json:"db_sslmode"`
	SSLRootCert string `json:"db_sslrootcert,omitempty"`
	SSLCert     string `json:"db_sslcert,omitempty"`
	SSLKey      string `json:"db_sslkey,omitempty"`
	SSLPassword string `json:"db_sslpassword,omitempty"`

	HeartbeatTopics   string `json:"db_heartbeat_topics,omitempty"`
	HeartbeatInterval int64  `json:"db_heartbeat_interval,omitempty"`
	HeartbeatQuery    string `json:"db_heartbeat_query,omitempty"`
}

func NewSyncXpertCluster(
	creds *auth.Credentials,
	projectName,
	productID string,
	attrs SyncXpertAttrs,
) *SyncXpertCluster {
	return &SyncXpertCluster{
		Attrs: attrs,
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
	}
}

func (o *SyncXpertCluster) GetAttrs() interface{} {
	return o.Attrs
}

// Создание заказа
func (o *SyncXpertCluster) Create(p CreateOrderPayload) error {

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

// Вертикальное масштабирование
func (o *SyncXpertCluster) VerticalScaling(flavor entities.Flavor, async bool) error {

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

	uri := o.generateOrderdActionUri("debezium_vertical_scaling")
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
	return nil
}

// Создание коннектора
func (o *SyncXpertCluster) CreateConnector(dbConfig SyncXpertDbConnectroConfig) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"database_type": "pg",
		"db_config":     dbConfig,
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

	uri := o.generateOrderdActionUri("debezium_add_connector")
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

// Удаление коннектора
func (o *SyncXpertCluster) DeleteConnector(name string) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"connector_name": name,
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

	uri1 := o.generateOrderdActionUri("debezium_delete_connector")
	uri2 := o.generateOrderdActionUri("debezium_cluster_del_connector")

	_, err = requests.SendRequest(o.Creds.AccessToken, uri1, "PATCH", payload, nil)
	if err != nil {
		_, err = requests.SendRequest(o.Creds.AccessToken, uri2, "PATCH", payload, nil)
		if err != nil {
			return err
		}
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}
	return nil
}

// Получения точки монтирования с указанным путем
func (o *SyncXpertCluster) GetExtraMount(path string) (*entities.ExtraMount, error) {
	vmItems, err := o.GetVMItems()
	if err != nil {
		return nil, err
	}

	extraMounts := vmItems[0].Data.Config.(entities.VMItemConfig).ExtraMounts
	if len(extraMounts) == 0 {
		return nil, nil
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

// Изменение группы доступа на всех ВМ в кластере
func (o *SyncXpertCluster) ChangeAccessGroup(changeAD entities.ADLogonGrants) error {

	vmItems, err := o.GetVMItems()
	if err != nil {
		return err
	}

	for _, vmItem := range vmItems {
		data := map[string]interface{}{
			"item_id": vmItem.ID,
			"order": map[string]interface{}{
				"attrs": map[string]interface{}{
					"members": changeAD.Groups,
					"role":    changeAD.Role,
				},
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

	return nil
}

// Добавление группы доступа на всех ВМ в кластере
func (o *SyncXpertCluster) AddAccessGroup(members entities.ADLogonGrants, permissions []string) error {

	vmItems, err := o.GetVMItems()
	if err != nil {
		return err
	}

	for _, vmItem := range vmItems {
		data := map[string]interface{}{
			"item_id": vmItem.ID,
			"order": map[string]interface{}{
				"attrs": map[string]interface{}{
					"members": members.Groups,
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

	return nil
}

// Удаление группы доступа на всех ВМ в кластере
func (o *SyncXpertCluster) DeleteAccessGroup(changeAD entities.ADLogonGrants) error {

	vmItems, err := o.GetVMItems()
	if err != nil {
		return err
	}

	for _, vmItem := range vmItems {
		data := map[string]interface{}{
			"item_id": vmItem.ID,
			"order": map[string]interface{}{
				"attrs": map[string]interface{}{
					"members": changeAD.Groups,
					"role":    changeAD.Role,
				},
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

	return nil
}

// Расширение точки монтирования на всех ВМ в кластере
func (o *SyncXpertCluster) ExpandMountPoint(mountPoint entities.ExtraMount) error {

	vmItems, err := o.GetVMItems()
	if err != nil {
		return err
	}

	stateOn := "on"
	stateOff := "off"

	for _, vmItem := range vmItems {
		if vmItem.Data.State != stateOn {
			return fmt.Errorf(
				"can't do action, vm.state is not equal %s, current state is %s", stateOn, stateOff,
			)
		}
	}

	extraMount, err := o.GetExtraMount(mountPoint.Path)
	if err != nil {
		return err
	}

	if extraMount == nil {
		return fmt.Errorf("can't find mount point with path = %s", mountPoint.Path)
	}

	increaseSize := mountPoint.Size - extraMount.Size
	if increaseSize < 10 {
		return fmt.Errorf(
			"can't change mount point less then 10 GB. Current is %d, new is %d",
			extraMount.Size, mountPoint.Size,
		)
	}
	for _, vmItem := range vmItems {
		data := map[string]interface{}{
			"item_id": vmItem.ID,
			"order": map[string]interface{}{
				"attrs": map[string]interface{}{
					"mount": mountPoint.Path,
					"size":  increaseSize,
				},
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
	}

	return nil
}
