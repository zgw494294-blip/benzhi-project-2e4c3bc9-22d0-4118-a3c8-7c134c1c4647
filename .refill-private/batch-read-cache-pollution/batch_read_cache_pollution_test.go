package batchreadcachepollution

import (
	"context"
	"testing"

	"github.com/benzhi-project/ancient-quality-gate/internal/application"
	"github.com/benzhi-project/ancient-quality-gate/internal/domain"
	"github.com/benzhi-project/ancient-quality-gate/internal/persistence"
)

func TestBatchReadCacheIsInvalidatedAfterMutation(t *testing.T) {
	store, err := persistence.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := application.New(store, []byte("test-key"))
	ctx := context.Background()

	created, err := service.CreateBatch(ctx, application.CreateBatchCommand{
		Title: "古籍", Edition: "初版", Owner: "质检组", IdempotencyKey: "create",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.GetBatch(ctx, created.Batch.BatchID); err != nil {
		t.Fatal(err)
	}
	updated, err := service.AddPage(ctx, application.AddPageCommand{
		BatchID: created.Batch.BatchID, PageID: "page-1", Sequence: 1,
		ImageDigest: "digest-123456", OCRText: "天地", CharacterCount: 2,
		Confidence: 0.99, ExpectedVersion: created.Batch.Version, IdempotencyKey: "page",
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := service.GetBatch(ctx, created.Batch.BatchID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != updated.Batch.Version || got.Status != domain.StatusEvidence {
		t.Fatalf("缓存中的批次状态未随持久化变更刷新: got version=%d status=%s, want version=%d status=%s", got.Version, got.Status, updated.Batch.Version, domain.StatusEvidence)
	}
}
