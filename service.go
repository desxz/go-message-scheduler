package main

import (
	"errors"

	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ErrDocumentNotFound    = errors.New("document not found")
	ErrInternalServerError = errors.New("internal server error")
)

type MessageRepository interface {
	RetrieveSentMessages() ([]Message, error)
}

type MessageServiceImpl struct {
	messageRepository MessageRepository
}

func NewMessageServiceImpl(mr MessageRepository) *MessageServiceImpl {
	return &MessageServiceImpl{
		messageRepository: mr,
	}
}

func (ms *MessageServiceImpl) RetrieveSentMessages() ([]Message, error) {
	sentMessages, err := ms.messageRepository.RetrieveSentMessages()
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrDocumentNotFound
		}
		return nil, ErrInternalServerError
	}

	return sentMessages, nil
}
