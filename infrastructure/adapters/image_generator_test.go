package adapters

import (
	"context"
	"generate-script-lambda/config"
	"testing"
)

func TestImageGenerator_Generate(t *testing.T) {
	logger := NewZerologWrapper()
	cf := NewContentFetcher(logger)
	dalleConfig, err := config.GetDaLLeConfig()
	if err != nil {
		t.Errorf("error getting dalle config: %v", err)
	}
	imageGenerator := NewImageGenerator(cf, dalleConfig, logger)
	_, err = imageGenerator.Generate(context.Background(), "little siren")
	if err != nil {
		t.Errorf("error generating image: %v", err)
	}
}
