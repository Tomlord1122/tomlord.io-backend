-- name: CreateMessage :one
INSERT INTO messages (user_id, post_slug, message)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CreateMessageWithBlogID :one
INSERT INTO messages (user_id, blog_id, post_slug, message)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetMessageByID :one
SELECT m.*, u.name as user_name, u.picture_url as user_picture_url,
       COALESCE(thumb_counts.count, 0) as thumb_count
FROM messages m
JOIN users u ON m.user_id = u.id
LEFT JOIN (
    SELECT message_id, COUNT(*) as count
    FROM message_thumbs
    GROUP BY message_id
) thumb_counts ON m.id = thumb_counts.message_id
WHERE m.id = $1;

-- name: GetMessagesByPostSlug :many
SELECT m.*, u.name as user_name, u.picture_url as user_picture_url,
       COALESCE(thumb_counts.count, 0) as thumb_count
FROM messages m
JOIN users u ON m.user_id = u.id
LEFT JOIN (
    SELECT message_id, COUNT(*) as count
    FROM message_thumbs
    GROUP BY message_id
) thumb_counts ON m.id = thumb_counts.message_id
WHERE m.post_slug = $1
ORDER BY m.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetMessagesByBlogID :many
SELECT m.*, u.name as user_name, u.picture_url as user_picture_url,
       COALESCE(thumb_counts.count, 0) as thumb_count
FROM messages m
JOIN users u ON m.user_id = u.id
LEFT JOIN (
    SELECT message_id, COUNT(*) as count
    FROM message_thumbs
    GROUP BY message_id
) thumb_counts ON m.id = thumb_counts.message_id
WHERE m.blog_id = $1
ORDER BY m.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetMessagesByBlogSlug :many
SELECT m.*, u.name as user_name, u.picture_url as user_picture_url,
       COALESCE(thumb_counts.count, 0) as thumb_count, b.slug as blog_slug
FROM messages m
JOIN users u ON m.user_id = u.id
JOIN blogs b ON m.blog_id = b.id
LEFT JOIN (
    SELECT message_id, COUNT(*) as count
    FROM message_thumbs
    GROUP BY message_id
) thumb_counts ON m.id = thumb_counts.message_id
WHERE b.slug = $1
ORDER BY m.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetMessagesByUser :many
SELECT m.*, u.name as user_name, u.picture_url as user_picture_url
FROM messages m
JOIN users u ON m.user_id = u.id
WHERE m.user_id = $1
ORDER BY m.created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateMessage :one
UPDATE messages
SET message = $2, updated_at = NOW()
WHERE id = $1 AND user_id = $3
RETURNING *;

-- name: DeleteMessage :exec
DELETE FROM messages
WHERE id = $1 AND user_id = $2;

-- name: DeleteMessageBySuperUser :exec
DELETE FROM messages
WHERE id = $1;

-- name: CountMessagesByPostSlug :one
SELECT COUNT(*) FROM messages
WHERE post_slug = $1;

-- name: CountMessagesByBlogID :one
SELECT COUNT(*) FROM messages
WHERE blog_id = $1;

-- name: CountMessagesByBlogSlug :one
SELECT COUNT(*) FROM messages m
JOIN blogs b ON m.blog_id = b.id
WHERE b.slug = $1;

-- name: UpdateMessageThumbCount :exec
UPDATE messages 
SET thumb_count = (
    SELECT COUNT(*) 
    FROM message_thumbs 
    WHERE message_id = $1
)
WHERE id = $1; 