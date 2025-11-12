package orders

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/requests"
)

type Attrser interface {
	GetAttrs() interface{}
}

type Order struct {
	Creds           *auth.Credentials        `json:"-"`
	Status          string                   `json:"status"`
	ID              string                   `json:"id"`
	Label           string                   `json:"label"`
	Category        string                   `json:"category"`
	CategoryV2      string                   `json:"category_v2"`
	Deletable       bool                     `json:"deletable"`
	ProjectName     string                   `json:"project_name"`
	ProductID       string                   `json:"product_id"`
	LastAction      LastAction               `json:"last_action"`
	NetSegment      entities.NetSegment      `json:"net_segment"`
	CreatedAt       string                   `json:"created_at"`
	UpdatedAt       string                   `json:"updated_at"`
	Data            []entities.Item          `json:"data"`
	FinancialSource entities.FinancialSource `json:"financial_source,omitempty"`
}

type LastAction struct {
	ID                 string `json:"id"`
	Status             string `json:"status"`
	HasPrintableOutput bool   `json:"has_printable_output"`
}

type CreateOrderPayload struct {
	Label        string
	Lifetime     int
	FinProjectID string
}

func (o *Order) GetOrder() *Order {
	return o
}

func (o *Order) ChangeLabel(label string) (err error) {

	created, err := o.itemCreated()
	if err != nil {
		return
	}
	if !created {
		return errors.New("item didn't created")
	}

	data := map[string]interface{}{
		"order": map[string]string{
			"label": label,
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}

	uri := fmt.Sprintf("order-service/api/v1/projects/%s/orders/%s", o.ProjectName, o.ID)
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}
	return
}

// DeleteTwoLayer вызов базового действия "Удалить рекурсивно"
func (o *Order) DeleteTwoLayer(async bool) error {
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

	uri := o.generateOrderdActionUri("delete_two_layer")
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

func (o *Order) GetLastActionOutputData() (any, error) {

	params := map[string]string{
		"include":  "total_count",
		"page":     "1",
		"per_page": "10",
	}
	uri := fmt.Sprintf(
		"order-service/api/v1/projects/%s/orders/%s/actions/history/%s/output",
		o.ProjectName, o.ID, o.LastAction.ID,
	)
	resp, err := requests.SendRequest(o.Creds.AccessToken, uri, "GET", nil, params)
	if err != nil {
		return "", err
	}

	var response struct {
		List []struct {
			Data   any    `json:"data"`
			Status string `json:"status"`
			Type   string `json:"type"`
		} `json:"list"`
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(bytes, &response)
	if err != nil {
		return "", err
	}

	for _, output := range response.List {
		if output.Type == "text" && output.Data != "" {
			return output.Data, nil
		}
	}
	return "", fmt.Errorf("can't find non-empty output message; output=%v", response.List)
}

func (o *Order) GetLastActionStatus() (status string, err error) {
	err = o.Sync()
	if err != nil {
		return status, err
	}
	if o.LastAction.Status == "" {
		err = errors.New("can't get last action status")
	} else {
		status = o.LastAction.Status
	}
	return
}

func (o *Order) GetOrderStatus() (string, error) {
	err := o.Sync()
	if err != nil {
		return "", err
	}
	return o.Status, nil
}

func (o *Order) GetParentItem() (*entities.Item, error) {

	_, err := o.itemCreated()
	if err != nil {
		return nil, err
	}

	var item *entities.Item
	for _, it := range o.Data {
		if it.Data.Parent == "" {
			item = &it
			break
		}
	}

	if item == nil {
		return nil, errors.New("can't find item without parent")
	}
	return item, nil
}

func (o *Order) GetParentItemID() (string, error) {
	item, err := o.GetParentItem()
	if err != nil {
		return "", err
	}
	return item.ID, nil
}

func (o *Order) GetState() (string, error) {
	item, err := o.GetParentItem()
	if err != nil {
		return "", err
	}
	return item.Data.State, nil
}

func (o *Order) GetVMItems() ([]entities.Item, error) {

	created, err := o.itemCreated()
	if err != nil {
		return nil, err
	}

	var items []entities.Item
	if created {
		for _, item := range o.Data {
			if item.Type == "vm" {
				items = append(items, item)
			}
		}
	}

	if len(items) == 0 {
		return nil, errors.New("can't find items with type 'vm'")
	}
	return items, nil
}

func (o *Order) Sync() error {

	uri := fmt.Sprintf(
		"order-service/api/v1/projects/%s/orders/%s?include=last_action",
		o.ProjectName, o.ID,
	)

	resp, err := requests.SendRequest(o.Creds.AccessToken, uri, "GET", nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, o)
	if err != nil {
		return err
	}
	return nil
}

func (o *Order) WaitSuccess(timeout int64) error {

	for {
		status, err := o.GetOrderStatus()
		if err != nil {
			return err
		}

		if isPending(status) {
			log.Printf("\nOrder status: still pending...")
			time.Sleep(time.Duration(timeout) * time.Second)
		} else {
			break
		}
	}

	if o.Status != "success" && o.Status != "deprovisioned" {
		return fmt.Errorf("order '%s' failed with status '%s'", o.ID, o.Status)
	}

	for {
		actionStatus, err := o.GetLastActionStatus()
		if err != nil {
			return err
		}

		if isPending(actionStatus) || isNew(actionStatus) {
			log.Printf("\nOrder action status: still pending...")
			time.Sleep(time.Duration(timeout) * time.Second)
		} else {
			break
		}
	}

	if isWarning(o.LastAction.Status) {
		return fmt.Errorf(
			"order last action ended with status '%s'. Check order on portal. order_id: '%s'",
			o.LastAction.Status, o.ID,
		)
	}

	if o.LastAction.Status != "success" {
		output, err := o.GetLastActionOutputData()
		if err != nil {
			return fmt.Errorf(
				"order last action failed with status '%s'; can't get last action output: %s."+
					"\nCheck order status on cloud portal",
				o.LastAction.Status, err.Error(),
			)
		}
		return fmt.Errorf(
			"order last action failed with status '%s'; last action output: %v",
			o.LastAction.Status, output,
		)
	}
	return nil
}

func (o *Order) createPayload(p CreateOrderPayload, attrser Attrser) ([]byte, error) {

	payload := map[string]interface{}{
		"order": map[string]interface{}{
			"label":                p.Label,
			"attrs":                attrser.GetAttrs(),
			"project_name":         o.ProjectName,
			"product_id":           o.ProductID,
			"count":                1,
			"financial_project_id": p.FinProjectID,
		},
	}

	if p.Lifetime != 0 {
		daysToSeconds := func(days int64) string {
			seconds := days * 24 * 60 * 60
			return strconv.FormatInt(seconds, 10)
		}
		order := payload["order"].(map[string]interface{})
		order["lifetime"] = daysToSeconds(int64(p.Lifetime))
	}

	if p.FinProjectID != "" {
		order := payload["order"].(map[string]interface{})
		order["financial_project_id"] = p.FinProjectID
	}

	return json.Marshal(payload)
}

func (o *Order) requiredState(required string) error {
	state, err := o.GetState()
	if err != nil {
		return err
	}
	if state != required {
		return fmt.Errorf("can't do action, because vm if vm.state is not equal %s, current state is %s", required, state)
	}
	return nil
}

func (o *Order) itemCreated() (bool, error) {
	if len(o.Data) == 0 {
		return false, errors.New("Order haven't items")
	}
	for _, it := range o.Data {
		if it.Data.Parent == "" {
			return true, nil
		}
	}
	return false, errors.New("Order havn't item without parent")
}

func (o *Order) ChangeFinancialProject(finProjectID string) (err error) {
	created, err := o.itemCreated()
	if err != nil {
		return
	}
	if !created {
		return errors.New("item didn't created")
	}
	orderCreatedDate, err := time.Parse(time.RFC3339, o.CreatedAt)
	if err != nil {
		return err
	}

	data := map[string]interface{}{
		"order_fin_projects": []map[string]string{
			{
				"financial_project_id": finProjectID,
				"start_date":           orderCreatedDate.Format("2006-01-02"),
			},
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := fmt.Sprintf("order-service/api/v1/projects/%s/orders/%s/order_fin_projects", o.ProjectName, o.ID)
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}
	return nil
}

func isPending(status string) bool {
	return status == "pending" || status == "changing" || status == "removing"
}

func isNew(status string) bool {
	return status == "new"
}

func isWarning(status string) bool {
	return status == "warning"
}

// URI для первичного создания заказа
func generateCreateOrderUri(projectName string) string {
	return fmt.Sprintf("order-service/api/v1/projects/%s/orders", projectName)
}

func (o *Order) generateOrderdActionUri(actionName string) string {
	return fmt.Sprintf(
		"order-service/api/v1/projects/%s/orders/%s/actions/%s",
		o.ProjectName, o.ID, actionName,
	)
}

func (o *Order) addCreatedWithOpenTofuTagToAttrs(attrs map[string]interface{}) map[string]interface{} {

	if attrs == nil {
		attrs = make(map[string]interface{})
	}

	attrs["created_with_opentofu"] = true
	return attrs
}

func (o *Order) GetVMAcls() ([]entities.AccessACL, error) {
	vmItems, err := o.GetVMItems()
	if err != nil {
		return nil, err
	}

	uniqueAcls := make(map[string]struct{})
	var vmAcls []entities.AccessACL

	for _, vm := range vmItems {
		for _, acl := range vm.Data.ACLs {
			members := make([]string, len(acl.Members))
			copy(members, acl.Members)
			sort.Strings(members)

			key := acl.Role + ":" + strings.Join(members, ",")

			if _, exists := uniqueAcls[key]; !exists {
				uniqueAcls[key] = struct{}{}
				vmAcls = append(vmAcls, acl)
			}
		}
	}

	return vmAcls, nil
}

func (o *Order) GetActiveProjectItems() ([]entities.Item, error) {

	created, err := o.itemCreated()
	if err != nil {
		return nil, err
	}

	var items []entities.Item
	if created {
		for _, item := range o.Data {
			if item.Type == "project" && item.Data.State == "on" {
				items = append(items, item)
			}
		}
	}

	if len(items) == 0 {
		return nil, errors.New("can't find items with type 'proect'")
	}
	return items, nil
}

func (o *Order) WaitLastActionEnded(timeout int64) error {

	for {
		actionStatus, err := o.GetLastActionStatus()
		if err != nil {
			return err
		}

		if isPending(actionStatus) || isNew(actionStatus) {
			log.Printf("\nOrder action status: still pending...")
			time.Sleep(time.Duration(timeout) * time.Second)
		} else {
			break
		}
	}
	return nil
}
