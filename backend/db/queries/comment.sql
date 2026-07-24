-- name: CreateComment :one
INSERT INTO comments (id, image_id, annotation_id, author, body, type)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListCommentsByImage :many
SELECT
    sqlc.embed(comments),
    CASE
        WHEN comments.annotation_id IS NOT NULL AND annotations.id IS NULL THEN 1
        ELSE 0
    END AS target_deleted
FROM comments
LEFT JOIN annotations ON annotations.id = comments.annotation_id
WHERE comments.image_id = ?
ORDER BY comments.created_at, comments.id;

-- name: GetComment :one
SELECT
    sqlc.embed(comments),
    CASE
        WHEN comments.annotation_id IS NOT NULL AND annotations.id IS NULL THEN 1
        ELSE 0
    END AS target_deleted
FROM comments
LEFT JOIN annotations ON annotations.id = comments.annotation_id
WHERE comments.id = ? AND comments.image_id = ?;

-- name: SetCommentResolved :execrows
UPDATE comments
SET resolved = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE id = ? AND image_id = ?;

-- name: DeleteComment :execrows
DELETE FROM comments
WHERE id = ? AND image_id = ?;
