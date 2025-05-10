package main

import (
	"context"
	"os"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	app := fiber.New()

	messagesMongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGODB_URI")))
	if err != nil {
		logger.Fatal("Failed to connect to MongoDB", zap.Error(err))
	}

	err = messagesMongoClient.Ping(ctx, nil)
	if err != nil {
		logger.Fatal("Failed to ping MongoDB", zap.Error(err))
	}

	messagesCollection := messagesMongoClient.Database(os.Getenv("MESSAGES_DB_NAME")).Collection(os.Getenv("MESSAGES_COLLECTION_NAME"))

	messagesRepository := NewMessageRepositoryImpl(messagesCollection)
	messageService := NewMessageServiceImpl(messagesRepository)
	messageHandler := NewMessageHandler(messageService)
	messageHandler.RegisterRoutes(app)

	if err := app.Listen(":3000"); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
