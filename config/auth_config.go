package config

import (
	"fmt"
	"os"
)

type AuthorizerConfig struct {
	ClientID      string
	ClientSecret  string
	TokenEndpoint string
}

func NewAuthorizerConfig() (*AuthorizerConfig, error) {
	clientID := os.Getenv("CLIENT_ID")
	if clientID == "" {
		return nil, fmt.Errorf("CLIENT_ID environment variable not set")
	}

	clientSecret := os.Getenv("CLIENT_SECRET")
	if clientSecret == "" {
		return nil, fmt.Errorf("CLIENT_SECRET environment variable not set")
	}

	tokenEndpoint := os.Getenv("TOKEN_ENDPOINT")
	if tokenEndpoint == "" {
		return nil, fmt.Errorf("TOKEN_ENDPOINT environment variable not set")
	}
	return &AuthorizerConfig{
		ClientID:      clientID,
		ClientSecret:  clientSecret,
		TokenEndpoint: tokenEndpoint,
	}, nil
}
