package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
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

	var sampleMixedMessages []Message
	if err := json.Unmarshal(sampleMixedMessageContentRawByte, &sampleMixedMessages); err != nil {
		assert.Fail(t, "Failed to unmarshal sample mixed messages")
		return
	}

	tests := []struct {
		name        string
		wantData    []Message
		wantErr     error
		beforeSuite func() (*mongo.Client, func())
	}{
		{
			name:     "should return retrived sent messages",
			wantData: []Message{sampleMixedMessages[0], sampleMixedMessages[4]},
			wantErr:  nil,
			beforeSuite: func() (*mongo.Client, func()) {
				client, cleanFunc, err := prepareTestMongoStore()
				assert.NoError(t, err)

				defer client.Disconnect(context.Background())

				bsonMessages := make([]interface{}, len(sampleMixedMessages))
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
			wantErr:  mongo.ErrNoDocuments,
			beforeSuite: func() (*mongo.Client, func()) {
				client, cleanFunc, err := prepareTestMongoStore()
				assert.NoError(t, err)

				defer client.Disconnect(context.Background())

				messageCollection := client.Database(testDB).Collection(testCollection)
				_, err = messageCollection.DeleteMany(context.Background(), nil)
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

				defer client.Disconnect(context.Background())

				bsonMessages := make([]interface{}, len(sampleMixedMessages))
				for _, message := range sampleMixedMessages {
					bsonMessages = append(bsonMessages, message)
				}

				// broke the data
				bsonMessages[0].(map[string]interface{})["status"] = 1

				messageCollection := client.Database(testDB).Collection(testCollection)
				_, err = messageCollection.InsertMany(context.Background(), bsonMessages)
				assert.NoError(t, err)

				return client, cleanFunc
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, cleanFunc := tt.beforeSuite()

			defer cleanFunc()

			messageRepository := NewMessageRepositoryImpl(client.Database("test").Collection("messages"))
			gotData, err := messageRepository.RetrieveSentMessages()
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.wantData, gotData)
		})
	}
}
