package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	gomock "go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestWorker_ProcessMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := NewMockWorkerMessageStore(ctrl)
	mockWebhookClient := NewMockWebhookClient(ctrl)

	tests := []struct {
		name        string
		messageID   string
		wantErr     bool
		wantProcess bool
		beforeSuite func()
	}{
		{
			name:        "successful message processing",
			messageID:   "1234567890abcdef12345678",
			wantErr:     false,
			wantProcess: true,
			beforeSuite: func() {
				message := &Message{
					ID:                       primitive.NewObjectID(),
					WebhookResponseMessageID: "",
					Content:                  "Test message",
					RecipientPhoneNumber:     "+1234567890",
					Status:                   "processing",
					CreatedAt:                time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
					SentAt:                   time.Date(2023, 10, 1, 0, 0, 10, 0, time.UTC),
				}

				mockRepo.EXPECT().FetchAndMarkProcessing(gomock.Any()).Return(message, nil)

				mockWebhookClient.EXPECT().PostMessage(gomock.Any(), &WebhookRequest{
					To:      message.RecipientPhoneNumber,
					Content: message.Content,
				}).Return(&WebhookResponse{
					Message:   "Accepted",
					MessageID: "webhook-message-id",
				}, nil)

				mockRepo.EXPECT().MarkAsSent(gomock.Any(), message.ID, "webhook-message-id").Return(nil)
			},
		},
		{
			name:        "failed message processing",
			messageID:   "1234567890abcdef12345678",
			wantErr:     true,
			wantProcess: true,
			beforeSuite: func() {
				message := &Message{
					ID:                       primitive.NewObjectID(),
					WebhookResponseMessageID: "",
					Content:                  "Test message",
					RecipientPhoneNumber:     "+1234567890",
					Status:                   "processing",
					CreatedAt:                time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
					SentAt:                   time.Date(2023, 10, 1, 0, 0, 10, 0, time.UTC),
				}

				mockRepo.EXPECT().FetchAndMarkProcessing(gomock.Any()).Return(message, nil)

				mockWebhookClient.EXPECT().PostMessage(gomock.Any(), &WebhookRequest{
					To:      message.RecipientPhoneNumber,
					Content: message.Content,
				}).Return(nil, assert.AnError)

				mockRepo.EXPECT().MarkAsFailed(gomock.Any(), message.ID).Return(nil)
			},
		},
		{
			name:        "no message to process",
			messageID:   "1234567890abcdef12345678",
			wantErr:     false,
			wantProcess: false,
			beforeSuite: func() {
				mockRepo.EXPECT().FetchAndMarkProcessing(gomock.Any()).Return(nil, mongo.ErrNoDocuments)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.beforeSuite()
			worker := NewWorkerInstance(tt.messageID, mockRepo, mockWebhookClient, zap.NewNop())
			process, err := worker.ProcessMessage(context.Background())
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.wantProcess, process)
		})
	}
}
