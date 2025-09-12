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

// Интерфейс для работы с продуктом Tarantool Data Grid
// Доступные действия: Создание заказа, отключение инстансов и включение инстансов.
type TarantoolCluster struct {
	Order
	Attrs TarantoolClusterAttrs `json:"attrs"`
}

// Аттрибуты для создания заказа Tarantool Data Grid
type TarantoolClusterAttrs struct {
	Domain           string              `json:"domain"`            // домен
	Platform         string              `json:"platform"`          // платформа
	DefaultNic       entities.DefaultNic `json:"default_nic"`       // сетевой сегмент
	OnSupport        bool                `json:"on_support"`        // группа сопровождения
	OsVersion        string              `json:"os_version"`        // версия дистрибутива
	AvailabilityZone string              `json:"availability_zone"` // зона доступности

	ADLogonGrants   []entities.ADLogonGrants `json:"ad_logon_grants"`  // группы доступа к АД
	ADIntegration   bool                     `json:"ad_integration"`   // интеграция АД
	GeoDistribution bool                     `json:"geo_distribution"` // Геораспределение

	AccessGroup         []string `json:"access_group"` // группы доступа Tarantool
	Layout              string   `json:"layout"`       // конфигурация кластера
	ExtraNics           []any    `json:"extra_nics"`
	TarantoolVersion    string   `json:"tarantool_version"` // версия приложения Tarantool
	CreatedWithOpenTofu bool     `json:"created_with_opentofu"`
}

type SwitchInstancesStateAttrs struct {
	Type      string   `json:"type"`
	Instances []string `json:"instances"`
}

type UpdateTarantoolVersionAttrs struct {
	TarantoolType       string `json:"tarantool_type"`
	NewTarantoolVersion string `json:"new_tarantool_version"`
}

func NewTarantoolCluster(creds *auth.Credentials, projectName, productID string, attrs TarantoolClusterAttrs) *TarantoolCluster {
	return &TarantoolCluster{
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
		Attrs: attrs,
	}
}

func (o *TarantoolCluster) GetAttrs() interface{} {
	return o.Attrs
}

// Создание заказа
func (o *TarantoolCluster) Create(p CreateOrderPayload) error {

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

	err = o.WaitSuccess(30)
	if err != nil {
		return err
	}

	return nil
}

// Запуск инстансов в зоне кластера
func (o *TarantoolCluster) EnableZoneInstances(attrs SwitchInstancesStateAttrs, zoneNumber int64) (err error) {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrsData := map[string]interface{}{
		"type":      attrs.Type,
		"instances": attrs.Instances,
	}

	data := map[string]interface{}{
		"item_id": itemID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrsData),
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	actionPostfix := fmt.Sprintf("tarantool_v2_start_instances_zone-%v", zoneNumber)

	uri := o.generateOrderdActionUri(actionPostfix)

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

// Отключение инстансов в зоне кластера
func (o *TarantoolCluster) DisableZoneInstances(attrs SwitchInstancesStateAttrs, zoneNumber int64) (err error) {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrsData := map[string]interface{}{
		"type":      attrs.Type,
		"instances": attrs.Instances,
	}

	data := map[string]interface{}{
		"item_id": itemID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrsData),
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	actionPostfix := fmt.Sprintf("tarantool_v2_stop_instances_zone-%v", zoneNumber)
	uri := o.generateOrderdActionUri(actionPostfix)

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

func (o *TarantoolCluster) GetZones() ([]string, error) {
	item, err := o.GetParentItem()
	if err != nil {
		return nil, err
	}

	config, ok := item.Data.Config.(entities.TarantoolClusterItemConfig)
	if !ok {
		return nil, fmt.Errorf("invalid config type, expected TarantoolClusterItemConfig")
	}

	clusterZones := make([]string, 0, len(config.Cluster))

	for zone := range config.Cluster {
		clusterZones = append(clusterZones, zone)
	}

	if len(clusterZones) == 0 {
		return nil, fmt.Errorf("no zones found in cluster configuration")
	}

	return clusterZones, nil
}

func (o *TarantoolCluster) GetZoneInstancesByStateFilter(zoneName, state string, cluster map[string][]entities.ZoneInstance) (instances []string, err error) {
	zoneInstances, ok := cluster[zoneName]
	if !ok {
		return nil, fmt.Errorf("zone %s not found in cluster", zoneName)
	}

	if state != "on" && state != "off" {
		return nil, fmt.Errorf("invalid state type, state must be euqal `on` or `off`. Current state: %s", state)
	}

	if len(zoneInstances) == 0 {
		return nil, fmt.Errorf("zone %s has no instances", zoneName)
	}

	for _, instance := range zoneInstances {
		if instance.State == state {
			instances = append(instances, instance.Instance)
		}
	}

	return instances, nil
}

func (o *TarantoolCluster) UpdateTarantoolVersion(attrs UpdateTarantoolVersionAttrs) (err error) {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrsData := map[string]interface{}{
		"tarantool_type":        attrs.TarantoolType,
		"new_tarantool_version": attrs.NewTarantoolVersion,
	}

	data := map[string]interface{}{
		"item_id": itemID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrsData),
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("tarantool_v2_update")

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
