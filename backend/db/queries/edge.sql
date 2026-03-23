-- name: CreateEdge :one
INSERT INTO edges (id, image_id, source_annotation_id, target_annotation_id, type)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: ListEdgesByImage :many
SELECT * FROM edges WHERE image_id = ? ORDER BY id;

-- name: DeleteEdge :exec
DELETE FROM edges WHERE id = ?;

-- name: DeleteEdgesByImage :exec
DELETE FROM edges WHERE image_id = ?;
