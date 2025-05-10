package main

import "errors"

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
	return nil, nil
}
