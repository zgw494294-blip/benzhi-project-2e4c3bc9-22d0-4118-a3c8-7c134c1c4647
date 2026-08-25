package application

import (
	"context"
	"encoding/json"

	"github.com/benzhi-project/ancient-quality-gate/internal/domain"
	"github.com/benzhi-project/ancient-quality-gate/internal/persistence"
)

type AddPageCommand struct {
	BatchID         string  `json:"-"`
	PageID          string  `json:"pageID"`
	Sequence        int     `json:"sequence"`
	ImageDigest     string  `json:"imageDigest"`
	OCRText         string  `json:"ocrText"`
	CharacterCount  int     `json:"characterCount"`
	Confidence      float64 `json:"confidence"`
	ExpectedVersion int64   `json:"expectedVersion"`
	IdempotencyKey  string  `json:"-"`
}
type AddOCRCommand struct {
	BatchID         string  `json:"-"`
	PageID          string  `json:"pageID"`
	OCRText         string  `json:"ocrText"`
	CharacterCount  int     `json:"characterCount"`
	Confidence      float64 `json:"confidence"`
	ExpectedVersion int64   `json:"expectedVersion"`
	IdempotencyKey  string  `json:"-"`
}

func (s *Service) AddPage(ctx context.Context, c AddPageCommand) (BatchResult, error) {
	var out BatchResult
	hash := requestHash(c)
	err := s.Store.Transact(ctx, func(tx *persistence.Tx) error {
		if raw, ok, e := tx.IdempotentResponse(ctx, "add-page:"+c.BatchID, c.IdempotencyKey, hash); e != nil {
			return e
		} else if ok {
			return json.Unmarshal(raw, &out)
		}
		b, e := tx.GetBatch(ctx, c.BatchID)
		if e != nil {
			return e
		}
		if b.Version != c.ExpectedVersion {
			return domain.VersionConflict("expectedVersion 与当前版本不一致")
		}
		p := domain.PageEvidence{PageID: c.PageID, BatchID: c.BatchID, Sequence: c.Sequence, ImageDigest: c.ImageDigest, OCRText: c.OCRText, CharacterCount: c.CharacterCount, Confidence: c.Confidence, ObservedAt: s.Now(), Revision: 1}
		existing, e := tx.ListLatestPages(ctx, c.BatchID)
		if e != nil {
			return e
		}
		if e = domain.ValidatePageRegistration(p, existing); e != nil {
			return e
		}
		id, e := tx.InsertPage(ctx, p)
		if e != nil {
			return e
		}
		expected := b.Version
		if e = b.MarkEvidence(); e != nil {
			return e
		}
		if e = tx.UpdateBatch(ctx, b, expected); e != nil {
			return e
		}
		if e = s.Chain.Append(ctx, tx, b.BatchID, "evidence.page_registered", p); e != nil {
			return e
		}
		out = BatchResult{Batch: b, EvidenceID: id}
		return tx.SaveIdempotentResponse(ctx, "add-page:"+c.BatchID, c.IdempotencyKey, hash, encode(out))
	})
	return out, err
}

func (s *Service) AddOCR(ctx context.Context, c AddOCRCommand) (BatchResult, error) {
	var out BatchResult
	hash := requestHash(c)
	err := s.Store.Transact(ctx, func(tx *persistence.Tx) error {
		if raw, ok, e := tx.IdempotentResponse(ctx, "add-ocr:"+c.BatchID, c.IdempotencyKey, hash); e != nil {
			return e
		} else if ok {
			return json.Unmarshal(raw, &out)
		}
		b, e := tx.GetBatch(ctx, c.BatchID)
		if e != nil {
			return e
		}
		if b.Version != c.ExpectedVersion {
			return domain.VersionConflict("expectedVersion 与当前版本不一致")
		}
		old, e := tx.LatestPage(ctx, c.BatchID, c.PageID)
		if e != nil {
			return e
		}
		p := old
		p.OCRText = c.OCRText
		p.CharacterCount = c.CharacterCount
		p.Confidence = c.Confidence
		p.ObservedAt = s.Now()
		p.Revision++
		if e = p.Validate(); e != nil {
			return e
		}
		id, e := tx.InsertPage(ctx, p)
		if e != nil {
			return e
		}
		expected := b.Version
		if e = b.MarkEvidenceRevision(); e != nil {
			return e
		}
		if e = tx.UpdateBatch(ctx, b, expected); e != nil {
			return e
		}
		if e = s.Chain.Append(ctx, tx, b.BatchID, "evidence.ocr_revised", p); e != nil {
			return e
		}
		out = BatchResult{Batch: b, EvidenceID: id}
		return tx.SaveIdempotentResponse(ctx, "add-ocr:"+c.BatchID, c.IdempotencyKey, hash, encode(out))
	})
	return out, err
}
