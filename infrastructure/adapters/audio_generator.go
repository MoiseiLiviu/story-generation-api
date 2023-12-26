package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/config"
	"io"
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
	logger           outbound.LoggerPort
	elevenLabsConfig *config.ElevenLabsConfig
}

func NewAudioGenerator(contentFetcher ContentFetcher, elevenLabsConfig *config.ElevenLabsConfig, logger outbound.LoggerPort) outbound.AudioGeneratorPort {
	return &audioGenerator{
		ContentFetcher:   contentFetcher,
		logger:           logger,
		elevenLabsConfig: elevenLabsConfig,
	}
}

func (a *audioGenerator) Generate(ctx context.Context, audioReq outbound.GenerateAudioRequest) (io.ReadCloser, error) {
	req, err := a.getRequest(ctx, audioReq.Text, audioReq.VoiceID)
	if err != nil {
		a.logger.ErrorWithFields(err,
			"Failed to create the HTTP request",
			map[string]interface{}{
				"action": "Creating HTTP Request",
				"URL":    a.elevenLabsConfig.ApiUrl + "/" + audioReq.VoiceID,
			})
		return nil, err

	}
	res, err := a.FetchContent(req)
	if err != nil {
		a.logger.Error(err, "Failed to fetch the content")
		return nil, err
	}

	return res.Body, nil
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
		a.logger.Error(err, "Failed to marshal the request body")
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.elevenLabsConfig.ApiUrl+"/"+voiceID, bytes.NewBuffer(jsonPayload))
	if err != nil {
		a.logger.ErrorWithFields(err, "Failed to create the HTTP request", map[string]interface{}{
			"action": "Creating HTTP Request",
			"URL":    a.elevenLabsConfig.ApiUrl + "/" + voiceID,
		})
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
