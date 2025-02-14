package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	fmt.Println("\nTesting device endpoint response times...")

	// Make multiple requests to a cached endpoint
	url := "http://localhost:8000/api/devices/get?id=test-device-1"

	for i := 0; i < 5; i++ {
		start := time.Now()
		resp, err := http.Get(url)
		if err != nil {
			log.Printf("Error making request: %v", err)
			continue
		}
		defer resp.Body.Close()

		duration := time.Since(start)
		log.Printf("Request %d - Response time: %v - Status: %d", i+1, duration, resp.StatusCode)
		time.Sleep(time.Second) // Wait a bit between requests
	}
}