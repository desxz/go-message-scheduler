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

func TestHandler_RetrieveSentMessages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	app := fiber.New()
	mockService := NewMockMessageService(ctrl)
	handler := NewMessageHandler(mockService)
	handler.RegisterRoutes(app)

	sentMessagesPath := "/sent-messages"
	sampleSentMessagesFilePath := "sample/sent_messages.json"
	sampleSentMessageContentRawByte, err := os.ReadFile(sampleSentMessagesFilePath)
	if err != nil {
		assert.Fail(t, "Failed to read sample sent messages file")
		return
	}

	var sampleSentMessages []Message
	if err := json.Unmarshal(sampleSentMessageContentRawByte, &sampleSentMessages); err != nil {
		assert.Fail(t, "Failed to unmarshal sample sent messages")
		return
	}

	sampleSentMessageContentByte, err := json.Marshal(sampleSentMessages)
	if err != nil {
		assert.Fail(t, "Failed to marshal sample sent messages")
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
			name:       "should return error when error is ErrDocumentNotFound",
			url:        sentMessagesPath,
			wantStatus: fiber.StatusNotFound,
			wantBody:   `Not Found`,
			beforeSuite: func() {
				mockService.EXPECT().RetriveSentMessages().Return(nil, ErrDocumentNotFound)
			},
		},
		{
			name:       "should return error when service fails",
			url:        sentMessagesPath,
			wantStatus: fiber.StatusInternalServerError,
			wantBody:   `Internal Server Error`,
			beforeSuite: func() {
				mockService.EXPECT().RetriveSentMessages().Return(nil, fiber.ErrInternalServerError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.beforeSuite()
			req := httptest.NewRequest(fiber.MethodGet, tt.url, nil)
			resp, err := app.Test(req, -1)
			defer resp.Body.Close()
			assert.NoError(t, err)
			assert.Equal(t, tt.wantStatus, resp.StatusCode)
			if tt.wantBody != "" {
				bodyBytes, _ := io.ReadAll(resp.Body)
				assert.Equal(t, tt.wantBody, string(bodyBytes))
			}
		})
	}
}
