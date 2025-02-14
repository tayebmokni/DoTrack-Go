package config

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoConfig struct {
	URI      string
	Database string
}

func NewMongoConfig() *MongoConfig {
	uri := getEnv("MONGODB_URI", "")
	if uri == "" {
		log.Fatal("MONGODB_URI environment variable is required")
	}

	return &MongoConfig{
		URI:      uri,
		Database: getEnv("MONGODB_DATABASE", "tracking"),
	}
}

func ConnectMongoDB(cfg *MongoConfig) (*mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Printf("Connecting to MongoDB at: %s", cfg.URI)

	clientOptions := options.Client().ApplyURI(cfg.URI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Ping the database
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	log.Printf("Successfully connected to MongoDB database: %s", cfg.Database)
	return client.Database(cfg.Database), nil
}