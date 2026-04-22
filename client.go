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
	var result Result
	if err := c.do(ctx, http.MethodPost, "/verify", map[string]string{"email": email}, &result); err != nil {
		return nil, err
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
	path := fmt.Sprintf("/verify/history?limit=%d&offset=%d",
		limitOrDefault(params.Limit),
		params.Offset,
	)
	var page HistoryPage
	if err := c.do(ctx, http.MethodGet, path, nil, &page); err != nil {
		return nil, err
	}
	return &page, nil
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	req, err := c.buildRequest(ctx, method, path, body)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("tickstem/verify: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("tickstem/verify: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return parseAPIError(resp.StatusCode, respBody)
	}

	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("tickstem/verify: decode response: %w", err)
		}
	}
	return nil
}

func (c *Client) buildRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	var bodyReader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("tickstem/verify: encode request body: %w", err)
		}
		bodyReader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("tickstem/verify: build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", "tickstem-go/"+Version)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	return req, nil
}

func limitOrDefault(limit int) int {
	if limit <= 0 || limit > 100 {
		return 20
	}
	return limit
}
