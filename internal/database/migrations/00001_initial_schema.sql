-- +goose Up
CREATE TABLE sites (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    url TEXT UNIQUE NOT NULL,
    added_at DATETIME NOT NULL
);

CREATE TABLE checks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_id INTEGER NOT NULL,
    status_code INTEGER,
    latency_ms INTEGER,
    checked_at DATETIME NOT NULL,
    error_msg TEXT,
    FOREIGN KEY (site_id) REFERENCES sites(id)
);

CREATE INDEX idx_checks_site_id ON checks(site_id);
CREATE INDEX idx_checks_site_id_checked_at ON checks(site_id, checked_at DESC);

-- +goose Down
DROP TABLE checks;
DROP TABLE sites;