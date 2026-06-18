-- name: AddBook :one
INSERT INTO books (user_id, title, authors, subjects, description, cover_art_url, work_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;
