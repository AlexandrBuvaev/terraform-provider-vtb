package orders

import (
	"encoding/json"
	"io"
	"strings"
	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/requests"
)

type RQaaS struct {
	Order
	Attrs RQaaSAttrs `json:"attrs"`
}

type RQaaSAttrs struct {
	Name                string                `json:"name"`
	Platform            string                `json:"platform"`
	DefaultNic          entities.DefaultNic   `json:"default_nic"`
	AvailabilityZone    string                `json:"availability_zone"`
	Cluster             entities.RQaaSCluster `json:"cluster"`
	CreatedWithOpenTofu bool                  `json:"created_with_opentofu"`
}

type QueueUserAttrs struct {
	APDCode     *string `json:"apd_code,omitempty"`
	CheckAccess *string `json:"check_access,omitempty"`
	Name        *string `json:"name,omitempty"`
	RisCode     *string `json:"ris_code,omitempty"`
	UserName    string  `json:"user_name"`
	Read        bool    `json:"read"`
	Write       bool    `json:"write"`
}

func NewRQaaS(creds *auth.Credentials, projectName, productID string, attrs RQaaSAttrs) *RQaaS {
	return &RQaaS{
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
		Attrs: attrs,
	}
}

func (o *RQaaS) GetAttrs() interface{} {
	return o.Attrs
}

func (o *RQaaS) Create(p CreateOrderPayload) error {
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

func (o *RQaaS) DeleteRQaaS() error {

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

	uri := o.generateOrderdActionUri("rqaas_remove_queue")
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

func (o *RQaaS) CreateUser(attrs QueueUserAttrs) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrsData := map[string]interface{}{
		"check_access": attrs.CheckAccess,
		"name":         attrs.Name,
		"ris_code":     attrs.RisCode,
		"read":         attrs.Read,
		"write":        attrs.Write,
		"user_name":    attrs.UserName,
	}

	if attrs.APDCode != nil {
		attrsData["apd_code"] = attrs.APDCode
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

	uri := o.generateOrderdActionUri("rqaas_user_add")
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

func (o *RQaaS) UpdateUserPermissions(attrs QueueUserAttrs) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrsData := map[string]interface{}{
		"user_name": attrs.UserName,
		"read":      attrs.Read,
		"write":     attrs.Write,
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

	uri := o.generateOrderdActionUri("rqaas_edit_access")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return nil
	}

	err = o.WaitSuccess(5)
	if err != nil {
		return err
	}

	return nil
}
