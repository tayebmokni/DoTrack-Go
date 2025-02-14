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
	return &MongoConfig{
		URI:      getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		Database: getEnv("MONGODB_DATABASE", "tracking"),
	}
}

func ConnectMongoDB(cfg *MongoConfig) (*mongo.Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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

	log.Printf("Connected to MongoDB: %s", cfg.Database)
	return client.Database(cfg.Database), nil
}
