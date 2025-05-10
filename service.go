package main

import (
	"errors"
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
		return nil, ErrInternalServerError
	}

	if len(sentMessages) == 0 {
		return nil, ErrDocumentNotFound
	}

	return sentMessages, nil
}
