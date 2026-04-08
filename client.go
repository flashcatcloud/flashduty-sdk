package flashduty

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	// maxResponseBodySize limits the response body size to prevent OOM attacks (10MB)
	maxResponseBodySize = 10 * 1024 * 1024

	// defaultMaxLogBodySize is the default maximum size for body content before truncation in logs
	defaultMaxLogBodySize = 2048
	// defaultLogPreviewSize is the default size of the preview shown for truncated log content
	defaultLogPreviewSize = 500
)

// Client represents a Flashduty API client
type Client struct {
	httpClient     *http.Client
	baseURL        *url.URL
	appKey         string
	userAgent      string
	logger         Logger
	requestHeaders http.Header         // static headers injected into every request
	requestHook    func(*http.Request) // callback invoked before every request
	optionErr      error               // collects errors from functional options
}

// makeRequest makes an HTTP request to the Flashduty API
func (c *Client) makeRequest(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var reqBody io.Reader
	var reqBodyBytes []byte

	if body != nil {
		var err error
		reqBodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("invalid request body: unable to serialize to JSON: %w", err)
		}
		reqBody = bytes.NewBuffer(reqBodyBytes)
	}

	// Parse path to handle query parameters correctly
	parsedPath, err := url.Parse(strings.TrimPrefix(path, "/"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse path: %w", err)
	}

	// Construct full URL with app_key query parameter
	fullURL := c.baseURL.ResolveReference(parsedPath)
	query := fullURL.Query()
	query.Set("app_key", c.appKey)
	fullURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, method, fullURL.String(), reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	// Apply static custom headers
	for key, values := range c.requestHeaders {
		for _, v := range values {
			req.Header.Set(key, v)
		}
	}
	// Apply request hook (e.g., trace context propagation)
	if c.requestHook != nil {
		c.requestHook(req)
	}

	logAttrs := traceLogAttrsFromRequest(req)
	logAttrs = append(logAttrs, "method", method, "url", sanitizeURL(fullURL), "body", truncateBody(string(reqBodyBytes)))
	c.logger.Info("duty request", logAttrs...)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to %s %s: %v", method, sanitizeURL(fullURL), sanitizeError(err))
	}

	return resp, nil
}

// SetUserAgent updates the User-Agent header for subsequent requests.
// This is useful when the user agent must change after client creation (e.g., per-session MCP client info).
func (c *Client) SetUserAgent(userAgent string) {
	c.userAgent = userAgent
}

// sanitizeURL removes sensitive query parameters from URL for safe logging
func sanitizeURL(u *url.URL) string {
	sanitized := *u
	q := sanitized.Query()
	if q.Has("app_key") {
		q.Set("app_key", "[REDACTED]")
		sanitized.RawQuery = q.Encode()
	}
	return sanitized.String()
}

// sanitizeError removes potential URL with sensitive data from error messages
func sanitizeError(err error) string {
	errStr := err.Error()
	idx := strings.Index(errStr, "app_key=")
	if idx == -1 {
		return errStr
	}

	endIdx := strings.IndexAny(errStr[idx:], "& ")
	if endIdx == -1 {
		return errStr[:idx] + "app_key=[REDACTED]"
	}
	return errStr[:idx] + "app_key=[REDACTED]" + errStr[idx+endIdx:]
}

func traceIDFromHeaders(headers http.Header) string {
	traceparent := headers.Get("traceparent")
	if traceparent == "" {
		return ""
	}

	parts := strings.Split(traceparent, "-")
	if len(parts) != 4 {
		return ""
	}

	traceID := parts[1]
	if len(traceID) != 32 {
		return ""
	}

	return traceID
}

func traceLogAttrsFromRequest(req *http.Request) []any {
	if req == nil {
		return nil
	}

	traceID := traceIDFromHeaders(req.Header)
	if traceID == "" {
		return nil
	}

	return []any{"trace_id", traceID}
}

// parseResponse parses the HTTP response into the given interface.
func parseResponse(logger Logger, resp *http.Response, v any) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	logAttrs := traceLogAttrsFromRequest(resp.Request)
	logAttrs = append(logAttrs,
		"status", resp.StatusCode,
		"body", truncateBody(string(body)),
	)

	requestID := resp.Header.Get("Flashcat-Request-Id")

	if resp.StatusCode >= 500 {
		logger.Error("duty response", logAttrs...)
		return fmt.Errorf("API server error (HTTP %d, request_id: %s): %s", resp.StatusCode, requestID, string(body))
	}

	if resp.StatusCode >= 400 {
		logger.Warn("duty response", logAttrs...)
		return fmt.Errorf("API client error (HTTP %d, request_id: %s): %s", resp.StatusCode, requestID, string(body))
	}

	logger.Info("duty response", logAttrs...)

	if v != nil {
		if err := json.Unmarshal(body, v); err != nil {
			return fmt.Errorf("invalid API response: failed to parse JSON (response size: %d bytes, request_id: %s): %w", len(body), requestID, err)
		}
	}

	return nil
}

// handleAPIError reads the response body and returns a detailed error message.
// This function should be called when resp.StatusCode != http.StatusOK.
func handleAPIError(logger Logger, resp *http.Response) error {
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodySize))
	if err != nil {
		return fmt.Errorf("API request failed (HTTP %d): unable to read response body: %v", resp.StatusCode, err)
	}

	logAttrs := traceLogAttrsFromRequest(resp.Request)
	logAttrs = append(logAttrs,
		"status", resp.StatusCode,
		"body", truncateBody(string(body)),
	)

	requestID := resp.Header.Get("Flashcat-Request-Id")

	if resp.StatusCode >= 500 {
		logger.Error("duty error", logAttrs...)
		return fmt.Errorf("API server error (HTTP %d, request_id: %s): %s", resp.StatusCode, requestID, string(body))
	}

	logger.Warn("duty error", logAttrs...)
	return fmt.Errorf("API client error (HTTP %d, request_id: %s): %s", resp.StatusCode, requestID, string(body))
}

// truncateBody truncates a string body if it exceeds the default max size for logging
func truncateBody(body string) string {
	bodyLen := len(body)
	if bodyLen <= defaultMaxLogBodySize {
		return body
	}

	previewSize := defaultLogPreviewSize
	if previewSize > bodyLen {
		previewSize = bodyLen
	}

	return fmt.Sprintf("[LARGE_BODY: truncated, size: %d bytes, preview: %s...]",
		bodyLen, body[:previewSize])
}
