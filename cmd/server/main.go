package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tracking/internal/api/router"
	"tracking/internal/cache"
	"tracking/internal/config"
	"tracking/internal/core/repository"
	"tracking/internal/core/service"
	"tracking/internal/protocol/server"
)

func main() {
	// Setup panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic: %v", r)
			os.Exit(1)
		}
	}()

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting application...")

	// Load configurations
	log.Println("Loading configuration...")
	cfg := config.LoadConfig()

	// Log startup information
	log.Printf("Configuration loaded successfully:")
	log.Printf("Host: %s", cfg.Host)
	log.Printf("Port: %s", cfg.Port)
	log.Printf("Base URL: %s", cfg.BaseURL)
	log.Printf("Test Mode: %v", cfg.TestMode)

	// Initialize Redis if URL is provided
	log.Println("Initializing Redis...")
	cache.Initialize(cfg.RedisURL)
	defer cache.Close()

	// Initialize repositories
	log.Println("Initializing repositories...")
	var deviceRepo repository.DeviceRepository
	var positionRepo repository.PositionRepository
	var orgMemberRepo repository.OrganizationMemberRepository

	// In test mode, always use in-memory repositories
	if cfg.TestMode {
		log.Println("Running in test mode - using in-memory repositories")
		deviceRepo = repository.NewInMemoryDeviceRepository()
		positionRepo = repository.NewInMemoryPositionRepository()
		orgMemberRepo = repository.NewInMemoryOrganizationMemberRepository()
	} else {
		// Try to connect to MongoDB
		mongoConfig := config.NewMongoConfig()
		log.Printf("Connecting to MongoDB at: %s", mongoConfig.URI)

		db, err := config.ConnectMongoDB(mongoConfig)
		if err != nil {
			log.Printf("Failed to connect to MongoDB: %v - falling back to in-memory storage", err)
			deviceRepo = repository.NewInMemoryDeviceRepository()
			positionRepo = repository.NewInMemoryPositionRepository()
			orgMemberRepo = repository.NewInMemoryOrganizationMemberRepository()
		} else {
			log.Printf("Successfully connected to MongoDB database: %s", mongoConfig.Database)
			deviceRepo = repository.NewMongoDeviceRepository(db)
			positionRepo = repository.NewMongoPositionRepository(db)
			orgMemberRepo = repository.NewMongoOrganizationMemberRepository(db)
		}
	}

	// Initialize services
	log.Println("Initializing services...")
	deviceService := service.NewDeviceService(deviceRepo, orgMemberRepo)
	positionService := service.NewPositionService(positionRepo, deviceRepo, orgMemberRepo)

	// Initialize HTTP router
	log.Println("Setting up HTTP router...")
	r := router.NewRouter(deviceService, positionService)

	// Initialize TCP server
	log.Printf("Initializing TCP server on port %d...", cfg.TCPPort)
	tcpServer := server.NewTCPServer(cfg.TCPPort, deviceRepo, positionRepo)
	if err := tcpServer.Start(); err != nil {
		log.Printf("Failed to start TCP server: %v", err)
		return
	}
	defer tcpServer.Stop()

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Handler: r,
	}

	// Channel to handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("HTTP server starting on %s", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				log.Printf("HTTP server failed to start: %v", err)
				os.Exit(1)
			}
		}
	}()

	// Wait for interrupt signal
	<-stop
	log.Println("Shutting down servers...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	log.Println("Servers stopped")
}