package db

import (
	_ "embed"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

type DB struct {
	*sql.DB
}

func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", path)
	raw, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	raw.SetMaxOpenConns(8)
	raw.SetMaxIdleConns(4)
	raw.SetConnMaxLifetime(time.Hour)

	if err := raw.Ping(); err != nil {
		return nil, err
	}
	if _, err := raw.Exec(schemaSQL); err != nil {
		return nil, fmt.Errorf("执行 schema 失败: %w", err)
	}
	return &DB{raw}, nil
}

func Now() int64 { return time.Now().Unix() }
