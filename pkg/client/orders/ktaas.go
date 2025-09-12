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

type KTaaS struct {
	Order
	Attrs KTaaSAttrs `json:"attrs"`
}

type KTaaSAttrs struct {
	Name                string                     `json:"name"`
	Accept              bool                       `json:"accept"`
	Platform            string                     `json:"platform"`
	DefaultNic          entities.DefaultNic        `json:"default_nic"`
	TopicFlavor         int64                      `json:"topic_flavor"`
	BillingFlavor       string                     `json:"billing_flavor"`
	Cluster             entities.KTaaSKafkaCluster `json:"cluster"`
	PartitionsNumber    int64                      `json:"partitions_number"`
	CreatedWithOpenTofu bool                       `json:"created_with_opentofu"`
}

type ChangeSizeTopicAttrs struct {
	TopicFlavor   int64  `json:"topic_flavor"`
	BillingFlavor string `json:"billing_flavor"`
}

type DeleteAclsAttrs struct {
	RawData entities.KTaaSAcls `json:"rawData"`
}

type DeleteGroupAclsAttrs struct {
	RawData entities.KtaaSGroupAcls `json:"rawData"`
}

func NewKTaaS(creds *auth.Credentials, projectName, productID string, attrs KTaaSAttrs) *KTaaS {
	return &KTaaS{
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
		Attrs: attrs,
	}
}

func (o *KTaaS) GetAttrs() interface{} {
	return o.Attrs
}

func (o *KTaaS) Create(p CreateOrderPayload) error {
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

	err = o.WaitSuccess(5)
	if err != nil {
		return err
	}

	err = o.Sync()
	if err != nil {
		return err
	}

	return nil
}

func (o *KTaaS) DeleteKTaaS() error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return nil
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

	uri := o.generateOrderdActionUri("ktaas_delete_topic")
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

func (o *KTaaS) KTaaSCreateAcls(acls []entities.KTaaSAcls) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"acls": acls,
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

	uri := o.generateOrderdActionUri("ktaas_create_acls")
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

func (o *KTaaS) KTaaSDeleteAcls(deleteAcls []DeleteAclsAttrs) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"selected": deleteAcls,
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

	uri := o.generateOrderdActionUri("ktaas_delete_acls")
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

func (o *KTaaS) KTaaSCreateGroupAcls(groupAcls []entities.KtaaSGroupAcls) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"group_acls": groupAcls,
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

	uri := o.generateOrderdActionUri("ktaas_create_group_acls")
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

func (o *KTaaS) KTaaSDeleteGroupAcls(deleteGroupAcls []DeleteGroupAclsAttrs) error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"selected": deleteGroupAcls,
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

	uri := o.generateOrderdActionUri("ktaas_delete_group_acls")
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

func (o *KTaaS) KTaaSChangeSizeTopic(attrs ChangeSizeTopicAttrs) error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrsData := map[string]interface{}{
		"topic_flavor":   attrs.TopicFlavor,
		"billing_flavor": attrs.BillingFlavor,
	}

	data := map[string]interface{}{
		"item_id": itemID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrsData),
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return nil
	}

	uri := o.generateOrderdActionUri("ktaas_change_size_topic")
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

func (o *KTaaS) KTaaSChangePartitionsTopic(partitionsNumber int64) error {
	if partitionsNumber%2 != 0 {
		return fmt.Errorf("partitionsNumber must be even, got %d", partitionsNumber)
	}

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"partitions_number": partitionsNumber,
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

	uri := o.generateOrderdActionUri("ktaas_change_partitions_topic")
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
