-- name: GetAll :many
SELECT * FROM tasks ORDER BY id ASC;

-- name: GetByID :one
SELECT * FROM tasks WHERE id = $1;

-- name: Create :one
INSERT INTO tasks (title)
VALUES ($1)
RETURNING *;

-- name: Update :one
UPDATE tasks
SET labels = $2, title = $3, description = $4, status = $5, due_date = $6, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: Delete :exec
DELETE FROM tasks WHERE id = $1;