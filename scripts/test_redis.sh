#!/bin/bash

# Test with Redis disabled
echo "Testing with Redis disabled..."
export REDIS_ACTIVE=false
go run cmd/server/main.go &
SERVER_PID=$!
sleep 2  # Wait for server to start

echo "Making requests with Redis disabled..."
go run cmd/test/test_redis.go

# Kill the server
kill $SERVER_PID

# Test with Redis enabled
echo -e "\nTesting with Redis enabled..."
export REDIS_ACTIVE=true
go run cmd/server/main.go &
SERVER_PID=$!
sleep 2  # Wait for server to start

echo "Making requests with Redis enabled..."
go run cmd/test/test_redis.go

# Kill the server
kill $SERVER_PID
