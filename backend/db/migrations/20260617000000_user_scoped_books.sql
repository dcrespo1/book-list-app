-- +goose Up
-- +goose StatementBegin

ALTER TABLE books
    ADD COLUMN user_id TEXT NOT NULL DEFAULT 'migrated',
    ADD COLUMN status  TEXT NOT NULL DEFAULT 'want_to_read',
    ADD COLUMN rating  INTEGER CHECK (rating BETWEEN 1 AND 5),
    ADD COLUMN notes   TEXT;

-- Replace single-column unique constraint with composite so the same book
-- can appear in multiple users' lists.
ALTER TABLE books DROP CONSTRAINT books_work_id_key;
ALTER TABLE books ADD CONSTRAINT books_work_id_user_id_key UNIQUE (work_id, user_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

ALTER TABLE books DROP CONSTRAINT books_work_id_user_id_key;
ALTER TABLE books ADD CONSTRAINT books_work_id_key UNIQUE (work_id);
ALTER TABLE books
    DROP COLUMN notes,
    DROP COLUMN rating,
    DROP COLUMN status,
    DROP COLUMN user_id;

-- +goose StatementEnd
