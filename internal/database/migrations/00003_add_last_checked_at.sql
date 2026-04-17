-- +goose Up
ALTER TABLE sites ADD COLUMN last_checked_at DATETIME;

-- +goose Down
ALTER TABLE sites DROP COLUMN last_checked_at;