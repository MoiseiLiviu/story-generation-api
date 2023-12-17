package config

import (
	"fmt"
	"os"
	"strconv"
)

type ElevenLabsConfig struct {
	ApiUrl          string
	ApiKey          string
	ModelId         string
	Stability       float64
	SimilarityBoost float64
}

func GetElevenLabsConfig() (*ElevenLabsConfig, error) {
	apiUrl := os.Getenv("ELEVEN_LABS_API_URL")
	if apiUrl == "" {
		return nil, fmt.Errorf("ELEVEN_LABS_API_URL must be set")
	}
	apiKey := os.Getenv("ELEVEN_LABS_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ELEVEN_LABS_API_KEY must be set")
	}
	modelId := os.Getenv("ELEVEN_LABS_MODEL_ID")
	if modelId == "" {
		return nil, fmt.Errorf("ELEVEN_LABS_MODEL_ID must be set")
	}
	stability := os.Getenv("ELEVEN_LABS_STABILITY")
	if stability == "" {
		return nil, fmt.Errorf("ELEVEN_LABS_STABILITY must be set")
	}
	stabilityVal, err := strconv.ParseFloat(stability, 32)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse eleven labs stability")
	}
	similarityBoost := os.Getenv("ELEVEN_LABS_SIMILARITY_BOOST")
	if similarityBoost == "" {
		return nil, fmt.Errorf("ELEVEN_LABS_SIMILARITY_BOOST must be set")
	}
	similarityBoostVal, err := strconv.ParseFloat(similarityBoost, 32)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse eleven labs similarity boost")
	}

	return &ElevenLabsConfig{
		ApiUrl:          apiUrl,
		ApiKey:          apiKey,
		ModelId:         modelId,
		Stability:       stabilityVal,
		SimilarityBoost: similarityBoostVal,
	}, nil
}
