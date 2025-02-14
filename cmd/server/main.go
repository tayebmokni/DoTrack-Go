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
	// Load configurations
	cfg := config.LoadConfig()
	mongoConfig := config.NewMongoConfig()

	// Log startup information
	log.Printf("Starting server with configuration:")
	log.Printf("Host: %s", cfg.Host)
	log.Printf("Port: %s", cfg.Port)
	log.Printf("Base URL: %s", cfg.BaseURL)

	// Initialize Redis if URL is provided
	cache.Initialize(cfg.RedisURL)
	defer cache.Close()

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
	positionService := service.NewPositionService(positionRepo, deviceRepo, orgMemberRepo)

	// Initialize HTTP router
	r := router.NewRouter(deviceService, positionService)

	// Initialize TCP server
	tcpServer := server.NewTCPServer(5023) // Standard port for GPS tracking devices
	if err := tcpServer.Start(); err != nil {
		log.Fatalf("Failed to start TCP server: %v", err)
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
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed to start: %v", err)
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

	// TCP server is stopped by defer tcpServer.Stop()
	log.Println("Servers stopped")
}