package references

import "time"

type Reference struct {
	ID         string    `bson:"_id,omitempty" json:"id"`
	ClientName string    `bson:"client_name" json:"client_name"`
	Category   string    `bson:"category" json:"category"`
	Summary    string    `bson:"summary" json:"summary"`
	Location   string    `bson:"location" json:"location"`
	LogoURL    string    `bson:"logo_url,omitempty" json:"logo_url,omitempty"`
	IsPublic   bool      `bson:"is_public" json:"is_public"`
	SortOrder  int       `bson:"sort_order" json:"sort_order"`
	CreatedAt  time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt  time.Time `bson:"updated_at" json:"updated_at"`
}

type UpsertRequest struct {
	ClientName string `json:"client_name" validate:"required"`
	Category   string `json:"category" validate:"required"`
	Summary    string `json:"summary" validate:"required"`
	Location   string `json:"location" validate:"required"`
	LogoURL    string `json:"logo_url" validate:"omitempty,url"`
	IsPublic   *bool  `json:"is_public"`
	SortOrder  *int   `json:"sort_order" validate:"omitempty,gte=0"`
}

type PublicListFilter struct {
	Category string
}

type AdminListFilter struct {
	Category string
}
