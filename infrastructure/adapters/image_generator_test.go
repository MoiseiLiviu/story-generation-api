package adapters

import (
	"context"
	"generate-script-lambda/config"
	"testing"
)

func TestImageGenerator_Generate(t *testing.T) {
	dalleConfig, err := config.GetDaLLeConfig()
	if err != nil {
		t.Fatal("Failed to get dalle config:", err)
	}
	logger := NewZerologWrapper()
	fetcher := NewContentFetcher(logger)
	generator := NewImageGenerator(fetcher, dalleConfig, logger)

	_, err = generator.Generate(context.Background(), "Hello world")
	if err != nil {
		t.Fatal("Failed to generate image:", err)
	}
}
