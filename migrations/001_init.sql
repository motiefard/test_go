CREATE TABLE IF NOT EXISTS inquiry_audits (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    request_id TEXT NOT NULL UNIQUE,
    inquired_at TEXT NOT NULL,
    duration_ms INTEGER NOT NULL,
    client_ip TEXT,
    http_status INTEGER NOT NULL,
    external_http_status INTEGER,
    masked_card TEXT NOT NULL,
    error_code TEXT,
    raw_response TEXT
);

CREATE INDEX IF NOT EXISTS idx_inquiry_audits_inquired_at ON inquiry_audits (inquired_at);
