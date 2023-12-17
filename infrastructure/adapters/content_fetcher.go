package adapters

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
)

type ContentFetcher interface {
	FetchContent(req *http.Request) ([]byte, error)
}

type contentFetcher struct{}

func NewContentFetcher() ContentFetcher {
	return &contentFetcher{}
}

func (c contentFetcher) FetchContent(req *http.Request) ([]byte, error) {
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		log.Error().Err(err).Str("method", req.Method).Str("url", req.URL.String()).Msg("Failed to execute HTTP request")
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		bodyPayload, err := io.ReadAll(res.Body)
		message := string(bodyPayload)
		log.Error().Err(err).
			Str("method", req.Method).Str("url", req.URL.String()).
			Int("status_code", res.StatusCode).
			Str("message", message).
			Msg("HTTP request returned non-OK status code")
		return nil, fmt.Errorf("HTTP request returned non-OK status code: %d", res.StatusCode)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Error().Err(err).Str("method", req.Method).Str("url", req.URL.String()).Msg("Failed to close response body")
		}
	}(res.Body)

	payload, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error().Err(err).Str("method", req.Method).Str("url", req.URL.String()).Msg("Failed to read response body")
		return nil, err
	}

	return payload, nil
}
