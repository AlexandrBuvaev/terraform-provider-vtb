package productcatalog

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"terraform-provider-vtb/pkg/client/auth"
	"terraform-provider-vtb/pkg/client/requests"
)

type ProductImageData struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Title string `json:"title"`
}

func GetProductImageData(creds *auth.Credentials, productName, env string) (*ProductImageData, error) {
	params := map[string]string{
		"env": env,
	}

	uri := fmt.Sprintf("product-catalog/api/v2/products/%s/optimized/", productName)

	resp, err := requests.SendRequest(creds.AccessToken, uri, "GET", nil, params)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"request to product-catalog with product name '%s' failed with status %d, response: %s",
			productName, resp.StatusCode, string(body),
		)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var productImageData ProductImageData
	if err := json.Unmarshal(body, &productImageData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &productImageData, nil
}
