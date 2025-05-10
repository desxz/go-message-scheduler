package client

import (
	"context"
	"net/http"
)

type WebhookResponse struct {
	Message   string `json:"message"`
	MessageID string `json:"messageId"`
}

type WebhookRequest struct {
	To      string `json:"to"`
	Content string `json:"content"`
}

type WebhookClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewWebhookClient(baseURL string, httpClient *http.Client) *WebhookClient {
	return &WebhookClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

func (c *WebhookClient) PostMessage(ctx context.Context, message *WebhookRequest) (*WebhookResponse, error) {
	return nil, nil
}
