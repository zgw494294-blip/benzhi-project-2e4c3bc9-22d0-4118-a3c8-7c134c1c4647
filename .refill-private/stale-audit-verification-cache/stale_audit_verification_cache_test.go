package stale_audit_verification_cache_test

import (
	"context"
	"testing"

	"github.com/benzhi-project/ancient-quality-gate/internal/application"
	"github.com/benzhi-project/ancient-quality-gate/internal/persistence"
)

func TestAuditVerificationRechecksPersistedChain(t *testing.T) {
	store, err := persistence.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	service := application.New(store, []byte("audit-cache-test-key"))
	_, err = service.CreateBatch(context.Background(), application.CreateBatchCommand{
		Title: "四库全书", Edition: "影印本", Owner: "质检组", IdempotencyKey: "create-audit-cache-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.VerifyAudit(context.Background()); err != nil {
		t.Fatalf("初次审计链验真失败: %v", err)
	}

	if _, err := store.DB().Exec(`UPDATE audit_events SET payload=? WHERE sequence_no=1`, `{"tampered":true}`); err != nil {
		t.Fatal(err)
	}
	if err := service.VerifyAudit(context.Background()); err == nil {
		t.Fatal("持久化审计事件被修改后，VerifyAudit 仍返回成功")
	}
}
