package service

import (
    "errors"
    "tracking/internal/core/model"
    "tracking/internal/core/repository"
)

type PositionService interface {
    AddPosition(deviceID string, latitude, longitude float64) (*model.Position, error)
    GetDevicePositions(deviceID string) ([]*model.Position, error)
    GetLatestPosition(deviceID string) (*model.Position, error)
}

type positionService struct {
    positionRepo repository.PositionRepository
}

func NewPositionService(positionRepo repository.PositionRepository) PositionService {
    return &positionService{
        positionRepo: positionRepo,
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
