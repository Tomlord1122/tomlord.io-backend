-- name: CreateMessage :one
INSERT INTO messages (user_id, post_slug, message)
VALUES ($1, $2, $3)
RETURNING id, user_id, post_slug, message, thumb_count, created_at, updated_at;

-- name: GetMessageByID :one
SELECT 
  m.id, m.user_id, m.post_slug, m.message, m.thumb_count, m.created_at, m.updated_at,
  u.name AS user_name, u.picture_url AS user_picture_url,
  COALESCE(thumb_counts.count, 0) AS thumb_count
FROM messages m
JOIN users u ON m.user_id = u.id
LEFT JOIN (
  SELECT message_id, COUNT(*) AS count
  FROM message_thumbs
  GROUP BY message_id
) thumb_counts ON m.id = thumb_counts.message_id
WHERE m.id = $1;


-- name: GetMessagesByBlogSlug :many
-- Note: blog slug now maps to post_slug directly
SELECT 
  m.id, m.user_id, m.post_slug, m.message, m.thumb_count, m.created_at, m.updated_at,
  u.name AS user_name, u.picture_url AS user_picture_url,
  COALESCE(thumb_counts.count, 0) AS thumb_count
FROM messages m
JOIN users u ON m.user_id = u.id
LEFT JOIN (
  SELECT message_id, COUNT(*) AS count
  FROM message_thumbs
  GROUP BY message_id
) thumb_counts ON m.id = thumb_counts.message_id
WHERE m.post_slug = $1
ORDER BY m.created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateMessage :one
UPDATE messages
SET message = $2, updated_at = NOW()
WHERE id = $1 AND user_id = $3
RETURNING id, user_id, post_slug, message, thumb_count, created_at, updated_at;

-- name: DeleteMessage :exec
DELETE FROM messages
WHERE id = $1 AND user_id = $2;

-- name: DeleteMessageBySuperUser :exec
DELETE FROM messages
WHERE id = $1;

-- name: UpdateMessageThumbCount :exec
UPDATE messages 
SET thumb_count = (
  SELECT COUNT(*) 
  FROM message_thumbs 
  WHERE message_id = $1
)
WHERE id = $1; 