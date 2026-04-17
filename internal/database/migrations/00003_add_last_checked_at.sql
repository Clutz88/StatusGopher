-- +goose Up

-- +goose StatementBegin
ALTER TABLE sites ADD COLUMN last_checked_at DATETIME;
-- +goose StatementEnd


-- +goose Down

-- +goose StatementBegin
ALTER TABLE sites DROP COLUMN last_checked_at;
-- +goose StatementEnd
