package references

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var ErrNotFound = errors.New("reference not found")

type Service struct {
	repo     Repository
	location *time.Location
}

func NewService(repo Repository, location *time.Location) *Service {
	return &Service{
		repo:     repo,
		location: location,
	}
}

func (s *Service) Create(ctx context.Context, req UpsertRequest) (Reference, error) {
	now := time.Now().In(s.location)
	isPublic := true
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}
	sortOrder := 0
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	item := Reference{
		ID:         primitive.NewObjectID().Hex(),
		ClientName: strings.TrimSpace(req.ClientName),
		Category:   strings.TrimSpace(req.Category),
		Summary:    strings.TrimSpace(req.Summary),
		Location:   strings.TrimSpace(req.Location),
		LogoURL:    strings.TrimSpace(req.LogoURL),
		IsPublic:   isPublic,
		SortOrder:  sortOrder,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := s.repo.Create(ctx, item); err != nil {
		return Reference{}, err
	}
	return item, nil
}

func (s *Service) Update(ctx context.Context, id string, req UpsertRequest) (Reference, error) {
	id = strings.TrimSpace(id)
	isPublic := true
	if req.IsPublic != nil {
		isPublic = *req.IsPublic
	}
	sortOrder := 0
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	set := bson.M{
		"client_name": strings.TrimSpace(req.ClientName),
		"category":    strings.TrimSpace(req.Category),
		"summary":     strings.TrimSpace(req.Summary),
		"location":    strings.TrimSpace(req.Location),
		"logo_url":    strings.TrimSpace(req.LogoURL),
		"is_public":   isPublic,
		"sort_order":  sortOrder,
		"updated_at":  time.Now().In(s.location),
	}

	updated, err := s.repo.Update(ctx, id, set)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Reference{}, ErrNotFound
		}
		return Reference{}, err
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	deleted, err := s.repo.Delete(ctx, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	if !deleted {
		return ErrNotFound
	}
	return nil
}

func (s *Service) ListPublic(ctx context.Context, filter PublicListFilter) ([]Reference, error) {
	filter.Category = strings.TrimSpace(filter.Category)
	return s.repo.ListPublic(ctx, filter)
}

func (s *Service) ListAdmin(ctx context.Context, filter AdminListFilter, limit, offset int64) ([]Reference, int64, error) {
	filter.Category = strings.TrimSpace(filter.Category)
	items, err := s.repo.ListAdmin(ctx, filter, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.CountAdmin(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
