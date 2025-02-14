package service

import (
    "errors"
    "tracking/internal/core/model"
    "tracking/internal/core/repository"
)

type DeviceService interface {
    CreateDevice(name, uniqueID string) (*model.Device, error)
    UpdateDevice(device *model.Device) error
    DeleteDevice(id string) error
    GetDevice(id string) (*model.Device, error)
    GetAllDevices() ([]*model.Device, error)
}

type deviceService struct {
    deviceRepo repository.DeviceRepository
}

func NewDeviceService(deviceRepo repository.DeviceRepository) DeviceService {
    return &deviceService{
        deviceRepo: deviceRepo,
    }
}

func (s *deviceService) CreateDevice(name, uniqueID string) (*model.Device, error) {
    if name == "" || uniqueID == "" {
        return nil, errors.New("invalid device data")
    }

    device := model.NewDevice(name, uniqueID)
    err := s.deviceRepo.Create(device)
    if err != nil {
        return nil, err
    }
    return device, nil
}

func (s *deviceService) UpdateDevice(device *model.Device) error {
    if device.ID == "" {
        return errors.New("invalid device ID")
    }
    return s.deviceRepo.Update(device)
}

func (s *deviceService) DeleteDevice(id string) error {
    if id == "" {
        return errors.New("invalid device ID")
    }
    return s.deviceRepo.Delete(id)
}

func (s *deviceService) GetDevice(id string) (*model.Device, error) {
    if id == "" {
        return nil, errors.New("invalid device ID")
    }
    return s.deviceRepo.FindByID(id)
}

func (s *deviceService) GetAllDevices() ([]*model.Device, error) {
    return s.deviceRepo.FindAll()
}
