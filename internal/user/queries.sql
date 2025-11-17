-- name: Exist :one
SELECT EXISTS (SELECT 1 FROM users WHERE id = $1) AS exists;

-- name: ExistsByEmail :one
SELECT EXISTS (SELECT 1 FROM users WHERE email = $1) AS exists;

-- name: Create :one
INSERT INTO users (email) VALUES ($1) RETURNING *;

-- name: GetByEmail :one
SELECT * FROM users WHERE email = $1;