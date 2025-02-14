package repository

import (
	"context"
	"time"
	"tracking/internal/core/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type DeviceRepository interface {
	Create(device *model.Device) error
	Update(device *model.Device) error
	Delete(id string) error
	FindByID(id string) (*model.Device, error)
	FindAll() ([]*model.Device, error)
	FindByUserID(userID string) ([]*model.Device, error)
	FindByUniqueID(uniqueID string) (*model.Device, error) // Added method
}

type MongoDeviceRepository struct {
	collection *mongo.Collection
}

func NewMongoDeviceRepository(db *mongo.Database) *MongoDeviceRepository {
	return &MongoDeviceRepository{
		collection: db.Collection("devices"),
	}
}

func (r *MongoDeviceRepository) Create(device *model.Device) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.InsertOne(ctx, device)
	return err
}

func (r *MongoDeviceRepository) Update(device *model.Device) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.ReplaceOne(ctx, bson.M{"id": device.ID}, device)
	return err
}

func (r *MongoDeviceRepository) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.DeleteOne(ctx, bson.M{"id": id})
	return err
}

func (r *MongoDeviceRepository) FindByID(id string) (*model.Device, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var device model.Device
	err := r.collection.FindOne(ctx, bson.M{"id": id}).Decode(&device)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &device, err
}

func (r *MongoDeviceRepository) FindAll() ([]*model.Device, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var devices []*model.Device
	if err = cursor.All(ctx, &devices); err != nil {
		return nil, err
	}
	return devices, nil
}

func (r *MongoDeviceRepository) FindByUserID(userID string) ([]*model.Device, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{"userid": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var devices []*model.Device
	if err = cursor.All(ctx, &devices); err != nil {
		return nil, err
	}
	return devices, nil
}

// Add new method to find device by uniqueId
func (r *MongoDeviceRepository) FindByUniqueID(uniqueID string) (*model.Device, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var device model.Device
	err := r.collection.FindOne(ctx, bson.M{"uniqueid": uniqueID}).Decode(&device)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &device, err
}