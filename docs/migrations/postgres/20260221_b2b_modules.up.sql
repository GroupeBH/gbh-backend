-- B2B modules schema for PostgreSQL.
-- Note: table "references" is quoted because REFERENCES is a SQL keyword.

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS rfp_leads (
  id BIGSERIAL PRIMARY KEY,
  organization TEXT NOT NULL,
  sector TEXT,
  domain TEXT NOT NULL,
  deadline TEXT,
  budget_range TEXT,
  contact_name TEXT,
  phone TEXT NOT NULL,
  email TEXT,
  description TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'new' CHECK (status IN ('new', 'reviewing', 'qualified', 'won', 'lost')),
  source TEXT NOT NULL DEFAULT 'website' CHECK (source IN ('website', 'whatsapp', 'manual')),
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rfp_leads_status_created_at
  ON rfp_leads (status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_rfp_leads_source_created_at
  ON rfp_leads (source, created_at DESC);

CREATE TABLE IF NOT EXISTS "references" (
  id BIGSERIAL PRIMARY KEY,
  client_name TEXT NOT NULL,
  category TEXT NOT NULL,
  summary TEXT NOT NULL,
  location TEXT NOT NULL,
  logo_url TEXT,
  is_public BOOLEAN NOT NULL DEFAULT TRUE,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_references_public_category_sort
  ON "references" (is_public, category, sort_order, created_at DESC);

CREATE TABLE IF NOT EXISTS case_studies (
  id BIGSERIAL PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  title TEXT NOT NULL,
  category TEXT NOT NULL,
  client_name TEXT NOT NULL,
  problem TEXT NOT NULL,
  solution TEXT NOT NULL,
  result TEXT NOT NULL,
  is_published BOOLEAN NOT NULL DEFAULT FALSE,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_case_studies_public_category_sort
  ON case_studies (is_published, category, sort_order, created_at DESC);

DROP TRIGGER IF EXISTS trg_rfp_leads_updated_at ON rfp_leads;
CREATE TRIGGER trg_rfp_leads_updated_at
BEFORE UPDATE ON rfp_leads
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_references_updated_at ON "references";
CREATE TRIGGER trg_references_updated_at
BEFORE UPDATE ON "references"
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

DROP TRIGGER IF EXISTS trg_case_studies_updated_at ON case_studies;
CREATE TRIGGER trg_case_studies_updated_at
BEFORE UPDATE ON case_studies
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

