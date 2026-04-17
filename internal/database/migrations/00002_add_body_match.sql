-- +goose Up
ALTER TABLE sites ADD COLUMN body_match TEXT;

-- +goose Down
ALTER TABLE sites DROP COLUMN body_match;