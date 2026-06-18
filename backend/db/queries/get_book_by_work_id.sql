-- name: GetBookByWorkID :one
SELECT * FROM books WHERE work_id = $1 AND user_id = $2;
