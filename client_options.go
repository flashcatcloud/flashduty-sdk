package flashduty

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Option configures the Client
type Option func(*Client)

// WithBaseURL sets the base URL for the API client.
// Invalid URLs are validated eagerly; NewClient returns an error if parsing fails.
func WithBaseURL(baseURL string) Option {
	parsedURL, err := url.Parse(baseURL)
	return func(c *Client) {
		if err == nil && parsedURL != nil {
			c.baseURL = parsedURL
		} else {
			c.optionErr = fmt.Errorf("invalid base URL %q: %w", baseURL, err)
		}
	}
}

// WithTimeout sets the HTTP client timeout
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

// WithUserAgent sets the User-Agent header
func WithUserAgent(ua string) Option {
	return func(c *Client) {
		c.userAgent = ua
	}
}

// WithHTTPClient sets a custom HTTP client. Nil values are ignored.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		if hc != nil {
			c.httpClient = hc
		}
	}
}

// WithLogger sets a custom logger for the SDK client. Nil values are ignored.
func WithLogger(l Logger) Option {
	return func(c *Client) {
		if l != nil {
			c.logger = l
		}
	}
}

// WithRequestHeaders sets static headers that will be included in every API request.
// These are applied after the SDK's own headers (Content-Type, Accept, User-Agent).
func WithRequestHeaders(headers http.Header) Option {
	return func(c *Client) {
		c.requestHeaders = headers
	}
}

// WithRequestHook sets a callback invoked on every outgoing HTTP request before it is sent.
// Use this to inject per-request headers such as W3C Trace Context (traceparent/tracestate).
// The hook receives the fully constructed *http.Request and may modify headers or other fields.
func WithRequestHook(hook func(*http.Request)) Option {
	return func(c *Client) {
		c.requestHook = hook
	}
}

// NewClient creates a new FlashDuty API client
func NewClient(appKey string, opts ...Option) (*Client, error) {
	if appKey == "" {
		return nil, fmt.Errorf("APP key is required")
	}

	baseURL, _ := url.Parse("https://api.flashcat.cloud")

	c := &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:   baseURL,
		appKey:    appKey,
		userAgent: "flashduty-go-sdk",
		logger:    defaultLogger,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.optionErr != nil {
		return nil, c.optionErr
	}

	return c, nil
}
