-- name: GetAllBooks :many
SELECT * FROM books WHERE user_id = $1 ORDER BY id DESC;
