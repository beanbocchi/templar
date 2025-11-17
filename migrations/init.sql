-- CreateEnum
CREATE TYPE Status AS ENUM ('Pending', 'Processing', 'Success', 'Canceled', 'Failed');

-- Templates table
CREATE TABLE templates (
    id UUID PRIMARY KEY, -- uuid
    name TEXT UNIQUE NOT NULL,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Template versions table
CREATE TABLE template_versions (
    id UUID PRIMARY KEY, -- uuid
    template_id UUID NOT NULL,
    version_number INTEGER NOT NULL,
    object_key TEXT NOT NULL, -- Object key in the object store system
    file_size INTEGER, -- In bytes
    file_hash TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (template_id) REFERENCES templates(id) ON UPDATE CASCADE ON DELETE CASCADE,
    UNIQUE(template_id, version_number)
);

-- Job table
CREATE TABLE jobs (
    id BIGINT PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL,
    template_id UUID NOT NULL,
    status Status DEFAULT 'pending',
    progress INTEGER DEFAULT 0,
    started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    error_message TEXT,
    metadata JSONB NOT NULL,
    FOREIGN KEY (template_id) REFERENCES templates(id) ON UPDATE CASCADE ON DELETE CASCADE,
    FOREIGN KEY (version_id) REFERENCES template_versions(id) ON UPDATE CASCADE ON DELETE CASCADE
);

-- Simple analytics table
CREATE TABLE analytics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    template_id UUID NOT NULL,
    version_id UUID NOT NULL,
    action TEXT NOT NULL, -- 'upload', 'download', ...
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    status Status NOT NULL,
    FOREIGN KEY (template_id) REFERENCES templates(id) ON UPDATE CASCADE,
    FOREIGN KEY (version_id) REFERENCES template_versions(id) ON UPDATE CASCADE
);

-- Indexes for performance
CREATE INDEX idx_template_versions_template_id ON template_versions(template_id);
CREATE INDEX idx_template_versions_created_at ON template_versions(created_at);
CREATE INDEX idx_cache_entries_version_id ON cache_entries(version_id);
CREATE INDEX idx_analytics_timestamp ON analytics(timestamp);
CREATE INDEX idx_analytics_template_id ON analytics(template_id);
