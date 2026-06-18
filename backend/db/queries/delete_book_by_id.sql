-- name: DeleteBookByID :exec
DELETE FROM books WHERE id = $1 AND user_id = $2;
