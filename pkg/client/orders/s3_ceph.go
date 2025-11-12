package orders

import (
	"encoding/json"
	//"errors"
	"fmt"
	"io"
	"strings"

	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/requests"
)

type S3CephAttrs struct {
	Platform            string `json:"platform"`
	AvailabilityZone    string `json:"availability_zone"`
	NetSegment          string `json:"net_segment,omitempty"`
	DataCenter          string `json:"data_center,omitempty"`
	StorageType         string `json:"storage_type"`
	CreatedWithOpenTofu bool   `json:"created_with_opentofu"`
}

type S3CephOrder struct {
	Order
	Attrs S3CephAttrs `json:"attrs"`
}

func NewS3CephOrder(
	creds *auth.Credentials,
	projectName,
	productID string,
	attrs S3CephAttrs,
) *S3CephOrder {
	return &S3CephOrder{
		Order: Order{
			ProductID:   productID,
			ProjectName: projectName,
			Creds:       creds,
		},
		Attrs: attrs,
	}
}

func (o *S3CephOrder) GetAttrs() interface{} {
	return o.Attrs
}

func (o *S3CephOrder) Create(p CreateOrderPayload) error {

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

func (o *S3CephOrder) Delete() error {

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{}

	data := map[string]interface{}{
		"item_id": item.ID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("s3_ceph_tenant_delete")

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

func (o *S3CephOrder) GetBucketItems() ([]entities.Item, error) {

	created, err := o.itemCreated()
	if err != nil {
		return nil, err
	}

	var items []entities.Item
	if created {
		for _, item := range o.Data {
			if item.Type == "s3_bucket" && item.Data.State == "on" {
				items = append(items, item)
			}
		}
	}
	return items, nil
}

func (o *S3CephOrder) GetBucket(Name string) (*entities.Item, error) {

	bucketItems, err := o.GetBucketItems()
	if err != nil {
		return nil, err
	}

	for _, item := range bucketItems {
		if item.Data.Config.(entities.S3CephBucketItemConfig).Name == Name {
			return &item, nil
		}
	}
	return nil, fmt.Errorf("there is no bucket with name '%s'", Name)
}

func (o *S3CephOrder) GetTenantUsers() ([]entities.S3CephTenantUsers, error) {

	item, err := o.GetParentItem()
	if err != nil {
		return nil, err
	}

	config := item.Data.Config.(entities.S3CephTenantItemConfig)
	return config.Users, nil
}

func (o *S3CephOrder) CreateBucket(name string, maxSizeGb int64, versioning bool) error {

	var uri string
	var data map[string]interface{}
	if err := o.requiredState("on"); err != nil {
		return err
	}

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"name":        name,
		"max_size_gb": maxSizeGb,
		"versioning":  versioning,
	}

	data = map[string]interface{}{
		"item_id": item.ID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	uri = o.generateOrderdActionUri("s3_ceph_bucket_add")

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

func (o *S3CephOrder) CreateUser(
	user *entities.S3CephTenantUsers,
	secretKey string,
) error {

	if err := o.requiredState("on"); err != nil {
		return err
	}

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"user_name":  user.UserName,
		"secret_key": secretKey,
		"access_key": user.AccessKey,
	}

	data := map[string]interface{}{
		"item_id": item.ID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("s3_ceph_user_add")

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

func (o *S3CephOrder) UpdateUser(
	user *entities.S3CephTenantUsers,
	secretKey string,
) error {

	if err := o.requiredState("on"); err != nil {
		return err
	}

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"user_name":  user.UserName,
		"secret_key": secretKey,
		"access_key": user.AccessKey,
	}

	data := map[string]interface{}{
		"item_id": item.ID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("s3_ceph_regenerate_keys")

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

func (o *S3CephOrder) DeleteUser(name string) error {

	if err := o.requiredState("on"); err != nil {
		return err
	}

	item, err := o.GetParentItem()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"user_name": name,
	}

	data := map[string]interface{}{
		"item_id": item.ID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("s3_ceph_user_delete")

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

func (o *S3CephOrder) DeleteBucket(name string) error {

	if err := o.requiredState("on"); err != nil {
		return err
	}

	bucket, err := o.GetBucket(name)
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"name": name,
	}

	data := map[string]interface{}{
		"item_id": bucket.ID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	uri := o.generateOrderdActionUri("s3_ceph_bucket_delete")

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

func (o *S3CephOrder) ChangeBucket(name string, maxSizeGb int64, versioning bool) error {

	var uri string
	var data map[string]interface{}
	if err := o.requiredState("on"); err != nil {
		return err
	}

	bucket, err := o.GetBucket(name)
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"name":        name,
		"max_size_gb": maxSizeGb,
		"versioning":  versioning,
	}

	data = map[string]interface{}{
		"item_id": bucket.ID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	uri = o.generateOrderdActionUri("s3_ceph_bucket_update")

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
