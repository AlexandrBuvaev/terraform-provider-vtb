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

type ArtemisOrder struct {
	Order
	Attrs ArtemisClusterAttrs `json:"attrs"`
}

type ArtemisClusterAttrs struct {
	AMQP                bool                     `json:"AMQP"`
	CORE                bool                     `json:"CORE"`
	Path                string                   `json:"path"`
	Size                int64                    `json:"size"`
	FileSystem          string                   `json:"file_system"`
	Domain              string                   `json:"domain"`
	Flavor              entities.Flavor          `json:"flavor"`
	LayoutID            string                   `json:"layout"`
	Platform            string                   `json:"platform"`
	OnSupport           bool                     `json:"on_support"`
	OsVersion           string                   `json:"os_version"`
	DefaultNic          entities.DefaultNic      `json:"default_nic"`
	ClusterName         string                   `json:"cluster_name"`
	ADIntegration       bool                     `json:"ad_integration"`
	ADLogonGrants       []entities.ADLogonGrants `json:"ad_logon_grants"`
	GeoDistribution     bool                     `json:"geo_distribution"`
	TypeLoadBalancer    string                   `json:"type_load_balancer"`
	TitleProtocols      any                      `json:"title_protocols,omitempty"`
	TitleForMountPoint  any                      `json:"title_for_mount_point"`
	ArtemisVersion      string                   `json:"artemis_version"`
	AvailabilityZone    string                   `json:"availability_zone"`
	CreatedWithOpenTofu bool                     `json:"created_with_opentofu"`
}

type VTBArtemisVerticalScalingAttrs struct {
	ReadIT                 bool            `json:"read_it"`
	Flavor                 entities.Flavor `json:"flavor"`
	CurrentFlavor          string          `json:"current_flavor"`
	StateServiceRAM        int64           `json:"state_service_ram"`
	StateServiceCPU        int64           `json:"state_service_cpu"`
	StateServiceFlavorName string          `json:"state_service_flavor_name"`
	ChangeRequest          bool            `json:"change_request,omitempty"` // Согласованное ЗНИ (для PROD)
}

type CascadeAddressCreateAttrs struct {
	CascadeAddressPrefix    string `json:"cascade_address_prefix"`
	CascadeAddressName      string `json:"cascade_address_name"`
	AddressFullPolicy       string `json:"address_full_policy"`
	MaxSizeBytes            string `json:"max_size_bytes"`
	SlowConsumerPolicy      string `json:"slow_consumer_policy"`
	SlowConsumerCheckPeriod int64  `json:"slow_consumer_check_period"`
	SlowConsumerThreshold   int64  `json:"slow_consumer_threshold"`

	NoExpiryDelay  bool  `json:"no_expiry_delay"`
	MaxExpiryDelay int64 `json:"max_expiry_delay"`
	MinExpiryDelay int64 `json:"min_expiry_delay"`
}

type CascadeAddressUpdateAttrs struct {
	CascadeAddressName      string `json:"cascade_address_name"`
	AddressFullPolicy       string `json:"address_full_policy"`
	SlowConsumerThreshold   int64  `json:"slow_consumer_threshold"`
	SlowConsumerPolicy      string `json:"slow_consumer_policy"`
	SlowConsumerCheckPeriod int64  `json:"slow_consumer_check_period"`

	MaxSize      string `json:"max_size"`
	MaxSizeBytes string `json:"max_size_bytes"`

	// Ограничения времени хранения на брокере
	ChangeExpiryDelay bool            `json:"change_expiry_delay"`
	MaxExpiryDelay    int64           `json:"max_expiry_delay"`
	MinExpiryDelay    int64           `json:"min_expiry_delay"`
	NewExpiryDelay    *NewExpiryDelay `json:"expiry_delay,omitempty"`
}

type NewExpiryDelay struct {
	NoExpiryDelay     bool  `json:"no_expiry_delay"`
	NewMaxExpiryDelay int64 `json:"new_max_expiry_delay"`
	NewMinExpiryDelay int64 `json:"new_min_expiry_delay"`
}

type ChangeRoleAttrs struct {
	Role                string
	SecurityPolicyNames string
	UserNames           []string
}

type GorizontalScalingAttrs struct {
	ReadIt   bool  `json:"read_it"`
	Quantity int64 `json:"quantity"`
}

type SwitchPluginAttrs struct {
	Limits      bool  `json:"Limits"`
	UniqueID    bool  `json:"UniqueID"`
	ReadIt      bool  `json:"read_it"`
	BlockSize   int64 `json:"blockSize,omitempty"`
	PacketLimit int64 `json:"packetLimit,omitempty"`
	RateLimit   int64 `json:"rateLimit,omitempty"`
	SizeLimit   int64 `json:"sizeLimit,omitempty"`
}

type ArtemisTUZ struct {
	Username      string `json:"user_name"`
	UserOwnerCert string `json:"user_owner_cert"`
}

func NewArtemisOrder(
	creds *auth.Credentials,
	projectName,
	productID string,
	attrs ArtemisClusterAttrs,
) *ArtemisOrder {
	return &ArtemisOrder{
		Attrs: attrs,
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
	}
}

func (o *ArtemisOrder) GetAttrs() interface{} {
	return o.Attrs
}

func (o *ArtemisOrder) Create(p CreateOrderPayload) error {

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

func (o *ArtemisOrder) Delete() error {

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

	uri := o.generateOrderdActionUri("vtb-artemis_remove_cluster")

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

func (o *ArtemisOrder) SwitchProtocol(amqp, core bool) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"AMQP":    amqp,
		"CORE":    core,
		"read_it": true,
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

	uri := fmt.Sprintf(
		"order-service/api/v1/projects/%s/orders/%s/actions/vtb-artemis_switch_protocol",
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
	return nil
}

func (o *ArtemisOrder) VerticalScaling(attrs VTBArtemisVerticalScalingAttrs) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrsData := map[string]interface{}{
		"read_it":                   attrs.ReadIT,
		"flavor":                    attrs.Flavor,
		"current_flavor":            attrs.CurrentFlavor,
		"state_service_ram":         attrs.StateServiceRAM,
		"state_service_cpu":         attrs.StateServiceCPU,
		"state_service_flavor_name": attrs.StateServiceFlavorName,
		"change_request":            attrs.ChangeRequest,
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

	uri := o.generateOrderdActionUri("vtb-artemis_vertical_scaling_cluster")
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

func (o *ArtemisOrder) GetExtraMount(path string) (*entities.ExtraMount, error) {

	items, err := o.GetVMItems()
	if err != nil {
		return nil, err
	}

	extraMounts := items[0].Data.Config.(entities.VMItemConfig).ExtraMounts

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

func (o *ArtemisOrder) ExpandMountPoint(prodFlag bool, mountPoint entities.ExtraMount, async bool) (err error) {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	appMount, err := o.GetExtraMount(mountPoint.Path)
	if err != nil {
		return err
	}

	expandSize := mountPoint.Size - appMount.Size

	attrs := map[string]interface{}{
		"read_it":          true,
		"current_flavor":   fmt.Sprintf("%v", appMount.Size),
		"state_service_mp": appMount.Size,
		"expand_size":      expandSize,
	}

	data := map[string]interface{}{
		"item_id": itemID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}

	if prodFlag {
		data["order"].(map[string]interface{})["attrs"].(map[string]interface{})["change_request"] = true
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("vtb-artemis_expand_mount")

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

func (o *ArtemisOrder) CreateTUZ(users []ArtemisTUZ) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"users": users,
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

	uri := o.generateOrderdActionUri("vtb-artemis_create_tuz")

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

func (o *ArtemisOrder) DeleteTUZ(users []string) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"users": users,
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

	uri := o.generateOrderdActionUri("vtb-artemis_delete_group_tuz")

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

func (o *ArtemisOrder) UpdateTUZ(users []ArtemisTUZ) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"users": users,
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

	uri := o.generateOrderdActionUri("vtb-artemis_group_update_tuz")

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

func (o *ArtemisOrder) CascadeAdressCreate(attrs []CascadeAddressCreateAttrs) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrsData := map[string]interface{}{
		"cascades": attrs,
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

	uri := o.generateOrderdActionUri("vtb-artemis_cascade_address_creation")

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

func (o *ArtemisOrder) CascadeAdressDelete(attrs []string) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrsData := map[string]interface{}{
		"cascades": attrs,
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

	uri := o.generateOrderdActionUri("vtb-artemis_cascades_address_delete")

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

func (o *ArtemisOrder) CascadeAdressUpdate(attrs CascadeAddressUpdateAttrs) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrsData := map[string]interface{}{
		"cascade_address_name":       attrs.CascadeAddressName,
		"address_full_policy":        attrs.AddressFullPolicy,
		"slow_consumer_threshold":    attrs.SlowConsumerThreshold,
		"slow_consumer_policy":       attrs.SlowConsumerPolicy,
		"slow_consumer_check_period": attrs.SlowConsumerCheckPeriod,
		"max_size":                   attrs.MaxSize,
		"max_size_bytes":             attrs.MaxSizeBytes,
		"change_expiry_delay":        attrs.ChangeExpiryDelay,
		"max_expiry_delay":           attrs.MaxExpiryDelay,
		"min_expiry_delay":           attrs.MaxExpiryDelay,
	}

	if attrs.NewExpiryDelay != nil {
		attrsData["expiry_delay"] = attrs.NewExpiryDelay
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

	uri := o.generateOrderdActionUri("vtb-artemis_cascade_address_update")

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

func (o *ArtemisOrder) ChangeUserInRole(attrs ChangeRoleAttrs) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrsData := map[string]interface{}{
		"role":                 attrs.Role,
		"security_policy_name": attrs.SecurityPolicyNames,
		"user_names":           attrs.UserNames,
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

	payloadStr := string(payload)
	payloadStr = strings.Replace(payloadStr, `"user_names":null`, `"user_names":[]`, 1)
	payload = []byte(payloadStr)

	uri := o.generateOrderdActionUri("vtb-artemis_change_tuz_in_role")

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

func (o *ArtemisOrder) GorizontalScaling(quantity int64) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"read_it":  true,
		"quantity": quantity,
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

	uri := o.generateOrderdActionUri("vtb-artemis_scale")

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

func (o *ArtemisOrder) UpdateReleaseVersion(read_it bool) error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"read_it": read_it,
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

	uri := o.generateOrderdActionUri("vtb-artemis_update")

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

func (o *ArtemisOrder) SwitchPlugins(attrs SwitchPluginAttrs) error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrsData := map[string]interface{}{
		"Limits":      attrs.Limits,
		"UniqueID":    attrs.UniqueID,
		"read_it":     attrs.ReadIt,
		"blockSize":   attrs.BlockSize,
		"packetLimit": attrs.PacketLimit,
		"rateLimit":   attrs.RateLimit,
		"sizeLimit":   attrs.SizeLimit,
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

	uri := o.generateOrderdActionUri("vtb-artemis_switch_plugin")

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

func (o *ArtemisOrder) SwitchArtemisVersion(artemisVersion string) error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"artemis_version": artemisVersion,
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

	uri := o.generateOrderdActionUri("vtb-artemis_switch_version_artemis")

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
