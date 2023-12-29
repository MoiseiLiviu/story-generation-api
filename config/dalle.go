package config

import (
	"fmt"
	"os"
)

type DaLLeConfig struct {
	ApiUrl string
	ApiKey string
	Size   string
	Model  string
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
	size := os.Getenv("DALLE_SIZE")
	if size == "" {
		return nil, fmt.Errorf("DALLE_SIZE must be set")
	}
	model := os.Getenv("DALLE_MODEL")
	if model == "" {
		return nil, fmt.Errorf("DALLE_MODEL must be set")
	}

	return &DaLLeConfig{
		ApiUrl: apiUrl,
		ApiKey: apiKey,
		Size:   size,
		Model:  model,
	}, nil
}
