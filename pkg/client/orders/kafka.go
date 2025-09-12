package orders

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"sort"
	"strings"

	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/entities"
	"terraform-provider-vtb/pkg/client/requests"
)

type Kafka struct {
	Order
	Attrs KafkaAttrs `json:"attrs"`
}

type KafkaBasicAttrs struct {
	ADIntegration       bool                     `json:"ad_integration"`
	ADLogonGrants       []entities.ADLogonGrants `json:"ad_logon_grants"`
	Domain              string                   `json:"domain"`
	DefaultNic          entities.DefaultNic      `json:"default_nic"`
	OneNodeFlavor       *entities.Flavor         `json:"one_node_flavor,omitempty"`
	Flavor              *entities.Flavor         `json:"flavor,omitempty"`
	ExtraMounts         []entities.ExtraMount    `json:"extra_mounts"`
	Platform            string                   `json:"platform"`
	OnSupport           bool                     `json:"on_support"`
	OsVersion           string                   `json:"os_version,omitempty"`
	ImageID             string                   `json:"image_id,omitempty"` // Для продукта Linux preprom(УСИ)
	AvailabilityZone    string                   `json:"availability_zone"`
	DataCenter          string                   `json:"data_center,omitempty"`
	CreatedWithOpenTofu bool                     `json:"created_with_opentofu"`
}

type KafkaAttrs struct {
	KafkaBasicAttrs
	GeoDistribution bool      `json:"geo_distribution"`
	LogRetention    int64     `json:"kafka_log_retention_minutes"`
	KafkaVersion    string    `json:"kafka_version"`
	Layout          string    `json:"layout"`
	LayoutName      string    `json:"layout_name"`
	ClusterName     string    `json:"cluster_name"`
	LayoutObj       LayoutObj `json:"layout_obj"`
	ClusterType     string    `json:"cluster_type"`
	Performance     *string   `json:"performance,omitempty"`
}

type LayoutObj struct {
	Name string `json:"name"`
	UUID string `json:"uuid"`
}

type UpgradeKafkaVersionAttrs struct {
	CurrentKafkaVersion string `json:"kafka_version"`
	NewKafkaVersion     string `json:"kafka_version_upgrade"`
	Accept              bool   `json:"accept"`
}

type KafkaQuotasBulkAction struct {
	QuotaType        string   `json:"quota_type"`
	ProducerByteRate int64    `json:"producer_byte_rate,omitempty"`
	ClientCNs        []string `json:"client_cns,omitempty"`
}

type TopicChangeParams struct {
	Operation        string
	CleanUpPolicy    string
	PartitionsNumber int64
	RetentionMs      int64
	SegmentBytes     int64
	CleanUpLimitBy   string
	RetentionBytes   int64
	CompressionType  string
}

type TopicChange struct {
	TopicNames []string
	Parameters TopicChangeParams
}

type HorizontalScalingAttrs struct {
	NewBrokersCount int64 `json:"new_brokers_count"`
	Accept          bool  `json:"accept"`
}

func NewKafka(creds *auth.Credentials, projectName, productID string, attrs KafkaAttrs) *Kafka {
	return &Kafka{
		Order: Order{
			Creds:       creds,
			ProjectName: projectName,
			ProductID:   productID,
		},
		Attrs: attrs,
	}
}

func (o *Kafka) GetBasicAttrs() *KafkaBasicAttrs {
	return &o.Attrs.KafkaBasicAttrs
}

func (o *Kafka) GetAttrs() interface{} {
	return o.Attrs
}

func (o *Kafka) Create(p CreateOrderPayload) error {

	payload, err := o.createPayload(p, o)
	if err != nil {
		return err
	}

	// prepare request
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

func (o *Kafka) GetState() (string, error) {
	_, err := o.itemCreated()
	if err != nil {
		return "", err
	}
	err = o.Sync()
	if err != nil {
		return "", err
	}
	item, err := o.GetClusterItem()
	if err != nil {
		return "", err
	}
	return item.Data.State, nil
}

func (o *Kafka) GetTopics() ([]entities.KafkaTopic, error) {
	item, err := o.GetClusterItem()
	if err != nil {
		return nil, err
	}
	return item.Data.Config.(entities.KafkaItemConfig).Topics, nil
}

func (o *Kafka) GetACLs() ([]entities.KafkaACL, error) {
	item, err := o.GetClusterItem()
	if err != nil {
		return nil, err
	}
	return item.Data.Config.(entities.KafkaItemConfig).ACLs, nil
}

func (o *Kafka) GetTransactionalACLs() ([]entities.KafkaTransactionalACL, error) {
	item, err := o.GetClusterItem()
	if err != nil {
		return nil, err
	}
	return item.Data.Config.(entities.KafkaItemConfig).TransactionalACLs, nil
}

func (o *Kafka) GetIdempotentACLs() ([]entities.KafkaIdempotentACL, error) {
	item, err := o.GetClusterItem()
	if err != nil {
		return nil, err
	}
	return item.Data.Config.(entities.KafkaItemConfig).IdempotentACLs, nil
}

func (o *Kafka) GetGroupACLs() ([]entities.KafkaGroupACL, error) {
	item, err := o.GetClusterItem()
	if err != nil {
		return nil, err
	}
	return item.Data.Config.(entities.KafkaItemConfig).GroupACLs, nil
}

func (o *Kafka) GetQuotas() ([]entities.KafkaQuota, error) {
	item, err := o.GetClusterItem()
	if err != nil {
		return nil, err
	}
	return item.Data.Config.(entities.KafkaItemConfig).Quotas, nil
}

func (o *Kafka) GetClusterItem() (*entities.Item, error) {
	created, err := o.itemCreated()
	if err != nil {
		return nil, err
	}
	if created {
		for _, item := range o.Data {
			if item.Type == "cluster" {
				return &item, nil
			}
		}
	}
	return nil, errors.New("can't find item with type 'cluster'")
}

func (o *Kafka) GetKafkaVMItems() ([]entities.Item, error) {
	items, err := o.GetVMItems()
	if err != nil {
		return nil, err
	}
	var kafkaItems []entities.Item
	for _, item := range items {
		for _, role := range item.Data.Config.(entities.VMItemConfig).NodeRoles {
			if role == "kafka" {
				kafkaItems = append(kafkaItems, item)
				break
			}
		}
	}
	if len(kafkaItems) == 0 {
		return nil, errors.New("can't find kafka items")
	}
	return kafkaItems, nil
}

func (o *Kafka) GetZookeeperVMItems() ([]entities.Item, error) {
	items, err := o.GetVMItems()
	if err != nil {
		return nil, err
	}
	var zookeeperItems []entities.Item
	for _, item := range items {
		for _, role := range item.Data.Config.(entities.VMItemConfig).NodeRoles {
			if role == "zookeeper" {
				zookeeperItems = append(zookeeperItems, item)
				break
			}
		}
	}
	if len(zookeeperItems) == 0 {
		return nil, errors.New("can't find zookeeper items")
	}
	return zookeeperItems, nil
}

func (o *Kafka) CreateTopics(topics []entities.KafkaTopic, async bool) error {

	// delete duplicates
	var unexistsTopics []entities.KafkaTopic
	for _, t := range topics {
		exists := false
		for i := 0; i < len(unexistsTopics); i++ {
			if t.TopicName == unexistsTopics[i].TopicName {
				exists = true
				break
			}
		}
		if !exists {
			unexistsTopics = append(unexistsTopics, t)
		}
	}
	// delete topics
	// get necessary information for topics
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}
	// prepare body
	var toCreate []interface{}
	for _, t := range unexistsTopics {

		targetTopic := map[string]interface{}{
			"topic_name":        t.TopicName,
			"partitions_number": t.PartitionsNumber,
			"compression_type":  t.CompressionType,
			// "segment_bytes":     t.SegmentSize,
		}

		if t.CleanupPolicy == "delete,compact" {
			targetTopic["cleanup_policy"] = "compact,delete"
		} else {
			targetTopic["cleanup_policy"] = t.CleanupPolicy
		}

		// !!!
		if t.CleanupPolicy == "compact" {
			targetTopic["retention_ms"] = 1800000
			targetTopic["_cleanup^limit_by"] = "time"
		}

		if t.CleanupPolicy == "delete" || t.CleanupPolicy == "delete,compact" {

			if t.RetentionBytes == 0 {
				targetTopic["_cleanup^limit_by"] = "time"
				targetTopic["retention_ms"] = t.RetentionMs
			} else if t.RetentionMs == 0 {
				targetTopic["_cleanup^limit_by"] = "size"
				targetTopic["retention_bytes"] = t.RetentionBytes
			} else {
				targetTopic["_cleanup^limit_by"] = "time,size"
				targetTopic["retention_ms"] = t.RetentionMs
				targetTopic["retention_bytes"] = t.RetentionBytes
			}
		}
		toCreate = append(toCreate, targetTopic)
	}

	attrs := map[string]interface{}{
		"topics": toCreate,
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
	uri := o.generateOrderdActionUri("kafka_create_topics")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}
	if !async {
		err = o.WaitSuccess(30)
		if err != nil {
			return err
		}
	}
	return err
}

func (o *Kafka) DeleteTopics(topics []entities.KafkaTopic, async bool) error {
	// default checking
	if len(topics) == 0 {
		return errors.New("for create topics length of slice can't be equal 0")
	}
	if err := o.requiredState("on"); err != nil {
		return err
	}
	err := o.Sync()
	if err != nil {
		return err
	}
	// delete duplicates
	var unduplicatedTopics []entities.KafkaTopic
	for _, t := range topics {
		duplicated := false
		for i := 0; i < len(unduplicatedTopics); i++ {
			if t.TopicName == unduplicatedTopics[i].TopicName {
				duplicated = true
				break
			}
		}
		if !duplicated {
			unduplicatedTopics = append(unduplicatedTopics, t)
		}
	}
	// compare exists and not exists
	currentTopics, err := o.GetTopics()
	if err != nil {
		return err
	}
	var existsTopics []entities.KafkaTopic
	for _, udt := range unduplicatedTopics {
		for _, t := range currentTopics {
			if udt.TopicName == t.TopicName {
				existsTopics = append(existsTopics, udt)
				break
			}
		}
	}
	// delete topics
	// get necessary information for topics
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}
	// prepare body
	var topicNames []string
	for _, t := range unduplicatedTopics {
		topicNames = append(topicNames, t.TopicName)
	}

	attrs := map[string]interface{}{
		"topics": topicNames,
	}

	data := map[string]interface{}{
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
		"item_id": itemID,
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// prepare request
	uri := o.generateOrderdActionUri("kafka_delete_topics")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}
	if !async {
		err = o.WaitSuccess(30)
		if err != nil {
			return err
		}
	}
	return err
}

func (o *Kafka) EditTopics(topics []entities.KafkaTopic, async bool) error {

	changes, err := o.prepareTopicChanges(topics)
	if err != nil {
		return err
	}

	data, err := o.editTopicBulkPreparePayload(changes)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// prepare request
	uri := o.generateOrderdActionUri("kafka_edit_topics_release")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}
	if !async {
		err = o.WaitSuccess(30)
		if err != nil {
			return err
		}
	}
	return err
}

func (o *Kafka) CreateACLs(acls []entities.KafkaACL, async bool) error {
	// default checking
	if len(acls) == 0 {
		return errors.New("for create topics length of slice can't be equal 0")
	}
	if err := o.requiredState("on"); err != nil {
		return err
	}
	err := o.Sync()
	if err != nil {
		return err
	}
	var validationErrors []string
	var allTopics []entities.KafkaACL
	var byName []entities.KafkaACL
	var byMask []entities.KafkaACL
	for _, a := range acls {
		err := a.Validate()
		if err != nil {
			validationErrors = append(validationErrors, err.Error())
		} else {
			switch a.Type {
			case "all_topics":
				allTopics = append(allTopics, a)
			case "by_name":
				byName = append(byName, a)
			case "by_mask":
				byMask = append(byMask, a)
			}
		}
	}
	if len(validationErrors) != 0 {
		return errors.New(strings.Join(validationErrors, "\n"))
	}
	// filter input acls
	var filteredByName []entities.KafkaACL
	var filteredByMask []entities.KafkaACL
	var filteredAllTopics []entities.KafkaACL
	// remove duplicates of acls with type of by_name
	for _, a := range byName {
		duplicated := false
		for i := 0; i < len(filteredByName); i++ {
			if a.ClientCN == filteredByName[i].ClientCN && a.ClientRole == filteredByName[i].ClientRole {
				sort.Strings(filteredByName[i].TopicNames)
				for _, tn := range a.TopicNames {
					tnDuplicate := false
					index := sort.SearchStrings(filteredByName[i].TopicNames, tn)
					if index >= len(a.TopicNames) && tn == filteredByName[i].TopicNames[index] {
						tnDuplicate = true
					}
					if !tnDuplicate {
						filteredByName[i].TopicNames = append(filteredByName[i].TopicNames, tn)
					}
				}
				duplicated = true
				break
			}
		}
		if !duplicated {
			filteredByName = append(filteredByName, a)
		}
	}
	// remove duplicates of acls with type of by_mask
	for _, a := range byMask {
		duplicated := false
		for i := 0; i < len(filteredByMask); i++ {
			if a.Equal(filteredByMask[i]) {
				duplicated = true
				break
			}
		}
		if !duplicated {
			filteredByMask = append(filteredByMask, a)
		}
	}
	// remove duplicates of acls with type of all_topics
	for _, a := range allTopics {
		duplicated := false
		for i := 0; i < len(filteredAllTopics); i++ {
			if a.Equal(filteredAllTopics[i]) {
				duplicated = true
				break
			}
		}
		if !duplicated {
			filteredAllTopics = append(filteredAllTopics, a)
		}
	}
	// compare exists and not exists
	currentACLs, err := o.GetACLs()
	if err != nil {
		return err
	}
	var currentACLsByName []entities.KafkaACL
	var currentACLsByMask []entities.KafkaACL
	var currentACLsAllTopics []entities.KafkaACL
	for _, a := range currentACLs {
		switch a.Type {
		case "all_topics":
			currentACLsAllTopics = append(currentACLsAllTopics, a)
		case "by_name":
			currentACLsByName = append(currentACLsByName, a)
		case "by_mask":
			currentACLsByMask = append(currentACLsByMask, a)
		}
	}

	var unexistsACLsByName []entities.KafkaACL
	var unexistsACLsByMask []entities.KafkaACL
	var unexistsACLsAllTopics []entities.KafkaACL
	for _, a := range filteredByName {
		exists := false
		for _, ca := range currentACLsByName {
			if a.Equal(ca) {
				exists = true
				break
			}
		}
		if !exists {
			unexistsACLsByName = append(unexistsACLsByName, a)
		}
	}
	for _, a := range filteredByMask {
		exists := false
		for _, ca := range currentACLsByMask {
			if a.Equal(ca) {
				exists = true
				break
			}
		}
		if !exists {
			unexistsACLsByMask = append(unexistsACLsByMask, a)
		}
	}
	for _, a := range filteredAllTopics {
		exists := false
		for _, ca := range currentACLsAllTopics {
			if a.Equal(ca) {
				exists = true
				break
			}
		}
		if !exists {
			unexistsACLsAllTopics = append(unexistsACLsAllTopics, a)
		}
	}

	unexistsACLs := unexistsACLsByName
	unexistsACLs = append(unexistsACLs, unexistsACLsByMask...)
	unexistsACLs = append(unexistsACLs, unexistsACLsAllTopics...)
	var toCreate []interface{}
	for _, a := range unexistsACLs {
		targetACL := map[string]interface{}{
			"client_cn":   a.ClientCN,
			"topic_type":  a.Type,
			"client_role": a.ClientRole,
		}
		switch a.Type {
		case "all_topics":
			targetACL["topic_name"] = a.Name
		case "by_name":
			targetACL["topic_names"] = a.TopicNames

		case "by_mask":
			targetACL["topic_name"] = a.Name
		}
		toCreate = append(toCreate, targetACL)
	}
	// delete topics
	// get necessary information for topics
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}
	// prepare body
	attrs := map[string]interface{}{
		"acls": toCreate,
	}

	data := map[string]interface{}{
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
		"item_id": itemID,
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// prepare request
	uri := o.generateOrderdActionUri("kafka_create_acls")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}
	if !async {
		err = o.WaitSuccess(30)
		if err != nil {
			return err
		}
	}
	return err
}

func (o *Kafka) DeleteACLs(acls []entities.KafkaACL, async bool) error {
	// default checking
	if len(acls) == 0 {
		return errors.New("for create topics length of slice can't be equal 0")
	}
	if err := o.requiredState("on"); err != nil {
		return err
	}
	err := o.Sync()
	if err != nil {
		return err
	}
	var validationErrors []string
	var allTopics []entities.KafkaACL
	var byName []entities.KafkaACL
	var byMask []entities.KafkaACL
	for _, a := range acls {
		err := a.Validate()
		if err != nil {
			validationErrors = append(validationErrors, err.Error())
		} else {
			switch a.Type {
			case "all_topics":
				allTopics = append(allTopics, a)
			case "by_name":
				byName = append(byName, a)
			case "by_mask":
				byMask = append(byMask, a)
			}
		}
	}
	if len(validationErrors) != 0 {
		return errors.New(strings.Join(validationErrors, "\n"))
	}
	// filter input acls
	var filteredByName []entities.KafkaACL
	var filteredByMask []entities.KafkaACL
	var filteredAllTopics []entities.KafkaACL
	// remove duplicates of acls with type of by_name
	for _, a := range byName {
		duplicated := false
		for i := 0; i < len(filteredByName); i++ {
			if a.ClientCN == filteredByName[i].ClientCN && a.ClientRole == filteredByName[i].ClientRole {
				sort.Strings(filteredByName[i].TopicNames)
				for _, tn := range a.TopicNames {
					tnDuplicate := false
					index := sort.SearchStrings(filteredByName[i].TopicNames, tn)
					if index >= len(a.TopicNames) && tn == filteredByName[i].TopicNames[index] {
						tnDuplicate = true
					}
					if !tnDuplicate {
						filteredByName[i].TopicNames = append(filteredByName[i].TopicNames, tn)
					}
				}
				duplicated = true
				break
			}
		}
		if !duplicated {
			filteredByName = append(filteredByName, a)
		}
	}
	// remove duplicates of acls with type of by_mask
	for _, a := range byMask {
		duplicated := false
		for i := 0; i < len(filteredByMask); i++ {
			if a.Equal(filteredByMask[i]) {
				duplicated = true
				break
			}
		}
		if !duplicated {
			filteredByMask = append(filteredByMask, a)
		}
	}
	// remove duplicates of acls with type of all_topics
	for _, a := range allTopics {
		duplicated := false
		for i := 0; i < len(filteredAllTopics); i++ {
			if a.Equal(filteredAllTopics[i]) {
				duplicated = true
				break
			}
		}
		if !duplicated {
			filteredAllTopics = append(filteredAllTopics, a)
		}
	}
	// compare exists and not exists
	currentACLs, err := o.GetACLs()
	if err != nil {
		return err
	}
	var currentACLsByName []entities.KafkaACL
	var currentACLsByMask []entities.KafkaACL
	var currentACLsAllTopics []entities.KafkaACL
	for _, a := range currentACLs {
		switch a.Type {
		case "all_topics":
			currentACLsAllTopics = append(currentACLsAllTopics, a)
		case "by_name":
			currentACLsByName = append(currentACLsByName, a)
		case "by_mask":
			currentACLsByMask = append(currentACLsByMask, a)
		}
	}

	var existsACLsByName []entities.KafkaACL
	var existsACLsByMask []entities.KafkaACL
	var existsACLsAllTopics []entities.KafkaACL
	for _, a := range filteredByName {
		for _, ca := range currentACLsByName {
			if a.ClientCN == ca.ClientCN && a.ClientRole == ca.ClientRole {
				existsACLsByName = append(existsACLsByName, a)
				break
			}
		}
	}
	for _, a := range filteredByMask {
		for _, ca := range currentACLsByMask {
			if a.Equal(ca) {
				existsACLsByMask = append(existsACLsByMask, a)
				break
			}
		}
	}
	for _, a := range filteredAllTopics {
		for _, ca := range currentACLsAllTopics {
			if a.Equal(ca) {
				existsACLsAllTopics = append(existsACLsAllTopics, a)
				break
			}
		}
	}

	existsACLs := existsACLsByName
	existsACLs = append(existsACLs, existsACLsByMask...)
	existsACLs = append(existsACLs, existsACLsAllTopics...)
	var toRemove []interface{}
	for _, a := range existsACLs {
		targetACL := map[string]interface{}{
			"client_cn":   a.ClientCN,
			"topic_type":  a.Type,
			"client_role": a.ClientRole,
		}
		switch a.Type {
		case "all_topics":
			targetACL["topic_name"] = a.Name
		case "by_name":
			targetACL["topic_names"] = a.TopicNames

		case "by_mask":
			targetACL["topic_name"] = a.Name
		}
		toRemove = append(toRemove, targetACL)
	}
	// delete topics
	// get necessary information for topics
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}
	// prepare body
	attrs := map[string]interface{}{
		"acls": toRemove,
	}

	data := map[string]interface{}{
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
		"item_id": itemID,
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// prepare request
	uri := o.generateOrderdActionUri("kafka_delete_acls")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}
	if !async {
		err = o.WaitSuccess(30)
		if err != nil {
			return err
		}
	}
	return err
}

func (o *Kafka) CreateTransactionalACLs(acls []entities.KafkaTransactionalACL, async bool) error {
	// default checking
	if len(acls) == 0 {
		return errors.New("for create topics length of slice can't be equal 0")
	}
	if err := o.requiredState("on"); err != nil {
		return err
	}
	err := o.Sync()
	if err != nil {
		return err
	}
	var validationErrors []string
	for _, a := range acls {
		err := a.Validate()
		if err != nil {
			validationErrors = append(validationErrors, err.Error())
		}
	}
	if len(validationErrors) != 0 {
		return errors.New(strings.Join(validationErrors, "\n"))
	}
	// filter input acls
	var filteredACLs []entities.KafkaTransactionalACL
	// remove duplicates of acls for all types
	for _, a := range acls {
		duplicated := false
		for i := 0; i < len(filteredACLs); i++ {
			if a.Equal(filteredACLs[i]) {
				duplicated = true
				break
			}
		}
		if !duplicated {
			filteredACLs = append(filteredACLs, a)
		}
	}
	// compare exists and not exists
	currentACLs, err := o.GetTransactionalACLs()
	if err != nil {
		return err
	}
	var unexistsACLs []entities.KafkaTransactionalACL
	for _, a := range filteredACLs {
		exists := false
		for _, ca := range currentACLs {
			if a.Equal(ca) {
				exists = true
				break
			}
		}
		if !exists {
			unexistsACLs = append(unexistsACLs, a)
		}
	}

	var toCreate []interface{}
	for _, a := range unexistsACLs {
		targetACL := map[string]interface{}{
			"client_cn":           a.ClientCN,
			"transaction_id":      a.Value,
			"transaction_id_type": a.Type,
		}
		toCreate = append(toCreate, targetACL)
	}
	// delete topics
	// get necessary information for topics
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}
	// prepare body
	attrs := map[string]interface{}{
		"acls": toCreate,
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
	uri := o.generateOrderdActionUri("kafka_create_transaction_acls")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}
	if !async {
		err = o.WaitSuccess(30)
		if err != nil {
			return err
		}
	}
	return err
}

func (o *Kafka) DeleteTransactionalACLs(acls []entities.KafkaTransactionalACL, async bool) error {
	// default checking
	if len(acls) == 0 {
		return errors.New("for create topics length of slice can't be equal 0")
	}
	if err := o.requiredState("on"); err != nil {
		return err
	}
	err := o.Sync()
	if err != nil {
		return err
	}
	var validationErrors []string
	for _, a := range acls {
		err := a.Validate()
		if err != nil {
			validationErrors = append(validationErrors, err.Error())
		}
	}
	if len(validationErrors) != 0 {
		return errors.New(strings.Join(validationErrors, "\n"))
	}
	// filter input acls
	var filteredACLs []entities.KafkaTransactionalACL
	// remove duplicates of acls for all types
	for _, a := range acls {
		duplicated := false
		for i := 0; i < len(filteredACLs); i++ {
			if a.Equal(filteredACLs[i]) {
				duplicated = true
				break
			}
		}
		if !duplicated {
			filteredACLs = append(filteredACLs, a)
		}
	}
	// compare exists and not exists
	currentACLs, err := o.GetTransactionalACLs()
	if err != nil {
		return err
	}
	var existsACLs []entities.KafkaTransactionalACL
	for _, a := range filteredACLs {
		for _, ca := range currentACLs {
			if a.Equal(ca) {
				existsACLs = append(existsACLs, a)
				break
			}
		}
	}

	var toRemove []interface{}
	for _, a := range existsACLs {
		targetACL := map[string]interface{}{
			"client_cn":           a.ClientCN,
			"transaction_id":      a.Value,
			"transaction_id_type": a.Type,
		}
		toRemove = append(toRemove, targetACL)
	}
	// delete topics
	// get necessary information for topics
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}
	// prepare body

	attrs := map[string]interface{}{
		"acls": toRemove,
	}

	data := map[string]interface{}{
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
		"item_id": itemID,
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// prepare request
	uri := o.generateOrderdActionUri("kafka_delete_transaction_acls")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}
	if !async {
		err = o.WaitSuccess(30)
		if err != nil {
			return err
		}
	}
	return err
}

func (o *Kafka) CreateIdempotentACLs(acls []entities.KafkaIdempotentACL, async bool) error {
	// default checking
	if len(acls) == 0 {
		return errors.New("for create topics length of slice can't be equal 0")
	}
	if err := o.requiredState("on"); err != nil {
		return err
	}
	err := o.Sync()
	if err != nil {
		return err
	}
	var validationErrors []string
	for _, a := range acls {
		err := a.Validate()
		if err != nil {
			validationErrors = append(validationErrors, err.Error())
		}
	}
	if len(validationErrors) != 0 {
		return errors.New(strings.Join(validationErrors, "\n"))
	}
	// filter input acls
	var filteredACLs []entities.KafkaIdempotentACL
	// remove duplicates of acls for all types
	for _, a := range acls {
		duplicated := false
		for i := 0; i < len(filteredACLs); i++ {
			if a.Equal(filteredACLs[i]) {
				duplicated = true
				break
			}
		}
		if !duplicated {
			filteredACLs = append(filteredACLs, a)
		}
	}
	// compare exists and not exists
	currentACLs, err := o.GetIdempotentACLs()
	if err != nil {
		return err
	}
	var unexistsACLs []entities.KafkaIdempotentACL
	for _, a := range filteredACLs {
		exists := false
		for _, ca := range currentACLs {
			if a.Equal(ca) {
				exists = true
				break
			}
		}
		if !exists {
			unexistsACLs = append(unexistsACLs, a)
		}
	}

	var toCreate []interface{}
	for _, a := range unexistsACLs {
		targetACL := map[string]interface{}{
			"client_cn": a.ClientCN,
		}
		toCreate = append(toCreate, targetACL)
	}
	// delete topics
	// get necessary information for topics
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}
	// prepare body
	attrs := map[string]interface{}{
		"acls": toCreate,
	}

	data := map[string]interface{}{
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
		"item_id": itemID,
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	// prepare request
	uri := o.generateOrderdActionUri("kafka_create_idempotent_acls_release")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}
	if !async {
		err = o.WaitSuccess(30)
		if err != nil {
			return err
		}
	}
	return err
}

func (o *Kafka) DeleteIdempotentACLs(acls []entities.KafkaIdempotentACL, async bool) error {
	// default checking
	if len(acls) == 0 {
		return errors.New("for create topics length of slice can't be equal 0")
	}
	if err := o.requiredState("on"); err != nil {
		return err
	}
	err := o.Sync()
	if err != nil {
		return err
	}
	var validationErrors []string
	for _, a := range acls {
		err := a.Validate()
		if err != nil {
			validationErrors = append(validationErrors, err.Error())
		}
	}
	if len(validationErrors) != 0 {
		return errors.New(strings.Join(validationErrors, "\n"))
	}
	// filter input acls
	var filteredACLs []entities.KafkaIdempotentACL
	// remove duplicates of acls for all types
	for _, a := range acls {
		duplicated := false
		for i := 0; i < len(filteredACLs); i++ {
			if a.Equal(filteredACLs[i]) {
				duplicated = true
				break
			}
		}
		if !duplicated {
			filteredACLs = append(filteredACLs, a)
		}
	}
	// compare exists and not exists
	currentACLs, err := o.GetIdempotentACLs()
	if err != nil {
		return err
	}
	var existsACLs []entities.KafkaIdempotentACL
	for _, a := range filteredACLs {
		for _, ca := range currentACLs {
			if a.Equal(ca) {
				existsACLs = append(existsACLs, a)
				break
			}
		}
	}

	var toRemove []interface{}
	for _, a := range existsACLs {
		targetACL := map[string]interface{}{
			"client_cn": a.ClientCN,
		}
		toRemove = append(toRemove, targetACL)
	}
	// delete topics
	// get necessary information for topics
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}
	// prepare body
	attrs := map[string]interface{}{
		"acls": toRemove,
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
	uri := o.generateOrderdActionUri("kafka_delete_idempotent_acls_release")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}
	if !async {
		err = o.WaitSuccess(30)
		if err != nil {
			return err
		}
	}
	return err
}

func (o *Kafka) GetKafkaExtraMount(path string) (*entities.ExtraMount, error) {
	items, err := o.GetKafkaVMItems()
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("get kafka vm items have zero length")
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

func (o *Kafka) GetZookeeperExtraMount(path string) (*entities.ExtraMount, error) {
	items, err := o.GetZookeeperVMItems()
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("get zookeeper vm items have zero length")
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

func (o *Kafka) ExpandKafkaMountPoint(mountPoint entities.ExtraMount, async bool) (err error) {

	if err = o.requiredState("on"); err != nil {
		return
	}

	itemID, err := o.GetParentItemID()
	if err != nil {
		return
	}

	extraMount, err := o.GetKafkaExtraMount(mountPoint.Path)
	if err != nil {
		return err
	}

	if extraMount == nil {
		return fmt.Errorf("can't find mount point with path = %s", mountPoint.Path)
	}

	if mountPoint == *extraMount {
		return fmt.Errorf("for expand kafka mount point '%s' new size equal current size", mountPoint.Path)
	}

	increaseSize := mountPoint.Size - extraMount.Size
	if increaseSize < 10 {
		return fmt.Errorf(
			"can't change mount point less then 10 GB. Current is %d, new is %d", extraMount.Size, mountPoint.Size,
		)
	}

	attrs := map[string]interface{}{
		"mount": mountPoint.Path,
		"size":  int(increaseSize),
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

	uri := o.generateOrderdActionUri("kafka_expand_mount_point")

	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	if !async {
		err = o.WaitSuccess(30)
		if err != nil {
			return err
		}
	}
	return err
}

func (o *Kafka) ChangeFlavor(flavor entities.Flavor) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"check_agree": true,
		"flavor":      flavor,
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

	uri := o.generateOrderdActionUri("resize_kafka_cluster_vms")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(30)
	if err != nil {
		return err
	}

	return err
}

func (o *Kafka) Stop(async bool) error {

	if err := o.requiredState("on"); err != nil {
		return err
	}
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

	uri := o.generateOrderdActionUri("stop_kafka")
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

func (o *Kafka) Start(async bool) error {

	if err := o.requiredState("off"); err != nil {
		return err
	}
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

	uri := o.generateOrderdActionUri("start_kafka")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	if !async {
		err = o.WaitSuccess(30)
		if err != nil {
			return err
		}
	}
	return err
}

func (o *Kafka) CreateOrUpdateQuotas(quotas []KafkaQuotasBulkAction, async bool) (err error) {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"quotas": quotas,
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

	uri := o.generateOrderdActionUri("kafka_create_quotas")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	if !async {
		err = o.WaitSuccess(30)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *Kafka) DeleteQuotas(quotas []KafkaQuotasBulkAction, async bool) error {

	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"quotas": quotas,
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

	uri := o.generateOrderdActionUri("kafka_delete_quotas")
	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	if !async {
		err = o.WaitSuccess(30)
		if err != nil {
			return err
		}
	}
	return nil
}

func setCleanUpLimitBy(change *TopicChange, planTopic entities.KafkaTopic) error {

	if change.Parameters.RetentionBytes != 0 && change.Parameters.RetentionMs != 0 {
		change.Parameters.CleanUpLimitBy = "[time,size]"
	} else if change.Parameters.RetentionMs != 0 {
		change.Parameters.CleanUpLimitBy = "time"
	} else if change.Parameters.RetentionBytes != 0 {
		change.Parameters.CleanUpLimitBy = "size"
	} else {
		return fmt.Errorf(
			"topic with name '%s' have cleanup.policy equal '%s', "+
				"retention.ms or retention.bytes must be set",
			planTopic.TopicName, planTopic.CleanupPolicy,
		)
	}
	return nil
}

func (o *Kafka) editTopicBulkPreparePayload(changes []TopicChange) (map[string]interface{}, error) {

	var rawChanges []map[string]interface{}
	for _, change := range changes {
		tmp := map[string]interface{}{
			"topic_names": change.TopicNames,
			"operation":   "change_cleanup_policy",
			"parameters": map[string]interface{}{
				"cleanup^policy":    change.Parameters.CleanUpPolicy,
				"_cleanup^limit_by": change.Parameters.CleanUpLimitBy,
				"compression^type":  change.Parameters.CompressionType,
			},
		}

		if change.Parameters.PartitionsNumber != 0 {
			tmp["parameters"].(map[string]interface{})["partitions_number"] = change.Parameters.PartitionsNumber
		}

		if change.Parameters.SegmentBytes != 0 {
			tmp["parameters"].(map[string]interface{})["segment^bytes"] = change.Parameters.SegmentBytes
		}

		if change.Parameters.RetentionBytes != 0 {
			tmp["parameters"].(map[string]interface{})["retention^bytes"] = change.Parameters.RetentionBytes
		}

		if change.Parameters.RetentionMs != 0 {
			tmp["parameters"].(map[string]interface{})["retention^ms"] = change.Parameters.RetentionMs
		}
		rawChanges = append(rawChanges, tmp)
	}

	itemID, err := o.GetParentItemID()
	if err != nil {
		return nil, err
	}

	attrs := map[string]interface{}{
		"changes": rawChanges,
	}

	data := map[string]interface{}{
		"item_id": itemID,
		"order": map[string]interface{}{
			"attrs": o.addCreatedWithOpenTofuTagToAttrs(attrs),
		},
	}
	return data, nil
}

func (o *Kafka) prepareTopicChanges(topics []entities.KafkaTopic) ([]TopicChange, error) {

	existTopics, err := o.GetTopics()
	if err != nil {
		return nil, err
	}

	var changes []TopicChange
	for _, topicToChange := range topics {
		exists := false

		for _, existTopics := range existTopics {

			if topicToChange.TopicName == existTopics.TopicName {
				if topicToChange.Equal(existTopics) {
					return nil, fmt.Errorf("topic with name '%s' hasn't changes", topicToChange.TopicName)
				}

				change := &TopicChange{
					TopicNames: []string{topicToChange.TopicName},
					Parameters: TopicChangeParams{
						SegmentBytes:     topicToChange.SegmentSize,
						PartitionsNumber: topicToChange.PartitionsNumber,
						RetentionMs:      topicToChange.RetentionMs,
						RetentionBytes:   topicToChange.RetentionBytes,
						CompressionType:  topicToChange.CompressionType,
					},
				}

				if topicToChange.CleanupPolicy == "delete,compact" {
					change.Parameters.CleanUpPolicy = "[compact,delete]"
				} else {
					change.Parameters.CleanUpPolicy = topicToChange.CleanupPolicy
				}

				if change.Parameters.CleanUpPolicy == "delete" ||
					change.Parameters.CleanUpPolicy == "[compact,delete]" {
					err := setCleanUpLimitBy(change, topicToChange)
					if err != nil {
						log.Fatal(err)
					}
				}

				changes = append(changes, *change)
				exists = true
				break
			}
		}

		if !exists {
			return nil,
				fmt.Errorf("topic with name '%s' dosn't exists in order", topicToChange.TopicName)
		}
	}
	return changes, nil
}

func (o *Kafka) UpgradeKafkaVersion(attrs UpgradeKafkaVersionAttrs) error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrsData := map[string]interface{}{
		"kafka_version":         attrs.CurrentKafkaVersion,
		"kafka_version_upgrade": attrs.NewKafkaVersion,
		"accept":                attrs.Accept,
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

	uri := o.generateOrderdActionUri("kafka_release_upgrade_new_version")

	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(30)
	if err != nil {
		return err
	}

	return nil
}

func (o *Kafka) HorizontalScaling(attrs HorizontalScalingAttrs) error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrsData := map[string]interface{}{
		"new_brokers_count": attrs.NewBrokersCount,
		"accept":            attrs.Accept,
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

	uri := o.generateOrderdActionUri("kafka_release_add_brokers")

	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(30)
	if err != nil {
		return err
	}

	return nil
}

func (o *Kafka) UpgradeKafkaDistribVersion() error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"accept": true,
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

	uri := o.generateOrderdActionUri("kafka_release_upgrade_version")

	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(30)
	if err != nil {
		return err
	}

	return nil

}

func (o *Kafka) EditClusterName(newName string) error {
	itemID, err := o.GetParentItemID()
	if err != nil {
		return err
	}

	attrs := map[string]interface{}{
		"accept":   true,
		"new_name": newName,
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

	uri := o.generateOrderdActionUri("kafka_edit_cluster_name")

	_, err = requests.SendRequest(o.Creds.AccessToken, uri, "PATCH", payload, nil)
	if err != nil {
		return err
	}

	err = o.WaitSuccess(30)
	if err != nil {
		return err
	}

	return nil
}
