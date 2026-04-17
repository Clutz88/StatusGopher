-- +goose Up
-- +goose StatementBegin
CREATE TABLE sites (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    url TEXT UNIQUE NOT NULL,
    added_at DATETIME NOT NULL
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE checks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    status_code INTEGER,
    latency_ms INTEGER,
    checked_at DATETIME NOT NULL,
    error_msg TEXT,
    FOREIGN KEY (site_id) REFERENCES sites(id)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_checks_site_id ON checks(site_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX idx_checks_site_id_checked_at ON checks(site_id, checked_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE checks;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE sites;
-- +goose StatementEnd