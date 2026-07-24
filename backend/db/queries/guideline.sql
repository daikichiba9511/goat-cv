-- name: CreateGuideline :one
INSERT INTO guidelines (id, project_id, title, body, display_order)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: ListGuidelinesByProject :many
SELECT * FROM guidelines
WHERE project_id = ?
ORDER BY display_order, title, id;

-- name: GetGuideline :one
SELECT * FROM guidelines
WHERE id = ? AND project_id = ?;

-- name: UpdateGuideline :one
UPDATE guidelines
SET title = ?,
    body = ?,
    display_order = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE id = ? AND project_id = ?
RETURNING *;

-- name: DeleteGuideline :execrows
DELETE FROM guidelines
WHERE id = ? AND project_id = ?;
