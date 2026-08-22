-- +goose Up
-- Fleetbase-style generic polymorphic file uploads:
-- widen files.uploadable_type CHECK so any entity can attach images/files
-- (trip POD photos, expense receipts, generic attachments), plus an entity
-- index for list-by-entity lookups.

PRAGMA foreign_keys = OFF;

CREATE TABLE files_new (
    id              TEXT PRIMARY KEY,
    filename        TEXT NOT NULL,
    original_name   TEXT NOT NULL,
    path            TEXT NOT NULL,
    size            INTEGER NOT NULL,
    mime_type       TEXT NOT NULL,
    uploadable_type TEXT NOT NULL CHECK (uploadable_type IN
        ('driver_license','vehicle_insurance','vehicle_permit','company_logo',
         'vehicle_rc','vehicle_fitness','vehicle_puc',
         'trip_pod','expense_receipt','logo','general')),
    uploadable_id   TEXT,
    created_at      DATETIME NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO files_new (id, filename, original_name, path, size, mime_type, uploadable_type, uploadable_id, created_at)
SELECT id, filename, original_name, path, size, mime_type, uploadable_type, uploadable_id, created_at FROM files;

DROP TABLE files;

ALTER TABLE files_new RENAME TO files;

CREATE INDEX IF NOT EXISTS idx_files_uploadable ON files(uploadable_type, uploadable_id);

PRAGMA foreign_keys = ON;

-- +goose Down
DROP INDEX IF EXISTS idx_files_uploadable;

-- Remove rows typed by the widened CHECK before restoring the narrower one.
DELETE FROM files WHERE uploadable_type IN ('trip_pod','expense_receipt','logo','general');

PRAGMA foreign_keys = OFF;

CREATE TABLE files_old (
    id              TEXT PRIMARY KEY,
    filename        TEXT NOT NULL,
    original_name   TEXT NOT NULL,
    path            TEXT NOT NULL,
    size            INTEGER NOT NULL,
    mime_type       TEXT NOT NULL,
    uploadable_type TEXT NOT NULL CHECK (uploadable_type IN
        ('driver_license','vehicle_insurance','vehicle_permit','company_logo',
         'vehicle_rc','vehicle_fitness','vehicle_puc')),
    uploadable_id   TEXT,
    created_at      DATETIME NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO files_old (id, filename, original_name, path, size, mime_type, uploadable_type, uploadable_id, created_at)
SELECT id, filename, original_name, path, size, mime_type, uploadable_type, uploadable_id, created_at FROM files;

DROP TABLE files;

ALTER TABLE files_old RENAME TO files;

PRAGMA foreign_keys = ON;
