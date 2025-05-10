package main

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MessageService interface {
	RetriveSentMessages() ([]Message, error)
}

type Message struct {
	ID                       primitive.ObjectID `bson:"_id,omitempty"`
	WebhookResponseMessageID string             `bson:"webhook_response_message_id" json:"webhook_response_message_id"`
	Content                  string             `bson:"content" json:"content"`
	RecipientPhoneNumber     string             `bson:"recipient_phone_number" json:"recipient_phone_number"`
	Status                   string             `bson:"status" json:"status"`
	CreatedAt                time.Time          `bson:"created_at" json:"created_at"`
	SentAt                   time.Time          `bson:"sent_at" json:"sent_at"`
}

type MessageHandler struct {
	messageService MessageService
}

func NewMessageHandler(ms MessageService) *MessageHandler {
	return &MessageHandler{
		messageService: ms,
	}
}

func (h *MessageHandler) RegisterRoutes(app *fiber.App) {
	app.Get("/sent-messages", h.RetriveSentMessages)
}

func (h *MessageHandler) RetriveSentMessages(c *fiber.Ctx) error {
	return nil
}
