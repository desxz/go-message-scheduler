package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient_PostMessage(t *testing.T) {
	tests := []struct {
		name        string
		message     *WebhookRequest
		want        *WebhookResponse
		wantErr     error
		beforeSuite func() *httptest.Server
	}{
		{
			name:    "successful message post",
			message: &WebhookRequest{To: "+1234567890", Content: "Test message"},
			want:    &WebhookResponse{Message: "Accepted", MessageID: "webhook-message-id"},
			beforeSuite: func() *httptest.Server {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/a4d12c37-21b5-4470-92ad-357329f2b48c", r.URL.Path)
					assert.Equal(t, http.MethodPost, r.Method)

					var req WebhookRequest
					err := json.NewDecoder(r.Body).Decode(&req)
					assert.NoError(t, err)
					assert.Equal(t, "+1234567890", req.To)
					assert.Equal(t, "Test message", req.Content)

					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(WebhookResponse{Message: "Accepted", MessageID: "webhook-message-id"})
				}))

				return server
			},
		},
		{
			name:    "failed message post",
			message: &WebhookRequest{To: "+1234567890", Content: "Test message"},
			want:    nil,
			wantErr: assert.AnError,
			beforeSuite: func() *httptest.Server {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, "/a4d12c37-21b5-4470-92ad-357329f2b48c", r.URL.Path)
					assert.Equal(t, http.MethodPost, r.Method)

					var req WebhookRequest
					err := json.NewDecoder(r.Body).Decode(&req)
					assert.NoError(t, err)
					assert.Equal(t, "+1234567890", req.To)
					assert.Equal(t, "Test message", req.Content)

					w.WriteHeader(http.StatusInternalServerError)
				}))

				return server
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.beforeSuite()
			defer server.Close()

			client := NewWebhookClient(server.URL, nil)
			got, err := client.PostMessage(nil, tt.message)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
