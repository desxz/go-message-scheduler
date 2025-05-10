package main

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type WorkerMessageStore interface {
	FetchAndMarkProcessing(ctx context.Context) (*Message, error)
	MarkAsSent(ctx context.Context, messageID primitive.ObjectID, webhookMessageID string) error
	MarkAsFailed(ctx context.Context, messageID primitive.ObjectID) error
}

type WebhookResponse struct {
	Message   string `json:"message"`
	MessageID string `json:"messageId"`
}

type WebhookRequest struct {
	To      string `json:"to"`
	Content string `json:"content"`
}

type WebhookClient interface {
	PostMessage(ctx context.Context, message *WebhookRequest) (*WebhookResponse, error)
}

type WorkerConfig struct {
	WorkerJobTimeout  time.Duration `mapstructure:"workerJobTimeout"`
	WorkerJobInterval time.Duration `mapstructure:"workerJobInterval"`
}

type WorkerInstance struct {
	ID                 string
	workerMessageStore WorkerMessageStore
	webhookClient      WebhookClient
	config             WorkerConfig
	logger             *zap.Logger
}

func NewWorkerInstance(id string, workerMessageStore WorkerMessageStore, webhookClient WebhookClient, logger *zap.Logger) *WorkerInstance {
	return &WorkerInstance{
		ID:                 id,
		workerMessageStore: workerMessageStore,
		webhookClient:      webhookClient,
		logger:             logger.With(zap.String("component", "worker"), zap.String("worker_id", id)),
	}
}

func (w *WorkerInstance) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	w.logger.Info("Worker started")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Worker received shutdown signal, stopping gracefully")
			// TODO: Implement graceful shutdown logic here
			return
		default:
			processed, err := w.ProcessMessage(ctx)
			if err != nil {
				w.logger.Error("Error processing message", zap.Error(err))
			}

			if !processed && err == nil {
				// no messg to process, sleep for a while
				time.Sleep(w.config.WorkerJobInterval)
			}
		}
	}
}

func (w *WorkerInstance) ProcessMessage(ctx context.Context) (bool, error) {
	opCtx, cancel := context.WithTimeout(ctx, w.config.WorkerJobTimeout)
	defer cancel()

	message, err := w.workerMessageStore.FetchAndMarkProcessing(opCtx)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
	}

	w.logger.Info("Processing message", zap.String("message_id", message.ID.Hex()))

	res, err := w.webhookClient.PostMessage(opCtx, &WebhookRequest{
		To:      message.RecipientPhoneNumber,
		Content: message.Content,
	})
	if err != nil {
		w.logger.Error("Failed to send message to webhook",
			zap.String("message_id", message.ID.Hex()),
			zap.Error(err))
		if err := w.workerMessageStore.MarkAsFailed(opCtx, message.ID); err != nil {
			w.logger.Error("Failed to mark message as failed",
				zap.String("message_id", message.ID.Hex()),
				zap.Error(err))
			return true, err
		}

		return true, err
	}

	if err := w.workerMessageStore.MarkAsSent(opCtx, message.ID, res.MessageID); err != nil {
		w.logger.Error("Failed to mark message as sent",
			zap.String("message_id", message.ID.Hex()),
			zap.Error(err))
		return true, err
	}

	w.logger.Info("Message processed successfully", zap.String("message_id", message.ID.Hex()))
	return true, nil
}
