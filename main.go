package main

import (
	"context"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

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

	// Swagger setup
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

	messagesCollection := messagesMongoClient.Database(os.Getenv("MESSAGES_DB_NAME")).Collection(os.Getenv("MESSAGES_COLLECTION_NAME"))

	messagesRepository := NewMessageRepositoryImpl(messagesCollection)
	messageService := NewMessageServiceImpl(messagesRepository)
	messageHandler := NewMessageHandler(messageService)
	messageHandler.RegisterRoutes(app)

	if err := app.Listen(":3000"); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
