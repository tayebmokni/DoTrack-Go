package server

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"tracking/internal/protocol/gt06"
	"tracking/internal/protocol/h02"
	"tracking/internal/protocol/teltonika"
)

type TCPServer struct {
	port            int
	listener        net.Listener
	gt06Decoder     *gt06.Decoder
	h02Decoder      *h02.Decoder
	teltonikaDecoder *teltonika.Decoder
}

func NewTCPServer(port int) *TCPServer {
	return &TCPServer{
		port:            port,
		gt06Decoder:     gt06.NewDecoder(),
		h02Decoder:      h02.NewDecoder(),
		teltonikaDecoder: teltonika.NewDecoder(),
	}
}

func (s *TCPServer) Start() error {
	var err error
	s.listener, err = net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to start TCP server: %v", err)
	}

	log.Printf("TCP server listening on port %d", s.port)

	go s.acceptConnections()
	return nil
}

func (s *TCPServer) Stop() {
	if s.listener != nil {
		s.listener.Close()
	}
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

func (s *TCPServer) handleConnection(conn net.Conn) {
	defer conn.Close()
	
	log.Printf("New connection from %s", conn.RemoteAddr().String())

	buffer := make([]byte, 4096)
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading from connection: %v", err)
			}
			return
		}

		data := buffer[:n]
		
		// Detect protocol and decode data
		if bytes.HasPrefix(data, []byte{0x78, 0x78}) {
			// GT06 protocol
			decodedData, err := s.gt06Decoder.Decode(data)
			if err != nil {
				log.Printf("Error decoding GT06 data: %v", err)
				continue
			}
			log.Printf("Received GT06 position: lat=%f, lon=%f", decodedData.Latitude, decodedData.Longitude)
			
		} else if bytes.HasPrefix(data, []byte("*HQ")) {
			// H02 protocol
			decodedData, err := s.h02Decoder.Decode(data)
			if err != nil {
				log.Printf("Error decoding H02 data: %v", err)
				continue
			}
			log.Printf("Received H02 position: lat=%f, lon=%f", decodedData.Latitude, decodedData.Longitude)
			
		} else {
			// Try Teltonika protocol
			decodedData, err := s.teltonikaDecoder.Decode(data)
			if err != nil {
				log.Printf("Error decoding Teltonika data: %v", err)
				continue
			}
			log.Printf("Received Teltonika position: lat=%f, lon=%f", decodedData.Latitude, decodedData.Longitude)
		}
	}
}
