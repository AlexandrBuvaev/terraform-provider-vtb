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
	GSLB_V1_PREFIX             = "gslb_cluster_v1"
	DELETE_CLUSTER             = "delete_cluster"
	GSLB_VERTICAL_SCALING      = "vertical_scaling"
	CHANGE_PASSWORDS           = "change_passwords"
	ENABLE_MM_HOSTS            = "enable_mm_hosts"
	DISABLE_MM_HOSTS           = "disable_mm_hosts"
	EXPAND_MOUNT_POINT_GSLB_V1 = "expand_mount_point_new"
	ENABLE_BGPAAS_GSLB_V1      = "enable_bgpaas"
	DISABLE_BGPAAS_GSLB_V1     = "disable_bgpaas"
)

type GSLBV1 struct {
	Order
	Attrs GSLBV1Attrs `json:"attrs"`
}

type GSLBV1Attrs struct {
	Domain           string                   `json:"domain"`
	DefaultNic       entities.DefaultNic      `json:"default_nic"`
	ADLogonGrants    []entities.ADLogonGrants `json:"ad_logon_grants"`
	Platform         string                   `json:"platform"`
	OnSupport        bool                     `json:"on_support"`
	OsVersion        string                   `json:"os_version"`
	ExtraMounts      []entities.ExtraMount    `json:"extra_mounts"`
	Flavor           entities.Flavor          `json:"flavor"`
	AvailabilityZone string                   `json:"availability_zone"`
	ExtraNics        []interface{}            `json:"extra_nics"`
	GeoDistribution  bool                     `json:"geo_distribution"`

	DesiredVersion   int64            `json:"desired_version"`
	OrderCredentials OrderCredentials `json:"order_credentials"`
	Layout           string           `json:"layout"`
	AddBGPaas        *bool            `json:"add_bgpaas"`
	DNSZone          string           `json:"dns_zone"`
}

type OrderCredentials struct {
	NginxLogin    string `json:"nginx_login"`
	ApiPassword   string `json:"api_password"`
	NginxPassword string `json:"nginx_password"`
}

func NewGSLBV1(creds *auth.Credentials, projectName, productID string, attrs GSLBV1Attrs) *GSLBV1 {
	return &GSLBV1{
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
		Attrs: attrs,
	}
}

func (o *GSLBV1) GetAttrs() interface{} {
	return o.Attrs
}

func (o *GSLBV1) Create(p CreateOrderPayload) error {
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

	return nil
}

func (o *GSLBV1) DeleteCluster() error {
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

	actionName, err := o.formatActionName(DELETE_CLUSTER)
	if err != nil {
		return err
	}

	_, err = requests.SendRequest(o.Creds.AccessToken, o.generateOrderdActionUri(actionName), "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}

	return nil
}

// update product actions
func (o *GSLBV1) VerticalScaling(flavor entities.Flavor) error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"accept": true,
		"flavor": flavor,
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

	actionName, err := o.formatActionName(GSLB_VERTICAL_SCALING)
	if err != nil {
		return err
	}
	_, err = requests.SendRequest(o.Creds.AccessToken, o.generateOrderdActionUri(actionName), "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}

	return nil
}

func (o *GSLBV1) ChangePasswords(targetCredential, password string) error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"select_credential": targetCredential,
	}

	switch targetCredential {
	case "nginx":
		attrs["nginx_password"] = password
	case "api":
		attrs["api_password"] = password
	default:
		return fmt.Errorf("invalid target_credential: '%s'. Expected: 'nginx' or 'api'", targetCredential)
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

	actionName, err := o.formatActionName(CHANGE_PASSWORDS)
	if err != nil {
		return err
	}

	_, err = requests.SendRequest(o.Creds.AccessToken, o.generateOrderdActionUri(actionName), "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}

	return nil
}

func (o *GSLBV1) EnableMMHosts(targetHosts []string, domain string) error {
	ItemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	filteredHosts, err := o.FilterTargetHosts("on", targetHosts)
	if err != nil {
		return err
	}

	if len(filteredHosts) == 0 {
		return nil
	}

	attrs := map[string]interface{}{
		"domain":       domain,
		"target_hosts": filteredHosts,
	}

	data := map[string]interface{}{
		"item_id": ItemID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	actionName, err := o.formatActionName(ENABLE_MM_HOSTS)
	if err != nil {
		return err
	}

	_, err = requests.SendRequest(o.Creds.AccessToken, o.generateOrderdActionUri(actionName), "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}

	return nil
}

func (o *GSLBV1) DisableMMHosts(targetHosts []string, domain string) error {
	ItemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	filteredHosts, err := o.FilterTargetHosts("off", targetHosts)
	if err != nil {
		return err
	}

	if len(filteredHosts) == 0 {
		return nil
	}

	attrs := map[string]interface{}{
		"domain":       domain,
		"target_hosts": filteredHosts,
	}

	data := map[string]interface{}{
		"item_id": ItemID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	actionName, err := o.formatActionName(DISABLE_MM_HOSTS)
	if err != nil {
		return err
	}

	_, err = requests.SendRequest(o.Creds.AccessToken, o.generateOrderdActionUri(actionName), "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}

	return nil
}

func (o *GSLBV1) ExpandMountPoint(mountPoint entities.ExtraMount) error {

	vmItems, err := o.GetVMItems()
	if err != nil {
		return err
	}

	extraMount, err := o.GetExtraMount(mountPoint.Path)
	if err != nil {
		return err
	}

	if extraMount == nil {
		err = fmt.Errorf("can't find mount point with path = %s", mountPoint.Path)
		return err
	}

	if mountPoint == *extraMount {
		return err
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
		"size":  increaseSize,
	}

	for _, vmItem := range vmItems {
		data := map[string]interface{}{
			"item_id": vmItem.ID,
			"order": map[string]interface{}{
				"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
			},
		}

		payload, err := json.Marshal(data)
		if err != nil {
			return err
		}

		_, err = requests.SendRequest(o.Creds.AccessToken, o.generateOrderdActionUri(EXPAND_MOUNT_POINT_GSLB_V1), "PATCH", payload, nil)
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

func (o *GSLBV1) SwitchBgPaaS(bgPaasFlag bool) error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	var actionName string
	attrs := map[string]interface{}{}
	switch bgPaasFlag {
	case true:
		actionName, err = o.formatActionName(ENABLE_BGPAAS_GSLB_V1)
		if err != nil {
			return err
		}
		attrs = map[string]interface{}{
			"add_bgpaas": true,
		}
	default:
		actionName, err = o.formatActionName(DISABLE_BGPAAS_GSLB_V1)
		if err != nil {
			return err
		}
		attrs = map[string]interface{}{
			"del_bgpaas": true,
		}
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

	_, err = requests.SendRequest(o.Creds.AccessToken, o.generateOrderdActionUri(actionName), "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}

	return nil
}

// helpers
func (o *GSLBV1) GetHostsWithTargetMMStatus(targetStatus string) ([]string, error) {
	targetMMStatusHosts := []string{}
	gslbAppItems, err := o.GetGSLBV1AppItems()
	if err != nil {
		return nil, err
	}

	for _, gslbAppItem := range gslbAppItems {
		gslbAppConfig, ok := gslbAppItem.Data.Config.(entities.GSLBAppItemConfig)
		if !ok {
			return nil, fmt.Errorf("invalid type for `gslbAppConfig`")
		}
		if gslbAppConfig.MaintenanceMode == targetStatus {
			targetMMStatusHosts = append(targetMMStatusHosts, gslbAppConfig.Name)
		}

	}

	return targetMMStatusHosts, nil

}

func (o *GSLBV1) GetGSLBV1AppConfigs() ([]entities.GSLBAppItemConfig, error) {
	gslbAppConfigs := []entities.GSLBAppItemConfig{}
	gslbAppItems, err := o.GetGSLBV1AppItems()
	if err != nil {
		return nil, err
	}

	for _, gslbAppItem := range gslbAppItems {
		gslbAppConfig, ok := gslbAppItem.Data.Config.(entities.GSLBAppItemConfig)
		if !ok {
			return nil, fmt.Errorf("invalid type for `gslbAppConfig`")
		}
		gslbAppConfigs = append(gslbAppConfigs, gslbAppConfig)

	}

	return gslbAppConfigs, nil

}

func (o *GSLBV1) GetGSLBV1AppItems() ([]entities.Item, error) {

	created, err := o.itemCreated()
	if err != nil {
		return nil, err
	}

	var items []entities.Item
	if created {
		for _, item := range o.Data {
			if item.Type == "gslb" && item.Data.Provider == "app_info" {
				items = append(items, item)
			}
		}
	}

	if len(items) == 0 {
		return nil, errors.New("can't find items with type 'gslb'")
	}
	return items, nil
}

func (o *GSLBV1) GetExtraMount(path string) (*entities.ExtraMount, error) {
	vmItems, err := o.GetVMItems()
	if err != nil {
		return nil, err
	}

	var size int64
	var fileSystem string
	for _, vmItem := range vmItems {
		extraMounts := vmItem.Data.Config.(entities.VMItemConfig).ExtraMounts
		if len(extraMounts) == 0 {
			return nil, fmt.Errorf("list extra mounts is empty")
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

func (o *GSLBV1) FilterTargetHosts(state string, targetHosts []string) ([]string, error) {
	var filteredHosts []string
	currentHosts, err := o.GetHostsWithTargetMMStatus(state)
	if err != nil {
		return nil, err
	}

	currentSet := make(map[string]struct{}, len(currentHosts))
	for _, host := range currentHosts {
		currentSet[host] = struct{}{}
	}

	// Фильтруем targetHosts: оставляем только те, что есть в currentSet
	for _, host := range targetHosts {
		if _, exists := currentSet[host]; !exists {
			filteredHosts = append(filteredHosts, host)
		}
	}

	return filteredHosts, nil

}

func (o *GSLBV1) formatActionName(actionName string) (string, error) {
	gslbv1Item, err := o.GetParentItem()
	if err != nil {
		return "", err
	}

	gslbv1ProviderName := gslbv1Item.Data.Provider

	if !strings.Contains(gslbv1ProviderName, GSLB_V1_PREFIX) {
		return "", fmt.Errorf("invalid format for `GSLBV1 Provider Name` not contains prefix: %s", GSLB_V1_PREFIX)
	}

	return fmt.Sprintf(
		"%s_%s",
		gslbv1ProviderName, actionName,
	), nil
}
