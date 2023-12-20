package adapters

import (
	"context"
	"fmt"
	"generate-script-lambda/config"
	"github.com/panjf2000/ants/v2"
	"strings"
	"testing"
)

func TestStoryScriptGeneratorIntegration_Generate(t *testing.T) {
	const wordsPerStory = 500

	gptConfig, err := config.GetGptConfig()
	if err != nil {
		t.Fatal("Failed to get gpt config:", err)
	}

	workerPool, err := ants.NewPool(10)
	if err != nil {
		t.Fatal("Failed to create worker pool:", err)
	}

	logger := NewZerologWrapper()

	generator := NewStoryScriptGenerator(wordsPerStory, gptConfig, workerPool, logger)

	ctx := context.Background()
	output, errCh := generator.Generate(ctx, "drunken mermaid")

	var builder strings.Builder

	for {
		select {
		case err, ok := <-errCh:
			if ok {
				t.Fatal("Received an error:", err)
			}
		case token, ok := <-output:
			if !ok {
				logger.Info(fmt.Sprintf("Generated script: %s", builder.String()))
				return
			} else {
				builder.WriteString(token)
			}
		}
	}
}
