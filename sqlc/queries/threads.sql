-- name: CreateThread :one
INSERT INTO threads (content)
VALUES ($1)
RETURNING *;

-- name: GetThreadByID :one
SELECT * FROM threads
WHERE id = $1;

-- name: ListThreads :many
SELECT * FROM threads
WHERE (
    sqlc.narg('cursor_created_at')::timestamptz IS NULL
    OR (created_at, id) < (sqlc.narg('cursor_created_at')::timestamptz, sqlc.narg('cursor_id')::uuid)
)
ORDER BY created_at DESC, id DESC
LIMIT $1;

-- name: UpdateThread :one
UPDATE threads
SET content = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteThread :exec
DELETE FROM threads
WHERE id = $1;
