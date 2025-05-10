package main

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	StatusSent       = "sent"
	StatusUnsent     = "unsent"
	StatusProcessing = "processing"
	StatusFailed     = "failed"
)

var (
	ErrDocumentDecodingFailed = errors.New("document decoding failed")
	ErrInvalidMessageID       = errors.New("invalid message ID")
)

type MessageRepositoryImpl struct {
	messageCollection *mongo.Collection
}

func NewMessageRepositoryImpl(collection *mongo.Collection) *MessageRepositoryImpl {
	return &MessageRepositoryImpl{
		messageCollection: collection,
	}
}

func (mr *MessageRepositoryImpl) FetchAndMarkProcessing(ctx context.Context) (*Message, error) {
	filter := bson.M{
		"status": StatusUnsent,
	}

	update := bson.M{
		"$set": bson.M{
			"status": StatusProcessing,
		},
	}

	opts := options.FindOneAndUpdate().
		SetSort(bson.M{"created_at": 1}).
		SetReturnDocument(options.After)

	var message Message
	err := mr.messageCollection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&message)

	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, mongo.ErrNoDocuments
		}
		return nil, err
	}

	return &message, nil
}

func (mr *MessageRepositoryImpl) MarkAsSent(ctx context.Context, messageID primitive.ObjectID, webhookMessageID string) error {
	now := time.Now()

	filter := bson.M{
		"_id": messageID,
	}

	update := bson.M{
		"$set": bson.M{
			"status":                      StatusSent,
			"sent_at":                     now,
			"webhook_response_message_id": webhookMessageID,
		},
	}

	// find one and update
	opts := options.FindOneAndUpdate().
		SetReturnDocument(options.After)
	var message Message
	err := mr.messageCollection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&message)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return mongo.ErrNoDocuments
		}
		return err
	}

	return nil
}

func (mr *MessageRepositoryImpl) MarkAsFailed(ctx context.Context, messageID primitive.ObjectID) error {
	filter := bson.M{
		"_id": messageID,
	}

	update := bson.M{
		"$set": bson.M{
			"status": StatusFailed,
		},
	}

	opts := options.FindOneAndUpdate().
		SetReturnDocument(options.After)
	var message Message
	err := mr.messageCollection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&message)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return mongo.ErrNoDocuments
		}
		return err
	}

	return nil
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
