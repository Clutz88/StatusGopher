-- +goose Up

-- +goose StatementBegin
ALTER TABLE sites ADD COLUMN body_match TEXT;
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
ALTER TABLE sites DROP COLUMN body_match;
-- +goose StatementEnd