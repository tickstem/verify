package verify_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tickstem/verify"
)

func serverFunc(fn func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(fn))
}

func TestVerify(t *testing.T) {
	t.Run("given valid email when server returns result then returns result", func(t *testing.T) {
		result := &verify.Result{
			ID:        "v1",
			Email:     "alice@example.com",
			Valid:     true,
			MXFound:   true,
			CreatedAt: time.Now(),
		}
		srv := serverFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/v1/verify", r.URL.Path)
			assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

			var body map[string]string
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			assert.Equal(t, "alice@example.com", body["email"])

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(result) //nolint:errcheck
		})
		defer srv.Close()

		client := verify.New("test-key", verify.WithBaseURL(srv.URL+"/v1"))
		got, err := client.Verify(context.Background(), "alice@example.com")

		require.NoError(t, err)
		assert.Equal(t, "v1", got.ID)
		assert.True(t, got.Valid)
	})

	t.Run("given 402 response when quota exceeded then returns quota error", func(t *testing.T) {
		srv := serverFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusPaymentRequired)
			json.NewEncoder(w).Encode(map[string]string{"error": "monthly verification quota exceeded"}) //nolint:errcheck
		})
		defer srv.Close()

		client := verify.New("test-key", verify.WithBaseURL(srv.URL+"/v1"))
		_, err := client.Verify(context.Background(), "alice@example.com")

		require.Error(t, err)
		assert.True(t, verify.IsQuotaExceeded(err))
	})

	t.Run("given 401 response when unauthorized then returns unauthorized error", func(t *testing.T) {
		srv := serverFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})
		defer srv.Close()

		client := verify.New("bad-key", verify.WithBaseURL(srv.URL+"/v1"))
		_, err := client.Verify(context.Background(), "alice@example.com")

		require.Error(t, err)
		assert.True(t, verify.IsUnauthorized(err))
	})
}

func TestListHistory(t *testing.T) {
	t.Run("given verifications exist when listing then returns history page", func(t *testing.T) {
		page := &verify.HistoryPage{
			Verifications: []*verify.Result{
				{ID: "v1", Email: "a@example.com", Valid: true},
				{ID: "v2", Email: "b@mailinator.com", Disposable: true},
			},
			Limit:  20,
			Offset: 0,
		}
		srv := serverFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/v1/verify/history", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(page) //nolint:errcheck
		})
		defer srv.Close()

		client := verify.New("test-key", verify.WithBaseURL(srv.URL+"/v1"))
		got, err := client.ListHistory(context.Background(), verify.ListHistoryParams{Limit: 20})

		require.NoError(t, err)
		require.Len(t, got.Verifications, 2)
		assert.Equal(t, "v1", got.Verifications[0].ID)
	})

	t.Run("given zero limit when listing then uses default limit", func(t *testing.T) {
		srv := serverFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "20", r.URL.Query().Get("limit"))
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(&verify.HistoryPage{}) //nolint:errcheck
		})
		defer srv.Close()

		client := verify.New("test-key", verify.WithBaseURL(srv.URL+"/v1"))
		_, err := client.ListHistory(context.Background(), verify.ListHistoryParams{})
		require.NoError(t, err)
	})
}

func TestAPIError(t *testing.T) {
	t.Run("given API error when formatting then includes status and message", func(t *testing.T) {
		err := &verify.APIError{StatusCode: 429, Message: "quota exceeded"}
		assert.Equal(t, "tickstem/verify: API error 429: quota exceeded", err.Error())
	})
}

func TestWithHTTPClient(t *testing.T) {
	t.Run("given custom http client when verifying then uses it", func(t *testing.T) {
		called := false
		transport := &recordingTransport{
			called: &called,
			inner:  http.DefaultTransport,
		}
		srv := serverFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(&verify.Result{}) //nolint:errcheck
		})
		defer srv.Close()

		client := verify.New("test-key",
			verify.WithBaseURL(srv.URL+"/v1"),
			verify.WithHTTPClient(&http.Client{Transport: transport}),
		)
		_, err := client.Verify(context.Background(), "alice@example.com")
		require.NoError(t, err)
		assert.True(t, called)
	})
}

type recordingTransport struct {
	called *bool
	inner  http.RoundTripper
}

func (rt *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	*rt.called = true
	return rt.inner.RoundTrip(req)
}
