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
}

type SyncXpertClusterConfig struct {
	BasicAuthUser      string `json:"basic_auth_user"`
	ClusterGroupID     string `json:"cluster_group_id"`
	DebeziumVersion    string `json:"debezium_version"`
	KafkaCertCname     string `json:"kafka_cert_cname"`
	BootstrapServers   string `json:"bootstrap_servers"`
	BasicAuthPassword  string `json:"basic_auth_password"`
	ConfigStorageTopic string `json:"config_storage_topic"`
	OffsetStorageTopic string `json:"offset_storage_topic"`
	StatusStorageTopic string `json:"status_storage_topic"`
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

	SSLMode     string `json:"db_sslmode"`
	SSLRootCert string `json:"db_sslrootcert,omitempty"`
	SSLCert     string `json:"db_sslcert,omitempty"`
	SSLKey      string `json:"db_sslkey,omitempty"`
	SSLPassword string `json:"db_sslpassword,omitempty"`

	HeartbeatTopics   string `json:"db_heartbeat_topics,omitempty"`
	HeartbeatInterval int64  `json:"db_heartbeat_interval,omitempty"`
	HeartbeatQuery    string `json:"db_heartbeat_query,omitempty"`
}

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

	uri := o.generateOrderdActionUri("debezium_add_connector_cluster")
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

func (o *SyncXpertCluster) DeleteConnector(name string, async bool) error {

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

	uri := o.generateOrderdActionUri("debezium_cluster_del_connector")
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
