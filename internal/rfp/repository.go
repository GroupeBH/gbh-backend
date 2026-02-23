package rfp

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Repository interface {
	Create(ctx context.Context, lead Lead) error
	List(ctx context.Context, filter ListFilter, limit, offset int64) ([]Lead, error)
	Count(ctx context.Context, filter ListFilter) (int64, error)
	GetByID(ctx context.Context, id string) (Lead, error)
	UpdateStatus(ctx context.Context, id string, status string, now time.Time) (Lead, error)
}

type MongoRepository struct {
	col *mongo.Collection
}

func NewRepository(col *mongo.Collection) *MongoRepository {
	return &MongoRepository{col: col}
}

func (r *MongoRepository) Create(ctx context.Context, lead Lead) error {
	_, err := r.col.InsertOne(ctx, lead)
	return err
}

func (r *MongoRepository) List(ctx context.Context, filter ListFilter, limit, offset int64) ([]Lead, error) {
	query := r.filterToBSON(filter)
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(limit).
		SetSkip(offset)

	cursor, err := r.col.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	items := make([]Lead, 0)
	for cursor.Next(ctx) {
		var lead Lead
		if err := cursor.Decode(&lead); err != nil {
			return nil, err
		}
		items = append(items, lead)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (r *MongoRepository) Count(ctx context.Context, filter ListFilter) (int64, error) {
	return r.col.CountDocuments(ctx, r.filterToBSON(filter))
}

func (r *MongoRepository) GetByID(ctx context.Context, id string) (Lead, error) {
	var lead Lead
	if err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&lead); err != nil {
		return Lead{}, err
	}
	return lead, nil
}

func (r *MongoRepository) UpdateStatus(ctx context.Context, id string, status string, now time.Time) (Lead, error) {
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": now,
		},
	}

	var updated Lead
	if err := r.col.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, opts).Decode(&updated); err != nil {
		return Lead{}, err
	}
	return updated, nil
}

func (r *MongoRepository) filterToBSON(filter ListFilter) bson.M {
	query := bson.M{}
	if filter.Status != "" {
		query["status"] = filter.Status
	}
	if filter.Source != "" {
		query["source"] = filter.Source
	}
	return query
}
