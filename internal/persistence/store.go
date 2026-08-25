package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
)

type Store struct {
	db                 *sql.DB
	cacheMu            sync.RWMutex
	idempotencyCache   map[string]cachedIdempotency
	pendingIdempotency []pendingIdempotency
}
type Tx struct {
	tx    *sql.Tx
	store *Store
}

func Open(path string) (*Store, error) {
	dsn := path
	if path == "" {
		dsn = "file:quality-gate.db?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("迁移数据库: %w", err)
	}
	return &Store{db: db, idempotencyCache: make(map[string]cachedIdempotency)}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Transact(ctx context.Context, fn func(*Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	wrapped := &Tx{tx: tx, store: s}
	if err := fn(wrapped); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.publishIdempotency()
	return nil
}

func (s *Store) DB() *sql.DB { return s.db }
