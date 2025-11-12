package clusterlayout

import (
	"fmt"
	"strings"
)

func selectClusterProductType(layout string) (string, error) {

	products := []string{
		"kafka",
		"debezium",
		"rabbitmq",
		"artemis",
		"balancer_v3",
		"balancer",
		"gslb",
		"airflow", // in airflow layout is like one_dc:webserver-2:scheduler-2:worker-2
		"tarantool",
	}

	for _, product := range products {
		switch product {
		case "tarantool":
			if strings.Contains(layout, "tarantool") {
				return "tarantool_v2", nil
			}
		case "airflow":
			if strings.Contains(layout, "worker") && strings.Contains(layout, "scheduler") {
				return product, nil
			}
		default:
			if strings.Contains(layout, product) {
				return product, nil
			}
		}
	}

	return "", fmt.Errorf(
		"layout invalid: there is no suitable product for %s.\nAvailable cluster products: [%s]",
		layout, strings.Join(products, ", "),
	)
}
