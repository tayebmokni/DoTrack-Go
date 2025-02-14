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
	AddPosition(deviceID string, latitude, longitude float64) (*model.Position, error)
	GetDevicePositions(deviceID string) ([]*model.Position, error)
	GetLatestPosition(deviceID string) (*model.Position, error)
	ProcessRawData(deviceID string, data []byte) (*model.Position, error)
}

type positionService struct {
	positionRepo    repository.PositionRepository
	teltonikaDecoder *teltonika.Decoder
	gt06Decoder      *gt06.Decoder
	h02Decoder       *h02.Decoder
}

func NewPositionService(positionRepo repository.PositionRepository) PositionService {
	return &positionService{
		positionRepo:     positionRepo,
		teltonikaDecoder: teltonika.NewDecoder(),
		gt06Decoder:      gt06.NewDecoder(),
		h02Decoder:       h02.NewDecoder(),
	}
}

func (s *positionService) AddPosition(deviceID string, latitude, longitude float64) (*model.Position, error) {
	if deviceID == "" {
		return nil, errors.New("invalid device ID")
	}

	position := model.NewPosition(deviceID, latitude, longitude)
	err := s.positionRepo.Create(position)
	if err != nil {
		return nil, err
	}
	return position, nil
}

func (s *positionService) GetDevicePositions(deviceID string) ([]*model.Position, error) {
	if deviceID == "" {
		return nil, errors.New("invalid device ID")
	}
	return s.positionRepo.FindByDeviceID(deviceID)
}

func (s *positionService) GetLatestPosition(deviceID string) (*model.Position, error) {
	if deviceID == "" {
		return nil, errors.New("invalid device ID")
	}
	return s.positionRepo.FindLatestByDeviceID(deviceID)
}

func (s *positionService) ProcessRawData(deviceID string, data []byte) (*model.Position, error) {
	if deviceID == "" {
		return nil, errors.New("invalid device ID")
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

	err := s.positionRepo.Create(position)
	if err != nil {
		return nil, err
	}

	return position, nil
}