-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;

-- name: GetAllUsers :many
SELECT id, created_at, updated_at, email, hashed_password FROM users
ORDER BY created_at DESC;

-- name: GetUserByEmail :one
SELECT id, created_at, updated_at, email, hashed_password
FROM users
WHERE email = $1;

-- name: DeleteAllUsers :exec
DELETE FROM users;