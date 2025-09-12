package entities

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func (c KafkaItemConfig) GetProviderType() (string, string) {
	return "kafka", "cluster"
}

type KafkaItemConfig struct {
	ClusterName              string                  `json:"cluster_name"`
	KafkaVersion             string                  `json:"kafka_version"`
	CertificateCn            string                  `json:"certificate_cn"`
	ConnectionURL            string                  `json:"connection_url"`
	KafkaClientURL           string                  `json:"kafka_client_url"`
	VtbKafkaVersion          string                  `json:"vtb_kafka_version"`
	ZookeeperVersion         string                  `json:"zookeeper_version"`
	GrafanaDashboardURL      string                  `json:"grafana_dashboard_url"`
	CertificateExpiration    string                  `json:"certificate_expiration"`
	CertificateValidFrom     string                  `json:"certificate_valid_from"`
	KafkaLogRetentionMinutes int                     `json:"kafka_log_retention_minutes"`
	Topics                   []KafkaTopic            `json:"topics"`
	ACLs                     []KafkaACL              `json:"acls"`
	TransactionalACLs        []KafkaTransactionalACL `json:"transaction_acls"`
	IdempotentACLs           []KafkaIdempotentACL    `json:"idempotent_acls"`
	GroupACLs                []KafkaGroupACL         `json:"group_acls"`
	Quotas                   []KafkaQuota            `json:"quotas"`
}

type KafkaTopic struct {
	TopicName        string `json:"topic_name" tfsdk:"topic_name"`
	CleanupPolicy    string `json:"cleanup_policy" tfsdk:"cleanup_policy"`
	RetentionMs      int64  `json:"retention_ms" tfsdk:"retention_ms"`
	RetentionBytes   int64  `json:"retention_bytes" tfsdk:"retention_bytes"`
	SegmentSize      int64  `json:"segment_bytes" tfsdk:"segment_bytes"`
	PartitionsNumber int64  `json:"partitions_number" tfsdk:"partitions"`
	CompressionType  string `json:"compression_type"`
}

type KafkaACL struct {
	ClientCN   string   `json:"client_cn"`
	ClientRole string   `json:"client_role"`
	Type       string   `json:"topic_type"`
	TopicNames []string `json:"topic_names,omitempty"`
	Name       string   `json:"topic_name,omitempty"`
}

type KafkaGroupACL struct {
	Host           string `json:"host"`
	Name           string `json:"name"`
	Operation      string `json:"operation"`
	Principal      string `json:"principal"`
	PatternType    string `json:"patternType"`
	ResourceType   string `json:"resourceType"`
	PermissionType string `json:"permissionType"`
}

type KafkaIdempotentACL struct {
	ClientCN string `json:"client_cn"`
}

type KafkaTransactionalACL struct {
	ClientCN string `json:"client_cn"`
	Value    string `json:"transaction_id"`
	Type     string `json:"transaction_id_type"`
}

type KafkaQuota struct {
	ProducerByteRate int64  `json:"producer_byte_rate"`
	ClientCN         string `json:"client_cn"`
}

func (t *KafkaTopic) UnmarshalJSON(data []byte) (err error) {

	rawData := struct {
		TopicName        string `json:"topic_name" tfsdk:"topic_name"`
		CleanupPolicy    string `json:"cleanup_policy" tfsdk:"cleanup_policy"`
		RetentionMs      string `json:"retention_ms" tfsdk:"retention_ms"`
		SegmentBytes     string `json:"segment_bytes" tfsdk:"segment_bytes"`
		RetentionBytes   string `json:"retention_bytes" tfsdk:"retention_bytes"`
		PartitionsNumber string `json:"partitions_number" tfsdk:"partitions"`
		CompressionType  string `json:"compression_type" tfsdk:"compression_type"`
	}{}

	err = json.Unmarshal(data, &rawData)
	if err != nil {
		return
	}

	t.TopicName = rawData.TopicName
	if rawData.CleanupPolicy == "compact,delete" {
		t.CleanupPolicy = "delete,compact"
	} else {
		t.CleanupPolicy = rawData.CleanupPolicy
	}
	t.CompressionType = rawData.CompressionType

	retentionMs, err := strconv.Atoi(rawData.RetentionMs)
	if err == nil {
		t.RetentionMs = int64(retentionMs)
	}

	retentionBytes, err := strconv.Atoi(rawData.RetentionBytes)
	if err == nil {
		t.RetentionBytes = int64(retentionBytes)
	}

	segmentSize, err := strconv.Atoi(rawData.SegmentBytes)
	if err == nil {
		t.SegmentSize = int64(segmentSize)
	}

	partitionsNumber, err := strconv.Atoi(rawData.PartitionsNumber)
	if err == nil {
		t.PartitionsNumber = int64(partitionsNumber)
	}
	return
}

func (t KafkaTopic) Equal(topic KafkaTopic) bool {

	if t.TopicName == topic.TopicName &&
		t.SegmentSize == topic.SegmentSize &&
		t.CleanupPolicy == topic.CleanupPolicy &&
		t.PartitionsNumber == topic.PartitionsNumber &&
		t.RetentionMs == topic.RetentionMs &&
		t.CompressionType == topic.CompressionType &&
		t.RetentionBytes == topic.RetentionBytes {
		return true
	}
	return false
}

func (a KafkaACL) Equal(acl KafkaACL) bool {
	topicNamesEqual := func() bool {
		if len(a.TopicNames) != len(acl.TopicNames) {
			return false
		}
		for _, topicNameLeft := range a.TopicNames {
			found := false
			for _, topicNameRight := range acl.TopicNames {
				if topicNameLeft == topicNameRight {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}

	if a.Type != acl.Type {
		return false
	}
	switch a.Type {
	case "all_topics":
		if a.ClientCN == acl.ClientCN && a.ClientRole == acl.ClientRole {
			return true
		}
	case "by_name":
		if a.ClientCN == acl.ClientCN && a.ClientRole == acl.ClientRole {
			return topicNamesEqual()
		}
	case "by_mask":
		if a.ClientCN == acl.ClientCN && a.ClientRole == acl.ClientRole && a.Name == acl.Name {
			return true
		}
	default:
		if a.ClientCN == acl.ClientCN && a.ClientRole == acl.ClientRole && a.Name == acl.Name {
			return topicNamesEqual()
		}
	}
	return false
}

func (a KafkaACL) Validate() (err error) {

	var validationErrors []string

	if a.Type != "all_topics" && a.Type != "by_name" && a.Type != "by_mask" {
		validationErrors = append(
			validationErrors,
			fmt.Sprintf(
				"ACL for client_cn equal '%s', and client_role equal '%s' have wrong topic_type equal '%s'. Available types is [%s]",
				a.ClientCN,
				a.ClientRole,
				a.Type, strings.Join([]string{"all_topics", "by_name", "by_mask"}, ", "),
			),
		)
	}

	if a.ClientCN == "" {
		validationErrors = append(
			validationErrors,
			fmt.Sprintf(
				"ACL for client_role equal '%s' with topic_type equal '%s' have empty client_cn",
				a.ClientRole, a.Type,
			),
		)
	}

	if a.ClientRole != "producer" && a.ClientRole != "consumer" {
		validationErrors = append(
			validationErrors,
			fmt.Sprintf(
				"ACL for client_cn equal '%s', and client_role equal '%s' with topic_type equal '%s' "+
					"must have topic_type equal 'producer' or 'consumer'",
				a.ClientCN, a.ClientRole, a.Type,
			),
		)
	}

	switch a.Type {
	case "all_topics":

		if a.Name != "*" {
			validationErrors = append(
				validationErrors,
				fmt.Sprintf(
					"ACL for client_cn equal '%s', and client_role equal '%s' with topic_type equal %s must have topic_name equal '*'",
					a.ClientCN, a.ClientRole, a.Type,
				),
			)
		}

		if len(a.TopicNames) != 0 {
			validationErrors = append(
				validationErrors,
				fmt.Sprintf(
					"ACL for client_cn equal '%s', and client_role equal '%s' with topic_type equal %s must have empty topic_names",
					a.ClientCN, a.ClientRole, a.Type,
				),
			)
		}
	case "by_mask":
		if a.Name == "" {
			validationErrors = append(validationErrors, fmt.Sprintf("ACL for client_cn equal '%s', and client_role equal '%s' with topic_type equal %s can't be with empty topic_name", a.ClientCN, a.ClientRole, a.Type))
		}
		if len(a.TopicNames) != 0 {
			validationErrors = append(validationErrors, fmt.Sprintf("ACL for client_cn equal '%s', and client_role equal '%s' with topic_type equal %s must have empty for topic_names", a.ClientCN, a.ClientRole, a.Type))
		}
	case "by_name":
		if a.Name != "" {
			validationErrors = append(validationErrors, fmt.Sprintf("ACL for client_cn equal '%s', and client_role equal '%s' with topic_type equal %s must be empty for topic_name", a.ClientCN, a.ClientRole, a.Type))
		}
		if len(a.TopicNames) == 0 {
			validationErrors = append(validationErrors, fmt.Sprintf("ACL for client_cn equal '%s', and client_role equal '%s' with topic_type equal %s must have empty topic_names", a.ClientCN, a.ClientRole, a.Type))
		}
	}

	if len(validationErrors) != 0 {
		return errors.New(strings.Join(validationErrors, "\n"))
	}
	return nil
}

func (a KafkaTransactionalACL) Equal(acl KafkaTransactionalACL) bool {
	if a.Type == acl.Type && a.ClientCN == acl.ClientCN && a.Value == acl.Value {
		return true
	}
	return false
}

func (a KafkaTransactionalACL) Validate() (err error) {

	var validationErrors []string

	if a.ClientCN == "" {
		validationErrors = append(
			validationErrors,
			fmt.Sprintf(
				"Transactional ACL with transaction_id equal '%s' with transaction_id_type equal %s can't have empty client_cn",
				a.Value, a.Type,
			),
		)
	}

	if a.Value == "" {
		validationErrors = append(
			validationErrors,
			fmt.Sprintf(
				"Transactional ACL with client_cn equal '%s' with transaction_id_type equal %s can't have empty transaction_id",
				a.ClientCN, a.Type,
			),
		)
	}

	if a.Type != "all_ids" && a.Type != "by_name" && a.Type != "by_mask" {
		validationErrors = append(
			validationErrors,
			fmt.Sprintf(
				"Transactional ACL for client_cn equal '%s', have wrong type equal %s. Available types is [%s]",
				a.ClientCN,
				a.Type,
				strings.Join([]string{"all_topics", "by_name", "by_mask"}, ", "),
			),
		)
	}

	switch a.Type {
	case "all_ids":
		if a.Value != "*" {
			validationErrors = append(
				validationErrors,
				fmt.Sprintf(
					"Transactional ACL for client_cn equal '%s' with type equal %s must have mask equal '*'",
					a.ClientCN, a.Type,
				),
			)
		}
	case "by_mask":
		if a.Value == "" {
			validationErrors = append(
				validationErrors,
				fmt.Sprintf(
					"Transactional ACL for client_cn equal '%s' with type equal %s can't be with empty mask",
					a.ClientCN, a.Type,
				),
			)
		}
	case "by_name":
		if a.Value == "" {
			validationErrors = append(
				validationErrors,
				fmt.Sprintf(
					"Transactional ACL for client_cn equal '%s' with type equal %s must be empty for mask",
					a.ClientCN, a.Type,
				),
			)
		}
	}

	if len(validationErrors) != 0 {
		return errors.New(strings.Join(validationErrors, "\n"))
	}
	return nil
}

func (a KafkaIdempotentACL) Equal(acl KafkaIdempotentACL) bool {
	return a.ClientCN == acl.ClientCN
}

func (a KafkaIdempotentACL) Validate() (err error) {

	var validationErrors []string

	if a.ClientCN == "" {
		validationErrors = append(validationErrors, "Idempotent ACL with client_cn is empty.")
	}

	if len(validationErrors) != 0 {
		return errors.New(strings.Join(validationErrors, "\n"))
	}
	return nil
}
