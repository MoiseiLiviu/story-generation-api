package adapters

import (
	"context"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/config"
	"testing"
)

func TestAudioGenerator_Generate(t *testing.T) {
	elevenLabsConfig, err := config.GetElevenLabsConfig()
	if err != nil {
		t.Fatal("Failed to get eleven labs config:", err)
	}

	logger := NewZerologWrapper()

	fetcher := NewContentFetcher(logger)

	generator := NewAudioGenerator(fetcher, elevenLabsConfig, logger)

	_, err = generator.Generate(context.Background(), outbound.GenerateAudioParams{
		Text:    "Hello world",
		VoiceID: "2EiwWnXFnvU5JabPnv8n",
	})
	if err != nil {
		t.Fatal("Failed to generate audio:", err)
	}
}
