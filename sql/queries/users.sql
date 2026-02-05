-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email)
VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: GetAllUsers :many
SELECT id, created_at, updated_at, email FROM users
ORDER BY created_at DESC;

-- name: DeleteAllUsers :exec
DELETE FROM users;