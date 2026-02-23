package db

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Collections struct {
	Services            *mongo.Collection
	ServiceTestimonials *mongo.Collection
	Appointments        *mongo.Collection
	ContactMessages     *mongo.Collection
	ReservationBlocks   *mongo.Collection
	Users               *mongo.Collection
	RFPLeads            *mongo.Collection
	References          *mongo.Collection
	CaseStudies         *mongo.Collection
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
		Services:            db.Collection("services"),
		ServiceTestimonials: db.Collection("service_testimonials"),
		Appointments:        db.Collection("appointments"),
		ContactMessages:     db.Collection("contact_messages"),
		ReservationBlocks:   db.Collection("reservation_blocks"),
		Users:               db.Collection("users"),
		RFPLeads:            db.Collection("rfp_leads"),
		References:          db.Collection("references"),
		CaseStudies:         db.Collection("case_studies"),
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

	_, err = cols.ServiceTestimonials.Indexes().CreateMany(indexTimeout, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "serviceId", Value: 1}, {Key: "createdAt", Value: -1}},
		},
	})
	if err != nil {
		return err
	}

	_, err = cols.Users.Indexes().CreateMany(indexTimeout, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "username", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true),
		},
	})
	if err != nil {
		return err
	}

	_, err = cols.RFPLeads.Indexes().CreateMany(indexTimeout, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "status", Value: 1}, {Key: "created_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "source", Value: 1}, {Key: "created_at", Value: -1}},
		},
	})
	if err != nil {
		return err
	}

	_, err = cols.References.Indexes().CreateMany(indexTimeout, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "is_public", Value: 1}, {Key: "category", Value: 1}, {Key: "sort_order", Value: 1}, {Key: "created_at", Value: -1}},
		},
	})
	if err != nil {
		return err
	}

	_, err = cols.CaseStudies.Indexes().CreateMany(indexTimeout, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "slug", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "is_published", Value: 1}, {Key: "category", Value: 1}, {Key: "sort_order", Value: 1}, {Key: "created_at", Value: -1}},
		},
	})
	if err != nil {
		return err
	}

	return nil
}
