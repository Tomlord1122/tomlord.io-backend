-- name: CreateMessageThumb :one
INSERT INTO message_thumbs (message_id, user_id)
VALUES ($1, $2)
ON CONFLICT (message_id, user_id) DO NOTHING
RETURNING *;

-- name: DeleteMessageThumb :exec
DELETE FROM message_thumbs
WHERE message_id = $1 AND user_id = $2;

-- name: CheckUserThumbedMessage :one
SELECT EXISTS(
    SELECT 1 FROM message_thumbs
    WHERE message_id = $1 AND user_id = $2
);

-- name: GetThumbCountForMessage :one
SELECT COUNT(*) FROM message_thumbs
WHERE message_id = $1;
