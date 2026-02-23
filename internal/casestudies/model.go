package casestudies

import "time"

type CaseStudy struct {
	ID          string    `bson:"_id,omitempty" json:"id"`
	Slug        string    `bson:"slug" json:"slug"`
	Title       string    `bson:"title" json:"title"`
	Category    string    `bson:"category" json:"category"`
	ClientName  string    `bson:"client_name" json:"client_name"`
	Problem     string    `bson:"problem" json:"problem"`
	Solution    string    `bson:"solution" json:"solution"`
	Result      string    `bson:"result" json:"result"`
	IsPublished bool      `bson:"is_published" json:"is_published"`
	SortOrder   int       `bson:"sort_order" json:"sort_order"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at" json:"updated_at"`
}

type UpsertRequest struct {
	Slug        string `json:"slug"`
	Title       string `json:"title" validate:"required"`
	Category    string `json:"category" validate:"required"`
	ClientName  string `json:"client_name" validate:"required"`
	Problem     string `json:"problem" validate:"required"`
	Solution    string `json:"solution" validate:"required"`
	Result      string `json:"result" validate:"required"`
	IsPublished *bool  `json:"is_published"`
	SortOrder   *int   `json:"sort_order" validate:"omitempty,gte=0"`
}

type PublicListFilter struct {
	Category string
}

type AdminListFilter struct {
	Category string
}
