package orders

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/requests"
)

type K8sProjectAttrs struct {
	Quota               entities.QuotaK8sProject        `json:"quota"`
	Roles               []entities.RolesK8sProject      `json:"roles"`
	Ingress             string                          `json:"ingress"`
	UserMark            string                          `json:"user_mark"`
	DataCenter          string                          `json:"data_center"`
	NetSegment          string                          `json:"net_segment"`
	ResourcePool        entities.ResourcePoolK8sProject `json:"resource_pool"`
	FinProjectID        string                          `json:"financial_project"`
	CreatedWithOpenTofu bool                            `json:"created_with_opentofu"`
}

type K8sProjectOrder struct {
	Order
	Attrs K8sProjectAttrs `json:"attrs"`
}

func (o *K8sProjectOrder) GetAttrs() interface{} {
	return o.Attrs
}

func NewK8sProjectOrder(
	creds *auth.Credentials,
	projectName,
	productID string,
	attrs K8sProjectAttrs,
) *K8sProjectOrder {
	return &K8sProjectOrder{
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
		Attrs: attrs,
	}
}

func (o *K8sProjectOrder) CreateOrder(p CreateOrderPayload) error {

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

	err = o.WaitSuccess(5)
	if err != nil {
		return err
	}
	return nil
}

func (o *K8sProjectOrder) DeleteOrder() error {

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

	uri := fmt.Sprintf(
		"order-service/api/v1/projects/%s/orders/%s/actions/delete_kubernetes_project",
		o.ProjectName, o.ID,
	)
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(5)
	if err != nil {
		return err
	}

	return nil
}

func (o *K8sProjectOrder) UpdateK8sProject(
	updatesQuota entities.QuotaK8sProject,
	updatesRoles []entities.RolesK8sProject,
) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"roles": updatesRoles,
		"quota": updatesQuota,
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
		"order-service/api/v1/projects/%s/orders/%s/actions/update_kubernetes_project",
		o.ProjectName, o.ID,
	)
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(5)
	if err != nil {
		return err
	}

	return nil
}

func (o *K8sProjectOrder) K8sProjectAddFullComponent(
	planComponent any,
	component string,
) error {

	roles := []entities.RolesK8sProject{}
	if componentData, ok := planComponent.(interface {
		GetRoles() []entities.RolesK8sProject
	}); ok {
		roles = componentData.GetRoles()
	}

	data, err := o.prepareComponentData(planComponent, component, roles)
	if err != nil {
		return err
	}

	return o.sendK8sProjectRequest(component, "add", data)
}

func (o *K8sProjectOrder) K8sProjectUpdateFullComponent(
	roles []entities.RolesK8sProject,
	component string,
) error {

	data, _ := o.prepareBaseData()
	data["order"].(map[string]interface{})["attrs"].(map[string]interface{})["roles"] = roles

	return o.sendK8sProjectRequest(component, "update", data)
}

func (o *K8sProjectOrder) K8sProjectAddOmniCertificate(
	appName,
	certName,
	component string,
) error {

	data, _ := o.prepareBaseData()

	data["order"].(map[string]interface{})["attrs"].(map[string]interface{})["app_name"] = appName
	data["order"].(map[string]interface{})["attrs"].(map[string]interface{})["client_name"] = certName

	return o.sendK8sProjectRequest(component, "add", data)
}

func (o *K8sProjectOrder) K8sProjectDeleteOmniCertificate(name, component string) error {

	data, _ := o.prepareBaseData()
	data["order"].(map[string]interface{})["attrs"].(map[string]interface{})["name"] = name

	return o.sendK8sProjectRequest(component, "delete", data)
}

func (o *K8sProjectOrder) K8sProjectAddComponent(component string) error {

	data, _ := o.prepareBaseData()

	return o.sendK8sProjectRequest(component, "add", data)
}

func (o *K8sProjectOrder) K8sProjectDeleteComponent(component string) error {

	data, _ := o.prepareBaseData()

	return o.sendK8sProjectRequest(component, "delete", data)
}

func (o *K8sProjectOrder) sendK8sProjectRequest(
	component,
	action string,
	data map[string]interface{},
) error {

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := fmt.Sprintf(
		"order-service/api/v1/projects/%s/orders/%s/actions/kubernetes_project_%s_%s",
		o.ProjectName, o.ID, action, component,
	)

	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(5)
	if err != nil {
		return err
	}

	return nil
}

func (o *K8sProjectOrder) prepareComponentData(
	planComponent any,
	component string,
	roles []entities.RolesK8sProject,
) (map[string]interface{}, error) {

	data, _ := o.prepareBaseData()

	data["order"].(map[string]interface{})["attrs"].(map[string]interface{})["roles"] = roles

	switch component {

	case "istio":

		var controlPlane string
		val := reflect.ValueOf(planComponent)

		if val.Kind() == reflect.Ptr {
			val = val.Elem()
		}
		field := val.FieldByName("ControlPlane")

		if field.IsValid() && field.Kind() == reflect.String {
			controlPlane = field.String()
		}

		data["order"].(map[string]interface{})["attrs"].(map[string]interface{})["control_plane"] = controlPlane

	case "chaos_mesh":

		data["order"].(map[string]interface{})["attrs"].(map[string]interface{})["roles"] = nil
	}

	return data, nil
}

func (o *K8sProjectOrder) prepareBaseData() (map[string]interface{}, error) {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return nil, err
	}

	data := map[string]interface{}{
		"item_id": itemID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(map[string]interface{}{}),
		},
	}

	return data, nil
}
