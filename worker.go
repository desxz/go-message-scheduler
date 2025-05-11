package main

import (
	"context"
	"sync"
	"time"

	"github.com/desxz/go-message-scheduler/client"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
)

type WorkerMessageStore interface {
	FetchAndMarkProcessing(ctx context.Context) (*Message, error)
	MarkAsSent(ctx context.Context, messageID primitive.ObjectID, webhookMessageID string) error
	MarkAsFailed(ctx context.Context, messageID primitive.ObjectID, reason string) error
}

type WorkerMessageCache interface {
	Set(ctx context.Context, key string, value string) error
}

type WebhookClient interface {
	PostMessage(ctx context.Context, message *client.WebhookRequest) (*client.WebhookResponse, error)
}

type WorkerConfig struct {
	ProcessMessageTimeout time.Duration `mapstructure:"processMessageTimeout"`
	WorkerJobInterval     time.Duration `mapstructure:"workerJobInterval"`
}

type WorkerInstance struct {
	ID                 string
	workerMessageStore WorkerMessageStore
	webhookClient      WebhookClient
	workerMessageCache WorkerMessageCache
	config             WorkerConfig
	validate           *validator.Validate
	logger             *zap.Logger
}

func NewWorkerInstance(id string, workerMessageStore WorkerMessageStore, webhookClient WebhookClient, workerMessageCache WorkerMessageCache, config WorkerConfig, logger *zap.Logger, validate *validator.Validate) *WorkerInstance {
	return &WorkerInstance{
		ID:                 id,
		workerMessageStore: workerMessageStore,
		workerMessageCache: workerMessageCache,
		webhookClient:      webhookClient,
		config:             config,
		validate:           validate,
		logger:             logger.With(zap.String("component", "worker"), zap.String("worker_id", id)),
	}
}

func (w *WorkerInstance) Start(ctx context.Context, wg *sync.WaitGroup, canFetchNewJob func() bool) {
	defer wg.Done()

	w.logger.Info("Worker started")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Worker received shutdown signal, stopping gracefully")
			return
		default:
		}

		if !canFetchNewJob() {
			w.logger.Debug("WorkerPool tarafından yeni iş alımı duraklatıldı, bekleniyor...")
			select {
			case <-ctx.Done():
				w.logger.Info("Worker (duraklatılmışken) context iptali nedeniyle durduruluyor.")
				return
			case <-time.After(w.config.WorkerJobInterval):
				continue
			}
		}

		processed, err := w.ProcessMessage(ctx)
		if err != nil {
			w.logger.Error("Error processing message", zap.Error(err))
		}

		if !processed && err == nil {
			w.logger.Info("Worker: No messages to process, sleeping", zap.Duration("interval", w.config.WorkerJobInterval))
			time.Sleep(w.config.WorkerJobInterval)
		}
	}
}

func (w *WorkerInstance) ProcessMessage(ctx context.Context) (bool, error) {
	message, err := w.workerMessageStore.FetchAndMarkProcessing(ctx)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, err
	}

	w.logger.Info("Processing message", zap.String("message_id", message.ID.Hex()))

	if err := w.validate.Struct(message); err != nil {
		w.logger.Error("Invalid message struct", zap.String("message_id", message.ID.Hex()), zap.Error(err))
		if err := w.workerMessageStore.MarkAsFailed(ctx, message.ID, "invalid message struct: "+err.Error()); err != nil {
			w.logger.Error("Failed to mark message as failed",
				zap.String("message_id", message.ID.Hex()),
				zap.Error(err))
			return true, err
		}
		return true, err
	}

	res, err := w.webhookClient.PostMessage(ctx, &client.WebhookRequest{
		To:      message.RecipientPhoneNumber,
		Content: message.Content,
	})
	if err != nil {
		w.logger.Error("Failed to send message to webhook",
			zap.String("message_id", message.ID.Hex()),
			zap.Error(err))
		if err := w.workerMessageStore.MarkAsFailed(ctx, message.ID, "failed to send webhook: "+err.Error()); err != nil {
			w.logger.Error("Failed to mark message as failed",
				zap.String("message_id", message.ID.Hex()),
				zap.Error(err))
			return true, err
		}

		return true, err
	}

	now := time.Now()
	if err := w.workerMessageStore.MarkAsSent(ctx, message.ID, res.MessageID); err != nil {
		w.logger.Error("Failed to mark message as sent",
			zap.String("message_id", message.ID.Hex()),
			zap.Error(err))
		return true, err
	}

	if err := w.workerMessageCache.Set(ctx, res.MessageID, now.Format(time.RFC3339)); err != nil {
		w.logger.Error("Failed to cache message ID",
			zap.String("message_id", message.ID.Hex()),
			zap.Error(err))
		return true, err
	}

	w.logger.Info("Message processed successfully", zap.String("message_id", message.ID.Hex()))
	return true, nil
}
