package services

import (
	"context"
	"generate-script-lambda/application/ports/inbound"
	"generate-script-lambda/config"
	"generate-script-lambda/infrastructure/adapters"
	"github.com/google/uuid"
	"github.com/panjf2000/ants/v2"
	"testing"
)

func TestSegmentTextGenerator_Generate(t *testing.T) {
	const wordsPerStory = 500

	gptConfig, err := config.GetGptConfig()
	if err != nil {
		t.Fatal("Failed to get gpt config:", err)
	}

	workerPool, err := ants.NewPool(10)
	if err != nil {
		t.Fatal("Failed to create worker pool:", err)
	}

	logger := adapters.NewZerologWrapper()

	scriptGenerator := adapters.NewStoryScriptGenerator(wordsPerStory, gptConfig, workerPool, logger)

	textGenerator := NewSegmentTextGenerator(logger, scriptGenerator, workerPool)

	ctx := context.Background()

	outCh, errCh := textGenerator.Generate(ctx, inbound.GenerateSegmentsParams{
		Input:   "drunken mermaid",
		StoryID: uuid.NewString(),
	})

	for {
		select {
		case err, ok := <-errCh:
			if ok {
				t.Fatal("Received an error:", err)
			}
		case _, ok := <-outCh:
			if !ok {
				return
			}
		}
	}
}
