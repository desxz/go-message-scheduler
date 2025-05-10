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

				// Create an invalid document that will cause cursor.All to fail
				invalidDoc := bson.M{
					"_id":                         oid,
					"webhook_response_message_id": "msg_123456789abcdef",
					"content":                     "Hello! This is a test message.",
					"recipient_phone_number":      "+15551234567",
					"status":                      StatusSent,
					"created_at":                  "not-a-valid-time-format",
					"sent_at":                     "2025-05-09T14:30:15Z",
				}

				// Insert multiple documents, one valid and one invalid to ensure the Find works but All fails
				messageCollection := client.Database(testDB).Collection(testCollection)

				// Insert valid messages first
				var docs []interface{}
				for _, msg := range sampleMixedMessages {
					docs = append(docs, msg)
				}

				// Insert the invalid document
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
