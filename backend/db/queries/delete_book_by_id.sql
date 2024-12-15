-- name: DeleteBookByID :exec
DELETE FROM books WHERE id = $1;
