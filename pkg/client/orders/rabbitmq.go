package orders

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"

	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/requests"
)

type RabbitMQ struct {
	Order
	Attrs RabbitMQAttrs `json:"attrs"`
}

type RabbitMQAttrs struct {
	ADIntegration       bool                     `json:"ad_integration"`
	ADLogonGrants       []entities.ADLogonGrants `json:"ad_logon_grants"`
	Domain              string                   `json:"domain"`
	DefaultNic          entities.DefaultNic      `json:"default_nic"`
	Flavor              entities.Flavor          `json:"flavor"`
	Platform            string                   `json:"platform"`
	OnSupport           bool                     `json:"on_support"`
	OsVersion           string                   `json:"os_version"`
	AvailabilityZone    string                   `json:"availability_zone"`
	Path                string                   `json:"path"`
	Size                int64                    `json:"size"`
	SizingCheck         bool                     `json:"sizing_check"`
	FileSystem          string                   `json:"file_system"`
	ExtraNics           []interface{}            `json:"extra_nics"`
	GeoDistribution     bool                     `json:"geo_distribution"`
	LayoutID            string                   `json:"layout"`
	ClusterName         string                   `json:"cluster_name"`
	RabbitMQVersion     string                   `json:"rabbitmq_version"`
	WebAdmingGroups     []string                 `json:"web_administrator_groups"`
	WebManagerGroups    []string                 `json:"web_manager_groups"`
	CreatedWithOpenTofu bool                     `json:"created_with_opentofu"`
}

type RabbitMQVerticalScalingAttrs struct {
	Mount             string                     `json:"mount"`
	Flavor            entities.Flavor            `json:"flavor"`
	Fstype            string                     `json:"fstype"`
	CheckAgree        bool                       `json:"check_agree"`
	SelectMode        string                     `json:"select_mode"`
	QuorumFlavor      entities.Flavor            `json:"quorum_flavor"`
	CurrentFlavor     string                     `json:"current_flavor"`
	QuorumExtraMounts entities.QuorumExtraMounts `json:"quorum_extra_mounts"`
	Size              int64                      `json:"size"`
}

type RabbitMQGorizontalScalingAttrs struct {
	CurrentLayout string `json:"current_layout"`
	NetSegment    string `json:"net_segment"`
	Layout        string `json:"layout"`
}

type RabbitMQUpgradeVersionAttrs struct {
	Maintance             bool   `json:"maintenance"`
	TargetRabbitMQVersion string `json:"target_rabbitmq_version"`
}

func NewRabbitMQ(
	creds *auth.Credentials,
	projectName,
	productID string,
	attrs RabbitMQAttrs,
) *RabbitMQ {
	return &RabbitMQ{
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
		Attrs: attrs,
	}
}

func (o *RabbitMQ) GetAttrs() interface{} {
	return o.Attrs
}

func (o *RabbitMQ) GetState() (string, error) {

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

func (o *RabbitMQ) GetRabbitMQNode() (*entities.Item, error) {
	VMs, err := o.GetVMItems()
	if err != nil {
		return nil, err
	}
	for i, vm := range VMs {
		vmConfig := VMs[i].Data.Config.(entities.VMItemConfig)
		if slices.Contains(vmConfig.NodeRoles, "rabbitmq") {
			return &vm, nil
		}
	}
	return nil, fmt.Errorf("can't find node with role rabbitmq")
}

func (o *RabbitMQ) Create(p CreateOrderPayload) error {

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

	ordersBody, _ := io.ReadAll(resp.Body)
	orderBody := strings.Trim(string(ordersBody), "[]")

	err = json.Unmarshal([]byte(orderBody), o)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}
	return nil
}

func (o *RabbitMQ) Delete(async bool) error {

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

	uri := o.generateOrderdActionUri("rabbitmq_remove_cluster_release")
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

type UpdateWebAccessGroupsAttrs struct {
	Accept   bool     `json:"accept"`
	Admins   []string `json:"administrator"`
	Managers []string `json:"manager"`

	ChangeRequest bool `json:"change_request,omitempty"`
}

func (o *RabbitMQ) UpdateWebAccessGroups(attrs UpdateWebAccessGroupsAttrs) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrsData := map[string]interface{}{
		"accept":         attrs.Accept,
		"administrator":  attrs.Admins,
		"manager":        attrs.Managers,
		"change_request": attrs.ChangeRequest,
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

	uri := o.generateOrderdActionUri("rabbitmq_edit_access_groups_on_the_web_release")
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

func (o *RabbitMQ) CreateVHosts(vhosts []entities.RabbitMQVhost) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"rabbitmq_vhosts": vhosts,
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

	uri := o.generateOrderdActionUri("rabbitmq_create_vhosts_release")
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

func (o *RabbitMQ) DeleteVHosts(vhosts []string) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"rabbitmq_vhosts_to_delete": vhosts,
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

	uri := o.generateOrderdActionUri("rabbitmq_delete_vhosts_release")
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

type RabbitMQUsers struct {
	RisCode string `json:"ris_code"`
	Name    string `json:"name"`
}

type RabbitqMQCreateUserAttrs struct {
	EnvPrefix     string          `json:"env_prefix"`
	Precheck      bool            `json:"precheck_client_user_name_exist"`
	RabbitMQUsers []RabbitMQUsers `json:"rabbitmq_users"`
}

func (o *RabbitMQ) CreateUsers(attrs *RabbitqMQCreateUserAttrs) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}
	attrsData := map[string]interface{}{
		"env_prefix":                      attrs.EnvPrefix,
		"precheck_client_user_name_exist": attrs.Precheck,
		"rabbitmq_users":                  attrs.RabbitMQUsers,
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

	uri := o.generateOrderdActionUri("rabbitmq_create_user_release")
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

func (o *RabbitMQ) DeleteUser(name string) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"name": name,
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

	uri := o.generateOrderdActionUri("rabbitmq_delete_users_release")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(10)
	if err != nil {
		return err
	}

	err = o.Sync()
	if err != nil {
		return err
	}
	return nil
}

func (o *RabbitMQ) UpdateVhostAccess(userName string, accesses []entities.RabbitMQVhostAccess) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"user_name":    userName,
		"vhost_access": accesses,
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

	uri := o.generateOrderdActionUri("rabbitmq_edit_vhost_access_release")
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

func (o *RabbitMQ) UpdateVhostAccessMultiply(input_permissions []entities.RabbitMQUserInputPermissions) error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"input_permissions": input_permissions,
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

	uri := o.generateOrderdActionUri("rabbitmq_edit_vhosts_access_release")
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

func (o *RabbitMQ) DeleteVhostAccess(access entities.RabbitMQVhostAccess) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrsData := map[string]interface{}{
		"user_name":  access.UserName,
		"vhost_name": access.VhostName,
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

	uri := o.generateOrderdActionUri("rabbitmq_delete_vhost_access_release")
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

func (o *RabbitMQ) VerticalScaling(attrs RabbitMQVerticalScalingAttrs) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrsData := map[string]interface{}{
		"mount":               attrs.Mount,
		"flavor":              attrs.Flavor,
		"fstype":              attrs.Fstype,
		"check_agree":         attrs.CheckAgree,
		"select_mode":         attrs.SelectMode,
		"quorum_flavor":       attrs.QuorumFlavor,
		"current_flavor":      attrs.CurrentFlavor,
		"quorum_extra_mounts": attrs.QuorumExtraMounts,
		"size":                attrs.Size,
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

	uri := o.generateOrderdActionUri("rabbitmq_vertical_scaling_release")
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

func (o *RabbitMQ) HorizontalScaling(attrs RabbitMQGorizontalScalingAttrs) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrsData := map[string]interface{}{
		"current_layout": attrs.CurrentLayout,
		"net_segment":    attrs.NetSegment,
		"layout":         attrs.Layout,
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

	uri := o.generateOrderdActionUri("rabbitmq_scaling_cluster")
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

func (o *RabbitMQ) UpgradeRabbitMQVersion(attrs RabbitMQUpgradeVersionAttrs) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrsData := map[string]interface{}{
		"maintance":               attrs.Maintance,
		"target_rabbitmq_version": attrs.TargetRabbitMQVersion,
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

	uri := o.generateOrderdActionUri("rabbitmq_upgrade_version")
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

func (o *RabbitMQ) UpdateReleaseVersion(accept bool) error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"accept": accept,
	}

	data := map[string]interface{}{
		"item_id": itemID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}

	uri := o.generateOrderdActionUri("rabbitmq_update_version_release")

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
