package audit_tail_rollback_test

import (
	"context"
	"testing"

	"github.com/benzhi-project/ancient-quality-gate/internal/application"
	"github.com/benzhi-project/ancient-quality-gate/internal/persistence"
)

func TestRolledBackAuditTailDoesNotBreakNextTransaction(t *testing.T) {
	store, err := persistence.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := application.New(store, []byte("audit-tail-test-key"))
	ctx := context.Background()

	_, err = store.DB().Exec(`CREATE TRIGGER fail_idempotency_insert
		BEFORE INSERT ON idempotency
		BEGIN
			SELECT RAISE(FAIL, 'forced idempotency failure');
		END`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.CreateBatch(ctx, application.CreateBatchCommand{
		Title: "回滚批次", Edition: "刻本", Owner: "质检组", IdempotencyKey: "rolled-back",
	})
	if err == nil {
		t.Fatal("注入故障的事务应当回滚")
	}
	if _, err = store.DB().Exec(`DROP TRIGGER fail_idempotency_insert`); err != nil {
		t.Fatal(err)
	}

	_, err = service.CreateBatch(ctx, application.CreateBatchCommand{
		Title: "有效批次", Edition: "抄本", Owner: "质检组", IdempotencyKey: "committed",
	})
	if err != nil {
		t.Fatalf("后续健康事务应当提交: %v", err)
	}
	if err = service.VerifyAudit(ctx); err != nil {
		t.Fatalf("回滚事务不得污染后续持久化审计链: %v", err)
	}
}
