package main

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MessageService interface {
	RetrieveSentMessages() ([]Message, error)
}

type Message struct {
	ID                       primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	WebhookResponseMessageID string             `bson:"webhook_response_message_id" json:"webhook_response_message_id"`
	Content                  string             `bson:"content" json:"content" validate:"max=160,min=1"`
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

// RetriveSentMessages godoc
// @Summary Retrieve all sent messages
// @Description Get all successfully sent messages
// @Tags messages
// @Accept json
// @Produce json
// @Success 200 {array} Message
// @Failure 404 {object} nil "No sent messages found"
// @Failure 500 {object} nil "Internal server error"
// @Router /sent-messages [get]
func (h *MessageHandler) RetriveSentMessages(c *fiber.Ctx) error {
	sentMessages, err := h.messageService.RetrieveSentMessages()
	if err != nil {
		if errors.Is(err, ErrDocumentNotFound) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.JSON(sentMessages)
}
