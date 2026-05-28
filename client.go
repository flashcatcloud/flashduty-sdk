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
	logAttrs = append(logAttrs, "method", method, "url", sanitizeURL(fullURL), "body", truncateBody(sanitizeBody(string(reqBodyBytes))))
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

// sensitiveBodyKeys enumerates normalized JSON keys whose values must be
// redacted before bodies are logged. The set intentionally covers common
// credential aliases seen in API payloads and echoed error responses.
var sensitiveBodyKeys = map[string]struct{}{
	"apikey":        {},
	"xapikey":       {},
	"accesskey":     {},
	"password":      {},
	"passwd":        {},
	"pwd":           {},
	"token":         {},
	"accesstoken":   {},
	"refreshtoken":  {},
	"idtoken":       {},
	"sessiontoken":  {},
	"authtoken":     {},
	"oauthtoken":    {},
	"bearertoken":   {},
	"authorization": {},
	"auth":          {},
	"secret":        {},
	"clientsecret":  {},
	"secretkey":     {},
	"privatekey":    {},
	"signingkey":    {},
	"credential":    {},
	"credentials":   {},
}

// redactChildrenKeys enumerates normalized JSON keys whose nested values are
// always redacted regardless of inner key name. These containers (env, headers)
// hold user-chosen keys that frequently carry credentials, so the allow-list
// approach in sensitiveBodyKeys cannot catch them.
var redactChildrenKeys = map[string]struct{}{
	"env":     {},
	"headers": {},
}

// sanitizeBody redacts values of well-known sensitive JSON keys so that
// secrets do not appear in request/response logs. It is best-effort: empty or
// non-JSON bodies pass through unchanged. Callers must still use sanitizeURL
// for URL-borne secrets.
func sanitizeBody(body string) string {
	if body == "" {
		return body
	}
	var v any
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		return body
	}

	sanitized, redacted := sanitizeJSONValue(v)
	if !redacted {
		return body
	}
	out, err := json.Marshal(sanitized)
	if err != nil {
		return body
	}
	return string(out)
}

func sanitizeJSONValue(v any) (any, bool) {
	switch value := v.(type) {
	case map[string]any:
		sanitized := make(map[string]any, len(value))
		redacted := false
		for key, item := range value {
			if isSensitiveBodyKey(key) {
				sanitized[key] = "[REDACTED]"
				redacted = true
				continue
			}

			// When the value is a container whose user-chosen keys may hold
			// credentials (env, headers), redact every leaf inside without
			// inspecting inner names — the allow-list cannot anticipate
			// arbitrary user-supplied key names like OPENAI_API_KEY.
			if shouldRedactChildren(key) {
				sanitized[key] = redactAllLeaves(item)
				redacted = true
				continue
			}

			sanitizedItem, itemRedacted := sanitizeJSONValue(item)
			sanitized[key] = sanitizedItem
			redacted = redacted || itemRedacted
		}
		return sanitized, redacted
	case []any:
		sanitized := make([]any, len(value))
		redacted := false
		for i, item := range value {
			sanitizedItem, itemRedacted := sanitizeJSONValue(item)
			sanitized[i] = sanitizedItem
			redacted = redacted || itemRedacted
		}
		return sanitized, redacted
	default:
		return v, false
	}
}

// redactAllLeaves walks v and replaces every non-container leaf with
// "[REDACTED]", preserving the surrounding map/slice shape so the log entry
// still hints at the payload structure.
func redactAllLeaves(v any) any {
	switch value := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(value))
		for key, item := range value {
			out[key] = redactAllLeaves(item)
		}
		return out
	case []any:
		out := make([]any, len(value))
		for i, item := range value {
			out[i] = redactAllLeaves(item)
		}
		return out
	default:
		return "[REDACTED]"
	}
}

func isSensitiveBodyKey(key string) bool {
	_, ok := sensitiveBodyKeys[normalizeSensitiveBodyKey(key)]
	return ok
}

func shouldRedactChildren(key string) bool {
	_, ok := redactChildrenKeys[normalizeSensitiveBodyKey(key)]
	return ok
}

func normalizeSensitiveBodyKey(key string) string {
	var b strings.Builder
	b.Grow(len(key))
	for _, r := range strings.ToLower(strings.TrimSpace(key)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
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
	sanitizedBody := sanitizeBody(string(body))

	logAttrs := traceLogAttrsFromRequest(resp.Request)
	logAttrs = append(logAttrs,
		"status", resp.StatusCode,
		"body", truncateBody(sanitizedBody),
	)

	requestID := resp.Header.Get("Flashcat-Request-Id")

	if resp.StatusCode >= 500 {
		logger.Error("duty response", logAttrs...)
		return fmt.Errorf("API server error (HTTP %d, request_id: %s): %s", resp.StatusCode, requestID, sanitizedBody)
	}

	if resp.StatusCode >= 400 {
		logger.Warn("duty response", logAttrs...)
		return fmt.Errorf("API client error (HTTP %d, request_id: %s): %s", resp.StatusCode, requestID, sanitizedBody)
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
	sanitizedBody := sanitizeBody(string(body))

	logAttrs := traceLogAttrsFromRequest(resp.Request)
	logAttrs = append(logAttrs,
		"status", resp.StatusCode,
		"body", truncateBody(sanitizedBody),
	)

	requestID := resp.Header.Get("Flashcat-Request-Id")

	if resp.StatusCode >= 500 {
		logger.Error("duty error", logAttrs...)
		return fmt.Errorf("API server error (HTTP %d, request_id: %s): %s", resp.StatusCode, requestID, sanitizedBody)
	}

	logger.Warn("duty error", logAttrs...)
	return fmt.Errorf("API client error (HTTP %d, request_id: %s): %s", resp.StatusCode, requestID, sanitizedBody)
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
