-- Drop indexes
DROP INDEX IF EXISTS "idx_analytics_template_id";
DROP INDEX IF EXISTS "idx_analytics_timestamp";
DROP INDEX IF EXISTS "template_versions_template_id_version_number_key";
DROP INDEX IF EXISTS "idx_template_versions_created_at";
DROP INDEX IF EXISTS "idx_template_versions_template_id";
DROP INDEX IF EXISTS "templates_name_key";

-- Drop tables
DROP TABLE IF EXISTS "analytics";
DROP TABLE IF EXISTS "jobs";
DROP TABLE IF EXISTS "template_versions";
DROP TABLE IF EXISTS "templates";


