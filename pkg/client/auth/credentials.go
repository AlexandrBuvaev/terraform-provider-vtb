package auth

import (
	"encoding/json"
	"io"
	"time"

	"terraform-provider-vtb/pkg/client/requests"
)

type Credentials struct {
	ClientId         string `json:"-"`
	ClientSecret     string `json:"-"`
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	IDToken          string `json:"id_token"`
	NotBeforePolicy  int    `json:"not-before-policy"`
	SessionState     string `json:"session_state"`
	Scope            string `json:"scope"`
	generatedtime    uint64 `json:"-"`
}

func NewCredentials(clientID, clientSecret string) (*Credentials, error) {

	resp, err := requests.SendAuthRequest(clientID, clientSecret, true)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var creds Credentials
	err = json.Unmarshal(body, &creds)
	if err != nil {
		return nil, err
	}

	creds.ClientId = clientID
	creds.ClientSecret = clientSecret

	now := time.Now()
	creds.generatedtime = uint64(now.Unix())

	go creds.startAutoUpdate()
	return &creds, nil
}

func (c *Credentials) startAutoUpdate() {
	var err error
	currentTime := time.Now().Unix()
	for {
		if uint64(c.generatedtime)+uint64(c.ExpiresIn)-200 <= uint64(currentTime) {
			for attempts := 0; attempts < 3; attempts++ {
				err = c.UpdateToken()
				if err == nil {
					break
				}
			}
			if err != nil {
				panic("Can't update token")
			}
			currentTime = time.Now().Unix()
		} else {
			time.Sleep(3500 * time.Second)
			currentTime = time.Now().Unix()
		}
	}
}

func (c *Credentials) UpdateToken() error {

	resp, err := requests.SendAuthRequest(c.ClientId, c.ClientSecret, true)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, c)
	if err != nil {
		return err
	}

	now := time.Now()
	c.generatedtime = uint64(now.Unix())
	return nil
}
