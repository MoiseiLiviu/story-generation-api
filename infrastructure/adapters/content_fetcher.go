package adapters

import (
	"fmt"
	"generate-script-lambda/application/ports/outbound"
	"io"
	"net/http"
	"time"
)

type ContentFetcher interface {
	FetchContent(req *http.Request) (*http.Response, error)
}

type contentFetcher struct {
	logger outbound.LoggerPort
}

func NewContentFetcher(logger outbound.LoggerPort) ContentFetcher {
	return &contentFetcher{
		logger: logger,
	}
}

func (c *contentFetcher) FetchContent(req *http.Request) (*http.Response, error) {
	const maxRetries int = 3
	const retryDelay = 5 * time.Second

	client := &http.Client{}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		res, err := client.Do(req)
		if err != nil {
			c.logger.ErrorWithFields(err, "Failed to send the HTTP request", map[string]interface{}{
				"method": req.Method,
				"URL":    req.URL.String(),
			})
			return nil, err
		}

		if res.StatusCode == http.StatusTooManyRequests ||
			res.StatusCode == http.StatusServiceUnavailable ||
			res.StatusCode == http.StatusGatewayTimeout ||
			res.StatusCode == http.StatusBadGateway ||
			res.StatusCode == http.StatusRequestTimeout {
			if attempt == maxRetries {
				return nil, fmt.Errorf("request failed after %d retries", maxRetries)
			}
			c.logger.WarnWithFields("Request failed, retrying...", map[string]interface{}{
				"method":  req.Method,
				"URL":     req.URL.String(),
				"attempt": attempt + 1,
				"status":  res.StatusCode,
			})

			time.Sleep(retryDelay)
			continue
		}

		if res.StatusCode != http.StatusOK {
			messagePayload, err := c.readResponseBodyPayload(res)
			if err != nil {
				return nil, err
			}
			message := string(messagePayload)
			c.logger.ErrorWithFields(err, "HTTP request returned non-OK status code", map[string]interface{}{
				"method":  req.Method,
				"URL":     req.URL.String(),
				"status":  res.StatusCode,
				"message": message,
			})
			return nil, fmt.Errorf("HTTP request returned non-OK status code: %d", res.StatusCode)
		}

		return res, nil
	}

	return nil, fmt.Errorf("failed to fetch content after retries")
}

func (c *contentFetcher) readResponseBodyPayload(res *http.Response) ([]byte, error) {
	payload, err := io.ReadAll(res.Body)
	if err != nil {
		c.logger.ErrorWithFields(err, "Failed to read the response body", map[string]interface{}{
			"method": res.Request.Method,
			"URL":    res.Request.URL.String(),
		})
		return nil, err
	}
	err = res.Body.Close()
	if err != nil {
		c.logger.ErrorWithFields(err, "Failed to close the response body", map[string]interface{}{
			"method": res.Request.Method,
			"URL":    res.Request.URL.String(),
		})
		return nil, err
	}

	return payload, nil
}
