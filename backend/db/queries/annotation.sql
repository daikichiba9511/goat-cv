-- name: CreateAnnotation :one
INSERT INTO annotations (id, image_id, type, coordinates, label_id)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetAnnotation :one
SELECT * FROM annotations WHERE id = ?;

-- name: ListAnnotationsByImage :many
SELECT * FROM annotations WHERE image_id = ? ORDER BY created_at;

-- name: UpdateAnnotation :one
UPDATE annotations SET type = ?, coordinates = ?, label_id = ? WHERE id = ?
RETURNING *;

-- name: DeleteAnnotation :exec
DELETE FROM annotations WHERE id = ?;

-- name: DeleteAnnotationsByImage :exec
DELETE FROM annotations WHERE image_id = ?;
