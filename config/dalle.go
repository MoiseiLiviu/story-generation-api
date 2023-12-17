package config

import (
	"fmt"
	"os"
)

type DaLLeConfig struct {
	ApiUrl string
	ApiKey string
}

func GetDaLLeConfig() (*DaLLeConfig, error) {
	apiUrl := os.Getenv("DALLE_API_URL")
	if apiUrl == "" {
		return nil, fmt.Errorf("DALLE_API_URL must be set")
	}
	apiKey := os.Getenv("DALLE_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("DALLE_API_KEY must be set")
	}

	return &DaLLeConfig{
		ApiUrl: apiUrl,
		ApiKey: apiKey,
	}, nil
}
