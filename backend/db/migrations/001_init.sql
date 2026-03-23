CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS label_definitions (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    color TEXT NOT NULL,
    category TEXT NOT NULL CHECK (category IN ('object', 'entity', 'key', 'value', 'table', 'cell')),
    UNIQUE (project_id, name)
);

CREATE TABLE IF NOT EXISTS images (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    original_width INTEGER NOT NULL,
    original_height INTEGER NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    rotation INTEGER NOT NULL DEFAULT 0 CHECK (rotation IN (0, 90, 180, 270)),
    flip_h BOOLEAN NOT NULL DEFAULT 0,
    flip_v BOOLEAN NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'annotated', 'in_review', 'approved', 'rejected', 'escalated')),
    uploaded_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS annotations (
    id TEXT PRIMARY KEY,
    image_id TEXT NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('bbox', 'polygon')),
    coordinates TEXT NOT NULL,
    label_id TEXT REFERENCES label_definitions(id) ON DELETE SET NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS edges (
    id TEXT PRIMARY KEY,
    image_id TEXT NOT NULL REFERENCES images(id) ON DELETE CASCADE,
    source_annotation_id TEXT NOT NULL REFERENCES annotations(id) ON DELETE CASCADE,
    target_annotation_id TEXT NOT NULL REFERENCES annotations(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK (type IN ('reading_order', 'key_value', 'table_cell')),
    CHECK (source_annotation_id != target_annotation_id),
    UNIQUE (source_annotation_id, target_annotation_id, type)
);

CREATE INDEX IF NOT EXISTS idx_images_project_id ON images(project_id);
CREATE INDEX IF NOT EXISTS idx_annotations_image_id ON annotations(image_id);
CREATE INDEX IF NOT EXISTS idx_edges_image_id ON edges(image_id);
CREATE INDEX IF NOT EXISTS idx_label_definitions_project_id ON label_definitions(project_id);
