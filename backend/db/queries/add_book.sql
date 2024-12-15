-- name: AddBook :one
INSERT INTO books (title, authors, subjects, description, cover_art_url, work_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;