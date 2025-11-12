package utils

import (
	"fmt"
	"terraform-provider-vtb/internal/client"
	"terraform-provider-vtb/pkg/client/orders"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type OrderCheckIsDeletedResult struct {
	Diagnostics diag.Diagnostics
	IsDeleted   bool
}

func CheckOrderIsDeleted(
	client *client.CloudClient,
	planOrderID types.String,
	inputType string,
	inputProvider string,
) OrderCheckIsDeletedResult {

	result := OrderCheckIsDeletedResult{}
	if planOrderID.IsUnknown() || planOrderID.IsNull() {
		return result
	}

	var state string

	if inputType == "vm" {
		order, err := orders.GetComputeOrder(
			client.Creds,
			client.ProjectName,
			planOrderID.ValueString(),
		)
		if err != nil {
			result.Diagnostics.AddError("Check resource is not deleted", err.Error())
			return result
		}

		item, err := order.GetParentItem()
		if err != nil {
			result.Diagnostics.AddError("Check resource is not deleted", err.Error())
			return result
		}
		state = item.Data.State
	}

	if inputType == "cluster" {
		switch inputProvider {
		case "kafka":
			{
				order, err := orders.GetKafkaOrder(
					client.Creds,
					client.ProjectName,
					planOrderID.ValueString(),
				)
				if err != nil {
					result.Diagnostics.AddError("Check resource is not deleted", err.Error())
					return result
				}
				item, err := order.GetParentItem()
				if err != nil {
					result.Diagnostics.AddError("Check resource is not deleted", err.Error())
					return result
				}
				state = item.Data.State
			}
		case "etcd":
			{
				order, err := orders.GetEtcdOrder(
					client.Creds,
					client.ProjectName,
					planOrderID.ValueString(),
				)
				if err != nil {
					result.Diagnostics.AddError("Check resource is not deleted", err.Error())
					return result
				}
				item, err := order.GetParentItem()
				if err != nil {
					result.Diagnostics.AddError("Check resource is not deleted", err.Error())
					return result
				}
				state = item.Data.State
			}
		case "clickhouse":
			{
				order, err := orders.GetClickhouseClusterOrder(
					client.Creds,
					client.ProjectName,
					planOrderID.ValueString(),
				)
				if err != nil {
					result.Diagnostics.AddError("Check resource is not deleted", err.Error())
					return result
				}
				item, err := order.GetParentItem()
				if err != nil {
					result.Diagnostics.AddError("Check resource is not deleted", err.Error())
					return result
				}
				state = item.Data.State
			}
		case "airflow":
			{
				order, err := orders.GetAirflowClusterOrder(
					client.Creds,
					client.ProjectName,
					planOrderID.ValueString(),
				)
				if err != nil {
					result.Diagnostics.AddError("Check resource is not deleted", err.Error())
					return result
				}
				item, err := order.GetParentItem()
				if err != nil {
					result.Diagnostics.AddError("Check resource is not deleted", err.Error())
					return result
				}
				state = item.Data.State
			}
		case "tarantool_v2":
			{
				order, err := orders.GetTarantoolClusterOrder(
					client.Creds,
					client.ProjectName,
					planOrderID.ValueString(),
				)
				if err != nil {
					result.Diagnostics.AddError("Check resource is not deleted", err.Error())
					return result
				}
				item, err := order.GetParentItem()
				if err != nil {
					result.Diagnostics.AddError("Check resource is not deleted", err.Error())
					return result
				}
				state = item.Data.State
			}
		case "balancer_v3":
			order, err := orders.GetBalancerV3Order(
				client.Creds,
				client.ProjectName,
				planOrderID.ValueString(),
			)
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			item, err := order.GetParentItem()
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			state = item.Data.State
		case "kubernetes":
			order, err := orders.GetK8sClusterOrder(
				client.Creds,
				client.ProjectName,
				planOrderID.ValueString(),
			)
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			item, err := order.GetParentItem()
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			state = item.Data.State
		case "scylladb":
			order, err := orders.GetScyllaDbClusterOrder(
				client.Creds,
				client.ProjectName,
				planOrderID.ValueString(),
			)
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			item, err := order.GetParentItem()
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			state = item.Data.State
		case "elasticsearch_os":
			order, err := orders.GetElasticSearchOrder(
				client.Creds,
				client.ProjectName,
				planOrderID.ValueString(),
			)
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			item, err := order.GetParentItem()
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			state = item.Data.State
		case "rabbitmq":
			order, err := orders.GetRabbitMQOrder(
				client.Creds,
				client.ProjectName,
				planOrderID.ValueString(),
			)
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			item, err := order.GetParentItem()
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			state = item.Data.State
		case "vtb-artemis":
			order, err := orders.GetArtemisOrder(
				client.Creds,
				client.ProjectName,
				planOrderID.ValueString(),
			)
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			item, err := order.GetParentItem()
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			state = item.Data.State
		case "debezium":
			order, err := orders.GetDebeziumOrder(
				client.Creds,
				client.ProjectName,
				planOrderID.ValueString(),
			)
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			item, err := order.GetParentItem()
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			state = item.Data.State
		case "gslb_cluster_v1":
			order, err := orders.GetGSLBV1Order(
				client.Creds,
				client.ProjectName,
				planOrderID.ValueString(),
			)
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			item, err := order.GetParentItem()
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			state = item.Data.State
		}
	}

	if inputType == "project" {
		if inputProvider == "kubernetes" {
			order, err := orders.GetK8sProjectOrder(
				client.Creds,
				client.ProjectName,
				planOrderID.ValueString(),
			)
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			item, err := order.GetParentItem()
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			state = item.Data.State
		}
	}

	if inputType == "app" {

		switch inputProvider {
		case "postgresql_v001":
			order, err := orders.GetPostgresqlOrder(
				client.Creds,
				client.ProjectName,
				planOrderID.ValueString(),
			)
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			item, err := order.GetParentItem()
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			state = item.Data.State
		case "redis":
			order, err := orders.GetRedisOrder(
				client.Creds,
				client.ProjectName,
				planOrderID.ValueString(),
			)
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			item, err := order.GetParentItem()
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			state = item.Data.State
		case "redis_sentinel":
			order, err := orders.GetRedisSentinelOrder(
				client.Creds,
				client.ProjectName,
				planOrderID.ValueString(),
			)
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			item, err := order.GetParentItem()
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			state = item.Data.State
		case "clickhouse":
			order, err := orders.GetClickHouseOrder(
				client.Creds,
				client.ProjectName,
				planOrderID.ValueString(),
			)
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			item, err := order.GetParentItem()
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			state = item.Data.State
		case "wildfly":
			order, err := orders.GetWildflyOrder(
				client.Creds,
				client.ProjectName,
				planOrderID.ValueString(),
			)
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			item, err := order.GetParentItem()
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			state = item.Data.State
		case "nginx":
			order, err := orders.GetNginxOrder(
				client.Creds,
				client.ProjectName,
				planOrderID.ValueString(),
			)
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			item, err := order.GetParentItem()
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			state = item.Data.State
		case "agent_orchestration":
			order, err := orders.GetAgentOrchestrationOrder(
				client.Creds,
				client.ProjectName,
				planOrderID.ValueString(),
			)
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			item, err := order.GetParentItem()
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			state = item.Data.State
		case "airflow":
			order, err := orders.GetAirflowStandaloneOrder(
				client.Creds,
				client.ProjectName,
				planOrderID.ValueString(),
			)
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			item, err := order.GetParentItem()
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			state = item.Data.State
		case "grafana":
			order, err := orders.GetGrafanaOrder(
				client.Creds,
				client.ProjectName,
				planOrderID.ValueString(),
			)
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			item, err := order.GetParentItem()
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			state = item.Data.State
		default:
			result.Diagnostics.AddError(
				"Check resource is not deleted",
				fmt.Sprintf("Unknown provider == %s", inputProvider),
			)
			return result
		}
	}

	if inputType == "saas" {
		switch inputProvider {
		case "rqaas":
			order, err := orders.GetRQaaSOrder(
				client.Creds,
				client.ProjectName,
				planOrderID.ValueString(),
			)
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			item, err := order.GetParentItem()
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			state = item.Data.State
		}
	}

	if inputType == "paas_ktaas" {
		switch inputProvider {
		case "ktaas":
			order, err := orders.GetKTaaSOrder(
				client.Creds,
				client.ProjectName,
				planOrderID.ValueString(),
			)
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			item, err := order.GetParentItem()
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			state = item.Data.State
		}
	}

	if inputType == "s3" {
		switch inputProvider {
		case "ceph":
			order, err := orders.GetS3CephOrder(
				client.Creds,
				client.ProjectName,
				planOrderID.ValueString(),
			)
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			item, err := order.GetParentItem()
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			state = item.Data.State
		}

	}

	if inputType == "container_space" {
		if inputProvider == "kubernetes" {
			order, err := orders.GetK8sContainerSpaceOrder(
				client.Creds,
				client.ProjectName,
				planOrderID.ValueString(),
			)
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			item, err := order.GetParentItem()
			if err != nil {
				result.Diagnostics.AddError("Check resource is not deleted", err.Error())
				return result
			}
			state = item.Data.State
		}
	}

	if state == "" {
		result.Diagnostics.AddError(
			"Check resource is not deleted",
			"Can't extract state from order",
		)
		return result
	}

	if state == "deleted" || state == "deprovisioned" {
		result.IsDeleted = true
	}
	return result
}
