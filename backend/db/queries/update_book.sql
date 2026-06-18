-- name: UpdateBook :one
UPDATE books
SET status = $1,
    rating = $2,
    notes  = $3
WHERE id = $4 AND user_id = $5
RETURNING *;
