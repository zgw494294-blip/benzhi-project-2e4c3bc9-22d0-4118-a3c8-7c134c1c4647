package application

import (
	"context"
	"encoding/json"

	"github.com/benzhi-project/ancient-quality-gate/internal/domain"
	"github.com/benzhi-project/ancient-quality-gate/internal/ledger"
	"github.com/benzhi-project/ancient-quality-gate/internal/persistence"
)

type QualityCommand struct {
	BatchID         string `json:"-"`
	ExpectedVersion int64  `json:"expectedVersion"`
	IdempotencyKey  string `json:"-"`
}
type QualityResponse struct {
	Batch  domain.DigitizationBatch `json:"batch"`
	Result domain.QualityResult     `json:"quality"`
}
type ResolveCommand struct {
	BatchID             string `json:"-"`
	IssueID             string `json:"-"`
	CorrectedEvidenceID string `json:"correctedEvidenceID"`
	ExpectedVersion     int64  `json:"expectedVersion"`
	IdempotencyKey      string `json:"-"`
}

func (s *Service) QualityCheck(ctx context.Context, c QualityCommand) (QualityResponse, error) {
	var out QualityResponse
	hash := requestHash(c)
	err := s.Store.Transact(ctx, func(tx *persistence.Tx) error {
		if raw, ok, e := tx.IdempotentResponse(ctx, "quality:"+c.BatchID, c.IdempotencyKey, hash); e != nil {
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
		if !b.CanQualityCheck() {
			return domain.StateError("当前状态不能执行质量检查")
		}
		pages, e := tx.ListLatestPages(ctx, c.BatchID)
		if e != nil {
			return e
		}
		issues, e := tx.ListIssues(ctx, c.BatchID)
		if e != nil {
			return e
		}
		result := domain.EvaluateQuality(b, pages, issues, s.Now())
		for index := range result.Issues {
			if result.Issues[index].IssueID == "" {
				result.Issues[index].IssueID = ledger.NewID("issue")
				result.Issues[index].Disposition = "open"
				if e = tx.InsertIssue(ctx, result.Issues[index]); e != nil {
					return e
				}
			}
		}
		expected := b.Version
		if e = b.BeginReview(); e != nil {
			return e
		}
		if e = tx.UpdateBatch(ctx, b, expected); e != nil {
			return e
		}
		if e = tx.InsertQuality(ctx, ledger.NewID("check"), b.BatchID, result); e != nil {
			return e
		}
		if e = s.Chain.Append(ctx, tx, b.BatchID, "quality.checked", result); e != nil {
			return e
		}
		out = QualityResponse{Batch: b, Result: result}
		return tx.SaveIdempotentResponse(ctx, "quality:"+c.BatchID, c.IdempotencyKey, hash, encode(out))
	})
	return out, err
}

func (s *Service) ResolveIssue(ctx context.Context, c ResolveCommand) (QualityResponse, error) {
	var out QualityResponse
	hash := requestHash(c)
	err := s.Store.Transact(ctx, func(tx *persistence.Tx) error {
		if raw, ok, e := tx.IdempotentResponse(ctx, "resolve:"+c.IssueID, c.IdempotencyKey, hash); e != nil {
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
		if b.Status != domain.StatusReview && b.Status != domain.StatusRejected {
			return domain.StateError("当前状态不能整改问题")
		}
		issue, e := tx.GetIssue(ctx, c.IssueID)
		if e != nil {
			return e
		}
		if issue.BatchID != c.BatchID {
			return domain.NotFound("质量问题不存在")
		}
		evidence, e := tx.GetEvidence(ctx, c.BatchID, c.CorrectedEvidenceID)
		if e != nil {
			return e
		}
		if e = domain.ResolveIssue(&issue, evidence, c.CorrectedEvidenceID, s.Now()); e != nil {
			return e
		}
		if e = tx.UpdateIssue(ctx, issue); e != nil {
			return e
		}
		issues, e := tx.ListIssues(ctx, c.BatchID)
		if e != nil {
			return e
		}
		pages, e := tx.ListLatestPages(ctx, c.BatchID)
		if e != nil {
			return e
		}
		result := domain.EvaluateQuality(b, pages, issues, s.Now())
		expected := b.Version
		b.Status = domain.StatusReview
		b.Version++
		if e = tx.UpdateBatch(ctx, b, expected); e != nil {
			return e
		}
		if e = tx.InsertQuality(ctx, ledger.NewID("check"), b.BatchID, result); e != nil {
			return e
		}
		if e = s.Chain.Append(ctx, tx, b.BatchID, "issue.resolved", map[string]any{"issue": issue, "quality": result}); e != nil {
			return e
		}
		out = QualityResponse{Batch: b, Result: result}
		return tx.SaveIdempotentResponse(ctx, "resolve:"+c.IssueID, c.IdempotencyKey, hash, encode(out))
	})
	return out, err
}
