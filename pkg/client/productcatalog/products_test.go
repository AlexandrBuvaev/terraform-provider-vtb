package productcatalog

import (
	"log"
	"terraform-provider-vtb/pkg/client/test"
	"testing"
)

func TestGetProductImageData(t *testing.T) {
	product, err := GetProductImageData(test.SharedCreds, "rqaas", "dev")
	if err != nil {
		log.Fatalf("Can't fetch from product-catalog, error: %v", err.Error())
	}

	log.Printf("product_id=%v", product.ID)
}
