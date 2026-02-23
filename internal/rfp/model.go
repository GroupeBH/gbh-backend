package rfp

import "time"

const (
	StatusNew       = "new"
	StatusReviewing = "reviewing"
	StatusQualified = "qualified"
	StatusWon       = "won"
	StatusLost      = "lost"

	SourceWebsite  = "website"
	SourceWhatsApp = "whatsapp"
	SourceManual   = "manual"
)

var validStatuses = map[string]struct{}{
	StatusNew:       {},
	StatusReviewing: {},
	StatusQualified: {},
	StatusWon:       {},
	StatusLost:      {},
}

var validSources = map[string]struct{}{
	SourceWebsite:  {},
	SourceWhatsApp: {},
	SourceManual:   {},
}

func IsValidStatus(value string) bool {
	_, ok := validStatuses[value]
	return ok
}

func IsValidSource(value string) bool {
	_, ok := validSources[value]
	return ok
}

type Lead struct {
	ID           string    `bson:"_id,omitempty" json:"id"`
	Organization string    `bson:"organization" json:"organization"`
	Sector       string    `bson:"sector,omitempty" json:"sector,omitempty"`
	Domain       string    `bson:"domain" json:"domain"`
	Deadline     string    `bson:"deadline,omitempty" json:"deadline,omitempty"`
	BudgetRange  string    `bson:"budget_range,omitempty" json:"budget_range,omitempty"`
	ContactName  string    `bson:"contact_name,omitempty" json:"contact_name,omitempty"`
	Phone        string    `bson:"phone" json:"phone"`
	Email        string    `bson:"email,omitempty" json:"email,omitempty"`
	Description  string    `bson:"description" json:"description"`
	Status       string    `bson:"status" json:"status"`
	Source       string    `bson:"source" json:"source"`
	CreatedAt    time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt    time.Time `bson:"updated_at" json:"updated_at"`
}

type CreateRequest struct {
	Organization string `json:"organization" validate:"required"`
	Sector       string `json:"sector"`
	Domain       string `json:"domain" validate:"required"`
	Deadline     string `json:"deadline"`
	BudgetRange  string `json:"budget_range"`
	ContactName  string `json:"contact_name"`
	Phone        string `json:"phone" validate:"required,phone"`
	Email        string `json:"email" validate:"omitempty,email"`
	Description  string `json:"description" validate:"required"`
	Source       string `json:"source" validate:"omitempty,oneof=website whatsapp manual"`
}

type AdminStatusUpdateRequest struct {
	Status string `json:"status" validate:"required,oneof=new reviewing qualified won lost"`
}

type ListFilter struct {
	Status string
	Source string
}
