package main

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
)

func Test_RetriveSentMessages(t *testing.T) {
	ctrl := gomock.NewController(t)
	app := fiber.New()
	mockService := NewMockMessageService(ctrl)
	handler := NewMessageHandler(mockService)
	handler.RegisterRoutes(app)

	sentMessagesPath := "/sent-messages"
	sampleSentMessagesFilePath := "sample/sent_messages.json"
	sampleSentMessageContentByte, err := os.ReadFile(sampleSentMessagesFilePath)
	if err != nil {
		assert.Fail(t, "Failed to read sample sent messages file")
		return
	}

	var sampleSentMessages []Message
	if err := json.Unmarshal(sampleSentMessageContentByte, &sampleSentMessages); err != nil {
		assert.Fail(t, "Failed to unmarshal sample sent messages")
		return
	}

	tests := []struct {
		name        string
		url         string
		wantStatus  int
		wantBody    string
		beforeSuite func()
	}{
		{
			name:       "should return retrived sent messages with status 200",
			url:        sentMessagesPath,
			wantStatus: fiber.StatusOK,
			wantBody:   string(sampleSentMessageContentByte),
			beforeSuite: func() {
				mockService.EXPECT().RetriveSentMessages().Return(sampleSentMessages, nil)
			},
		},
		{
			name:       "should return error when messages are not found",
			url:        sentMessagesPath,
			wantStatus: fiber.StatusNotFound,
			wantBody:   `{"error":"No messages found"}`,
			beforeSuite: func() {
				mockService.EXPECT().RetriveSentMessages().Return(nil, nil)
			},
		},
		{
			name:       "should return error when service fails",
			url:        sentMessagesPath,
			wantStatus: fiber.StatusInternalServerError,
			wantBody:   `{"error":"Internal server error"}`,
			beforeSuite: func() {
				mockService.EXPECT().RetriveSentMessages().Return(nil, fiber.ErrInternalServerError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.beforeSuite()
			req := httptest.NewRequest("GET", tt.url, nil)
			resp, err := app.Test(req, -1)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantBody != "" {
				bodyBytes, _ := io.ReadAll(resp.Body)
				assert.Equal(t, tt.wantBody, string(bodyBytes))
			}
		})
	}
}
