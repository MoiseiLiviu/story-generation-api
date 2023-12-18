package adapters

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/config"
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
	logger outbound.LoggerPort
	conf   *config.AuthorizerConfig
}

func NewCognitoAuthorizer(logger outbound.LoggerPort, conf *config.AuthorizerConfig) Authorizer {
	return &cognitoAuthorizer{
		logger: logger,
		conf:   conf,
	}
}

func (a *cognitoAuthorizer) Authorize(ctx context.Context) (string, error) {
	a.logger.Info("Authorizing with Cognito")
	clientCredentials := base64.StdEncoding.EncodeToString([]byte(a.conf.ClientID + ":" + a.conf.ClientSecret))

	requestBody := strings.NewReader("grant_type=client_credentials")

	req, err := http.NewRequestWithContext(ctx, "POST", a.conf.TokenEndpoint, requestBody)
	if err != nil {
		a.logger.Error(err, "Failed to create the HTTP request")
		return "", err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+clientCredentials)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		a.logger.Error(err, "Failed to send the HTTP request")
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			a.logger.Error(err, "Failed to close the response body")
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		a.logger.Error(err, "Failed to read the response body")
		return "", err
	}

	var tokenResponse TokenResponse
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		a.logger.Error(err, "Failed to unmarshal the response body")
		return "", err
	}
	a.logger.Info("Successfully authorized with Cognito")

	return tokenResponse.AccessToken, nil
}
