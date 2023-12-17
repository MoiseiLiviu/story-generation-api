package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"generate-script-lambda/application/ports/outbound"
	"github.com/rs/zerolog/log"
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
	authorizer  Authorizer
}

func NewStorySaver(storyApiUrl string, authorizer Authorizer) outbound.StorySaverPort {
	return &storySaver{
		storyApiUrl: storyApiUrl,
		authorizer:  authorizer,
	}
}

func (s *storySaver) Save(ctx context.Context, params outbound.SaveStoryParams) error {
	token, err := s.authorizer.Authorize(ctx)
	if err != nil {
		log.Err(err).Msg("Failed to authorize")
		return err
	}
	storyRequest := StoryRequest{
		Input:  params.Input,
		ID:     params.ID,
		UserID: params.UserID,
	}
	payload, err := json.Marshal(storyRequest)
	if err != nil {
		log.Err(err).Msg("Failed to marshal story request")
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.storyApiUrl, bytes.NewReader(payload))
	if err != nil {
		log.Err(err).Msg("Failed to create request")
	}

	req.Header.Add("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Err(err).Msg("Failed to send request")
		return err
	}

	defer func(closer io.ReadCloser) {
		err := closer.Close()
		if err != nil {
			log.Err(err).Msg("Failed to close response body")
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusCreated {
		log.Error().Msgf("Received unexpected status code %d", resp.StatusCode)
		return err
	}

	return nil
}
