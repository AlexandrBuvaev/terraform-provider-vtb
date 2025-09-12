package orders

import (
	"encoding/json"
	"fmt"
	"io"

	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/requests"
)

func GetComputeOrder(creds *auth.Credentials, projectName, orderID string) (*Compute, error) {
	orderType := &Compute{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*Compute), nil
}

func GetNginxOrder(creds *auth.Credentials, projectName, orderID string) (*Nginx, error) {
	orderType := &Nginx{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*Nginx), nil
}

func GetAgentOrchestrationOrder(creds *auth.Credentials, projectName, orderID string) (*AgentOrchestration, error) {
	orderType := &AgentOrchestration{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*AgentOrchestration), nil
}

func GetRedisOrder(creds *auth.Credentials, projectName, orderID string) (*Redis, error) {
	orderType := &Redis{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*Redis), nil
}

func GetRedisSentinelOrder(creds *auth.Credentials, projectName, orderID string) (*RedisSentinel, error) {
	orderType := &RedisSentinel{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*RedisSentinel), nil
}

func GetClickHouseOrder(creds *auth.Credentials, projectName, orderID string) (*ClickHouse, error) {
	orderType := &ClickHouse{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*ClickHouse), nil
}

func GetClickhouseClusterOrder(creds *auth.Credentials, projectName, orderID string) (*ClickHouseCluster, error) {
	orderType := &ClickHouseCluster{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*ClickHouseCluster), nil
}

func GetWildflyOrder(creds *auth.Credentials, projectName, orderID string) (*Wildfly, error) {
	orderType := &Wildfly{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*Wildfly), nil
}

func GetPostgresqlOrder(creds *auth.Credentials, projectName, orderID string) (*PostgresqlOrder, error) {
	orderType := &PostgresqlOrder{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*PostgresqlOrder), nil
}

func GetEtcdOrder(creds *auth.Credentials, projectName, orderID string) (*EtcdOrder, error) {
	orderType := &EtcdOrder{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*EtcdOrder), nil
}
func GetGrafanaOrder(creds *auth.Credentials, projectName, orderID string) (*GrafanaOrder, error) {
	orderType := &GrafanaOrder{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*GrafanaOrder), nil
}

func GetKafkaOrder(creds *auth.Credentials, projectName, orderID string) (*Kafka, error) {
	orderType := &Kafka{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*Kafka), nil
}

func GetRabbitMQOrder(creds *auth.Credentials, projectName, orderID string) (*RabbitMQ, error) {
	orderType := &RabbitMQ{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*RabbitMQ), nil
}

func GetArtemisOrder(creds *auth.Credentials, projectName, orderID string) (*ArtemisOrder, error) {
	orderType := &ArtemisOrder{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*ArtemisOrder), nil
}

func GetOpenMessagingOrder(creds *auth.Credentials, projectName, orderID string) (*OpenMessagingOrder, error) {
	orderType := &OpenMessagingOrder{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*OpenMessagingOrder), nil
}

func GetK8sClusterOrder(creds *auth.Credentials, projectName, orderID string) (*K8sClusterOrder, error) {
	orderType := &K8sClusterOrder{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*K8sClusterOrder), nil
}

func GetK8sProjectOrder(creds *auth.Credentials, projectName, orderID string) (*K8sProjectOrder, error) {
	orderType := &K8sProjectOrder{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*K8sProjectOrder), nil
}

func GetDebeziumOrder(creds *auth.Credentials, projectName, orderID string) (*SyncXpertCluster, error) {
	orderType := &SyncXpertCluster{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*SyncXpertCluster), nil
}

func GetBalancerV3Order(creds *auth.Credentials, projectName, orderID string) (*BalancerV3, error) {
	orderType := &BalancerV3{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err

	}
	return result.(*BalancerV3), nil
}

func GetAirflowClusterOrder(creds *auth.Credentials, projectName, orderID string) (*AirflowCluster, error) {
	orderType := &AirflowCluster{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*AirflowCluster), nil
}

func GetAirflowStandaloneOrder(creds *auth.Credentials, projectName, orderID string) (*AirflowStandalone, error) {
	orderType := &AirflowStandalone{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*AirflowStandalone), nil
}

func GetTarantoolClusterOrder(creds *auth.Credentials, projectName, orderID string) (*TarantoolCluster, error) {
	orderType := &TarantoolCluster{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*TarantoolCluster), nil
}

func GetRQaaSOrder(creds *auth.Credentials, projectName, orderID string) (*RQaaS, error) {
	orderType := &RQaaS{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*RQaaS), nil
}

func GetElasticSearchOrder(creds *auth.Credentials, projectName, orderID string) (*ElasticSearch, error) {
	orderType := &ElasticSearch{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*ElasticSearch), nil
}

func GetScyllaDbClusterOrder(creds *auth.Credentials, projectName, orderID string) (*ScyllaDbCluster, error) {
	orderType := &ScyllaDbCluster{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*ScyllaDbCluster), nil
}

func GetKTaaSOrder(creds *auth.Credentials, projectName, orderID string) (*KTaaS, error) {
	orderType := &KTaaS{}
	result, err := getOrder(creds, projectName, orderID, orderType)
	if err != nil {
		return nil, err
	}
	return result.(*KTaaS), nil
}

func getOrder(creds *auth.Credentials, projectName, orderID string, orderType interface{}) (interface{}, error) {
	var order interface{}

	switch t := orderType.(type) {
	case *Compute:
		order = &Compute{
			Order: Order{Creds: creds},
		}
	case *Nginx:
		order = &Nginx{
			Order: Order{Creds: creds},
		}
	case *AgentOrchestration:
		order = &AgentOrchestration{
			Order: Order{Creds: creds},
		}
	case *Redis:
		order = &Redis{
			Order: Order{Creds: creds},
		}
	case *RedisSentinel:
		order = &RedisSentinel{
			Order: Order{Creds: creds},
		}
	case *ClickHouse:
		order = &ClickHouse{
			Order: Order{Creds: creds},
		}
	case *ClickHouseCluster:
		order = &ClickHouseCluster{
			Order: Order{Creds: creds},
		}
	case *Wildfly:
		order = &Wildfly{
			Order: Order{Creds: creds},
		}
	case *PostgresqlOrder:
		order = &PostgresqlOrder{
			Order: Order{Creds: creds},
		}
	case *EtcdOrder:
		order = &EtcdOrder{
			Order: Order{Creds: creds},
		}
	case *Kafka:
		order = &Kafka{
			Order: Order{Creds: creds},
		}
	case *RabbitMQ:
		order = &RabbitMQ{
			Order: Order{Creds: creds},
		}
	case *ArtemisOrder:
		order = &ArtemisOrder{
			Order: Order{Creds: creds},
		}
	case *OpenMessagingOrder:
		order = &OpenMessagingOrder{
			Order: Order{Creds: creds},
		}
	case *K8sClusterOrder:
		order = &K8sClusterOrder{
			Order: Order{Creds: creds},
		}
	case *K8sProjectOrder:
		order = &K8sProjectOrder{
			Order: Order{Creds: creds},
		}
	case *SyncXpertCluster:
		order = &SyncXpertCluster{
			Order: Order{Creds: creds},
		}
	case *BalancerV3:
		order = &BalancerV3{
			Order: Order{Creds: creds},
		}
	case *AirflowCluster:
		order = &AirflowCluster{
			Order: Order{Creds: creds},
		}
	case *AirflowStandalone:
		order = &AirflowStandalone{
			Order: Order{Creds: creds},
		}
	case *GrafanaOrder:
		order = &GrafanaOrder{
			Order: Order{Creds: creds},
		}
	case *TarantoolCluster:
		order = &TarantoolCluster{
			Order: Order{Creds: creds},
		}
	case *RQaaS:
		order = &RQaaS{
			Order: Order{Creds: creds},
		}
	case *ElasticSearch:
		order = &ElasticSearch{
			Order: Order{Creds: creds},
		}
	case *KTaaS:
		order = &KTaaS{
			Order: Order{Creds: creds},
		}
	case *ScyllaDbCluster:
		order = &ScyllaDbCluster{
			Order: Order{Creds: creds},
		}
	default:
		return nil, fmt.Errorf("unsupported order type %T", t)
	}

	uri := fmt.Sprintf(
		"order-service/api/v1/projects/%s/orders/%s?include=last_action",
		projectName, orderID,
	)

	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, nil)
	if err != nil {
		return order, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return order, err
	}

	err = json.Unmarshal(body, order)
	if err != nil {
		return order, err
	}

	// Обратная совместимость для старых заказов (созданных до перехода на AZ вместо DC)
	if attrs, ok := order.(interface{ GetBasicAttrs() *BasicAttrs }); ok {
		dc := attrs.GetBasicAttrs().DataCenter
		if dc != "" {
			zone, err := convertDCtoAZ(dc)
			if err != nil {
				return nil, err
			}
			attrs.GetBasicAttrs().AvailabilityZone = zone
		}
	}
	return order, nil
}

func convertDCtoAZ(dcCode string) (string, error) {

	dataCenterToZone := map[string]string{
		"20": "msk-east",
		"2":  "msk-east",
		"50": "msk-north",
		"5":  "msk-north",
		"t":  "msk-t1",
		"7":  "msk-north",
		"55": "msk-north",
	}

	zoneCode, ok := dataCenterToZone[dcCode]
	if !ok {
		return "", fmt.Errorf("can't map data_center=%s to availability zone", dcCode)
	}
	return zoneCode, nil
}
