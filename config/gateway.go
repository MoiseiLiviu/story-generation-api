package config

import (
	"fmt"
	"os"
)

type GatewayConfig struct {
	ApiUrl string
}

func GetGatewayConfig() (*GatewayConfig, error) {
	apiUrl := os.Getenv("GATEWAY_URL")
	if apiUrl == "" {
		return nil, fmt.Errorf("GATEWAY_URL must be set")
	}

	return &GatewayConfig{
		ApiUrl: apiUrl,
	}, nil
}
