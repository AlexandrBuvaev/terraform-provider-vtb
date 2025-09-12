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

type AirflowCluster struct {
	Order
	Attrs AirflowClusterAttrs `json:"attrs"`
}

type AirflowClusterAttrs struct {
	BasicAttrs
	OnBackup        bool            `json:"on_rubackup"`
	Flavor          entities.Flavor `json:"-"`
	FlavorWorker    entities.Flavor `json:"flavor_worker"`
	FlavorScheduler entities.Flavor `json:"flavor_scheduler"`
	FlavorWebserver entities.Flavor `json:"flavor_webserver"`
	ExtraNics       []any           `json:"extra_nics"`
	Layout          string          `json:"layout"`
	GeoDistribution bool            `json:"geo_distribution"`
	AirflowClusterConfig
}

type AirflowClusterConfig struct {
	AirflowVersion   string                   `json:"airflow_version"`
	Executor         string                   `json:"executor"`
	ClusterName      string                   `json:"cluster_name"`
	DeployGrants     []entities.ADLogonGrants `json:"deploy_grants"`
	WebConsoleGrants []entities.ADLogonGrants `json:"web_console_grants"`
	RabbitMQConfig   AirflowRabbitConfig      `json:"rabbitmq_config"`
	PostgresqlConfig AirflowPostgresConfig    `json:"postgresql_config"`
}

type AirflowRabbitConfig struct {
	BrokerVHost    string `json:"broker_vhost"`
	RabbitMQCertCN string `json:"rabbitmq_cert_CN"`
	BrokerOrderID  string `json:"broker_order_id"`
}

type AirflowPostgresConfig struct {
	DBOrderID    string `json:"db_order_id"`
	DBUser       string `json:"db_user"`
	DBDatabase   string `json:"db_database"`
	DBPassword   string `json:"db_password"`
	Notification bool   `json:"pg_notification_accept"`
}

func NewAirflowCluster(
	creds *auth.Credentials,
	projectName,
	productID string,
	attrs AirflowClusterAttrs,
) *AirflowCluster {
	return &AirflowCluster{
		Attrs: attrs,
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
	}
}

func (o *AirflowCluster) GetOrder() *Order {
	return &o.Order
}

func (o *AirflowCluster) GetAttrs() interface{} {
	return o.Attrs
}

func (o *AirflowCluster) Create(p CreateOrderPayload) error {

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

func (o *AirflowCluster) GetAirflowVMItems() ([]entities.Item, error) {
	items, err := o.GetVMItems()
	if err != nil {
		return nil, err
	}
	var airflowItems []entities.Item
	for _, item := range items {
		vmConfig := item.Data.Config.(entities.VMItemConfig)
		if slices.Contains(vmConfig.NodeRoles, "webserver") || slices.Contains(vmConfig.NodeRoles, "scheduler") || slices.Contains(vmConfig.NodeRoles, "worker") {
			airflowItems = append(airflowItems, item)
		}
	}
	if len(airflowItems) == 0 {
		return nil, errors.New("can't find airflow vm items")
	}
	return airflowItems, nil
}

func (o *AirflowCluster) GetAirflowItemsByType(itemRole string) ([]entities.Item, error) {
	items, err := o.GetVMItems()
	if err != nil {
		return nil, err
	}
	var airflowItems []entities.Item
	for _, item := range items {
		vmConfig := item.Data.Config.(entities.VMItemConfig)
		if slices.Contains(vmConfig.NodeRoles, itemRole) {
			airflowItems = append(airflowItems, item)
		}
	}
	if len(airflowItems) == 0 {
		return nil, errors.New("can't find airflow vm item with scheduler role")
	}
	return airflowItems, nil
}

func (o *AirflowCluster) GetAirflowExtraMount(path string) (*entities.ExtraMount, error) {
	items, err := o.GetAirflowVMItems()
	if err != nil {
		return nil, err
	}
	extraMounts := items[0].Data.Config.(entities.VMItemConfig).ExtraMounts
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

type AirflowVerticalScalingAttrs struct {
	CheckAgree      bool
	FlavorWorker    entities.Flavor
	FlavorScheduler entities.Flavor
	FlavorWebserver entities.Flavor
	Executor        string
}

func (o *AirflowCluster) AirflowVerticalScaling(attrs AirflowVerticalScalingAttrs) error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrsData := map[string]interface{}{
		"check_agree":      attrs.CheckAgree,
		"flavor_worker":    attrs.FlavorWorker,
		"flavor_scheduler": attrs.FlavorScheduler,
		"flavor_webserver": attrs.FlavorWebserver,
		"executor":         attrs.Executor,
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

type AirflowHorizontalScalingAttrs struct {
	FlavorWorker      entities.Flavor
	NewWorkerCount    int
	FlavorScheduler   entities.Flavor
	NewSchedulerCount int
}

func (o *AirflowCluster) AirflowHorizontalScaling(input AirflowHorizontalScalingAttrs) error {
	attrs := map[string]interface{}{
		"worker_checkbox":    false,
		"scheduler_checkbox": false,
	}

	if input.NewWorkerCount != 0 {
		attrs["flavor_worker"] = input.FlavorWorker
		attrs["new_worker_count"] = input.NewWorkerCount
		attrs["worker_checkbox"] = true
	}

	if input.NewSchedulerCount != 0 {
		attrs["flavor_scheduler"] = input.FlavorScheduler
		attrs["new_scheduler_count"] = input.NewSchedulerCount
		attrs["scheduler_checkbox"] = true
	}

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
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
	uri := o.generateOrderdActionUri("airflow_add_node")
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
