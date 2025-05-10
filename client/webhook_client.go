package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type WebhookClientConfig struct {
	Timeout int    `json:"timeout"`
	Path    string `json:"path"`
}

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
	config     *WebhookClientConfig
}

func NewWebhookClient(baseURL string, httpClient *http.Client, config *WebhookClientConfig) *WebhookClient {
	return &WebhookClient{
		baseURL:    baseURL,
		httpClient: httpClient,
		config:     config,
	}
}

func (c *WebhookClient) PostMessage(ctx context.Context, message *WebhookRequest) (*WebhookResponse, error) {
	body, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+c.config.Path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("failed to post message, status code: %d", resp.StatusCode)
	}

	var webhookResponse WebhookResponse
	if err := json.NewDecoder(resp.Body).Decode(&webhookResponse); err != nil {
		return nil, err
	}

	return &webhookResponse, nil
}
