package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/config"
	"github.com/donovanhide/eventsource"
	"github.com/panjf2000/ants/v2"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
)

const DoneSignal = "[DONE]"
const MaxRetries = 3

type chatGptRequest struct {
	Stream   bool             `json:"stream"`
	Model    string           `json:"model"`
	Messages []chatGptMessage `json:"messages"`
}

type chatGptMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatGptChunkBody struct {
	Choices []chatGptResponseChoice `json:"choices"`
}

type chatGptResponseChoice struct {
	Index int `json:"index"`
	Delta struct {
		Content string `json:"content"`
	} `json:"delta"`
}

type storyScriptGenerator struct {
	wordsPerStory int
	gptConfig     *config.GptConfig
	workerPool    *ants.Pool
}

func NewStoryScriptGenerator(wordsPerStory int, gptConfig *config.GptConfig, workerPool *ants.Pool) outbound.StoryScriptGeneratorPort {
	return &storyScriptGenerator{
		wordsPerStory: wordsPerStory,
		gptConfig:     gptConfig,
		workerPool:    workerPool,
	}
}

func (s *storyScriptGenerator) Generate(ctx context.Context, input string) (<-chan string, <-chan error) {
	out := make(chan string)
	errCh := make(chan error)

	retryCount := 0

	newCtx, cancel := context.WithCancel(ctx)

	err := s.workerPool.Submit(func() {
		defer close(out)
		defer close(errCh)
		defer cancel()
		req, err := s.createRequest(ctx, input)
		if err != nil {
			log.Error().Err(err).Str("input", input).Msg("Failed to create request for streaming script")
			errCh <- err
			return
		}

		stream, err := eventsource.SubscribeWithRequest("", req)
		if err != nil {
			log.Error().Err(err).Msg("Failed to subscribe to script stream")
			errCh <- err
			return
		}
		for {
			select {
			case <-newCtx.Done():
				return
			case ev := <-stream.Events:
				if ev.Data() != DoneSignal {
					payload, err := s.extractPayload(ev)
					if err != nil {
						errCh <- err
						cancel()
						return
					} else {
						out <- payload
					}
				}
				retryCount = 0
			case err := <-stream.Errors:
				if err == io.EOF {
					log.Info().Msg("Reached end of stream")
					return
				} else if retryCount < MaxRetries {
					log.Warn().Err(err).Int("retryCount", retryCount).Msg("Error occurred during streaming, will retry")
					retryCount++
					continue
				}
				log.Error().Err(err).Msg("Maximum retries reached for streaming script")
				errCh <- err
				cancel()
				return
			}
		}
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to submit task to worker pool for streaming script")
		errCh <- err
	}

	return out, errCh
}

func (s *storyScriptGenerator) extractPayload(event eventsource.Event) (string, error) {
	var chunkBody chatGptChunkBody
	err := json.Unmarshal([]byte(event.Data()), &chunkBody)
	if err != nil {
		log.Error().Err(err).Msg("Failed to unmarshal script stream event data")
		return "", err
	}
	fmt.Println(fmt.Sprintf("Event content: %s", chunkBody.Choices[0].Delta.Content))

	return chunkBody.Choices[0].Delta.Content, nil
}

func (s *storyScriptGenerator) createRequest(ctx context.Context, input string) (*http.Request, error) {
	promptMessage := chatGptMessage{
		Role: "system",
		Content: fmt.Sprintf("Write a story on the topic: %s."+
			"The start of the story should be a short, quick description of the scenery, written in squared brackets.\n"+
			"Example: [White castle with a cloudy sky]\n"+
			"The squared brackets descriptions:\n"+
			"- Should not contain any names\n"+
			"- Should be descriptive in a short manner (at most one sentence).\n"+
			"- Should be used only 4 times per story\n"+
			"- Should be used in a meaningful way (only when the scenery changes drastically)\n"+
			"- Should not be part of the storytelling (similar to a theater play, just to set the scenery)\n"+
			"The story should be of about %d words.", input, s.wordsPerStory),
	}

	promptReq := chatGptRequest{
		Stream:   true,
		Model:    s.gptConfig.Model,
		Messages: []chatGptMessage{promptMessage},
	}

	payloadBytes, err := json.Marshal(promptReq)
	if err != nil {
		log.Error().Err(err).Msg("Error encoding request payload for script stream")
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.gptConfig.ApiUrl, bytes.NewBuffer(payloadBytes))
	if err != nil {
		log.Error().Err(err).Msg("Error creating HTTP request for script stream")
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+s.gptConfig.ApiKey)
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}
