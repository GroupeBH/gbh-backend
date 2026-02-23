DROP TRIGGER IF EXISTS trg_case_studies_updated_at ON case_studies;
DROP TRIGGER IF EXISTS trg_references_updated_at ON "references";
DROP TRIGGER IF EXISTS trg_rfp_leads_updated_at ON rfp_leads;

DROP TABLE IF EXISTS case_studies;
DROP TABLE IF EXISTS "references";
DROP TABLE IF EXISTS rfp_leads;

DROP FUNCTION IF EXISTS set_updated_at();

