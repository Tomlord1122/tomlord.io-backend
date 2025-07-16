-- name: CreateBlog :one
INSERT INTO blogs (title, slug, date, lang, duration, tags, description, is_published)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetBlogByID :one
SELECT * FROM blogs
WHERE id = $1;

-- name: GetBlogBySlug :one
SELECT * FROM blogs
WHERE slug = $1;

-- name: GetBlogs :many
SELECT * FROM blogs
WHERE is_published = true
ORDER BY date DESC
LIMIT $1 OFFSET $2;

-- name: GetBlogsByTag :many
SELECT * FROM blogs
WHERE is_published = true AND $1 = ANY(tags)
ORDER BY date DESC
LIMIT $2 OFFSET $3;

-- name: GetBlogsByLang :many
SELECT * FROM blogs
WHERE is_published = true AND lang = $1
ORDER BY date DESC
LIMIT $2 OFFSET $3;

-- name: GetAllBlogs :many
SELECT * FROM blogs
ORDER BY date DESC
LIMIT $1 OFFSET $2;

-- name: UpdateBlog :one
UPDATE blogs
SET title = $2, date = $3, lang = $4, duration = $5, tags = $6, description = $7, is_published = $8, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateBlogBySlug :one
UPDATE blogs
SET title = $2, date = $3, lang = $4, duration = $5, tags = $6, description = $7, is_published = $8, updated_at = NOW()
WHERE slug = $1
RETURNING *;

-- name: DeleteBlog :exec
DELETE FROM blogs
WHERE id = $1;

-- name: DeleteBlogBySlug :exec
DELETE FROM blogs
WHERE slug = $1;

-- name: CountBlogs :one
SELECT COUNT(*) FROM blogs
WHERE is_published = true;

-- name: CountBlogsByTag :one
SELECT COUNT(*) FROM blogs
WHERE is_published = true AND $1 = ANY(tags);

-- name: GetBlogWithMessageCount :one
SELECT 
    b.*,
    COALESCE(msg_count.count, 0) as message_count
FROM blogs b
LEFT JOIN (
    SELECT blog_id, COUNT(*) as count
    FROM messages
    GROUP BY blog_id
) msg_count ON b.id = msg_count.blog_id
WHERE b.id = $1;

-- name: GetBlogWithMessageCountBySlug :one
SELECT 
    b.*,
    COALESCE(msg_count.count, 0) as message_count
FROM blogs b
LEFT JOIN (
    SELECT blog_id, COUNT(*) as count
    FROM messages
    GROUP BY blog_id
) msg_count ON b.id = msg_count.blog_id
WHERE b.slug = $1; 