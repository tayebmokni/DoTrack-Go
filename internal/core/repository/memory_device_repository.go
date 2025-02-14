package repository

import (
	"fmt"
	"sync"
	"tracking/internal/core/model"
)

type inMemoryDeviceRepository struct {
	devices map[string]*model.Device
	mutex   sync.RWMutex
}

func NewInMemoryDeviceRepository() DeviceRepository {
	return &inMemoryDeviceRepository{
		devices: make(map[string]*model.Device),
	}
}

func (r *inMemoryDeviceRepository) Create(device *model.Device) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.devices[device.ID]; exists {
		return fmt.Errorf("device with ID %s already exists", device.ID)
	}

	r.devices[device.ID] = device
	return nil
}

func (r *inMemoryDeviceRepository) Update(device *model.Device) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.devices[device.ID]; !exists {
		return fmt.Errorf("device with ID %s not found", device.ID)
	}

	r.devices[device.ID] = device
	return nil
}

func (r *inMemoryDeviceRepository) Delete(id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.devices[id]; !exists {
		return fmt.Errorf("device with ID %s not found", id)
	}

	delete(r.devices, id)
	return nil
}

func (r *inMemoryDeviceRepository) FindByID(id string) (*model.Device, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if device, exists := r.devices[id]; exists {
		return device, nil
	}
	return nil, nil
}

func (r *inMemoryDeviceRepository) FindByUniqueID(uniqueID string) (*model.Device, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, device := range r.devices {
		if device.UniqueID == uniqueID {
			return device, nil
		}
	}
	return nil, nil
}

func (r *inMemoryDeviceRepository) FindByUser(userID string) ([]*model.Device, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var result []*model.Device
	for _, device := range r.devices {
		if device.UserID == userID {
			result = append(result, device)
		}
	}
	return result, nil
}

func (r *inMemoryDeviceRepository) FindByUserID(userID string) ([]*model.Device, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var result []*model.Device
	for _, device := range r.devices {
		if device.UserID == userID {
			result = append(result, device)
		}
	}
	return result, nil
}

func (r *inMemoryDeviceRepository) FindByOrganization(orgID string) ([]*model.Device, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var result []*model.Device
	for _, device := range r.devices {
		if device.OrganizationID == orgID {
			result = append(result, device)
		}
	}
	return result, nil
}

func (r *inMemoryDeviceRepository) FindAll() ([]*model.Device, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	devices := make([]*model.Device, 0, len(r.devices))
	for _, device := range r.devices {
		devices = append(devices, device)
	}
	return devices, nil
}