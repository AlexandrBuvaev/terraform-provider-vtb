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

type AirflowOrder interface {
	GetOrder() *Order
}

type AirflowStandalone struct {
	Order
	Attrs AirflowStandaloneAttrs `json:"attrs"`
}

type AirflowStandaloneAttrs struct {
	BasicAttrs
	OnBackup        bool  `json:"on_rubackup"`
	ExtraNics       []any `json:"extra_nics"`
	GeoDistribution bool  `json:"geo_distribution"`
	AirflowStandaloneConfig
}

type AirflowStandaloneConfig struct {
	AirflowVersion   string                   `json:"airflow_version"`
	Executor         string                   `json:"executor"`
	ClusterName      string                   `json:"cluster_name"`
	DeployGrants     []entities.ADLogonGrants `json:"deploy_grants"`
	WebConsoleGrants []entities.ADLogonGrants `json:"web_console_grants"`
	PostgresqlConfig AirflowPostgresConfig    `json:"postgresql_config"`
	ClientCerts      []AirflowClientCert      `json:"client_certificates"`
}

type AirflowClientCert struct {
	CertificateCN string `json:"certificate_cn"`
	Password      string `json:"password"`
}

func NewAirflowStandalone(
	creds *auth.Credentials,
	projectName,
	productID string,
	attrs AirflowStandaloneAttrs,
) *AirflowStandalone {
	return &AirflowStandalone{
		Attrs: attrs,
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
	}
}

func (o *AirflowStandalone) GetAttrs() interface{} {
	return o.Attrs
}

func (o *AirflowStandalone) GetID() string {
	return o.ID
}

func (o *AirflowStandalone) GetOrder() *Order {
	return &o.Order
}

func (o *AirflowStandalone) Create(p CreateOrderPayload) error {

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

func (o *AirflowStandalone) GetAirflowVMItem() (*entities.Item, error) {
	items, err := o.GetVMItems()
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		vmConfig := item.Data.Config.(entities.VMItemConfig)
		if slices.Contains(vmConfig.NodeRoles, "webserver") && slices.Contains(vmConfig.NodeRoles, "scheduler") {
			return &item, nil
		}
	}
	return nil, fmt.Errorf("can't find node with roles scheduler, webserver")
}

func (o *AirflowStandalone) GetAirflowExtraMount(path string) (*entities.ExtraMount, error) {
	item, err := o.GetAirflowVMItem()
	if err != nil {
		return nil, err
	}
	extraMounts := item.Data.Config.(entities.VMItemConfig).ExtraMounts
	//	var em e.VMExtraMount
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

func (o *AirflowStandalone) AirflowVerticalScaling(flavor entities.Flavor) error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"check_agree": true,
		"flavor":      flavor,
		"executor":    "localexecutor",
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
	// prepare request
	uri := o.generateOrderdActionUri("airflow_vertical_scaling")
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

func AirflowChangeDeployGroups(airflowOrder AirflowOrder, deployGrantsPlan []string) error {
	o := airflowOrder.GetOrder()

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"groups": deployGrantsPlan,
		"role":   "airflow_deploy",
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
	// prepare request
	uri := o.generateOrderdActionUri("airflow_change_deploy_group")
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

func AirflowChangeWebConsoleGroups(airflowOrder AirflowOrder, consoleGroupsPlan []entities.ADLogonGrants) error {
	o := airflowOrder.GetOrder()

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"ldap_groups": consoleGroupsPlan,
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
	// prepare request
	uri := o.generateOrderdActionUri("airflow_change_web_access")
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

func AirflowAddClientCert(airflowOrder AirflowOrder, cert AirflowClientCert) error {
	o := airflowOrder.GetOrder()

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"tls_password":     cert.Password,
		"client_cert_name": cert.CertificateCN,
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
	// prepare request
	uri := o.generateOrderdActionUri("airflow_create_client_cert")
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

func AirflowUpdateProduct(airflowOrder AirflowOrder, executor string) error {
	o := airflowOrder.GetOrder()

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"check_agree": true,
		"executor":    executor,
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
	// prepare request
	uri := o.generateOrderdActionUri("airflow_upgrade_product")
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

func AirflowChangeDBPassword(airflowOrder AirflowOrder, newPas string) error {
	o := airflowOrder.GetOrder()

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"db_password": newPas,
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
	// prepare request
	uri := o.generateOrderdActionUri("airflow_change_db_password")
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

func AirflowExpandMountPoint(airflowOrder AirflowOrder, mountPoint string, delta int64) error {
	o := airflowOrder.GetOrder()

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"size":  delta,
		"mount": mountPoint,
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
	// prepare request
	uri := o.generateOrderdActionUri("airflow_expand_mount_point")
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

// AirflowDeleteTwoLayer вызов базового действия "Удалить рекурсивно"
func AirflowDeleteTwoLayer(airflowOrder AirflowOrder, async bool) error {
	o := airflowOrder.GetOrder()

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

	uri := o.generateOrderdActionUri("airflow_delete_two_layer")
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

func (o *AirflowStandalone) ExpandMountPoint(mountPoint string, delta int64) error {
	vmItems, err := o.GetVMItems()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"size":  delta,
		"mount": mountPoint,
	}

	data := map[string]interface{}{
		"item_id": vmItems[0].ID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
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

	return nil
}

func (o *AirflowStandalone) AddAccessGroup(members entities.ADLogonGrants, premissions []string) error {

	vmItems, err := o.GetVMItems()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"members": members.Groups,
		"role": map[string]interface{}{
			"group_name":  members.Role,
			"permissions": premissions,
		},
	}

	data := map[string]interface{}{
		"item_id": vmItems[0].ID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
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
	return err
}

func (o *AirflowStandalone) ChangeAccessGroup(changeAD entities.ADLogonGrants) error {

	vmItems, err := o.GetVMItems()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"members": changeAD.Groups,
		"role":    changeAD.Role,
	}

	data := map[string]interface{}{
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
		"item_id": vmItems[0].ID,
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
	return err
}

func (o *AirflowStandalone) DeleteAccessGroup(changeAD entities.ADLogonGrants) error {

	vmItems, err := o.GetVMItems()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"members": changeAD.Groups,
		"role":    changeAD.Role,
	}

	data := map[string]interface{}{
		"item_id": vmItems[0].ID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
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
	return err
}
