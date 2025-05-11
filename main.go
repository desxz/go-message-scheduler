package main

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"sync"

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
	ctx := context.Background()
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	app := fiber.New()

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

	poolCtx := context.Background()

	redisDB, err := strconv.Atoi(os.Getenv("REDIS_DB"))
	if err != nil {
		logger.Fatal("Failed to parse REDIS_DB", zap.Error(err))
	}

	messageCache := NewRedisCache(os.Getenv("REDIS_URI"), os.Getenv("REDIS_PASSWORD"), redisDB, config.Cache)

	validate := validator.New()

	poolWg := &sync.WaitGroup{}
	pool := NewWorkerPool(config.Pool.NumWorkers, messagesRepository, webhookClient, messageCache, *config, logger, poolWg, config.Pool.InitialJobFetch, validate)
	pool.Start()
	defer pool.Shutdown(poolCtx)

	messageHandler.RegisterRoutes(app)

	workerPoolHandler := NewWorkerPoolHandler(pool)
	workerPoolHandler.RegisterRoutes(app)

	go func() {
		if err := app.Listen(os.Getenv("PORT")); err != nil {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	<-pool.poolCtx.Done()
}
