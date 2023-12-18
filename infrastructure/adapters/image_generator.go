package adapters

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"generate-script-lambda/application/ports/outbound"
	"generate-script-lambda/config"
	"net/http"
)

type DalleApiRequest struct {
	Prompt         string `json:"prompt"`
	Size           string `json:"size"`
	Number         int    `json:"n"`
	ResponseFormat string `json:"response_format"`
}

type DalleApiResponse struct {
	Data []struct {
		B64Json string `json:"b64_json"`
	} `json:"data"`
}

type imageGenerator struct {
	ContentFetcher
	logger      outbound.LoggerPort
	dalleConfig *config.DaLLeConfig
}

func NewImageGenerator(contentFetcher ContentFetcher, dalleConfig *config.DaLLeConfig, logger outbound.LoggerPort) outbound.ImageGeneratorPort {
	return &imageGenerator{
		logger:         logger,
		ContentFetcher: contentFetcher,
		dalleConfig:    dalleConfig,
	}
}

func (i *imageGenerator) Generate(ctx context.Context, description string) ([]byte, error) {
	req, err := i.getRequest(ctx, description)
	if err != nil {
		i.logger.Error(err, "Failed to create the HTTP request")
		return nil, err
	}

	var dalleRes DalleApiResponse

	rawRes, err := i.FetchContent(req)
	if err != nil {
		i.logger.Error(err, "Failed to fetch the content")
		return nil, err
	}

	err = json.Unmarshal(rawRes, &dalleRes)
	if err != nil {
		i.logger.Error(err, "Failed to unmarshal the response")
		return nil, err
	}

	decodedImage, err := base64.StdEncoding.DecodeString(dalleRes.Data[0].B64Json)
	if err != nil {
		i.logger.Error(err, "Failed to decode the image")
		return nil, err
	}

	return decodedImage, nil
}

func (i *imageGenerator) getRequest(ctx context.Context, text string) (*http.Request, error) {
	reqBody := DalleApiRequest{
		Prompt:         fmt.Sprintf("%s, in a cartoon style", text),
		Size:           "256x256",
		Number:         1,
		ResponseFormat: "b64_json",
	}

	jsonPayload, err := json.Marshal(reqBody)
	if err != nil {
		i.logger.Error(err, "Failed to marshal the request body")
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", i.dalleConfig.ApiUrl, bytes.NewBuffer(jsonPayload))
	if err != nil {
		i.logger.Error(err, "Failed to create the HTTP request")
		return nil, err
	}

	reqHeaders := map[string]string{
		"Authorization": "Bearer " + i.dalleConfig.ApiKey,
		"Content-Type":  "application/json",
	}
	for key, value := range reqHeaders {
		req.Header.Add(key, value)
	}

	return req, nil
}
