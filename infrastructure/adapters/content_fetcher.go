package adapters

import (
	"fmt"
	"generate-script-lambda/application/ports/outbound"
	"io"
	"net/http"
)

type ContentFetcher interface {
	FetchContent(req *http.Request) ([]byte, error)
}

type contentFetcher struct {
	logger outbound.LoggerPort
}

func NewContentFetcher(logger outbound.LoggerPort) ContentFetcher {
	return &contentFetcher{
		logger: logger,
	}
}

func (c *contentFetcher) FetchContent(req *http.Request) ([]byte, error) {
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		c.logger.ErrorWithFields(err, "Failed to send the HTTP request", map[string]interface{}{
			"method": req.Method,
			"URL":    req.URL.String(),
		})
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		bodyPayload, err := io.ReadAll(res.Body)
		message := string(bodyPayload)
		c.logger.ErrorWithFields(err, "HTTP request returned non-OK status code", map[string]interface{}{
			"method":  req.Method,
			"URL":     req.URL.String(),
			"status":  res.StatusCode,
			"message": message,
		})
		return nil, fmt.Errorf("HTTP request returned non-OK status code: %d", res.StatusCode)
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			c.logger.ErrorWithFields(err, "Failed to close the response body", map[string]interface{}{
				"method": req.Method,
				"URL":    req.URL.String(),
			})
		}
	}(res.Body)

	payload, err := io.ReadAll(res.Body)
	if err != nil {
		c.logger.ErrorWithFields(err, "Failed to read the response body", map[string]interface{}{
			"method": req.Method,
			"URL":    req.URL.String(),
		})
		return nil, err
	}

	return payload, nil
}
