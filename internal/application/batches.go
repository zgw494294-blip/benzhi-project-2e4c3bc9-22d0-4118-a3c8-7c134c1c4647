package application

import (
	"context"
	"encoding/json"

	"github.com/benzhi-project/ancient-quality-gate/internal/domain"
	"github.com/benzhi-project/ancient-quality-gate/internal/ledger"
	"github.com/benzhi-project/ancient-quality-gate/internal/persistence"
)

type CreateBatchCommand struct {
	Title          string `json:"title"`
	Edition        string `json:"edition"`
	Owner          string `json:"owner"`
	IdempotencyKey string `json:"-"`
}
type BatchResult struct {
	Batch      domain.DigitizationBatch `json:"batch"`
	EvidenceID string                   `json:"evidenceID,omitempty"`
}

func (s *Service) CreateBatch(ctx context.Context, c CreateBatchCommand) (BatchResult, error) {
	var out BatchResult
	hash := requestHash(c)
	err := s.Store.Transact(ctx, func(tx *persistence.Tx) error {
		if raw, ok, e := tx.IdempotentResponse(ctx, "create-batch", c.IdempotencyKey, hash); e != nil {
			return e
		} else if ok {
			return json.Unmarshal(raw, &out)
		}
		now := s.Now()
		b := domain.DigitizationBatch{BatchID: ledger.NewID("batch"), Title: c.Title, Edition: c.Edition, Owner: c.Owner, Status: domain.StatusDraft, Version: 1, CreatedAt: now, UpdatedAt: now}
		if e := b.Validate(); e != nil {
			return e
		}
		if e := tx.InsertBatch(ctx, b); e != nil {
			return e
		}
		if e := s.Chain.Append(ctx, tx, b.BatchID, "batch.created", b); e != nil {
			return e
		}
		out.Batch = b
		return tx.SaveIdempotentResponse(ctx, "create-batch", c.IdempotencyKey, hash, encode(out))
	})
	return out, err
}

func (s *Service) GetBatch(ctx context.Context, id string) (domain.DigitizationBatch, error) {
	return s.Store.GetBatch(ctx, id)
}
