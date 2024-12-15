-- name: GetBookByWorkID :one
SELECT * FROM books WHERE work_id = $1;
