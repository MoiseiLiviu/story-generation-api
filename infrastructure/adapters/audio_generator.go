package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/config"
	"github.com/rs/zerolog/log"
	"net/http"
)

type ElevenLabsRequest struct {
	Text          string        `json:"text"`
	ModelId       string        `json:"model_id"`
	VoiceSettings VoiceSettings `json:"voice_settings"`
}

type VoiceSettings struct {
	Stability       float64 `json:"stability"`
	SimilarityBoost float64 `json:"similarity_boost"`
}

type audioGenerator struct {
	ContentFetcher
	elevenLabsConfig *config.ElevenLabsConfig
}

func NewAudioGenerator(contentFetcher ContentFetcher, elevenLabsConfig *config.ElevenLabsConfig) outbound.AudioGeneratorPort {
	return &audioGenerator{
		ContentFetcher:   contentFetcher,
		elevenLabsConfig: elevenLabsConfig,
	}
}

func (a *audioGenerator) Generate(ctx context.Context, generateAudioParams outbound.GenerateAudioParams) ([]byte, error) {
	req, err := a.getRequest(ctx, generateAudioParams.Text, generateAudioParams.VoiceID)
	if err != nil {
		log.Error().Err(err).Str("action", "Fetching Audio").Str("text", generateAudioParams.Text).Msg("Failed to construct the HTTP request for audio fetching")
		return nil, err
	}

	return a.FetchContent(req)
}

func (a *audioGenerator) getRequest(ctx context.Context, text string, voiceID string) (*http.Request, error) {
	reqBody := ElevenLabsRequest{
		Text:    text,
		ModelId: a.elevenLabsConfig.ModelId,
		VoiceSettings: VoiceSettings{
			Stability:       a.elevenLabsConfig.Stability,
			SimilarityBoost: a.elevenLabsConfig.SimilarityBoost,
		},
	}

	jsonPayload, err := json.Marshal(reqBody)
	if err != nil {
		log.Error().Err(err).Str("action", "Marshalling JSON").Interface("ElevenLabsRequest", reqBody).Msg("Failed to marshal the request body for ElevenLabs API")
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.elevenLabsConfig.ApiUrl+"/"+voiceID, bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Error().Err(err).Str("action", "Creating HTTP Request").Str("URL", a.elevenLabsConfig.ApiUrl+"/"+voiceID).Msg("Failed to create the HTTP POST request")
		return nil, err
	}

	reqHeaders := map[string]string{
		"Accept":       "audio/mpeg",
		"xi-api-key":   a.elevenLabsConfig.ApiKey,
		"Content-Type": "application/json",
	}
	for key, value := range reqHeaders {
		req.Header.Add(key, value)
	}

	return req, nil
}
