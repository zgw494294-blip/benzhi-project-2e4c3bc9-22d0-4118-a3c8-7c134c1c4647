package snapshot_cache_alias_test

import (
	"context"
	"strings"
	"testing"

	"github.com/benzhi-project/ancient-quality-gate/internal/application"
	"github.com/benzhi-project/ancient-quality-gate/internal/persistence"
)

func TestSnapshotCacheDoesNotShareMutableState(t *testing.T) {
	store, err := persistence.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	service := application.New(store, []byte("snapshot-cache-test-key"))
	ctx := context.Background()
	created, err := service.CreateBatch(ctx, application.CreateBatchCommand{
		Title: "资治通鉴", Edition: "宋刻本", Owner: "质检组", IdempotencyKey: "create",
	})
	if err != nil {
		t.Fatal(err)
	}
	page, err := service.AddPage(ctx, application.AddPageCommand{
		BatchID: created.Batch.BatchID, PageID: "page-1", Sequence: 1,
		ImageDigest: "sha256-image-1", OCRText: "天地玄黄", CharacterCount: 4, Confidence: 0.99,
		ExpectedVersion: created.Batch.Version, IdempotencyKey: "page",
	})
	if err != nil {
		t.Fatal(err)
	}
	quality, err := service.QualityCheck(ctx, application.QualityCommand{
		BatchID: page.Batch.BatchID, ExpectedVersion: page.Batch.Version, IdempotencyKey: "quality",
	})
	if err != nil {
		t.Fatal(err)
	}
	review, err := service.Review(ctx, application.ReviewCommand{
		BatchID: quality.Batch.BatchID, Approved: true, Reviewer: "复核专家",
		ExpectedVersion: quality.Batch.Version, IdempotencyKey: "review",
	})
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := service.Freeze(ctx, application.FreezeCommand{
		BatchID: review.Batch.BatchID, IssuedTo: "发布组",
		ExpectedVersion: review.Batch.Version, IdempotencyKey: "freeze",
	})
	if err != nil {
		t.Fatal(err)
	}

	consumerCopy, err := store.GetSnapshot(ctx, frozen.Batch.BatchID)
	if err != nil {
		t.Fatal(err)
	}
	consumerCopy.Pages[0].OCRText = "被其他读取方临时改写"

	verified, err := service.VerifyCredential(ctx, frozen.Credential.CredentialID)
	if err != nil {
		t.Fatal(err)
	}
	if !verified.Valid {
		t.Fatalf("缓存读取方的局部修改污染了后续凭据验真: %s", strings.TrimSpace(verified.Reason))
	}
}
