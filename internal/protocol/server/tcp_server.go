package server

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"
	"tracking/internal/core/model"
	"tracking/internal/core/repository"
	"tracking/internal/protocol/gt06"
	"tracking/internal/protocol/h02"
	"tracking/internal/protocol/teltonika"
)

type DeviceConnection struct {
	conn          net.Conn
	deviceID      string
	protocol      string
	authenticated bool
	lastSeen      int64
}

type TCPServer struct {
	port             int
	listener         net.Listener
	deviceRepo       repository.DeviceRepository
	positionRepo     repository.PositionRepository
	gt06Decoder      *gt06.Decoder
	h02Decoder       *h02.Decoder
	teltonikaDecoder *teltonika.Decoder
	connections      map[string]*DeviceConnection
	mutex            sync.RWMutex
}

func NewTCPServer(port int, deviceRepo repository.DeviceRepository, positionRepo repository.PositionRepository) *TCPServer {
	return &TCPServer{
		port:             port,
		deviceRepo:       deviceRepo,
		positionRepo:     positionRepo,
		gt06Decoder:      gt06.NewDecoder(),
		h02Decoder:       h02.NewDecoder(),
		teltonikaDecoder: teltonika.NewDecoder(),
		connections:      make(map[string]*DeviceConnection),
	}
}

func (s *TCPServer) Start() error {
	var err error
	s.listener, err = net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to start TCP server: %v", err)
	}

	log.Printf("TCP server listening on port %d", s.port)
	log.Printf("Supported protocols: GT06, H02, Teltonika")

	go s.acceptConnections()
	return nil
}

func (s *TCPServer) Stop() {
	if s.listener != nil {
		s.listener.Close()
	}

	// Close all active connections
	s.mutex.Lock()
	for _, conn := range s.connections {
		conn.conn.Close()
	}
	s.connections = make(map[string]*DeviceConnection)
	s.mutex.Unlock()
}

func (s *TCPServer) acceptConnections() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				return
			}
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		go s.handleConnection(conn)
	}
}

func (s *TCPServer) authenticateDevice(data []byte, protocol string) (*model.Device, error) {
	var deviceID string

	// Extract device identifier based on protocol
	switch protocol {
	case "gt06":
		if len(data) < 10 {
			return nil, fmt.Errorf("data too short for GT06 protocol")
		}
		deviceID = fmt.Sprintf("%X", data[4:10]) // IMEI in GT06
	case "h02":
		parts := strings.Split(string(data), ",")
		if len(parts) < 3 {
			return nil, fmt.Errorf("invalid H02 protocol format")
		}
		deviceID = parts[2] // IMEI in H02
	case "teltonika":
		if len(data) < 8 {
			return nil, fmt.Errorf("data too short for Teltonika protocol")
		}
		deviceID = fmt.Sprintf("%X", data[0:8]) // IMEI in Teltonika
	default:
		return nil, fmt.Errorf("unknown protocol")
	}

	// Check if it's a test device
	if strings.HasPrefix(deviceID, "test-") || strings.HasPrefix(deviceID, "demo-") {
		log.Printf("Accepting test device: %s", deviceID)
		return &model.Device{
			ID:         deviceID,
			Name:       "Test Device",
			UniqueID:   deviceID,
			Status:     "active",
			Protocol:   protocol,
			CreatedAt:  time.Now(),
			LastUpdate: time.Now(),
		}, nil
	}

	// Find device in database
	device, err := s.deviceRepo.FindByUniqueID(deviceID)
	if err != nil {
		return nil, fmt.Errorf("error finding device: %v", err)
	}
	if device == nil {
		return nil, fmt.Errorf("device not found: %s", deviceID)
	}

	return device, nil
}

func (s *TCPServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()
	log.Printf("New connection from %s", remoteAddr)

	deviceConn := &DeviceConnection{
		conn:          conn,
		authenticated: false,
	}

	buffer := make([]byte, 4096)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from connection: %v", err)
			}
			if deviceConn.deviceID != "" {
				s.mutex.Lock()
				delete(s.connections, deviceConn.deviceID)
				s.mutex.Unlock()
				log.Printf("Device disconnected: %s", deviceConn.deviceID)
			}
			return
		}

		data := buffer[:n]
		log.Printf("Received %d bytes from %s", n, remoteAddr)

		// Detect protocol and handle authentication
		var protocol string
		if bytes.HasPrefix(data, []byte{0x78, 0x78}) {
			protocol = "gt06"
		} else if bytes.HasPrefix(data, []byte("*HQ")) {
			protocol = "h02"
		} else {
			protocol = "teltonika"
		}

		if !deviceConn.authenticated {
			device, err := s.authenticateDevice(data, protocol)
			if err != nil {
				log.Printf("Authentication failed for %s: %v", remoteAddr, err)
				return
			}

			deviceConn.deviceID = device.ID
			deviceConn.protocol = protocol
			deviceConn.authenticated = true

			// Store connection
			s.mutex.Lock()
			s.connections[device.ID] = deviceConn
			s.mutex.Unlock()

			log.Printf("Device authenticated: %s (%s)", device.ID, protocol)

			// Send authentication response based on protocol
			var response []byte
			switch protocol {
			case "gt06":
				response = s.gt06Decoder.GenerateResponse(0x01, device.ID)
			case "h02":
				response = []byte("*HQ,OK#")
			case "teltonika":
				response = []byte{0x01}
			}

			if _, err := conn.Write(response); err != nil {
				log.Printf("Error sending auth response to %s: %v", device.ID, err)
				return
			}

			continue
		}

		// Handle protocol-specific data
		var response []byte
		var processErr error
		var position *model.Position

		// Process data based on protocol
		switch protocol {
		case "gt06":
			decodedData, err := s.gt06Decoder.Decode(data)
			if err == nil {
				position = s.gt06Decoder.ToPosition(deviceConn.deviceID, decodedData)
				msgType := data[3] // Protocol number in GT06 packet
				response = s.gt06Decoder.GenerateResponse(msgType, deviceConn.deviceID)
			} else {
				processErr = err
			}

		case "h02":
			decodedData, err := s.h02Decoder.Decode(data)
			if err == nil {
				position = s.h02Decoder.ToPosition(deviceConn.deviceID, decodedData)
				response = []byte("*HQ,OK#")
			} else {
				processErr = err
			}

		default: // teltonika
			decodedData, err := s.teltonikaDecoder.Decode(data)
			if err == nil {
				position = s.teltonikaDecoder.ToPosition(deviceConn.deviceID, decodedData)
				response = []byte{0x01}
			} else {
				processErr = err
			}
		}

		if processErr != nil {
			log.Printf("Error processing data from %s: %v", deviceConn.deviceID, processErr)
			continue
		}

		// Store position and update device status if position is valid
		if position != nil {
			if err := s.positionRepo.Create(position); err != nil {
				log.Printf("Error storing position for device %s: %v", deviceConn.deviceID, err)
			} else {
				// Update device's last position and status
				device, err := s.deviceRepo.FindByID(deviceConn.deviceID)
				if err == nil && device != nil {
					device.PositionID = position.ID
					device.LastUpdate = position.Timestamp
					device.Status = "active"
					if err := s.deviceRepo.Update(device); err != nil {
						log.Printf("Error updating device status: %v", err)
					}
				}
			}
		}

		// Send response to device
		if response != nil {
			if _, err := conn.Write(response); err != nil {
				log.Printf("Error sending response to %s: %v", deviceConn.deviceID, err)
				continue
			}
		}
	}
}