package service

import (
	"context"
	"errors"
	"fmt"
	"time"
	"tracking/internal/cache"
	"tracking/internal/core/model"
	"tracking/internal/core/repository"
)

type DeviceService interface {
	CreateDevice(name, uniqueID string, userID, organizationID string) (*model.Device, error)
	UpdateDevice(device *model.Device) error
	DeleteDevice(id string) error
	GetDevice(id string) (*model.Device, error)
	GetAllDevices() ([]*model.Device, error)
	GetUserDevices(userID string) ([]*model.Device, error)
	GetOrganizationDevices(organizationID string) ([]*model.Device, error)
	ValidateDeviceAccess(deviceID, userID string) error
}

type deviceService struct {
	deviceRepo    repository.DeviceRepository
	orgMemberRepo repository.OrganizationMemberRepository
}

const (
	deviceCacheDuration      = 5 * time.Minute
	deviceListCacheDuration  = 2 * time.Minute
	deviceCacheKeyPrefix     = "device:"
	deviceListCacheKeyPrefix = "devices:"
)

func NewDeviceService(deviceRepo repository.DeviceRepository, orgMemberRepo repository.OrganizationMemberRepository) DeviceService {
	return &deviceService{
		deviceRepo:    deviceRepo,
		orgMemberRepo: orgMemberRepo,
	}
}

func (s *deviceService) CreateDevice(name, uniqueID string, userID, organizationID string) (*model.Device, error) {
	if name == "" || uniqueID == "" {
		return nil, errors.New("invalid device data")
	}

	// If creating for an organization, verify user is a member
	if organizationID != "" {
		member, err := s.orgMemberRepo.FindByUserAndOrg(userID, organizationID)
		if err != nil {
			return nil, err
		}
		if member == nil {
			return nil, errors.New("user is not a member of the organization")
		}
	}

	device := model.NewDevice(name, uniqueID)
	device.SetOwnership(userID, organizationID)
	err := s.deviceRepo.Create(device)
	if err != nil {
		return nil, err
	}

	// Invalidate relevant cache entries
	ctx := context.Background()
	cache.Delete(ctx, fmt.Sprintf("%s%s", deviceListCacheKeyPrefix, userID))
	if organizationID != "" {
		cache.Delete(ctx, fmt.Sprintf("%s%s", deviceListCacheKeyPrefix, organizationID))
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

	ctx := context.Background()
	cacheKey := fmt.Sprintf("%s%s", deviceCacheKeyPrefix, id)

	// Try to get from cache first
	var device model.Device
	err := cache.Get(ctx, cacheKey, &device)
	if err == nil {
		return &device, nil
	}

	// If not in cache, get from database
	device_ptr, err := s.deviceRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// Cache the result if found
	if device_ptr != nil {
		cache.Set(ctx, cacheKey, device_ptr, deviceCacheDuration)
	}

	return device_ptr, nil
}

func (s *deviceService) GetAllDevices() ([]*model.Device, error) {
	return s.deviceRepo.FindAll()
}

func (s *deviceService) GetUserDevices(userID string) ([]*model.Device, error) {
	if userID == "" {
		return nil, errors.New("invalid user ID")
	}

	ctx := context.Background()
	cacheKey := fmt.Sprintf("%s%s", deviceListCacheKeyPrefix, userID)

	// Try to get from cache first
	var devices []*model.Device
	err := cache.Get(ctx, cacheKey, &devices)
	if err == nil {
		return devices, nil
	}

	// If not in cache, get from database
	devices, err = s.deviceRepo.FindByUserID(userID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	cache.Set(ctx, cacheKey, devices, deviceListCacheDuration)

	return devices, nil
}

func (s *deviceService) GetOrganizationDevices(organizationID string) ([]*model.Device, error) {
	if organizationID == "" {
		return nil, errors.New("invalid organization ID")
	}
	devices, err := s.deviceRepo.FindAll()
	if err != nil {
		return nil, err
	}

	var orgDevices []*model.Device
	for _, device := range devices {
		if device.OrganizationID == organizationID {
			orgDevices = append(orgDevices, device)
		}
	}
	return orgDevices, nil
}

func (s *deviceService) ValidateDeviceAccess(deviceID, userID string) error {
	if deviceID == "" || userID == "" {
		return errors.New("invalid device or user ID")
	}

	device, err := s.deviceRepo.FindByID(deviceID)
	if err != nil {
		return err
	}
	if device == nil {
		return errors.New("device not found")
	}

	// Check if user owns the device directly
	if device.UserID == userID {
		return nil
	}

	// If device belongs to an organization, check organization membership
	if device.OrganizationID != "" {
		member, err := s.orgMemberRepo.FindByUserAndOrg(userID, device.OrganizationID)
		if err != nil {
			return err
		}
		if member != nil {
			return nil
		}
	}

	return errors.New("unauthorized access to device")
}