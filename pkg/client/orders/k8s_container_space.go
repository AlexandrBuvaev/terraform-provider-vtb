package orders

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/contextkeys"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/references"
	"terraform-provider-vtb/pkg/client/requests"
)

type K8sContainerSpaceAttrs struct {
	Ratio               entities.RatioK8sContainerSpace        `json:"ratio"`
	Domain              string                                 `json:"domain"`
	Worker              entities.WorkerK8sContainerSpace       `json:"worker"`
	Platform            string                                 `json:"platform"`
	DataCenter          string                                 `json:"data_center"`
	DefaultNic          entities.DefaultNic                    `json:"default_nic"`
	ControlPlane        entities.ControlPlaneK8sContainerSpace `json:"control_plane"`
	IngressEnabled      bool                                   `json:"ingress_enabled"`
	Ingress             entities.IngressK8sContainerSpace      `json:"ingress"`
	CreatedWithOpenTofu bool                                   `json:"created_with_opentofu"`
}

type K8sContainerSpaceOrder struct {
	Order
	Attrs K8sContainerSpaceAttrs `json:"attrs"`
}

func (o *K8sContainerSpaceOrder) GetAttrs() interface{} {
	return o.Attrs
}

func (o *K8sContainerSpaceOrder) GetOrderID() string {
	return o.Order.ID
}

func NewK8sContainerSpaceOrder(
	creds *auth.Credentials,
	projectName,
	productID string,
	attrs K8sContainerSpaceAttrs,
) *K8sContainerSpaceOrder {
	return &K8sContainerSpaceOrder{
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
		Attrs: attrs,
	}
}

func (o *K8sContainerSpaceOrder) Create(p CreateOrderPayload) error {

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

func (o *K8sContainerSpaceOrder) Delete() error {

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

	actionName, err := o.formatActionName(DELETE_ORDER)
	if err != nil {
		return err
	}
	uri := fmt.Sprintf(
		"order-service/api/v1/projects/%s/orders/%s/actions/%s",
		o.ProjectName, o.ID, actionName,
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

func (o *K8sContainerSpaceOrder) K8sContainerSpaceRegionConfigureComponents(
	region entities.K8sConfigRegion,
	action string,
) error {

	data, _ := o.prepareBaseData()

	dataAttrs := map[string]interface{}{
		"name":       region.Name,
		"components": region.Components,
	}

	attrs := data["order"].(map[string]interface{})["attrs"].(map[string]interface{})

	for key, val := range dataAttrs {
		attrs[key] = val
	}

	return o.sendK8sContainerSpaceRequest(action, data)
}

func (o *K8sContainerSpaceOrder) K8sContainerSpaceRegionIngressChangeFlavor(
	component any,
	componentName string,
) error {

	data, _ := o.prepareBaseData()

	var flavor entities.K8sConfigNodeFlavor
	var name string

	if region, ok := any(component).(entities.K8sConfigRegion); ok {
		flavor = region.Flavor
		name = region.Name
	}

	if ingress, ok := any(component).(entities.K8sConfigIngress); ok {
		flavor = ingress.Flavor
		name = ingress.Name
	}

	componentData := map[string]interface{}{
		"flavor":    flavor,
		"name":      name,
		"agreement": true,
	}

	attrs := data["order"].(map[string]interface{})["attrs"].(map[string]interface{})

	for key, val := range componentData {
		attrs[key] = val
	}
	action := fmt.Sprintf("%s_%s", componentName, "flavor")

	return o.sendK8sContainerSpaceRequest(action, data)
}

func (o *K8sContainerSpaceOrder) K8sContainerSpaceRegionRequestsRatioConfig(
	region entities.K8sConfigRegion,
	action string,
	ratio *references.K8sClusterRatioData,
	config CommonActionParams,
) error {

	data, _ := o.prepareBaseData()

	flavorEmpty := entities.K8sConfigNodeFlavor{}
	ratioData := map[string]interface{}{
		"name":                   config.NetSegment,
		"container_cpu_ratio":    ratio.ContainerCPURatio,
		"container_memory_ratio": ratio.ContainerMemoryRatio,
	}

	componentData := map[string]interface{}{
		"flavor":                 flavorEmpty,
		"name":                   region.Name,
		"container_cpu_ratio":    region.ContainerCPURatio,
		"container_memory_ratio": region.ContainerMemoryRatio,
		"pod_cpu_max":            region.PodCPUMax,
		"pod_memory_max":         region.PodMemoryMax,
		"ratio":                  ratioData,
	}

	attrs := data["order"].(map[string]interface{})["attrs"].(map[string]interface{})

	for key, val := range componentData {
		attrs[key] = val
	}

	return o.sendK8sContainerSpaceRequest(action, data)
}

func (o *K8sContainerSpaceOrder) K8sContainerSpaceConfigureRegionIngress(
	action string,
	component string,
	name string,
) error {

	data, _ := o.prepareBaseData()

	componentData := map[string]interface{}{
		"name": name,
	}

	attrs := data["order"].(map[string]interface{})["attrs"].(map[string]interface{})

	for key, val := range componentData {
		attrs[key] = val
	}
	var actionName string
	switch action {
	case "delete":
		actionName = fmt.Sprintf("delete_%s", component)
	case "uncordon":
		actionName = fmt.Sprintf("%s_uncordon_nodes", component)
	}
	return o.sendK8sContainerSpaceRequest(actionName, data)
}

func (o *K8sContainerSpaceOrder) K8sContainerSpaceAddNodes(
	componentName string,
	newNodesCount int64,
	config CommonActionParams,
	component string,
) error {
	regionAddNodesLimit := config.RegionAddNodesMax
	ingressAddNodesLimit := config.IngressAddNodesMax

	var limit int64
	switch component {
	case "region":
		limit = regionAddNodesLimit
	case "ingress":
		limit = ingressAddNodesLimit
	default:
		limit = newNodesCount
	}

	data, _ := o.prepareBaseData()
	zone, _ := convertDCtoAZ(config.DataCenter)

	remainingNodes := newNodesCount

	for remainingNodes > 0 {
		currentBatch := limit
		if remainingNodes < limit {
			currentBatch = remainingNodes
		}

		componentData := map[string]interface{}{
			"availability_zone": zone,
			"net_segment":       config.NetSegment,
			"platform":          config.Platform,
			"name":              componentName,
			"new_nodes_count":   currentBatch,
		}

		attrs := data["order"].(map[string]interface{})["attrs"].(map[string]interface{})

		for key, val := range componentData {
			attrs[key] = val
		}

		err := o.sendK8sContainerSpaceRequest(fmt.Sprintf("%s_add_nodes", component), data)
		if err != nil {
			return fmt.Errorf("failed to add nodes: %v", err)
		}

		remainingNodes -= currentBatch
	}

	return nil
}

func (o *K8sContainerSpaceOrder) K8sContainerSpaceIngressSettings(
	ingress entities.K8sConfigIngress,
) error {

	data, _ := o.prepareBaseData()
	tcpUdpSettings := make([]entities.K8sConfigIngressTcpUdp, 0)

	if len(ingress.TcpUdpSettings) > 0 {
		tcpUdpSettings = ingress.TcpUdpSettings
	}

	componentData := map[string]interface{}{
		"features":         ingress.Features,
		"tcp_udp_settings": tcpUdpSettings,
		"name":             ingress.Name,
		"agreement":        true,
	}

	attrs := data["order"].(map[string]interface{})["attrs"].(map[string]interface{})

	for key, val := range componentData {
		attrs[key] = val
	}

	return o.sendK8sContainerSpaceRequest("ingress_settings", data)
}

func (o *K8sContainerSpaceOrder) K8sContainerSpaceCreateProject(
	quota entities.QuotaK8sSpaceProject,
	roles []entities.RolesK8sProject,
	userMark string,
) error {

	data, _ := o.prepareBaseData()

	dataAttrs := map[string]interface{}{
		"quota":     quota,
		"roles":     roles,
		"user_mark": userMark,
	}

	attrs := data["order"].(map[string]interface{})["attrs"].(map[string]interface{})

	for key, val := range dataAttrs {
		attrs[key] = val
	}

	return o.sendK8sContainerSpaceRequest("create_project", data)
}

func (o *K8sContainerSpaceOrder) K8sContainerSpaceUpdateProject(
	quota entities.QuotaK8sSpaceProject,
	roles []entities.RolesK8sProject,
	itemID string,
) error {

	data := map[string]interface{}{
		"item_id": itemID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(map[string]interface{}{}),
		},
	}

	dataAttrs := map[string]interface{}{
		"quota": quota,
		"roles": roles,
	}

	attrs := data["order"].(map[string]interface{})["attrs"].(map[string]interface{})

	for key, val := range dataAttrs {
		attrs[key] = val
	}

	return o.sendK8sContainerSpaceRequest("update_project", data)
}

func (o *K8sContainerSpaceOrder) K8sContainerSpaceDeleteProject(
	itemID string,
) error {

	data := map[string]interface{}{
		"item_id": itemID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(map[string]interface{}{}),
		},
	}

	return o.sendK8sContainerSpaceRequest("delete_project", data)
}

func (o *K8sContainerSpaceOrder) sendK8sContainerSpaceRequest(
	action string,
	data map[string]interface{},
) error {

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	actionName, err := o.formatActionName(action)
	if err != nil {
		return err
	}

	uri := fmt.Sprintf(
		"order-service/api/v1/projects/%s/orders/%s/actions/%s",
		o.ProjectName, o.ID, actionName,
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

func (o *K8sContainerSpaceOrder) formatActionName(actionName string) (string, error) {
	spaceItem, err := o.GetParentItem()
	if err != nil {
		return "", err
	}
	item := spaceItem.Data.Config.(entities.K8sContainerSpaceConfig)

	versions := strings.Split(item.ControlPlane.ProductVersion, ".")
	if len(versions) < 2 {
		return "", err
	}

	return fmt.Sprintf(
		"%s_%s_space_%s",
		KUBERNETES_V1_PREFIX, versions[1], actionName,
	), nil
}

func (o *K8sContainerSpaceOrder) prepareBaseData() (map[string]interface{}, error) {

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

func (o *K8sContainerSpaceOrder) K8sProjectAddFullComponent(
	ctx context.Context,
	planComponent any,
	component string,
) error {

	roles := []entities.RolesK8sProject{}
	if componentData, ok := planComponent.(interface {
		GetRoles() []entities.RolesK8sProject
	}); ok {
		roles = componentData.GetRoles()
	}

	data, err := o.prepareComponentData(ctx, planComponent, component, roles)
	if err != nil {
		return err
	}

	return o.sendK8sProjectRequest(component, "add", data)
}

func (o *K8sContainerSpaceOrder) K8sProjectUpdateFullComponent(
	ctx context.Context,
	roles []entities.RolesK8sProject,
	component string,
) error {

	itemID, err := getItemIDFromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to process item ID : %w", err)
	}

	data, _ := o.prepareProjectBaseData(itemID)
	data["order"].(map[string]interface{})["attrs"].(map[string]interface{})["roles"] = roles

	return o.sendK8sProjectRequest(component, "update", data)
}

func (o *K8sContainerSpaceOrder) K8sProjectAddOmniCertificate(
	ctx context.Context,
	appName,
	certName,
	component string,
) error {

	itemID, err := getItemIDFromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to process item ID : %w", err)
	}

	data, _ := o.prepareProjectBaseData(itemID)

	data["order"].(map[string]interface{})["attrs"].(map[string]interface{})["app_name"] = appName
	data["order"].(map[string]interface{})["attrs"].(map[string]interface{})["client_name"] = certName

	return o.sendK8sProjectRequest(component, "add", data)
}

func (o *K8sContainerSpaceOrder) K8sProjectDeleteOmniCertificate(ctx context.Context, name, component string) error {

	itemID, err := getItemIDFromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to process item ID : %w", err)
	}

	data, _ := o.prepareProjectBaseData(itemID)
	data["order"].(map[string]interface{})["attrs"].(map[string]interface{})["name"] = name

	return o.sendK8sProjectRequest(component, "delete", data)
}

func (o *K8sContainerSpaceOrder) K8sProjectAddComponent(ctx context.Context, component string) error {

	itemID, err := getItemIDFromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to process item ID : %w", err)
	}

	data, _ := o.prepareProjectBaseData(itemID)

	return o.sendK8sProjectRequest(component, "add", data)
}

func (o *K8sContainerSpaceOrder) K8sProjectDeleteComponent(ctx context.Context, component string) error {

	itemID, err := getItemIDFromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to process item ID : %w", err)
	}

	data, _ := o.prepareProjectBaseData(itemID)

	return o.sendK8sProjectRequest(component, "delete", data)
}

func (o *K8sContainerSpaceOrder) sendK8sProjectRequest(
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

func (o *K8sContainerSpaceOrder) prepareComponentData(
	ctx context.Context,
	planComponent any,
	component string,
	roles []entities.RolesK8sProject,
) (map[string]interface{}, error) {

	itemID, err := getItemIDFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to process item ID : %w", err)
	}

	data, _ := o.prepareProjectBaseData(itemID)

	data["order"].(map[string]interface{})["attrs"].(map[string]interface{})["roles"] = roles

	switch component {

	case "istio":

		var controlPlane string
		val := reflect.ValueOf(planComponent)
		method := val.MethodByName("GetControlPlane")
		if method.IsValid() {
			res := method.Call(nil)
			if len(res) > 0 {
				controlPlane = res[0].Interface().(string)
			}
		}

		data["order"].(map[string]interface{})["attrs"].(map[string]interface{})["control_plane"] = controlPlane

	case "chaos_mesh":

		data["order"].(map[string]interface{})["attrs"].(map[string]interface{})["roles"] = nil
	}

	return data, nil
}

func (o *K8sContainerSpaceOrder) prepareProjectBaseData(itemID string) (map[string]interface{}, error) {

	data := map[string]interface{}{
		"item_id": itemID,
		"order": map[string]interface{}{
			"attrs": map[string]interface{}{
				"created_with_terraform": true,
			},
		},
	}

	return data, nil
}

func getItemIDFromContext(ctx context.Context) (string, error) {
	val := ctx.Value(contextkeys.ItemIDKey)
	if val == nil {
		return "", fmt.Errorf("item ID not found in context")
	}

	id, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("invalid item ID type in context")
	}

	return id, nil
}
