package main

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	StatusSent = "sent"
)

var (
	ErrDocumentDecodingFailed = errors.New("document decoding failed")
)

type MessageRepositoryImpl struct {
	messageCollection *mongo.Collection
}

func NewMessageRepositoryImpl(collection *mongo.Collection) *MessageRepositoryImpl {
	return &MessageRepositoryImpl{
		messageCollection: collection,
	}
}

func (mr *MessageRepositoryImpl) RetrieveSentMessages() ([]Message, error) {
	ctx := context.Background()

	filter := bson.M{"status": StatusSent} // not actually required, but for clarity
	sort := bson.M{"sent_at": -1}

	cursor, err := mr.messageCollection.Find(ctx, filter, options.Find().SetSort(sort))
	if err != nil {
		return nil, err
	}

	defer cursor.Close(ctx)

	var messages []Message
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, ErrDocumentDecodingFailed // breaks the worker I know
	}

	return messages, nil
}
