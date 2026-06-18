-- name: GetBookByID :one
SELECT * FROM books WHERE id = $1 AND user_id = $2;
