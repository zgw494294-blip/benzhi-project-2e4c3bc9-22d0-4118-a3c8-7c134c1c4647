package application

import (
	"context"
	"testing"

	"github.com/benzhi-project/ancient-quality-gate/internal/persistence"
)

func TestFailedPageCanBeCorrectedReviewedAndFrozen(t *testing.T) {
	store, err := persistence.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := New(store, []byte("test-signing-key"))
	ctx := context.Background()

	created, err := service.CreateBatch(ctx, CreateBatchCommand{
		Title: "永乐大典", Edition: "影印本", Owner: "质检组", IdempotencyKey: "create",
	})
	if err != nil {
		t.Fatal(err)
	}
	page, err := service.AddPage(ctx, AddPageCommand{
		BatchID: created.Batch.BatchID, PageID: "page-1", Sequence: 1,
		ImageDigest: "sha256-image-1", OCRText: "天", CharacterCount: 1, Confidence: 0.40,
		ExpectedVersion: created.Batch.Version, IdempotencyKey: "page",
	})
	if err != nil {
		t.Fatal(err)
	}
	checked, err := service.QualityCheck(ctx, QualityCommand{
		BatchID: page.Batch.BatchID, ExpectedVersion: page.Batch.Version, IdempotencyKey: "quality-failed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked.Result.Passed || len(checked.Result.Issues) != 1 {
		t.Fatalf("预期生成一个质量问题: %+v", checked.Result)
	}

	revised, err := service.AddOCR(ctx, AddOCRCommand{
		BatchID: checked.Batch.BatchID, PageID: "page-1", OCRText: "天地玄黄",
		CharacterCount: 4, Confidence: 0.99, ExpectedVersion: checked.Batch.Version,
		IdempotencyKey: "ocr-correction",
	})
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := service.ResolveIssue(ctx, ResolveCommand{
		BatchID: checked.Batch.BatchID, IssueID: checked.Result.Issues[0].IssueID,
		CorrectedEvidenceID: revised.EvidenceID, ExpectedVersion: revised.Batch.Version,
		IdempotencyKey: "resolve",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !resolved.Result.Passed {
		t.Fatalf("整改复验应当通过: %+v", resolved.Result)
	}
	reviewed, err := service.Review(ctx, ReviewCommand{
		BatchID: resolved.Batch.BatchID, Approved: true, Reviewer: "复核专家",
		Comment: "整改完成", ExpectedVersion: resolved.Batch.Version, IdempotencyKey: "review",
	})
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := service.Freeze(ctx, FreezeCommand{
		BatchID: reviewed.Batch.BatchID, IssuedTo: "发布组",
		ExpectedVersion: reviewed.Batch.Version, IdempotencyKey: "freeze",
	})
	if err != nil {
		t.Fatal(err)
	}
	verified, err := service.VerifyCredential(ctx, frozen.Credential.CredentialID)
	if err != nil {
		t.Fatal(err)
	}
	if !verified.Valid || verified.FrozenVersion != frozen.Batch.Version {
		t.Fatalf("发布凭据应有效: %+v", verified)
	}
	if err := service.VerifyAudit(ctx); err != nil {
		t.Fatalf("审计链应完整: %v", err)
	}
}

func TestIdempotencyRejectsChangedRequest(t *testing.T) {
	store, err := persistence.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := New(store, nil)
	ctx := context.Background()
	_, err = service.CreateBatch(ctx, CreateBatchCommand{Title: "甲", Edition: "刻本", Owner: "甲", IdempotencyKey: "same"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.CreateBatch(ctx, CreateBatchCommand{Title: "乙", Edition: "刻本", Owner: "甲", IdempotencyKey: "same"})
	if err == nil {
		t.Fatal("相同幂等键用于不同请求时应当失败")
	}
}
