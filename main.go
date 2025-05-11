package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/desxz/go-message-scheduler/client"
	_ "github.com/desxz/go-message-scheduler/docs"
)

// @title Go Message Scheduler API
// @version 1.0
// @description API for scheduling and managing messages
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email muratgun545@gmail.com
// @license.name MIT License
// @license.url https://opensource.org/licenses/MIT
// @host localhost:3000
// @BasePath /
func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, _ := zap.NewProduction()
	defer logger.Sync()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	app.Get("/swagger/*", swagger.New(swagger.Config{
		Title:        "Message Scheduler API",
		DeepLinking:  false,
		DocExpansion: "list",
	}))

	messagesMongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGODB_URI")))
	if err != nil {
		logger.Fatal("Failed to connect to MongoDB", zap.Error(err))
	}

	err = messagesMongoClient.Ping(ctx, nil)
	if err != nil {
		logger.Fatal("Failed to ping MongoDB", zap.Error(err))
	}

	config, err := NewConfig(os.Getenv("CONFIG_PATH"), os.Getenv("CONFIG_ENV"))
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	messagesCollection := messagesMongoClient.Database(os.Getenv("MESSAGES_DB_NAME")).Collection(os.Getenv("MESSAGES_COLLECTION_NAME"))

	messagesRepository := NewMessageRepositoryImpl(messagesCollection)
	messageService := NewMessageServiceImpl(messagesRepository)
	messageHandler := NewMessageHandler(messageService)
	messageHandler.RegisterRoutes(app)

	webhookHttpClient := http.Client{
		Timeout: config.WebhookClient.Timeout,
	}
	webhookClient := client.NewWebhookClient(config.WebhookClient.Host, &webhookHttpClient, &config.WebhookClient)

	redisDB, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		logger.Fatal("Failed to parse REDIS_DB", zap.Error(err))
	}

	messageCache := NewRedisCache(os.Getenv("REDIS_URI"), os.Getenv("REDIS_PASSWORD"), redisDB, config.Cache)

	validate := validator.New()

	rateLimiter := NewRateLimiter(config.RateLimiter, logger)

	poolWg := &sync.WaitGroup{}
	pool := NewWorkerPool(config.Pool.NumWorkers, messagesRepository, webhookClient, messageCache, *config, logger, poolWg, config.Pool.InitialJobFetch, validate, rateLimiter)
	pool.Start()

	workerPoolHandler := NewWorkerPoolHandler(pool)
	workerPoolHandler.RegisterRoutes(app)

	serverShutdown := make(chan struct{})
	go func() {
		logger.Info("Starting server", zap.String("port", os.Getenv("PORT")))
		if err := app.Listen(os.Getenv("PORT")); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server error", zap.Error(err))
		}
		close(serverShutdown)
	}()

	<-quit
	logger.Info("Shutdown signal received")

	_, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	logger.Info("Shutting down HTTP server...")
	if err := app.Shutdown(); err != nil {
		logger.Error("Error shutting down server:", zap.Error(err))
	}
	<-serverShutdown
	logger.Info("HTTP Server shutdown complete")

	logger.Info("Shutting down worker pool...")
	workerShutdownCtx, workerCancel := context.WithTimeout(context.Background(), config.Pool.Timeout)
	defer workerCancel()

	if err := pool.Shutdown(workerShutdownCtx); err != nil {
		logger.Error("Error during worker pool shutdown:", zap.Error(err))
	}
	logger.Info("Worker pool shutdown complete")

	logger.Info("Closing Redis connection...")
	if err := messageCache.Close(); err != nil {
		logger.Error("Error closing Redis connection:", zap.Error(err))
	}
	logger.Info("Redis connection closed")

	logger.Info("Disconnecting from MongoDB...")
	if err := messagesMongoClient.Disconnect(ctx); err != nil {
		logger.Error("Error disconnecting from MongoDB:", zap.Error(err))
	}
	logger.Info("MongoDB disconnected")

	logger.Info("Graceful shutdown completed")

}
