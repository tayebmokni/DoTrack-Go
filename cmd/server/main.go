package main

import (
	"log"
	"net/http"

	"tracking/internal/api/router"
	"tracking/internal/config"
	"tracking/internal/core/repository"
	"tracking/internal/core/service"
)

func main() {
	// Load MongoDB configuration
	mongoConfig := config.NewMongoConfig()

	// Connect to MongoDB
	db, err := config.ConnectMongoDB(mongoConfig)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Initialize repositories with MongoDB
	deviceRepo := repository.NewMongoDeviceRepository(db)
	positionRepo := repository.NewMongoPositionRepository(db)

	// Initialize services
	deviceService := service.NewDeviceService(deviceRepo)
	positionService := service.NewPositionService(positionRepo)

	// Initialize router
	r := router.NewRouter(deviceService, positionService)

	// Start server
	log.Printf("Server starting on :8000")
	if err := http.ListenAndServe(":8000", r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}