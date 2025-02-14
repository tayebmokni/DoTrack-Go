package repository

import (
	"context"
	"time"
	"tracking/internal/core/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type OrganizationRepository interface {
	Create(org *model.Organization) error
	Update(org *model.Organization) error
	Delete(id string) error
	FindByID(id string) (*model.Organization, error)
	FindAll() ([]*model.Organization, error)
}

type MongoOrganizationRepository struct {
	collection *mongo.Collection
}

func NewMongoOrganizationRepository(db *mongo.Database) *MongoOrganizationRepository {
	return &MongoOrganizationRepository{
		collection: db.Collection("organizations"),
	}
}

func (r *MongoOrganizationRepository) Create(org *model.Organization) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.InsertOne(ctx, org)
	return err
}

func (r *MongoOrganizationRepository) Update(org *model.Organization) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.ReplaceOne(ctx, bson.M{"id": org.ID}, org)
	return err
}

func (r *MongoOrganizationRepository) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.DeleteOne(ctx, bson.M{"id": id})
	return err
}

func (r *MongoOrganizationRepository) FindByID(id string) (*model.Organization, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var org model.Organization
	err := r.collection.FindOne(ctx, bson.M{"id": id}).Decode(&org)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &org, err
}

func (r *MongoOrganizationRepository) FindAll() ([]*model.Organization, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var orgs []*model.Organization
	if err = cursor.All(ctx, &orgs); err != nil {
		return nil, err
	}
	return orgs, nil
}
