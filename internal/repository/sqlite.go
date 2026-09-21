package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type AuditRecord struct {
	RequestID          string
	InquiredAt         time.Time
	Duration           time.Duration
	ClientIP           string
	HTTPStatus         int
	ExternalHTTPStatus *int
	MaskedCard         string
	ErrorCode          string
	RawResponse        json.RawMessage
}

type AuditRepository interface {
	Save(ctx context.Context, rec AuditRecord) error
}

type SQLite struct {
	db *sql.DB
}

func Open(dbPath, migrationsPath string) (*SQLite, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &SQLite{db: db}
	if err := s.migrate(migrationsPath); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLite) Close() error { return s.db.Close() }

func (s *SQLite) migrate(dir string) error {
	path := filepath.Join(dir, "001_init.sql")
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read migration: %w", err)
	}
	if _, err := s.db.Exec(string(sqlBytes)); err != nil {
		return fmt.Errorf("apply migration: %w", err)
	}
	return nil
}

func (s *SQLite) Save(ctx context.Context, rec AuditRecord) error {
	var ext any
	if rec.ExternalHTTPStatus != nil {
		ext = *rec.ExternalHTTPStatus
	}
	var raw any
	if len(rec.RawResponse) > 0 {
		raw = string(rec.RawResponse)
	}
	var errCode any
	if rec.ErrorCode != "" {
		errCode = rec.ErrorCode
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO inquiry_audits (
			request_id, inquired_at, duration_ms, client_ip, http_status,
			external_http_status, masked_card, error_code, raw_response
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		rec.RequestID,
		rec.InquiredAt.UTC().Format(time.RFC3339Nano),
		rec.Duration.Milliseconds(),
		rec.ClientIP,
		rec.HTTPStatus,
		ext,
		rec.MaskedCard,
		errCode,
		raw,
	)
	if err != nil {
		return fmt.Errorf("insert audit: %w", err)
	}
	return nil
}
