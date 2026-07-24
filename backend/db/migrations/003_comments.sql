CREATE TABLE IF NOT EXISTS comments (
    id TEXT PRIMARY KEY,
    image_id TEXT NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    -- Why: Annotation削除後も対象IDとQA記録を残し、Image向けCommentと区別できる論理参照にする。
    annotation_id TEXT,
    author TEXT NOT NULL CHECK (length(trim(author)) > 0),
    body TEXT NOT NULL CHECK (length(trim(body)) > 0),
    type TEXT NOT NULL CHECK (type IN ('question', 'issue', 'note')),
    resolved BOOLEAN NOT NULL DEFAULT 0 CHECK (resolved IN (0, 1)),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_comments_image_created
ON comments(image_id, created_at, id);

CREATE INDEX IF NOT EXISTS idx_comments_annotation_id
ON comments(annotation_id);
