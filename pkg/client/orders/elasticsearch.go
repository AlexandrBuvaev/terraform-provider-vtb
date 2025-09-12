package orders

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/requests"
)

type ElasticSearch struct {
	Order
	Attrs ElasticSearchAttrs `json:"attrs"`
}

type ElasticSearchAttrs struct {
	Accept                 bool                     `json:"accept"`
	Domain                 string                   `json:"domain"`
	Platform               string                   `json:"platform"`
	OnSupport              bool                     `json:"on_support"`
	OsVersion              string                   `json:"os_version"`
	DefaultNic             entities.DefaultNic      `json:"default_nic"`
	FlavorData             entities.Flavor          `json:"flavor_data"`
	NodesCount             NodesCountElastic        `json:"nodes_count"`
	ClusterName            string                   `json:"cluster_name"`
	FlavorMaster           entities.Flavor          `json:"flavor_master"`
	ADIntegration          bool                     `json:"ad_integration"`
	AdmAppGroups           []string                 `json:"adm_app_groups"`
	UserAppGroups          []string                 `json:"user_app_groups"`
	FluentdPassword        string                   `json:"fluentd_password"`
	AvailabilityZone       string                   `json:"availability_zone"`
	DataExtraMounts        []entities.ExtraMount    `json:"data_extra_mounts"`
	FlavorCoordinator      entities.Flavor          `json:"flavor_coordinator"`
	MasterExtraMounts      []entities.ExtraMount    `json:"master_extra_mounts"`
	ElasticSearchVersion   string                   `json:"elasticsearch_version"`
	CoordinatorExtraMounts []entities.ExtraMount    `json:"coordinator_extra_mounts"`
	ADLogonGrants          []entities.ADLogonGrants `json:"ad_logon_grants"`

	GeoDistribution *bool    `json:"geo_distribution,omitempty"`
	SystemAdmGroup  []string `json:"system_adm_groups,omitempty"`
	Layout          string   `json:"layout,omitempty"`

	InstallKibana     bool                  `json:"install_kibana"`
	KibanaPassword    string                `json:"kibana_password"`
	FlavorKibana      *entities.Flavor      `json:"flavor_kibana,omitempty"`
	KibanaLocation    *string               `json:"kibana_location,omitempty"`
	KibanaExtraMounts []entities.ExtraMount `json:"kibana_extra_mounts,omitempty"`

	CreatedWithOpenTofu bool `json:"created_with_opentofu"`
}

type NodesCountElastic struct {
	Data        int64 `json:"data"`
	Master      int64 `json:"master"`
	Coordinator int64 `json:"coordinator"`
}

func NewElasticSearch(creds *auth.Credentials, projectName, productID string, attrs ElasticSearchAttrs) *ElasticSearch {
	return &ElasticSearch{
		Order: Order{
			ProjectName: projectName,
			ProductID:   productID,
			Creds:       creds,
		},
		Attrs: attrs,
	}
}

func (o *ElasticSearch) Create(p CreateOrderPayload) error {

	payload, err := o.createPayload(p, o)
	if err != nil {
		return err
	}

	uri := generateCreateOrderUri(o.ProjectName)
	resp, err := requests.SendRequest(o.Creds.AccessToken, uri, "POST", payload, nil)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	trimmedBody := strings.Trim(string(bytes), "[]")
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

func (o *ElasticSearch) GetAttrs() interface{} {
	return o.Attrs
}

func (o *ElasticSearch) GetVMItemsByItemsRole(itemRole string) ([]entities.Item, error) {

	items, err := o.GetVMItems()
	if err != nil {
		return nil, err
	}

	var resultItems []entities.Item
	for _, item := range items {
		vmConfig, ok := item.Data.Config.(entities.VMItemConfig)
		if !ok {
			return nil, errors.New("invalid type for vmItem")
		}

		if slices.Contains(vmConfig.NodeRoles, itemRole) {
			resultItems = append(resultItems, item)
		}
	}

	return resultItems, nil
}

func (o *ElasticSearch) GetDataVmItems() (dataVmItems []entities.Item, err error) {
	dataVmItems, err = o.GetVMItemsByItemsRole("data")
	if err != nil {
		return nil, err
	}
	return dataVmItems, nil
}

func (o *ElasticSearch) GetCoordinatorVmItems() (coordinatorVmItems []entities.Item, err error) {
	coordinatorVmItems, err = o.GetVMItemsByItemsRole("coordinator")
	if err != nil {
		return nil, err
	}
	return coordinatorVmItems, nil
}

func (o *ElasticSearch) GetMasterVmItems() (masterVmItems []entities.Item, err error) {
	masterVmItems, err = o.GetVMItemsByItemsRole("master")
	if err != nil {
		return nil, err
	}
	return masterVmItems, nil
}

func (o *ElasticSearch) GetKibanaVmItems() (kibanaVmItems []entities.Item, err error) {
	kibanaVmItems, err = o.GetVMItemsByItemsRole("kibana")
	if err != nil {
		return nil, err
	}
	return kibanaVmItems, nil
}

func (o *ElasticSearch) ChangeFlavorDataNodes(flavor entities.Flavor) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"flavor":      flavor,
		"check_agree": true,
		"checking_the_reading_of_the_instruction": true,
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

	uri := o.generateOrderdActionUri("vertical_resize_opensearch_data_nodes")
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

func (o *ElasticSearch) ChangeFlavorCoordinatorNodes(flavor entities.Flavor) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"flavor":      flavor,
		"check_agree": true,
		"checking_the_reading_of_the_instruction": true,
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

	uri := o.generateOrderdActionUri("vertical_resize_opensearch_coordinator_nodes")
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

func (o *ElasticSearch) ChangeFlavorMasterNodes(flavor entities.Flavor) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"flavor":      flavor,
		"check_agree": true,
		"checking_the_reading_of_the_instruction": true,
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

	uri := o.generateOrderdActionUri("vertical_resize_opensearch_master_nodes")
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

func (o *ElasticSearch) ScaleOpenSearchCluster(nodesCount NodesCountElastic, environment string) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"on_support":     true,
		"nodes_count":    nodesCount,
		"ad_integration": true,
	}

	if strings.EqualFold(environment, "dev") {
		attrs["on_support"] = false
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

	uri := o.generateOrderdActionUri("enlarge_elastic_cluster")

	if strings.EqualFold(environment, "prod") {
		uri = o.generateOrderdActionUri("enlarge_elastic_cluster_geodistribution")
	}

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

func (o *ElasticSearch) ChangeOpenSearchKibanaPassword(newPassword string) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"user_name":       "kibana_server",
		"kibana_password": newPassword,
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

	uri := o.generateOrderdActionUri("elasticsearch_opensearch_reset_kibana_password")
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

func (o *ElasticSearch) ChangeOpenSearchFluentdPassword(newPassword string) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"user_name":        "fluentd",
		"fluentd_password": newPassword,
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

	uri := o.generateOrderdActionUri("elasticsearch_opensearch_reset_fluentd_password")
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

func (o *ElasticSearch) DeleteElasticSearch() error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"item_id": itemID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(map[string]interface{}{}),
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("delete_elasticsearch_opensearch")
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

func (o *ElasticSearch) GetDataVmItemExtraMount(path string) (*entities.ExtraMount, error) {

	dataVmItems, err := o.GetDataVmItems()
	if err != nil {
		return nil, err
	}

	vmItem, ok := dataVmItems[0].Data.Config.(entities.VMItemConfig)
	if !ok {
		return nil, fmt.Errorf("invalid type for vm item config")
	}

	for _, em := range vmItem.ExtraMounts {
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

func (o *ElasticSearch) ExpandDataAppExtraMount(extraMount entities.ExtraMount) error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	dataAppExtraMount, err := o.GetDataVmItemExtraMount("/app")
	if err != nil {
		return err
	}

	increaseSize := extraMount.Size - dataAppExtraMount.Size

	attrs := map[string]interface{}{
		"size":  increaseSize,
		"mount": "/app",
	}

	data := map[string]interface{}{
		"item_id": itemID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}

	uri := o.generateOrderdActionUri("expand_mount_data_opensearch")
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

	return nil
}
