package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"generate-script-lambda/application/ports/outbound"
	"io"
	"net/http"
)

type StoryRequest struct {
	Input  string `json:"input"`
	ID     string `json:"id"`
	UserID string `json:"user_id"`
}

type storySaver struct {
	storyApiUrl string
	logger      outbound.LoggerPort
	authorizer  Authorizer
}

func NewStorySaver(storyApiUrl string, authorizer Authorizer, logger outbound.LoggerPort) outbound.StorySaverPort {
	return &storySaver{
		logger:      logger,
		storyApiUrl: storyApiUrl,
		authorizer:  authorizer,
	}
}

func (s *storySaver) Save(ctx context.Context, params outbound.SaveStoryParams) error {
	token, err := s.authorizer.Authorize(ctx)
	if err != nil {
		s.logger.Error(err, "Failed to authorize")
		return err
	}
	storyRequest := StoryRequest{
		Input:  params.Input,
		ID:     params.ID,
		UserID: params.UserID,
	}
	payload, err := json.Marshal(storyRequest)
	if err != nil {
		s.logger.Error(err, "Failed to marshal the request")
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.storyApiUrl, bytes.NewReader(payload))
	if err != nil {
		s.logger.Error(err, "Failed to create the HTTP request")
	}

	req.Header.Add("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Error(err, "Failed to send the HTTP request")
		return err
	}

	defer func(closer io.ReadCloser) {
		err := closer.Close()
		if err != nil {
			s.logger.Error(err, "Failed to close the response body")
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		bodyPayload, err := io.ReadAll(resp.Body)
		if err != nil {
			s.logger.Error(err, "Failed to read the response body")
			return err
		}
		message := string(bodyPayload)
		s.logger.ErrorWithFields(err, "HTTP request returned non-OK status code", map[string]interface{}{
			"method": req.Method,
			"URL":    req.URL.String(),
			"status": resp.StatusCode,
			"body":   message,
		})
		return fmt.Errorf("save story request failed with status code: %d", resp.StatusCode)
	}

	return nil
}
