package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	mongoImage     = "mongo:7.0.4"
	testDB         = "test"
	testCollection = "messages"
)

func prepareTestMongoStore() (*mongo.Client, func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	mongodbContainer, err := mongodb.Run(ctx, mongoImage)
	if err != nil {
		return nil, nil, err
	}

	cleanFunc := func() {
		if cerr := mongodbContainer.Terminate(ctx); cerr != nil {
			return
		}
	}

	uri, err := mongodbContainer.ConnectionString(ctx)
	if err != nil {
		cleanFunc()
		return nil, nil, err
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		cleanFunc()
		return nil, nil, err
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		cleanFunc()
		return nil, nil, err
	}

	return client, cleanFunc, nil
}

func TestRepository_RetrieveSendMessage(t *testing.T) {
	sampleMixedMessagesFilePath := "sample/mixed_status_messages.json"
	sampleMixedMessageContentRawByte, err := os.ReadFile(sampleMixedMessagesFilePath)
	if err != nil {
		assert.Fail(t, "Failed to read sample mixed messages file")
		return
	}

	sampleMixedMessages := make([]Message, 5)
	if err := json.Unmarshal(sampleMixedMessageContentRawByte, &sampleMixedMessages); err != nil {
		assert.Fail(t, "Failed to unmarshal sample mixed messages, got error: %v", err)
		return
	}

	tests := []struct {
		name        string
		wantData    []Message
		wantErr     error
		beforeSuite func() (*mongo.Client, func())
	}{
		{
			name:     "should return retrieved sent messages",
			wantData: []Message{sampleMixedMessages[4], sampleMixedMessages[0]}, // from newest to oldest sent_at
			wantErr:  nil,
			beforeSuite: func() (*mongo.Client, func()) {
				client, cleanFunc, err := prepareTestMongoStore()
				assert.NoError(t, err)

				var bsonMessages []interface{}
				for _, message := range sampleMixedMessages {
					bsonMessages = append(bsonMessages, message)
				}

				messageCollection := client.Database(testDB).Collection(testCollection)
				_, err = messageCollection.InsertMany(context.Background(), bsonMessages)
				assert.NoError(t, err)

				return client, cleanFunc
			},
		},
		{
			name:     "should return error when messages are not found",
			wantData: nil,
			wantErr:  nil,
			beforeSuite: func() (*mongo.Client, func()) {
				client, cleanFunc, err := prepareTestMongoStore()
				assert.NoError(t, err)

				return client, cleanFunc
			},
		},
		{
			name:     "should return error when decoding fails",
			wantData: nil,
			wantErr:  ErrDocumentDecodingFailed,
			beforeSuite: func() (*mongo.Client, func()) {
				client, cleanFunc, err := prepareTestMongoStore()
				assert.NoError(t, err)

				oid, _ := primitive.ObjectIDFromHex("645f6e1a8b45c23d9812ab20")

				invalidDoc := bson.M{
					"_id":                         oid,
					"webhook_response_message_id": "msg_123456789abcdef",
					"content":                     "Hello! This is a test message.",
					"recipient_phone_number":      "+15551234567",
					"status":                      StatusSent,
					"created_at":                  "not-a-valid-time-format",
					"sent_at":                     "2025-05-09T14:30:15Z",
				}

				messageCollection := client.Database(testDB).Collection(testCollection)

				var docs []interface{}
				for _, msg := range sampleMixedMessages {
					docs = append(docs, msg)
				}

				docs = append(docs, invalidDoc)

				_, err = messageCollection.InsertMany(context.Background(), docs)
				assert.NoError(t, err)

				return client, cleanFunc
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, cleanFunc := tt.beforeSuite()

			defer client.Disconnect(context.Background())
			defer cleanFunc()

			messageRepository := NewMessageRepositoryImpl(client.Database(testDB).Collection(testCollection))
			gotData, err := messageRepository.RetrieveSentMessages()
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.wantData, gotData)
		})
	}
}

func TestRepository_FetchAndMarkProcessing(t *testing.T) {
	sampleMixedMessagesFilePath := "sample/mixed_status_messages.json"
	sampleMixedMessageContentRawByte, err := os.ReadFile(sampleMixedMessagesFilePath)
	if err != nil {
		assert.Fail(t, "Failed to read sample mixed messages file")
		return
	}

	var sampleMixedMessages []Message
	if err := json.Unmarshal(sampleMixedMessageContentRawByte, &sampleMixedMessages); err != nil {
		assert.Fail(t, "Failed to unmarshal sample mixed messages, got error: %v", err)
		return
	}

	tests := []struct {
		name          string
		wantMessageID primitive.ObjectID
		wantErr       bool
		wantStatus    string
		beforeSuite   func() (*mongo.Client, func())
	}{
		{
			name:          "should return message and mark it as processing",
			wantMessageID: sampleMixedMessages[3].ID,
			wantErr:       false,
			wantStatus:    StatusProcessing,
			beforeSuite: func() (*mongo.Client, func()) {
				client, cleanFunc, err := prepareTestMongoStore()
				assert.NoError(t, err)

				var bsonMessages []interface{}
				for _, message := range sampleMixedMessages {
					bsonMessages = append(bsonMessages, message)
				}

				messageCollection := client.Database(testDB).Collection(testCollection)
				_, err = messageCollection.InsertMany(context.Background(), bsonMessages)
				assert.NoError(t, err)

				return client, cleanFunc
			},
		},
		{
			name:          "should return error when no documents are found",
			wantMessageID: primitive.NilObjectID,
			wantErr:       true,
			wantStatus:    "",
			beforeSuite: func() (*mongo.Client, func()) {
				client, cleanFunc, err := prepareTestMongoStore()
				assert.NoError(t, err)

				return client, cleanFunc
			},
		},
		{
			name:          "should return error when decoding fails",
			wantMessageID: primitive.NilObjectID,
			wantErr:       true,
			wantStatus:    "",
			beforeSuite: func() (*mongo.Client, func()) {
				client, cleanFunc, err := prepareTestMongoStore()
				assert.NoError(t, err)

				oid, _ := primitive.ObjectIDFromHex("645f6e1a8b45c23d9812ab20")

				invalidDoc := bson.M{
					"_id":                         oid,
					"webhook_response_message_id": "msg_123456789abcdef",
					"content":                     "Hello! This is a test message.",
					"recipient_phone_number":      "+15551234567",
					"status":                      StatusUnsent,
					"created_at":                  "not-a-valid-time-format",
					"sent_at":                     "2025-05-09T14:30:15Z",
				}

				messageCollection := client.Database(testDB).Collection(testCollection)

				_, err = messageCollection.InsertOne(context.Background(), invalidDoc)
				assert.NoError(t, err)

				return client, cleanFunc
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, cleanFunc := tt.beforeSuite()

			defer client.Disconnect(context.Background())
			defer cleanFunc()

			messageRepository := NewMessageRepositoryImpl(client.Database(testDB).Collection(testCollection))
			gotData, err := messageRepository.FetchAndMarkProcessing(context.Background())
			assert.Equal(t, tt.wantErr, err != nil)
			if !tt.wantErr {
				assert.Equal(t, tt.wantMessageID, gotData.ID)
				assert.Equal(t, tt.wantStatus, gotData.Status)
			}
		})
	}
}

func TestRepository_MarkAsSent(t *testing.T) {
	sampleMixedMessagesFilePath := "sample/mixed_status_messages.json"
	sampleMixedMessageContentRawByte, err := os.ReadFile(sampleMixedMessagesFilePath)
	if err != nil {
		assert.Fail(t, "Failed to read sample mixed messages file")
		return
	}

	var sampleMixedMessages []Message
	if err := json.Unmarshal(sampleMixedMessageContentRawByte, &sampleMixedMessages); err != nil {
		assert.Fail(t, "Failed to unmarshal sample mixed messages, got error: %v", err)
		return
	}

	brokenDataID, _ := primitive.ObjectIDFromHex("645f6e1a8b45c23d9812ab20")

	tests := []struct {
		name                 string
		wantErr              bool
		wantWebhookMessageID string
		wantStatus           string
		markID               primitive.ObjectID
		beforeSuite          func() (*mongo.Client, func())
	}{
		{
			name:                 "should mark message as sent",
			wantErr:              false,
			wantWebhookMessageID: "webhook-message-id-1234567890",
			wantStatus:           StatusSent,
			markID:               sampleMixedMessages[0].ID,
			beforeSuite: func() (*mongo.Client, func()) {
				client, cleanFunc, err := prepareTestMongoStore()
				assert.NoError(t, err)

				var bsonMessages []interface{}
				for _, message := range sampleMixedMessages {
					bsonMessages = append(bsonMessages, message)
				}

				messageCollection := client.Database(testDB).Collection(testCollection)
				_, err = messageCollection.InsertMany(context.Background(), bsonMessages)
				assert.NoError(t, err)

				return client, cleanFunc
			},
		},
		{
			name:                 "should return error when message ID is invalid",
			wantErr:              true,
			wantWebhookMessageID: "",
			wantStatus:           "",
			markID:               primitive.NewObjectID(),
			beforeSuite: func() (*mongo.Client, func()) {
				client, cleanFunc, err := prepareTestMongoStore()
				assert.NoError(t, err)

				var bsonMessages []interface{}
				for _, message := range sampleMixedMessages {
					bsonMessages = append(bsonMessages, message)
				}

				messageCollection := client.Database(testDB).Collection(testCollection)
				_, err = messageCollection.InsertMany(context.Background(), bsonMessages)
				assert.NoError(t, err)

				return client, cleanFunc
			},
		},
		{
			name:                 "should return error when decoding fails",
			wantErr:              true,
			wantWebhookMessageID: "",
			wantStatus:           "",
			markID:               brokenDataID,
			beforeSuite: func() (*mongo.Client, func()) {
				client, cleanFunc, err := prepareTestMongoStore()
				assert.NoError(t, err)

				invalidDoc := bson.M{
					"_id":                         brokenDataID,
					"webhook_response_message_id": "msg_123456789abcdef",
					"content":                     "Hello! This is a test message.",
					"recipient_phone_number":      "+15551234567",
					"status":                      StatusUnsent,
					"created_at":                  "not-a-valid-time-format",
					"sent_at":                     "2025-05-09T14:30:15Z",
				}

				messageCollection := client.Database(testDB).Collection(testCollection)

				_, err = messageCollection.InsertOne(context.Background(), invalidDoc)
				assert.NoError(t, err)

				return client, cleanFunc
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, cleanFunc := tt.beforeSuite()

			defer client.Disconnect(context.Background())
			defer cleanFunc()

			messageRepository := NewMessageRepositoryImpl(client.Database(testDB).Collection(testCollection))
			err := messageRepository.MarkAsSent(context.Background(), tt.markID, tt.wantWebhookMessageID)
			assert.Equal(t, tt.wantErr, err != nil)

			if !tt.wantErr {
				var updatedMessage Message
				err = client.Database(testDB).Collection(testCollection).FindOne(context.Background(), bson.M{"_id": tt.markID}).Decode(&updatedMessage)
				assert.NoError(t, err)
				assert.Equal(t, StatusSent, updatedMessage.Status)
				assert.Equal(t, tt.wantWebhookMessageID, updatedMessage.WebhookResponseMessageID)
			}
		})
	}
}

func TestRepository_MarkAsFailed(t *testing.T) {
	sampleMixedMessagesFilePath := "sample/mixed_status_messages.json"
	sampleMixedMessageContentRawByte, err := os.ReadFile(sampleMixedMessagesFilePath)
	if err != nil {
		assert.Fail(t, "Failed to read sample mixed messages file")
		return
	}

	var sampleMixedMessages []Message
	if err := json.Unmarshal(sampleMixedMessageContentRawByte, &sampleMixedMessages); err != nil {
		assert.Fail(t, "Failed to unmarshal sample mixed messages, got error: %v", err)
		return
	}

	brokenDataID, _ := primitive.ObjectIDFromHex("645f6e1a8b45c23d9812ab20")

	tests := []struct {
		name        string
		wantErr     bool
		wantStatus  string
		markID      primitive.ObjectID
		beforeSuite func() (*mongo.Client, func())
	}{
		{
			name:       "should mark message as failed",
			wantErr:    false,
			wantStatus: StatusFailed,
			markID:     sampleMixedMessages[0].ID,
			beforeSuite: func() (*mongo.Client, func()) {
				client, cleanFunc, err := prepareTestMongoStore()
				assert.NoError(t, err)

				var bsonMessages []interface{}
				for _, message := range sampleMixedMessages {
					bsonMessages = append(bsonMessages, message)
				}

				messageCollection := client.Database(testDB).Collection(testCollection)
				_, err = messageCollection.InsertMany(context.Background(), bsonMessages)
				assert.NoError(t, err)

				return client, cleanFunc
			},
		},
		{
			name:       "should return error when message ID is invalid",
			wantErr:    true,
			wantStatus: "",
			markID:     primitive.NewObjectID(),
			beforeSuite: func() (*mongo.Client, func()) {
				client, cleanFunc, err := prepareTestMongoStore()
				assert.NoError(t, err)

				var bsonMessages []interface{}
				for _, message := range sampleMixedMessages {
					bsonMessages = append(bsonMessages, message)
				}

				messageCollection := client.Database(testDB).Collection(testCollection)
				_, err = messageCollection.InsertMany(context.Background(), bsonMessages)
				assert.NoError(t, err)

				return client, cleanFunc
			},
		},
		{
			name:       "should return error when decoding fails",
			wantErr:    true,
			wantStatus: "",
			markID:     brokenDataID,
			beforeSuite: func() (*mongo.Client, func()) {
				client, cleanFunc, err := prepareTestMongoStore()
				assert.NoError(t, err)

				invalidDoc := bson.M{
					"_id":                         brokenDataID,
					"webhook_response_message_id": "msg_123456789abcdef",
					"content":                     "Hello! This is a test message.",
					"recipient_phone_number":      "+15551234567",
					"status":                      StatusUnsent,
					"created_at":                  "not-a-valid-time-format",
					"sent_at":                     "2025-05-09T14:30:15Z",
				}

				messageCollection := client.Database(testDB).Collection(testCollection)

				_, err = messageCollection.InsertOne(context.Background(), invalidDoc)
				assert.NoError(t, err)

				return client, cleanFunc
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, cleanFunc := tt.beforeSuite()

			defer client.Disconnect(context.Background())
			defer cleanFunc()

			messageRepository := NewMessageRepositoryImpl(client.Database(testDB).Collection(testCollection))
			err := messageRepository.MarkAsFailed(context.Background(), tt.markID, "failed")
			assert.Equal(t, tt.wantErr, err != nil)

			if !tt.wantErr {
				var updatedMessage Message
				err = client.Database(testDB).Collection(testCollection).FindOne(context.Background(), bson.M{"_id": tt.markID}).Decode(&updatedMessage)
				assert.NoError(t, err)
				assert.Equal(t, StatusFailed, updatedMessage.Status)
			}
		})
	}
}
