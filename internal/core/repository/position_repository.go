package repository

import (
	"context"
	"time"
	"tracking/internal/core/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PositionRepository interface {
	Create(position *model.Position) error
	FindByDeviceID(deviceID string) ([]*model.Position, error)
	FindLatestByDeviceID(deviceID string) (*model.Position, error)
}

type MongoPositionRepository struct {
	collection *mongo.Collection
}

func NewMongoPositionRepository(db *mongo.Database) *MongoPositionRepository {
	return &MongoPositionRepository{
		collection: db.Collection("positions"),
	}
}

func (r *MongoPositionRepository) Create(position *model.Position) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.InsertOne(ctx, position)
	return err
}

func (r *MongoPositionRepository) FindByDeviceID(deviceID string) ([]*model.Position, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{"deviceid": deviceID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var positions []*model.Position
	if err = cursor.All(ctx, &positions); err != nil {
		return nil, err
	}
	return positions, nil
}

func (r *MongoPositionRepository) FindLatestByDeviceID(deviceID string) (*model.Position, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.FindOne().SetSort(bson.M{"timestamp": -1})
	var position model.Position
	err := r.collection.FindOne(ctx, bson.M{"deviceid": deviceID}, opts).Decode(&position)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &position, err
}