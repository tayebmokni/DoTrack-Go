package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// Sample packets based on real device data
var (
	// GT06 location packet (with GPS info and status)
	gt06Packet = []byte{
		0x78, 0x78, // Start bytes
		0x11,       // Packet length
		0x12,       // Protocol number (location)
		0x0F,       // GPS status
		0x0C, 0x46, 0x58, 0x1E, // Latitude
		0x6E, 0x31, 0x22, 0x50, // Longitude
		0x28,             // Speed
		0x01, 0x44,       // Course
		0x23, 0x02, 0x14, // Date (YY-MM-DD)
		0x12, 0x15, 0x13, // Time (HH-MM-SS)
		0x06, 0x0D,       // Checksum (calculated)
		0x0D, 0x0A,       // End bytes
	}

	// H02 location packet (ASCII format)
	h02Packet = []byte("*HQ,V1,867567021398618,V,2237.7514,N,11408.6214,E,6,2,15,110,10,1,6")
)

type rawDataRequest struct {
	DeviceID string `json:"deviceId"`
	RawData  string `json:"rawData"` // Base64 encoded data
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func main() {
	// Wait for servers to start
	fmt.Println("Waiting for servers to start...")
	time.Sleep(5 * time.Second) // Increased delay to ensure server is ready

	// Get a test JWT token with retries
	var token string
	var err error
	for i := 0; i < 5; i++ {
		token, err = getTestToken()
		if err == nil {
			break
		}
		fmt.Printf("Attempt %d: Failed to get test token, retrying... (%v)\n", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		fmt.Printf("Failed to get test token after retries: %v\n", err)
		return
	}

	fmt.Println("Successfully obtained test token")

	// Test GT06 protocol
	fmt.Println("\nTesting GT06 protocol...")
	testProtocol("test-gt06-device", gt06Packet, token)

	time.Sleep(2 * time.Second)

	// Test H02 protocol
	fmt.Println("\nTesting H02 protocol...")
	testProtocol("test-h02-device", h02Packet, token)
}

func testProtocol(deviceID string, packet []byte, token string) {
	// Encode packet in base64
	rawData := base64.StdEncoding.EncodeToString(packet)

	// Create request body
	reqBody := rawDataRequest{
		DeviceID: deviceID,
		RawData:  rawData,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Printf("Error marshaling request: %v\n", err)
		return
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", "http://localhost:8000/api/positions/raw", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Send request with retries
	client := &http.Client{Timeout: 5 * time.Second}
	var resp *http.Response
	for i := 0; i < 3; i++ {
		resp, err = client.Do(req)
		if err == nil {
			break
		}
		fmt.Printf("Attempt %d: Failed to send request, retrying... (%v)\n", i+1, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		fmt.Printf("Error sending request after retries: %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Read response
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	fmt.Printf("Response status: %d\n", resp.StatusCode)
	fmt.Printf("Response body: %s\n", string(body))
}

func getTestToken() (string, error) {
	// Create login request
	loginReq := loginRequest{
		Email:    "test@example.com",
		Password: "test123",
	}

	jsonData, err := json.Marshal(loginReq)
	if err != nil {
		return "", fmt.Errorf("error marshaling login request: %v", err)
	}

	// Send POST request to test login endpoint with timeout
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post("http://localhost:8000/api/auth/test-login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error making login request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	var loginResp loginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return "", fmt.Errorf("error decoding login response: %v", err)
	}

	return loginResp.AccessToken, nil
}