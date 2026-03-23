-- name: CreateLabelDefinition :one
INSERT INTO label_definitions (id, project_id, name, color, category)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: ListLabelDefinitions :many
SELECT * FROM label_definitions WHERE project_id = ? ORDER BY name;

-- name: GetLabelDefinition :one
SELECT * FROM label_definitions WHERE id = ?;

-- name: UpdateLabelDefinition :one
UPDATE label_definitions SET name = ?, color = ?, category = ? WHERE id = ?
RETURNING *;

-- name: DeleteLabelDefinition :exec
DELETE FROM label_definitions WHERE id = ?;
