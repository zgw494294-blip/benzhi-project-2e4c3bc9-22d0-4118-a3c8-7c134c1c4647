package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/benzhi-project/ancient-quality-gate/internal/domain"
)

type cachedIdempotency struct {
	requestHash string
	response    []byte
}

type pendingIdempotency struct {
	key   string
	entry cachedIdempotency
}

func idempotencyCacheKey(operation, key string) string {
	return operation + "\x00" + key
}

func (t *Tx) IdempotentResponse(ctx context.Context, operation, key, hash string) ([]byte, bool, error) {
	cacheKey := idempotencyCacheKey(operation, key)
	t.store.cacheMu.RLock()
	cached, ok := t.store.idempotencyCache[cacheKey]
	t.store.cacheMu.RUnlock()
	if ok {
		if cached.requestHash != hash {
			return nil, false, domain.Conflict("同一 idempotencyKey 对应了不同请求")
		}
		return append([]byte(nil), cached.response...), true, nil
	}

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
	pending := pendingIdempotency{
		key: idempotencyCacheKey(operation, key),
		entry: cachedIdempotency{
			requestHash: hash,
			response:    append([]byte(nil), response...),
		},
	}
	t.store.cacheMu.Lock()
	t.store.pendingIdempotency = append(t.store.pendingIdempotency, pending)
	t.store.cacheMu.Unlock()
	_, e := t.tx.ExecContext(ctx, `INSERT INTO idempotency(operation,idempotency_key,request_hash,response,created_at) VALUES(?,?,?,?,?)`, operation, key, hash, string(response), time.Now().UTC().Format(time.RFC3339Nano))
	return e
}

func (s *Store) publishIdempotency() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	for _, pending := range s.pendingIdempotency {
		s.idempotencyCache[pending.key] = pending.entry
	}
	s.pendingIdempotency = nil
}
