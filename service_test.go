package main

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
)

func TestService_RetrieveSentMessages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockMessageRepository(ctrl)
	mockService := NewMessageServiceImpl(mockRepo)

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

	tests := []struct {
		name        string
		wantData    []Message
		wantErr     error
		beforeSuite func()
	}{
		{
			name:     "should return retrived sent messages",
			wantData: sampleSentMessages,
			wantErr:  nil,
			beforeSuite: func() {
				mockRepo.EXPECT().RetrieveSentMessages().Return(sampleSentMessages, nil)
			},
		},
		{
			name:     "should return error when messages are not found",
			wantData: nil,
			wantErr:  ErrDocumentNotFound,
			beforeSuite: func() {
				mockRepo.EXPECT().RetrieveSentMessages().Return([]Message{}, nil)
			},
		},
		{
			name:     "should return error when there is an internal server error",
			wantData: nil,
			wantErr:  ErrInternalServerError,
			beforeSuite: func() {
				mockRepo.EXPECT().RetrieveSentMessages().Return(nil, ErrInternalServerError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.beforeSuite()
			got, err := mockService.RetrieveSentMessages()
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.wantData, got)
		})
	}
}
