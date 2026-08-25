package application

import (
	"context"
	"encoding/json"

	"github.com/benzhi-project/ancient-quality-gate/internal/domain"
	"github.com/benzhi-project/ancient-quality-gate/internal/ledger"
	"github.com/benzhi-project/ancient-quality-gate/internal/persistence"
)

type ReviewCommand struct {
	BatchID         string `json:"-"`
	Approved        bool   `json:"approved"`
	Reviewer        string `json:"reviewer"`
	Comment         string `json:"comment"`
	ExpectedVersion int64  `json:"expectedVersion"`
	IdempotencyKey  string `json:"-"`
}
type ReviewResponse struct {
	Batch    domain.DigitizationBatch `json:"batch"`
	Decision domain.ReviewDecision    `json:"decision"`
}
type FreezeCommand struct {
	BatchID         string `json:"-"`
	IssuedTo        string `json:"issuedTo"`
	ExpectedVersion int64  `json:"expectedVersion"`
	IdempotencyKey  string `json:"-"`
}
type FreezeResponse struct {
	Batch          domain.DigitizationBatch `json:"batch"`
	SnapshotDigest string                   `json:"snapshotDigest"`
	Credential     domain.ReleaseCredential `json:"credential"`
}
type VerifyResponse struct {
	Valid         bool                     `json:"valid"`
	Reason        string                   `json:"reason"`
	Credential    domain.ReleaseCredential `json:"credential"`
	FrozenVersion int64                    `json:"frozenVersion"`
}

func (s *Service) Review(ctx context.Context, c ReviewCommand) (ReviewResponse, error) {
	var out ReviewResponse
	hash := requestHash(c)
	err := s.Store.Transact(ctx, func(tx *persistence.Tx) error {
		if raw, ok, e := tx.IdempotentResponse(ctx, "review:"+c.BatchID, c.IdempotencyKey, hash); e != nil {
			return e
		} else if ok {
			return json.Unmarshal(raw, &out)
		}
		if c.Reviewer == "" {
			return domain.Invalid("reviewer 不能为空", "reviewer")
		}
		b, e := tx.GetBatch(ctx, c.BatchID)
		if e != nil {
			return e
		}
		if b.Version != c.ExpectedVersion {
			return domain.VersionConflict("expectedVersion 与当前版本不一致")
		}
		if !b.CanReview() {
			return domain.StateError("当前状态不能专家复核")
		}
		quality, e := tx.LatestQuality(ctx, c.BatchID)
		if e != nil {
			return e
		}
		issues, e := tx.ListIssues(ctx, c.BatchID)
		if e != nil {
			return e
		}
		if c.Approved {
			if quality == nil || !quality.Passed {
				return domain.QualityError("质量门禁尚未通过")
			}
			for _, i := range issues {
				if i.ResolvedAt == nil && (i.Severity == "critical" || i.Severity == "major") {
					return domain.QualityError("仍有严重问题未关闭")
				}
			}
		}
		expected := b.Version
		if c.Approved {
			e = b.Approve()
		} else {
			e = b.Reject()
		}
		if e != nil {
			return e
		}
		b.UpdatedAt = s.Now()
		if e = tx.UpdateBatch(ctx, b, expected); e != nil {
			return e
		}
		decision := domain.ReviewDecision{Approved: c.Approved, Reviewer: c.Reviewer, Comment: c.Comment, ReviewedAt: s.Now()}
		if e = tx.InsertReview(ctx, ledger.NewID("review"), b.BatchID, decision); e != nil {
			return e
		}
		if e = s.Chain.Append(ctx, tx, b.BatchID, "review.decided", decision); e != nil {
			return e
		}
		out = ReviewResponse{Batch: b, Decision: decision}
		return tx.SaveIdempotentResponse(ctx, "review:"+c.BatchID, c.IdempotencyKey, hash, encode(out))
	})
	return out, err
}

func (s *Service) Freeze(ctx context.Context, c FreezeCommand) (FreezeResponse, error) {
	var out FreezeResponse
	hash := requestHash(c)
	err := s.Store.Transact(ctx, func(tx *persistence.Tx) error {
		if raw, ok, e := tx.IdempotentResponse(ctx, "freeze:"+c.BatchID, c.IdempotencyKey, hash); e != nil {
			return e
		} else if ok {
			return json.Unmarshal(raw, &out)
		}
		if c.IssuedTo == "" {
			return domain.Invalid("issuedTo 不能为空", "issuedTo")
		}
		b, e := tx.GetBatch(ctx, c.BatchID)
		if e != nil {
			return e
		}
		if b.Version != c.ExpectedVersion {
			return domain.VersionConflict("expectedVersion 与当前版本不一致")
		}
		if !b.CanFreeze() {
			return domain.StateError("批次尚未批准，不能冻结")
		}
		pages, e := tx.ListLatestPages(ctx, c.BatchID)
		if e != nil {
			return e
		}
		issues, e := tx.ListIssues(ctx, c.BatchID)
		if e != nil {
			return e
		}
		quality, e := tx.LatestQuality(ctx, c.BatchID)
		if e != nil {
			return e
		}
		if quality == nil || !quality.Passed {
			return domain.QualityError("质量门禁尚未通过")
		}
		expected := b.Version
		if e = b.Freeze(); e != nil {
			return e
		}
		b.UpdatedAt = s.Now()
		if e = tx.UpdateBatch(ctx, b, expected); e != nil {
			return e
		}
		snapshot, e := ledger.BuildSnapshot(b, pages, issues, quality, s.Now())
		if e != nil {
			return e
		}
		credential := s.Signer.Issue(b.BatchID, snapshot.Digest, c.IssuedTo)
		if e = tx.InsertSnapshot(ctx, snapshot); e != nil {
			return e
		}
		if e = tx.InsertCredential(ctx, credential); e != nil {
			return e
		}
		if e = s.Chain.Append(ctx, tx, b.BatchID, "batch.frozen", map[string]any{"digest": snapshot.Digest, "credentialID": credential.CredentialID}); e != nil {
			return e
		}
		out = FreezeResponse{Batch: b, SnapshotDigest: snapshot.Digest, Credential: credential}
		return tx.SaveIdempotentResponse(ctx, "freeze:"+c.BatchID, c.IdempotencyKey, hash, encode(out))
	})
	return out, err
}

func (s *Service) VerifyCredential(ctx context.Context, id string) (VerifyResponse, error) {
	credential, e := s.Store.GetCredential(ctx, id)
	if e != nil {
		return VerifyResponse{}, e
	}
	snapshot, e := s.Store.GetSnapshot(ctx, credential.BatchID)
	if e != nil {
		return VerifyResponse{}, e
	}
	valid, reason := s.Signer.Verify(credential, snapshot)
	return VerifyResponse{Valid: valid, Reason: reason, Credential: credential, FrozenVersion: snapshot.Batch.Version}, nil
}

func (s *Service) VerifyAudit(ctx context.Context) error {
	records, e := s.Store.Audits(ctx)
	if e != nil {
		return e
	}
	return ledger.Verify(records)
}
