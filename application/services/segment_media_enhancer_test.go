package services

import (
	"context"
	"generate-script-lambda/application/ports/inbound"
	"generate-script-lambda/channel_utils"
	"generate-script-lambda/config"
	"generate-script-lambda/infrastructure/adapters"
	"github.com/google/uuid"
	"github.com/panjf2000/ants/v2"
	"testing"
)

func TestSegmentMediaEnhancer_Enhance(t *testing.T) {
	const wordsPerStory = 500

	gptConfig, err := config.GetGptConfig()
	if err != nil {
		t.Fatal("Failed to get gpt config:", err)
	}

	dalleConfig, err := config.GetDaLLeConfig()
	if err != nil {
		t.Fatal("Failed to get dalle config:", err)
	}

	elevenLabsConfig, err := config.GetElevenLabsConfig()
	if err != nil {
		t.Fatal("Failed to get eleven labs config:", err)
	}

	workerPool, err := ants.NewPool(20)
	if err != nil {
		t.Fatal("Failed to create worker pool:", err)
	}

	logger := adapters.NewZerologWrapper()

	scriptGenerator := adapters.NewStoryScriptGenerator(wordsPerStory, gptConfig, workerPool, logger)

	textGenerator := NewSegmentTextGenerator(logger, scriptGenerator, workerPool)

	fetcher := adapters.NewContentFetcher(logger)

	imageGenerator := adapters.NewImageGenerator(fetcher, dalleConfig, logger)

	audioGenerator := adapters.NewAudioGenerator(fetcher, elevenLabsConfig, logger)

	enhancer := NewSegmentMediaEnhancer(logger, imageGenerator, audioGenerator, workerPool)

	ctx := context.Background()

	segmentsCh, generatorErrCh := textGenerator.Generate(ctx, inbound.GenerateSegmentsParams{
		Input:   "drunken mermaid",
		StoryID: uuid.NewString(),
	})

	enhancedSegmentsCh, enhancerErrCh := enhancer.Enhance(ctx, segmentsCh, "2EiwWnXFnvU5JabPnv8n")
	mergedErrCh, err := channel_utils.MergeChannels(workerPool, generatorErrCh, enhancerErrCh)
	if err != nil {
		t.Fatal("Failed to merge error channels:", err)
	}

	for {
		select {
		case err, ok := <-mergedErrCh:
			if ok {
				t.Fatal("Received an error:", err)
			}
		case _, ok := <-enhancedSegmentsCh:
			if !ok {
				return
			}
		}
	}
}
