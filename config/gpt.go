package config

import (
	"fmt"
	"os"
)

type GptConfig struct {
	ApiUrl string
	ApiKey string
	Model  string
}

func GetGptConfig() (*GptConfig, error) {
	model := os.Getenv("GPT_MODEL")
	if model == "" {
		return nil, fmt.Errorf("GPT_MODEL must be set")
	}
	apiUrl := os.Getenv("GPT_API_URL")
	if apiUrl == "" {
		return nil, fmt.Errorf("GPT_API_URL must be set")
	}
	apiKey := os.Getenv("GPT_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("GPT_API_KEY must be set")
	}
	return &GptConfig{
		ApiUrl: apiUrl,
		ApiKey: apiKey,
		Model:  model,
	}, nil
}
