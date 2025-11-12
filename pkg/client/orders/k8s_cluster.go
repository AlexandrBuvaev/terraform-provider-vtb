package orders

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/references"
	"terraform-provider-vtb/pkg/client/requests"
)

const (
	KUBERNETES_V1_PREFIX = "kubernetes_v1"
)

type CommonActionParams struct {
	Platform            string
	NetSegment          string
	DataCenter          string
	RegionAddNodesMax   int64
	IngressAddNodesMax  int64
	RegionIngressAddMax int64
}

type K8sClusterAttrs struct {
	DataCenter           string                            `json:"data_center"`
	Platform             string                            `json:"platform"`
	Domain               string                            `json:"domain"`
	BalancerDnsZone      string                            `json:"balancer_dns_zone"`
	ControlPanelSize     string                            `json:"control_panel_size"`
	ContainerCPURatio    int64                             `json:"container_cpu_ratio"`
	ContainerMemoryRatio int64                             `json:"container_memory_ratio"`
	DefaultNic           entities.DefaultNic               `json:"default_nic"`
	Version              entities.VersionK8sCluster        `json:"version"`
	Worker               entities.WorkerIngressK8sCluster  `json:"worker"`
	Ingress              entities.WorkerIngressK8sCluster  `json:"ingress"`
	Features             entities.K8sClusterConfigFeatures `json:"features"`
	CreatedWithOpenTofu  bool                              `json:"created_with_opentofu"`
	GslbOnly             bool                              `json:"gslb_only"`
}

type K8sClusterOrder struct {
	Order
	Attrs K8sClusterAttrs `json:"attrs"`
}

func (o *K8sClusterOrder) GetAttrs() interface{} {
	return o.Attrs
}

func NewK8sClusterOrder(
	creds *auth.Credentials,
	projectName,
	productID string,
	attrs K8sClusterAttrs,
) *K8sClusterOrder {
	return &K8sClusterOrder{
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
		Attrs: attrs,
	}
}

func (o *K8sClusterOrder) Create(p CreateOrderPayload) error {

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

func (o *K8sClusterOrder) Delete() error {

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

func (o *K8sClusterOrder) K8sClusterUpdate(
	config entities.K8sClusterConfig,
) error {

	data, _ := o.prepareBaseData()

	componentData := make(map[string]interface{})
	componentData["unavailable"] = 0
	componentData["version"] = map[string]interface{}{
		"k8s_version":     config.Version,
		"product_version": config.ProductVersion,
	}

	attrs := data["order"].(map[string]interface{})["attrs"].(map[string]interface{})

	for key, val := range componentData {
		attrs[key] = val
	}

	return o.sendK8sClusterUpdateRequest(config.ProductVersion, data)
}

// control plane
func (o *K8sClusterOrder) K8sClusterChangeFlavor(
	component entities.K8sClusterConfigControlPlane,
	componentName string,
) error {

	data, _ := o.prepareBaseData()

	componentData := map[string]interface{}{
		"flavor":    component.Flavor,
		"role_name": component.RoleName,
		"agreement": true,
	}

	attrs := data["order"].(map[string]interface{})["attrs"].(map[string]interface{})

	for key, val := range componentData {
		attrs[key] = val
	}
	action := fmt.Sprintf("%s_%s", componentName, "flavor")

	return o.sendK8sClusterRequest(action, data)
}

// add delete
func (o *K8sClusterOrder) K8sClusterRegionConfigureComponents(
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

	return o.sendK8sClusterRequest(action, data)
}

func (o *K8sClusterOrder) K8sClusterRegionIngressChangeFlavor(
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

	return o.sendK8sClusterRequest(action, data)
}

func (o *K8sClusterOrder) K8sClusterIngressSettings(
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

	return o.sendK8sClusterRequest("ingress_settings", data)
}

func (o *K8sClusterOrder) K8sClusterRequestsRatioConfig(
	planConfig entities.Limits,
	limits *references.K8sClusterRatioData,
) error {

	data, _ := o.prepareBaseData()

	componentData := map[string]interface{}{
		"container_cpu_ratio":    planConfig.ContainerCPURatio,
		"container_memory_ratio": planConfig.ContainerMemoryRatio,
		"limits":                 limits,
	}

	attrs := data["order"].(map[string]interface{})["attrs"].(map[string]interface{})

	for key, val := range componentData {
		attrs[key] = val
	}

	return o.sendK8sClusterRequest("requests_ratio_config", data)
}

func (o *K8sClusterOrder) K8sClusterConfigureIstioCP(
	istioCP entities.K8sClusterConfigIstioControlPlane,
	action string,
) error {

	data, _ := o.prepareBaseData()
	componentData := make(map[string]interface{})
	componentData["flavor"] = istioCP.Flavor
	componentData["options"] = map[string]interface{}{
		"eventrouter": istioCP.Options.Eventrouter,
	}

	if action != "create" {
		componentData["name"] = istioCP.Name
	}

	attrs := data["order"].(map[string]interface{})["attrs"].(map[string]interface{})

	for key, val := range componentData {
		attrs[key] = val
	}

	actionName := fmt.Sprintf("%s_istio_control_plane", action)

	return o.sendK8sClusterRequest(actionName, data)
}

func (o *K8sClusterOrder) K8sClusterConfigureIstioOptions(
	istioOptions *entities.K8sClusterConfigIstioOptions,
	optionsFlavor string,
) error {

	data, _ := o.prepareBaseData()
	componentData := map[string]interface{}{
		"options_flavor": optionsFlavor,
		"options":        istioOptions,
	}

	attrs := data["order"].(map[string]interface{})["attrs"].(map[string]interface{})

	for key, val := range componentData {
		attrs[key] = val
	}

	return o.sendK8sClusterRequest("configure_istio_options", data)
}

func (o *K8sClusterOrder) K8sClusterAddRegion(
	region entities.K8sConfigRegion,
	config CommonActionParams,
) error {

	data, _ := o.prepareBaseData()
	zone, _ := convertDCtoAZ(config.DataCenter)
	nodesLimit := config.RegionIngressAddMax

	componentData := map[string]interface{}{
		"availability_zone":      zone,
		"net_segment":            config.NetSegment,
		"platform":               config.Platform,
		"flavor":                 region.Flavor,
		"iscodes":                region.IsCodes,
		"container_cpu_ratio":    region.ContainerCPURatio,
		"container_memory_ratio": region.ContainerMemoryRatio,
		"pod_cpu_max":            region.PodCPUMax,
		"pod_memory_max":         region.PodMemoryMax,
	}

	if region.Size > nodesLimit {
		componentData["size"] = nodesLimit
	} else {
		componentData["size"] = region.Size
	}

	attrs := data["order"].(map[string]interface{})["attrs"].(map[string]interface{})

	for key, val := range componentData {
		attrs[key] = val
	}

	return o.sendK8sClusterRequest("add_region", data)
}

func (o *K8sClusterOrder) K8sClusterConfigureRegionIngress(
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
	return o.sendK8sClusterRequest(actionName, data)
}

func (o *K8sClusterOrder) K8sClusterRegionRequestsRatioConfig(
	region entities.K8sConfigRegion,
	action string,
) error {

	data, _ := o.prepareBaseData()

	componentData := map[string]interface{}{
		"flavor":                 region.Flavor,
		"name":                   region.Name,
		"container_cpu_ratio":    region.ContainerCPURatio,
		"container_memory_ratio": region.ContainerMemoryRatio,
		"pod_cpu_max":            region.PodCPUMax,
		"pod_memory_max":         region.PodMemoryMax,
	}

	attrs := data["order"].(map[string]interface{})["attrs"].(map[string]interface{})

	for key, val := range componentData {
		attrs[key] = val
	}

	return o.sendK8sClusterRequest(action, data)
}

func (o *K8sClusterOrder) K8sClusterRegionSetCodes(
	region entities.K8sConfigRegion,
	action string,
) error {

	data, _ := o.prepareBaseData()

	componentData := map[string]interface{}{
		"iscodes": region.IsCodes,
		"name":    region.Name,
	}

	attrs := data["order"].(map[string]interface{})["attrs"].(map[string]interface{})

	for key, val := range componentData {
		attrs[key] = val
	}

	return o.sendK8sClusterRequest(fmt.Sprintf("region_set_%s", action), data)
}

func (o *K8sClusterOrder) K8sClusterAddIngress(
	ingress entities.K8sConfigIngress,
	config CommonActionParams,
) error {

	data, _ := o.prepareBaseData()
	zone, _ := convertDCtoAZ(config.DataCenter)
	nodesLimit := config.RegionIngressAddMax

	componentData := map[string]interface{}{
		"availability_zone": zone,
		"net_segment":       config.NetSegment,
		"platform":          config.Platform,
		"flavor":            ingress.Flavor,
		"iscodes":           ingress.IsCodes,
	}

	if ingress.Size > nodesLimit {
		componentData["size"] = nodesLimit
	} else {
		componentData["size"] = ingress.Size
	}

	attrs := data["order"].(map[string]interface{})["attrs"].(map[string]interface{})

	for key, val := range componentData {
		attrs[key] = val
	}

	return o.sendK8sClusterRequest("add_ingress", data)
}

// region ingress infra
// func (o *K8sClusterOrder) K8sClusterDeleteNodes(
// 	componentName string,
// 	nodes []string,
// 	componentType string,
// ) error {

// 	data, _ := o.prepareBaseData()

// 	componentData := map[string]interface{}{
// 		"name":  componentName,
// 		"nodes": nodes,
// 	}

// 	attrs := data["order"].(map[string]interface{})["attrs"].(map[string]interface{})

// 	for key, val := range componentData {
// 		attrs[key] = val
// 	}

// 	if componentType == "infra" {
// 		attrs["role_name"] = componentType
// 	}

// 	return o.sendK8sClusterRequest(fmt.Sprintf("%s_delete_nodes", componentType), data)
// }

// region ingress infra
func (o *K8sClusterOrder) K8sClusterAddNodes(
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

		if component == "infra" {
			attrs["role_name"] = component
		}

		err := o.sendK8sClusterRequest(fmt.Sprintf("%s_add_nodes", component), data)
		if err != nil {
			return fmt.Errorf("failed to add nodes: %v", err)
		}

		remainingNodes -= currentBatch
	}

	return nil
}

// all components cluster actions
func (o *K8sClusterOrder) K8sClusterManageComponent(actionType, component string) error {

	data, _ := o.prepareBaseData()

	var action string
	if component != "visibility" {
		action = prepareActionName(actionType, component)
	} else {
		action = "visibility"
	}

	return o.sendK8sClusterRequest(action, data)
}

func (o *K8sClusterOrder) K8sClusterAddIstio(
	istio entities.K8sClusterConfigIstio,
) error {

	data, _ := o.prepareBaseData()
	cpFlavor := istio.ControlPlanes[0].Flavor
	cpOptions := istio.ControlPlanes[0].Options

	componentData := map[string]interface{}{
		"istio_flavor":          istio.OptionsFlavor,
		"istio_options":         istio.Options,
		"control_plane_flavor":  cpFlavor,
		"control_plane_options": cpOptions,
	}

	attrs := data["order"].(map[string]interface{})["attrs"].(map[string]interface{})

	for key, val := range componentData {
		attrs[key] = val
	}

	return o.sendK8sClusterRequest("add_istio", data)
}

func (o *K8sClusterOrder) K8sClusterProducts(products []string) error {

	data, _ := o.prepareBaseData()

	attrs := data["order"].(map[string]interface{})["attrs"].(map[string]interface{})
	attrs["products"] = products

	return o.sendK8sClusterRequest("products", data)
}

func (o *K8sClusterOrder) sendK8sClusterRequest(
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

func (o *K8sClusterOrder) sendK8sClusterUpdateRequest(
	targetVersion string,
	data map[string]interface{},
) error {

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	versions := strings.Split(targetVersion, ".")
	if len(versions) < 2 {
		return err
	}
	version := versions[1]

	actionName := fmt.Sprintf(
		"%s_%s_cluster_%s",
		KUBERNETES_V1_PREFIX, version, "update",
	)

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

func (o *K8sClusterOrder) formatActionName(actionName string) (string, error) {
	clusterItem, err := o.GetParentItem()
	if err != nil {
		return "", err
	}
	item := clusterItem.Data.Config.(entities.K8sClusterConfig)

	versions := strings.Split(item.ProductVersion, ".")
	if len(versions) < 2 {
		return "", err
	}

	return fmt.Sprintf(
		"%s_%s_cluster_%s",
		KUBERNETES_V1_PREFIX, versions[1], actionName,
	), nil
}

func (o *K8sClusterOrder) prepareBaseData() (map[string]interface{}, error) {

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

func prepareActionName(formattedName, component string) string {

	return fmt.Sprintf("%s_%s", formattedName, component)
}
