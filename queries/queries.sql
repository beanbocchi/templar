-- name: GetTemplate :one
SELECT * FROM templates
WHERE "id" = ?;

-- name: ListTemplates :many
SELECT * FROM templates
WHERE (
    "name" LIKE '%' || sqlc.narg('search') || '%' OR
    "description" LIKE '%' || sqlc.narg('search') || '%'  OR
    sqlc.narg('search') IS NULL 
)
ORDER BY created_at DESC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');

-- name: CreateTemplate :one
INSERT INTO "templates" ("id", "name", "description")
VALUES (?, ?, ?)
RETURNING *;

-- name: UpdateTemplate :one
UPDATE templates 
SET "name" = ?, "description" = ? 
WHERE "id" = ? 
RETURNING *;

-- name: DeleteTemplate :exec
DELETE FROM "templates" WHERE "id" = ?;

-- name: GetTemplateVersion :one
SELECT * FROM "template_versions"
WHERE (
  ("template_id" = ? AND "version_number" = ?) OR
  ("object_key" = ?)
);

-- name: ListTemplateVersions :many
SELECT * FROM "template_versions"
WHERE "template_id" = ?;

-- name: CreateTemplateVersion :one
INSERT INTO "template_versions" ("id", "template_id", "version_number", "object_key", "file_size", "file_hash")
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: DeleteTemplateVersion :exec
DELETE FROM "template_versions" WHERE "id" = ?;

-- name: ListJobs :many
SELECT * FROM "jobs"
ORDER BY "created_at" DESC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');

-- name: CreateJob :one
INSERT INTO "jobs" ("type", "template_id", "version_number", "status", "progress", "started_at", "completed_at", "error_message", "metadata")
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateJob :one
UPDATE jobs
SET 
  "status" = COALESCE(sqlc.narg('status'), "status"),
  "progress" = COALESCE(sqlc.narg('progress'), "progress"),
  "completed_at" = COALESCE(sqlc.narg('completed_at'), "completed_at"),
  "error_message" = COALESCE(sqlc.narg('error_message'), "error_message")
WHERE "id" = sqlc.arg('id')
RETURNING *;

-- name: DeleteJob :exec
DELETE FROM "jobs" WHERE "id" = ?;

-- name: ListAnalytics :many
SELECT * FROM "analytics"
ORDER BY "timestamp" DESC
LIMIT ?
OFFSET ?;

-- name: CreateAnalytic :one
INSERT INTO "analytics" ("id", "template_id", "version_id", "action", "timestamp", "status")
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- -- CreateTable
-- CREATE TABLE "templates" (
--     "id" TEXT NOT NULL PRIMARY KEY,
--     "name" TEXT NOT NULL,
--     "description" TEXT,
--     "created_at" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
--     "updated_at" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
-- );

-- -- CreateTable
-- CREATE TABLE "template_versions" (
--     "id" TEXT NOT NULL PRIMARY KEY,
--     "template_id" TEXT NOT NULL,
--     "version_number" INTEGER NOT NULL,
--     "object_key" TEXT NOT NULL,
--     "file_size" INTEGER,
--     "file_hash" TEXT,
--     "created_at" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
--     CONSTRAINT "template_versions_template_id_fkey" FOREIGN KEY ("template_id") REFERENCES "templates" ("id") ON DELETE CASCADE ON UPDATE CASCADE
-- );

-- -- CreateTable
-- CREATE TABLE "jobs" (
--     "id" BIGINT NOT NULL PRIMARY KEY,
--     "type" TEXT NOT NULL,
--     "template_id" TEXT NOT NULL,
--     "version_id" TEXT,
--     "status" TEXT NOT NULL DEFAULT 'Pending',
--     "progress" INTEGER NOT NULL DEFAULT 0,
--     "started_at" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
--     "completed_at" DATETIME,
--     "error_message" TEXT,
--     "metadata" TEXT NOT NULL,
--     CONSTRAINT "jobs_template_id_fkey" FOREIGN KEY ("template_id") REFERENCES "templates" ("id") ON DELETE CASCADE ON UPDATE CASCADE,
--     CONSTRAINT "jobs_version_id_fkey" FOREIGN KEY ("version_id") REFERENCES "template_versions" ("id") ON DELETE CASCADE ON UPDATE CASCADE
-- );

-- -- CreateTable
-- CREATE TABLE "analytics" (
--     "id" INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
--     "template_id" TEXT NOT NULL,
--     "version_id" TEXT NOT NULL,
--     "action" TEXT NOT NULL,
--     "timestamp" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
--     "status" TEXT NOT NULL,
--     CONSTRAINT "analytics_template_id_fkey" FOREIGN KEY ("template_id") REFERENCES "templates" ("id") ON DELETE RESTRICT ON UPDATE CASCADE,
--     CONSTRAINT "analytics_version_id_fkey" FOREIGN KEY ("version_id") REFERENCES "template_versions" ("id") ON DELETE RESTRICT ON UPDATE CASCADE
-- );

-- -- CreateIndex
-- CREATE UNIQUE INDEX "templates_name_key" ON "templates"("name");

-- -- CreateIndex
-- CREATE INDEX "idx_template_versions_template_id" ON "template_versions"("template_id");

-- -- CreateIndex
-- CREATE INDEX "idx_template_versions_created_at" ON "template_versions"("created_at");

-- -- CreateIndex
-- CREATE UNIQUE INDEX "template_versions_template_id_version_number_key" ON "template_versions"("template_id", "version_number");

-- -- CreateIndex
-- CREATE INDEX "idx_analytics_timestamp" ON "analytics"("timestamp");

-- -- CreateIndex
-- CREATE INDEX "idx_analytics_template_id" ON "analytics"("template_id");