// Package verify provides a Go client for the Tickstem email verification API.
//
// Usage:
//
//	client := verify.New(os.Getenv("TICKSTEM_API_KEY"))
//
//	result, err := client.Verify(ctx, "user@example.com")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result.Valid, result.Disposable, result.RoleBased)
package verify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://api.tickstem.dev/v1"

// Client is a Tickstem email verification API client. Create one with New.
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

type Option func(*Client)

// WithBaseURL overrides the API base URL. Useful for local testing.
func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) {
		c.httpClient = hc
	}
}

func New(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:  apiKey,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Verify checks a single email address and returns the verification result.
func (c *Client) Verify(ctx context.Context, email string) (*Result, error) {
	body, err := json.Marshal(map[string]string{"email": email})
	if err != nil {
		return nil, fmt.Errorf("tickstem/verify: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/verify", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("tickstem/verify: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tickstem/verify: do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tickstem/verify: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, parseAPIError(resp.StatusCode, respBody)
	}

	var result Result
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("tickstem/verify: decode response: %w", err)
	}
	return &result, nil
}

// ListHistoryParams configures the ListHistory request.
type ListHistoryParams struct {
	Limit  int
	Offset int
}

// ListHistory returns past verification results for the authenticated account.
func (c *Client) ListHistory(ctx context.Context, params ListHistoryParams) (*HistoryPage, error) {
	url := fmt.Sprintf("%s/verify/history?limit=%d&offset=%d",
		c.baseURL,
		limitOrDefault(params.Limit),
		params.Offset,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("tickstem/verify: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tickstem/verify: do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tickstem/verify: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, parseAPIError(resp.StatusCode, respBody)
	}

	var page HistoryPage
	if err := json.Unmarshal(respBody, &page); err != nil {
		return nil, fmt.Errorf("tickstem/verify: decode response: %w", err)
	}
	return &page, nil
}

func limitOrDefault(limit int) int {
	if limit <= 0 || limit > 100 {
		return 20
	}
	return limit
}
