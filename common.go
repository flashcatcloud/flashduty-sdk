package flashduty

import (
	"context"
	"fmt"
	"net/http"
)

type dataEnvelope[T any] struct {
	Error *DutyError `json:"error,omitempty"`
	Data  T          `json:"data,omitempty"`
}

type optionalDataEnvelope[T any] struct {
	Error *DutyError `json:"error,omitempty"`
	Data  *T         `json:"data,omitempty"`
}

type emptyEnvelope struct {
	Error *DutyError `json:"error,omitempty"`
}

func getData[T any](c *Client, ctx context.Context, path string, errPrefix string) (*T, error) {
	resp, err := c.makeRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errPrefix, err)
	}
	defer func() { _ = resp.Body.Close() }()

	return parseData[T](c, resp)
}

func postData[T any](c *Client, ctx context.Context, path string, body any, errPrefix string) (*T, error) {
	resp, err := c.makeRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errPrefix, err)
	}
	defer func() { _ = resp.Body.Close() }()

	return parseData[T](c, resp)
}

func getOptionalData[T any](c *Client, ctx context.Context, path string, errPrefix string) (*T, error) {
	resp, err := c.makeRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errPrefix, err)
	}
	defer func() { _ = resp.Body.Close() }()

	return parseOptionalData[T](c, resp)
}

func postOptionalData[T any](c *Client, ctx context.Context, path string, body any, errPrefix string) (*T, error) {
	resp, err := c.makeRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errPrefix, err)
	}
	defer func() { _ = resp.Body.Close() }()

	return parseOptionalData[T](c, resp)
}

func postEmpty(c *Client, ctx context.Context, path string, body any, errPrefix string) error {
	resp, err := c.makeRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return fmt.Errorf("%s: %w", errPrefix, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return handleAPIError(c.logger, resp)
	}

	var result emptyEnvelope
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return err
	}
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func parseData[T any](c *Client, resp *http.Response) (*T, error) {
	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result dataEnvelope[T]
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return &result.Data, nil
}

func parseOptionalData[T any](c *Client, resp *http.Response) (*T, error) {
	if resp.StatusCode != http.StatusOK {
		return nil, handleAPIError(c.logger, resp)
	}

	var result optionalDataEnvelope[T]
	if err := parseResponse(c.logger, resp, &result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return result.Data, nil
}
