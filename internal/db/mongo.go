package db

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Collections struct {
	Services          *mongo.Collection
	Appointments      *mongo.Collection
	ContactMessages   *mongo.Collection
	ReservationBlocks *mongo.Collection
}

func Connect(ctx context.Context, uri, dbName string) (*mongo.Client, *Collections, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		return nil, nil, err
	}

	db := client.Database(dbName)

	cols := &Collections{
		Services:          db.Collection("services"),
		Appointments:      db.Collection("appointments"),
		ContactMessages:   db.Collection("contact_messages"),
		ReservationBlocks: db.Collection("reservation_blocks"),
	}

	return client, cols, nil
}

func EnsureIndexes(ctx context.Context, cols *Collections) error {
	indexTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := cols.Services.Indexes().CreateMany(indexTimeout, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "slug", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	})
	if err != nil {
		return err
	}

	_, err = cols.Appointments.Indexes().CreateMany(indexTimeout, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "date", Value: 1}, {Key: "time", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "date", Value: 1}},
		},
	})
	if err != nil {
		return err
	}

	_, err = cols.ReservationBlocks.Indexes().CreateMany(indexTimeout, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "date", Value: 1}, {Key: "time", Value: 1}},
		},
	})
	if err != nil {
		return err
	}

	return nil
}
