package requests

import (
	"bytes"
	"crypto/tls"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var PortalAPI, PortalAuthURL, PortalConsoleUrl string = func() (apiUrl, authUrl, consoleUrl string) {
	switch os.Getenv("PORTAL_STAND") {
	case "blue":
		return "api.blue.cloud.vtb.ru",
			"auth.blue.cloud.vtb.ru",
			"console.blue.cloud.vtb.ru"
	case "dev":
		return "dev-api.cloud.vtb.ru",
			"dev-api.cloud.vtb.ru",
			"console.cloud.vtb.ru"
	case "test":
		return "test-api.cloud.vtb.ru",
			"test-api.cloud.vtb.ru",
			"console.cloud.vtb.ru"
	default:
		return "api.cloud.vtb.ru",
			"auth.cloud.vtb.ru",
			"console.cloud.vtb.ru"
	}
}()

func SendRequest(
	AccessToken,
	uri,
	method string,
	payload []byte,
	parameters map[string]string,
) (*http.Response, error) {

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// prepare url-encoded data
	data := url.Values{}
	for key, value := range parameters {
		data.Set(key, value)
	}
	encodedData := data.Encode()

	// prepare url
	url := fmt.Sprintf("https://%s/%s", PortalAPI, uri)
	if len(parameters) > 0 {
		url = url + "?" + encodedData
	}

	// prepare client
	client := http.Client{
		Timeout: 60 * time.Second,
	}
	request, err := http.NewRequest(method, url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", AccessToken))

	// send request
	attempt := 0
	var resp *http.Response
	for attempt < 5 {
		resp, err = client.Do(request)
		if err == nil {
			break
		}
		attempt++
		time.Sleep(10 * time.Second)
	}

	if err != nil && attempt == 5 {
		return nil, fmt.Errorf("retry limit exceeded, all attemps was failed: %v", err)
	}

	if !isStatusCodeAcceptable(resp.StatusCode) {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return resp, err
		}
		return resp, fmt.Errorf(
			"action (with X-Request-Id: %s) was failed with status %s.\nError message: %s",
			resp.Header.Get("X-Request-Id"),
			resp.Status,
			string(body),
		)
	}
	return resp, nil
}

func SendAuthRequest(clientID, secret string, isNew bool) (*http.Response, error) {

	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// prepare url-encoded data
	data := url.Values{}
	data.Set("client_id", clientID)
	data.Set("client_secret", secret)

	if isNew {
		data.Set("grant_type", "client_credentials")
		data.Set("scope", "openid")
	} else {
		data.Set("refresh_token", secret)
		data.Set("grant_type", "refresh_token")
	}
	body := strings.NewReader(data.Encode())

	// prepare url
	url := fmt.Sprintf(
		"https://%s/auth/realms/Portal/protocol/openid-connect/token",
		PortalAuthURL,
	)

	// prepare client
	client := http.Client{
		Timeout: 30 * time.Second,
	}
	request, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// send request
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	// handling status
	if !isStatusCodeAcceptable(resp.StatusCode) {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"auth request has status == %s\nError message: %s",
			resp.Status, string(body),
		)
	}
	return resp, nil
}

func isStatusCodeAcceptable(statusCode int) bool {
	acceptableCodes := []int{200, 201, 204}
	for _, code := range acceptableCodes {
		if code == statusCode {
			return true
		}
	}
	return false
}
