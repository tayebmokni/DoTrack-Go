package repository

import (
	"sync"
	"time"
	"tracking/internal/core/model"
)

type inMemoryPositionRepository struct {
	positions map[string]*model.Position
	mutex     sync.RWMutex
}

func NewInMemoryPositionRepository() PositionRepository {
	return &inMemoryPositionRepository{
		positions: make(map[string]*model.Position),
	}
}

func (r *inMemoryPositionRepository) Create(position *model.Position) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.positions[position.ID] = position
	return nil
}

func (r *inMemoryPositionRepository) FindByID(id string) (*model.Position, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	if position, exists := r.positions[id]; exists {
		return position, nil
	}
	return nil, nil
}

func (r *inMemoryPositionRepository) FindByDeviceID(deviceID string) ([]*model.Position, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var result []*model.Position
	for _, position := range r.positions {
		if position.DeviceID == deviceID {
			result = append(result, position)
		}
	}
	return result, nil
}

func (r *inMemoryPositionRepository) FindLatestByDeviceID(deviceID string) (*model.Position, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var latest *model.Position
	var latestTime time.Time

	for _, position := range r.positions {
		if position.DeviceID == deviceID {
			if latest == nil || position.Timestamp.After(latestTime) {
				latest = position
				latestTime = position.Timestamp
			}
		}
	}
	return latest, nil
}
