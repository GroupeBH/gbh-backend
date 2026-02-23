package references

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Repository interface {
	Create(ctx context.Context, item Reference) error
	Update(ctx context.Context, id string, set bson.M) (Reference, error)
	Delete(ctx context.Context, id string) (bool, error)
	ListPublic(ctx context.Context, filter PublicListFilter) ([]Reference, error)
	ListAdmin(ctx context.Context, filter AdminListFilter, limit, offset int64) ([]Reference, error)
	CountAdmin(ctx context.Context, filter AdminListFilter) (int64, error)
}

type MongoRepository struct {
	col *mongo.Collection
}

func NewRepository(col *mongo.Collection) *MongoRepository {
	return &MongoRepository{col: col}
}

func (r *MongoRepository) Create(ctx context.Context, item Reference) error {
	_, err := r.col.InsertOne(ctx, item)
	return err
}

func (r *MongoRepository) Update(ctx context.Context, id string, set bson.M) (Reference, error) {
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	update := bson.M{"$set": set}

	var updated Reference
	if err := r.col.FindOneAndUpdate(ctx, bson.M{"_id": id}, update, opts).Decode(&updated); err != nil {
		return Reference{}, err
	}
	return updated, nil
}

func (r *MongoRepository) Delete(ctx context.Context, id string) (bool, error) {
	res, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return false, err
	}
	return res.DeletedCount > 0, nil
}

func (r *MongoRepository) ListPublic(ctx context.Context, filter PublicListFilter) ([]Reference, error) {
	query := bson.M{"is_public": true}
	if filter.Category != "" {
		query["category"] = filter.Category
	}

	opts := options.Find().SetSort(bson.D{
		{Key: "sort_order", Value: 1},
		{Key: "created_at", Value: -1},
	})

	cursor, err := r.col.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	items := make([]Reference, 0)
	for cursor.Next(ctx) {
		var ref Reference
		if err := cursor.Decode(&ref); err != nil {
			return nil, err
		}
		items = append(items, ref)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *MongoRepository) ListAdmin(ctx context.Context, filter AdminListFilter, limit, offset int64) ([]Reference, error) {
	query := bson.M{}
	if filter.Category != "" {
		query["category"] = filter.Category
	}

	opts := options.Find().
		SetSort(bson.D{
			{Key: "sort_order", Value: 1},
			{Key: "created_at", Value: -1},
		}).
		SetLimit(limit).
		SetSkip(offset)

	cursor, err := r.col.Find(ctx, query, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	items := make([]Reference, 0)
	for cursor.Next(ctx) {
		var ref Reference
		if err := cursor.Decode(&ref); err != nil {
			return nil, err
		}
		items = append(items, ref)
	}
	if err := cursor.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *MongoRepository) CountAdmin(ctx context.Context, filter AdminListFilter) (int64, error) {
	query := bson.M{}
	if filter.Category != "" {
		query["category"] = filter.Category
	}
	return r.col.CountDocuments(ctx, query)
}
