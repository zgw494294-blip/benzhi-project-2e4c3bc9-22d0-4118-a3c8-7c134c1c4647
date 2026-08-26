package credential_statement_lifecycle

import (
	"context"
	"testing"

	"github.com/benzhi-project/ancient-quality-gate/internal/application"
	"github.com/benzhi-project/ancient-quality-gate/internal/persistence"
)

func TestRepeatedCredentialVerificationKeepsReadResourceValid(t *testing.T) {
	store, err := persistence.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := application.New(store, []byte("credential-statement-test"))
	ctx := context.Background()

	created, err := service.CreateBatch(ctx, application.CreateBatchCommand{Title: "四库全书", Edition: "影印本", Owner: "质检组", IdempotencyKey: "create"})
	if err != nil {
		t.Fatal(err)
	}
	page, err := service.AddPage(ctx, application.AddPageCommand{
		BatchID: created.Batch.BatchID, PageID: "page-1", Sequence: 1,
		ImageDigest: "digest-123456", OCRText: "天地玄黄", CharacterCount: 4,
		Confidence: 0.99, ExpectedVersion: created.Batch.Version, IdempotencyKey: "page",
	})
	if err != nil {
		t.Fatal(err)
	}
	quality, err := service.QualityCheck(ctx, application.QualityCommand{BatchID: created.Batch.BatchID, ExpectedVersion: page.Batch.Version, IdempotencyKey: "quality"})
	if err != nil || !quality.Result.Passed {
		t.Fatalf("quality check failed: result=%+v err=%v", quality.Result, err)
	}
	review, err := service.Review(ctx, application.ReviewCommand{BatchID: created.Batch.BatchID, Approved: true, Reviewer: "复核专家", ExpectedVersion: quality.Batch.Version, IdempotencyKey: "review"})
	if err != nil {
		t.Fatal(err)
	}
	frozen, err := service.Freeze(ctx, application.FreezeCommand{BatchID: created.Batch.BatchID, IssuedTo: "发布组", ExpectedVersion: review.Batch.Version, IdempotencyKey: "freeze"})
	if err != nil {
		t.Fatal(err)
	}

	first, err := service.VerifyCredential(ctx, frozen.Credential.CredentialID)
	if err != nil || !first.Valid {
		t.Fatalf("first verification failed: result=%+v err=%v", first, err)
	}
	second, err := service.VerifyCredential(ctx, frozen.Credential.CredentialID)
	if err != nil || !second.Valid {
		t.Fatalf("repeated verification must stay valid: result=%+v err=%v", second, err)
	}
}
