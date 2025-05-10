package main

import (
	"errors"

	"go.mongodb.org/mongo-driver/mongo"
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
	return nil, nil
}
