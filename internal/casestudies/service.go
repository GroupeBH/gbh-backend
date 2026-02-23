package casestudies

import (
	"context"
	"errors"
	"strings"
	"time"

	"gbh-backend/internal/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ErrNotFound    = errors.New("case study not found")
	ErrSlugExists  = errors.New("slug already exists")
	ErrInvalidSlug = errors.New("invalid slug")
)

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

func (s *Service) Create(ctx context.Context, req UpsertRequest) (CaseStudy, error) {
	slug := normalizeSlug(req.Slug, req.Title)
	if slug == "" {
		return CaseStudy{}, ErrInvalidSlug
	}

	isPublished := false
	if req.IsPublished != nil {
		isPublished = *req.IsPublished
	}
	sortOrder := 0
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	now := time.Now().In(s.location)
	item := CaseStudy{
		ID:          primitive.NewObjectID().Hex(),
		Slug:        slug,
		Title:       strings.TrimSpace(req.Title),
		Category:    strings.TrimSpace(req.Category),
		ClientName:  strings.TrimSpace(req.ClientName),
		Problem:     strings.TrimSpace(req.Problem),
		Solution:    strings.TrimSpace(req.Solution),
		Result:      strings.TrimSpace(req.Result),
		IsPublished: isPublished,
		SortOrder:   sortOrder,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, item); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return CaseStudy{}, ErrSlugExists
		}
		return CaseStudy{}, err
	}
	return item, nil
}

func (s *Service) Update(ctx context.Context, id string, req UpsertRequest) (CaseStudy, error) {
	id = strings.TrimSpace(id)
	slug := normalizeSlug(req.Slug, req.Title)
	if slug == "" {
		return CaseStudy{}, ErrInvalidSlug
	}

	isPublished := false
	if req.IsPublished != nil {
		isPublished = *req.IsPublished
	}
	sortOrder := 0
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	set := bson.M{
		"slug":         slug,
		"title":        strings.TrimSpace(req.Title),
		"category":     strings.TrimSpace(req.Category),
		"client_name":  strings.TrimSpace(req.ClientName),
		"problem":      strings.TrimSpace(req.Problem),
		"solution":     strings.TrimSpace(req.Solution),
		"result":       strings.TrimSpace(req.Result),
		"is_published": isPublished,
		"sort_order":   sortOrder,
		"updated_at":   time.Now().In(s.location),
	}

	updated, err := s.repo.Update(ctx, id, set)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return CaseStudy{}, ErrNotFound
		}
		if mongo.IsDuplicateKeyError(err) {
			return CaseStudy{}, ErrSlugExists
		}
		return CaseStudy{}, err
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

func (s *Service) ListPublic(ctx context.Context, filter PublicListFilter) ([]CaseStudy, error) {
	filter.Category = strings.TrimSpace(filter.Category)
	return s.repo.ListPublic(ctx, filter)
}

func (s *Service) GetPublishedBySlug(ctx context.Context, slug string) (CaseStudy, error) {
	item, err := s.repo.GetPublishedBySlug(ctx, strings.TrimSpace(slug))
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return CaseStudy{}, ErrNotFound
		}
		return CaseStudy{}, err
	}
	return item, nil
}

func (s *Service) ListAdmin(ctx context.Context, filter AdminListFilter, limit, offset int64) ([]CaseStudy, int64, error) {
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

func normalizeSlug(slug, title string) string {
	raw := strings.TrimSpace(slug)
	if raw == "" {
		raw = strings.TrimSpace(title)
	}
	return utils.Slugify(raw)
}
