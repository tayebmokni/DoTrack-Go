package service

import (
	"bytes"
	"errors"
	"tracking/internal/core/model"
	"tracking/internal/core/repository"
	"tracking/internal/protocol/gt06"
	"tracking/internal/protocol/h02"
	"tracking/internal/protocol/teltonika"
)

type PositionService interface {
	AddPosition(deviceID string, latitude, longitude float64, userID string) (*model.Position, error)
	GetDevicePositions(deviceID string, userID string) ([]*model.Position, error)
	GetLatestPosition(deviceID string, userID string) (*model.Position, error)
	ProcessRawData(deviceID string, data []byte, userID string) (*model.Position, error)
}

type positionService struct {
	positionRepo     repository.PositionRepository
	deviceRepo       repository.DeviceRepository
	teltonikaDecoder *teltonika.Decoder
	gt06Decoder      *gt06.Decoder
	h02Decoder       *h02.Decoder
}

func NewPositionService(positionRepo repository.PositionRepository, deviceRepo repository.DeviceRepository) PositionService {
	return &positionService{
		positionRepo:     positionRepo,
		deviceRepo:       deviceRepo,
		teltonikaDecoder: teltonika.NewDecoder(),
		gt06Decoder:      gt06.NewDecoder(),
		h02Decoder:       h02.NewDecoder(),
	}
}

func (s *positionService) validateDeviceAccess(deviceID, userID string) (*model.Device, error) {
	if deviceID == "" {
		return nil, errors.New("invalid device ID")
	}

	device, err := s.deviceRepo.FindByID(deviceID)
	if err != nil {
		return nil, err
	}
	if device == nil {
		return nil, errors.New("unauthorized device: device not registered in the system")
	}

	// Check if user owns the device directly
	if device.UserID == userID {
		return device, nil
	}

	// If device belongs to an organization, we would check organization membership here
	// For now, if the device has an organization ID and no direct user ownership,
	// we'll deny access until organization membership check is implemented
	if device.OrganizationID != "" && device.UserID != userID {
		return nil, errors.New("unauthorized access: device belongs to an organization")
	}

	return nil, errors.New("unauthorized access to device")
}

func (s *positionService) AddPosition(deviceID string, latitude, longitude float64, userID string) (*model.Position, error) {
	_, err := s.validateDeviceAccess(deviceID, userID)
	if err != nil {
		return nil, err
	}

	position := model.NewPosition(deviceID, latitude, longitude)
	err = s.positionRepo.Create(position)
	if err != nil {
		return nil, err
	}
	return position, nil
}

func (s *positionService) GetDevicePositions(deviceID string, userID string) ([]*model.Position, error) {
	_, err := s.validateDeviceAccess(deviceID, userID)
	if err != nil {
		return nil, err
	}
	return s.positionRepo.FindByDeviceID(deviceID)
}

func (s *positionService) GetLatestPosition(deviceID string, userID string) (*model.Position, error) {
	_, err := s.validateDeviceAccess(deviceID, userID)
	if err != nil {
		return nil, err
	}
	return s.positionRepo.FindLatestByDeviceID(deviceID)
}

func (s *positionService) ProcessRawData(deviceID string, data []byte, userID string) (*model.Position, error) {
	device, err := s.validateDeviceAccess(deviceID, userID)
	if err != nil {
		return nil, err
	}

	var position *model.Position

	// Detect protocol and use appropriate decoder
	if bytes.HasPrefix(data, []byte{0x78, 0x78}) {
		// GT06 protocol
		decodedData, err := s.gt06Decoder.Decode(data)
		if err != nil {
			return nil, err
		}
		position = s.gt06Decoder.ToPosition(deviceID, decodedData)
	} else if bytes.HasPrefix(data, []byte("*HQ")) {
		// H02 protocol
		decodedData, err := s.h02Decoder.Decode(data)
		if err != nil {
			return nil, err
		}
		position = s.h02Decoder.ToPosition(deviceID, decodedData)
	} else {
		// Default to Teltonika protocol
		decodedData, err := s.teltonikaDecoder.Decode(data)
		if err != nil {
			return nil, err
		}
		position = s.teltonikaDecoder.ToPosition(deviceID, decodedData)
	}

	err = s.positionRepo.Create(position)
	if err != nil {
		return nil, err
	}

	// Update device's last position and status
	device.PositionID = position.ID
	device.LastUpdate = position.Timestamp
	device.Status = "active"
	err = s.deviceRepo.Update(device)
	if err != nil {
		return nil, err
	}

	return position, nil
}