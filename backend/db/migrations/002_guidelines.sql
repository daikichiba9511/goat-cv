CREATE TABLE IF NOT EXISTS guidelines (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    display_order INTEGER NOT NULL CHECK (display_order >= 0),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_guidelines_project_order
ON guidelines(project_id, display_order, title, id);
