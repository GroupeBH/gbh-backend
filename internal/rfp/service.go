package rfp

import (
	"context"
	"errors"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var (
	ErrInvalidSource = errors.New("invalid source")
	ErrInvalidStatus = errors.New("invalid status")
	ErrNotFound      = errors.New("rfp not found")
)

type Notifier interface {
	SendRFPLeadNotification(ctx context.Context, lead Lead) (string, error)
	SendRFPLeadConfirmation(ctx context.Context, lead Lead) (string, error)
}

type Service struct {
	repo     Repository
	location *time.Location
	notifier Notifier
}

func NewService(repo Repository, location *time.Location, notifier Notifier) *Service {
	return &Service{
		repo:     repo,
		location: location,
		notifier: notifier,
	}
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Lead, error) {
	source := strings.ToLower(strings.TrimSpace(req.Source))
	if source == "" {
		source = SourceWebsite
	}
	if !IsValidSource(source) {
		return Lead{}, ErrInvalidSource
	}

	now := time.Now().In(s.location)
	lead := Lead{
		ID:           primitive.NewObjectID().Hex(),
		Organization: strings.TrimSpace(req.Organization),
		Sector:       strings.TrimSpace(req.Sector),
		Domain:       strings.TrimSpace(req.Domain),
		Deadline:     strings.TrimSpace(req.Deadline),
		BudgetRange:  strings.TrimSpace(req.BudgetRange),
		ContactName:  strings.TrimSpace(req.ContactName),
		Phone:        strings.TrimSpace(req.Phone),
		Email:        strings.TrimSpace(req.Email),
		Description:  strings.TrimSpace(req.Description),
		Status:       StatusNew,
		Source:       source,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Create(ctx, lead); err != nil {
		return Lead{}, err
	}

	return lead, nil
}

func (s *Service) ListAdmin(ctx context.Context, filter ListFilter, limit, offset int64) ([]Lead, int64, error) {
	filter.Status = strings.ToLower(strings.TrimSpace(filter.Status))
	filter.Source = strings.ToLower(strings.TrimSpace(filter.Source))

	if filter.Status != "" && !IsValidStatus(filter.Status) {
		return nil, 0, ErrInvalidStatus
	}
	if filter.Source != "" && !IsValidSource(filter.Source) {
		return nil, 0, ErrInvalidSource
	}

	items, err := s.repo.List(ctx, filter, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.repo.Count(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Service) GetAdminByID(ctx context.Context, id string) (Lead, error) {
	lead, err := s.repo.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Lead{}, ErrNotFound
		}
		return Lead{}, err
	}
	return lead, nil
}

func (s *Service) UpdateStatus(ctx context.Context, id, status string) (Lead, error) {
	id = strings.TrimSpace(id)
	status = strings.ToLower(strings.TrimSpace(status))
	if !IsValidStatus(status) {
		return Lead{}, ErrInvalidStatus
	}

	updated, err := s.repo.UpdateStatus(ctx, id, status, time.Now().In(s.location))
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return Lead{}, ErrNotFound
		}
		return Lead{}, err
	}
	return updated, nil
}

func (s *Service) NotifyNewLead(ctx context.Context, lead Lead) error {
	if s.notifier == nil {
		return nil
	}
	_, err := s.notifier.SendRFPLeadNotification(ctx, lead)
	return err
}

func (s *Service) NotifyLeadConfirmation(ctx context.Context, lead Lead) error {
	if s.notifier == nil {
		return nil
	}
	if strings.TrimSpace(lead.Email) == "" {
		return nil
	}
	_, err := s.notifier.SendRFPLeadConfirmation(ctx, lead)
	return err
}
