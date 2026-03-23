-- name: CreateImage :one
INSERT INTO images (id, project_id, filename, original_width, original_height, width, height, rotation, flip_h, flip_v)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetImage :one
SELECT * FROM images WHERE id = ?;

-- name: ListImagesByProject :many
SELECT * FROM images WHERE project_id = ? ORDER BY uploaded_at DESC;

-- name: ListImagesByProjectAndStatus :many
SELECT * FROM images WHERE project_id = ? AND status = ? ORDER BY uploaded_at DESC;

-- name: UpdateImageTransform :one
UPDATE images SET
    rotation = ?,
    flip_h = ?,
    flip_v = ?,
    width = ?,
    height = ?
WHERE id = ?
RETURNING *;

-- name: UpdateImageStatus :one
UPDATE images SET status = ? WHERE id = ?
RETURNING *;

-- name: DeleteImage :exec
DELETE FROM images WHERE id = ?;
