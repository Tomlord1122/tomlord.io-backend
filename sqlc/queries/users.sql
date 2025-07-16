-- name: CreateUser :one
INSERT INTO users (google_id, email, name, picture_url)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: GetUserByGoogleID :one
SELECT * FROM users
WHERE google_id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: UpdateUser :one
UPDATE users
SET name = $2, picture_url = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateUserByGoogleID :one
UPDATE users
SET email = $2, name = $3, picture_url = $4, updated_at = NOW()
WHERE google_id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1; 