# B2B Modules (RFP, References, Case Studies)

## Architecture backend (Go)

```text
internal/
  rfp/
    model.go
    repository.go
    service.go
    handler.go
  references/
    model.go
    repository.go
    service.go
    handler.go
  casestudies/
    model.go
    repository.go
    service.go
    handler.go
  httpx/
    httpx.go
```

## Router Chi (`/api/v1`)

Public:
- `POST /api/v1/rfp`
- `GET /api/v1/references?category=...`
- `GET /api/v1/case-studies`
- `GET /api/v1/case-studies/{slug}`

Admin:
- `GET /api/v1/admin/rfp?status=&source=&limit=&offset=`
- `GET /api/v1/admin/rfp/{id}`
- `PATCH /api/v1/admin/rfp/{id}`
- `GET /api/v1/admin/references?category=&limit=&offset=`
- `POST /api/v1/admin/references`
- `PUT /api/v1/admin/references/{id}`
- `DELETE /api/v1/admin/references/{id}`
- `GET /api/v1/admin/case-studies?category=&limit=&offset=`
- `POST /api/v1/admin/case-studies`
- `PUT /api/v1/admin/case-studies/{id}`
- `DELETE /api/v1/admin/case-studies/{id}`

Legacy routes `/api/...` are kept unchanged.

## Validation rules

RFP:
- required: `organization`, `domain`, `phone`, `description`
- `email`: format email if provided
- `source`: `website | whatsapp | manual` (default `website`)
- `status`: managed by admin only (`new | reviewing | qualified | won | lost`)

References:
- required: `client_name`, `category`, `summary`, `location`
- optional: `logo_url` (URL), `is_public` (default `true`), `sort_order` (default `0`)

Case studies:
- required: `title`, `category`, `client_name`, `problem`, `solution`, `result`
- `slug`: normalized from `slug` (or `title` if empty), must stay unique
- optional: `is_published` (default `false`), `sort_order` (default `0`)

## JSON examples

Create RFP request:

```json
{
  "organization": "ANAPI",
  "sector": "Public",
  "domain": "Formations",
  "deadline": "2026-03-30",
  "budget_range": "10k-25k USD",
  "contact_name": "Jean Mutombo",
  "phone": "+243812000000",
  "email": "jean@anapi.cd",
  "description": "Formation sur la gestion de projet",
  "source": "website"
}
```

Create RFP response:

```json
{
  "success": true,
  "message": "rfp submitted",
  "id": "67b8f9e7b7ab8f6d70df5d11"
}
```

Public references response:

```json
{
  "items": [
    {
      "id": "67b8f9e7b7ab8f6d70df5d12",
      "client_name": "SNEL",
      "category": "Formations",
      "summary": "Sessions de formation technique",
      "location": "Kinshasa",
      "is_public": true,
      "sort_order": 1,
      "created_at": "2026-02-21T09:00:00Z",
      "updated_at": "2026-02-21T09:00:00Z"
    }
  ]
}
```

Admin list response (pagination):

```json
{
  "items": [],
  "limit": 20,
  "offset": 0,
  "total": 0
}
```

