package repository

import (
	"context"
	"time"
	"tracking/internal/core/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type OrganizationMemberRepository interface {
	Create(member *model.OrganizationMember) error
	Update(member *model.OrganizationMember) error
	Delete(id string) error
	FindByID(id string) (*model.OrganizationMember, error)
	FindByUserAndOrg(userID, orgID string) (*model.OrganizationMember, error)
	FindByOrganization(orgID string) ([]*model.OrganizationMember, error)
}

type MongoOrganizationMemberRepository struct {
	collection *mongo.Collection
}

func NewMongoOrganizationMemberRepository(db *mongo.Database) *MongoOrganizationMemberRepository {
	return &MongoOrganizationMemberRepository{
		collection: db.Collection("organization_members"),
	}
}

func (r *MongoOrganizationMemberRepository) Create(member *model.OrganizationMember) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.InsertOne(ctx, member)
	return err
}

func (r *MongoOrganizationMemberRepository) Update(member *model.OrganizationMember) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.ReplaceOne(ctx, bson.M{"id": member.ID}, member)
	return err
}

func (r *MongoOrganizationMemberRepository) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := r.collection.DeleteOne(ctx, bson.M{"id": id})
	return err
}

func (r *MongoOrganizationMemberRepository) FindByID(id string) (*model.OrganizationMember, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var member model.OrganizationMember
	err := r.collection.FindOne(ctx, bson.M{"id": id}).Decode(&member)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &member, err
}

func (r *MongoOrganizationMemberRepository) FindByUserAndOrg(userID, orgID string) (*model.OrganizationMember, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var member model.OrganizationMember
	err := r.collection.FindOne(ctx, bson.M{
		"userid":         userID,
		"organizationid": orgID,
	}).Decode(&member)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &member, err
}

func (r *MongoOrganizationMemberRepository) FindByOrganization(orgID string) ([]*model.OrganizationMember, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, bson.M{"organizationid": orgID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var members []*model.OrganizationMember
	if err = cursor.All(ctx, &members); err != nil {
		return nil, err
	}
	return members, nil
}
