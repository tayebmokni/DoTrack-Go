package main

import (
	"fmt"
	"log"
	"net/http"

	"tracking/internal/api/router"
	"tracking/internal/config"
	"tracking/internal/core/repository"
	"tracking/internal/core/service"
)

func main() {
	// Load configurations
	cfg := config.LoadConfig()
	mongoConfig := config.NewMongoConfig()

	// Log startup information
	log.Printf("Starting server with configuration:")
	log.Printf("Host: %s", cfg.Host)
	log.Printf("Port: %s", cfg.Port)
	log.Printf("Base URL: %s", cfg.BaseURL)

	// Connect to MongoDB
	db, err := config.ConnectMongoDB(mongoConfig)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	log.Printf("Connected to MongoDB database: %s", mongoConfig.Database)

	// Initialize repositories
	deviceRepo := repository.NewMongoDeviceRepository(db)
	positionRepo := repository.NewMongoPositionRepository(db)
	orgMemberRepo := repository.NewMongoOrganizationMemberRepository(db)

	// Initialize services
	deviceService := service.NewDeviceService(deviceRepo, orgMemberRepo)
	positionService := service.NewPositionService(positionRepo, deviceRepo)

	// Initialize router
	r := router.NewRouter(deviceService, positionService)

	// Start server
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	log.Printf("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}