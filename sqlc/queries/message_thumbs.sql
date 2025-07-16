-- name: CreateMessageThumb :one
INSERT INTO message_thumbs (message_id, user_id)
VALUES ($1, $2)
ON CONFLICT (message_id, user_id) DO NOTHING
RETURNING *;

-- name: DeleteMessageThumb :exec
DELETE FROM message_thumbs
WHERE message_id = $1 AND user_id = $2;

-- name: GetMessageThumbsByUser :many
SELECT mt.*, m.post_slug
FROM message_thumbs mt
JOIN messages m ON mt.message_id = m.id
WHERE mt.user_id = $1
ORDER BY mt.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CheckUserThumbedMessage :one
SELECT EXISTS(
    SELECT 1 FROM message_thumbs
    WHERE message_id = $1 AND user_id = $2
);

-- name: GetThumbCountForMessage :one
SELECT COUNT(*) FROM message_thumbs
WHERE message_id = $1;

-- name: GetMessageThumbsWithUsers :many
SELECT mt.*, u.name as user_name, u.picture_url as user_picture_url
FROM message_thumbs mt
JOIN users u ON mt.user_id = u.id
WHERE mt.message_id = $1
ORDER BY mt.created_at DESC
LIMIT $2 OFFSET $3; 