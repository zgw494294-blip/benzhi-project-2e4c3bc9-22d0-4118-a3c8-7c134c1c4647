package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/benzhi-project/ancient-quality-gate/internal/domain"
)

func (t *Tx) IdempotentResponse(ctx context.Context, operation, key, hash string) ([]byte, bool, error) {
	var existingHash, response string
	err := t.tx.QueryRowContext(ctx, `SELECT request_hash,response FROM idempotency WHERE operation=? AND idempotency_key=?`, operation, key).Scan(&existingHash, &response)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if existingHash != hash {
		return nil, false, domain.Conflict("同一 idempotencyKey 对应了不同请求")
	}
	return []byte(response), true, nil
}

func (t *Tx) SaveIdempotentResponse(ctx context.Context, operation, key, hash string, response []byte) error {
	_, e := t.tx.ExecContext(ctx, `INSERT INTO idempotency(operation,idempotency_key,request_hash,response,created_at) VALUES(?,?,?,?,?)`, operation, key, hash, string(response), time.Now().UTC().Format(time.RFC3339Nano))
	return e
}
