-- +goose Up
-- +goose StatementBegin
CREATE TABLE books (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    authors TEXT NOT NULL,       -- Comma-separated list of authors
    subjects TEXT,               -- Comma-separated list of subjects
    description TEXT,
    cover_art_url TEXT,
    work_id TEXT UNIQUE NOT NULL -- Unique identifier from Open Library
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE books;
-- +goose StatementEnd
