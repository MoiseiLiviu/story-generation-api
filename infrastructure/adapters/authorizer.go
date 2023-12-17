package adapters

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"generate-script-lambda/config"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"strings"
)

type Authorizer interface {
	Authorize(ctx context.Context) (string, error)
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

type cognitoAuthorizer struct {
	conf *config.AuthorizerConfig
}

func NewCognitoAuthorizer(conf *config.AuthorizerConfig) Authorizer {
	return &cognitoAuthorizer{
		conf: conf,
	}
}

func (a *cognitoAuthorizer) Authorize(ctx context.Context) (string, error) {
	log.Info().Msg("Authorizing with Cognito...")
	clientCredentials := base64.StdEncoding.EncodeToString([]byte(a.conf.ClientID + ":" + a.conf.ClientSecret))

	requestBody := strings.NewReader("grant_type=client_credentials")

	req, err := http.NewRequestWithContext(ctx, "POST", a.conf.TokenEndpoint, requestBody)
	if err != nil {
		log.Err(err).Msg("Failed to create request!")
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+clientCredentials)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Err(err).Msg("Failed to send request!")
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Err(err).Msg("Failed to close response body!")
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Err(err).Msg("Failed to read response body!")
		return "", err
	}

	var tokenResponse TokenResponse
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		log.Err(err).Msg("Failed to unmarshal response body!")
		return "", err
	}
	log.Info().Msg("Successfully authorized with Cognito")

	return tokenResponse.AccessToken, nil
}
