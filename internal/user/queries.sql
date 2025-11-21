-- name: Exist :one
SELECT EXISTS (SELECT 1 FROM users WHERE id = $1) AS exists;

-- name: ExistsByEmail :one
SELECT EXISTS (SELECT 1 FROM users WHERE email = $1) AS exists;

-- name: Create :one
INSERT INTO users (email, username, avatar_url) VALUES ($1, $2, $3) RETURNING *;

-- name: UpdateAbout :one
UPDATE users SET about_me = $2 WHERE id = $1 RETURNING *;

-- name: GetByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetByEmail :one
SELECT * FROM users WHERE email = $1;