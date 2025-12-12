-- name: GetPageByName :one
SELECT * FROM pages
WHERE name = $1;

-- name: UpsertPage :one
INSERT INTO pages (name, content)
VALUES ($1, $2)
ON CONFLICT (name) DO UPDATE
SET content = EXCLUDED.content, updated_at = NOW()
RETURNING *;

-- name: ListPages :many
SELECT * FROM pages
ORDER BY updated_at DESC
LIMIT $1 OFFSET $2;

-- name: DeletePageByName :exec
DELETE FROM pages
WHERE name = $1;
