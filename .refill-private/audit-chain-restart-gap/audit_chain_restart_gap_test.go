package audit_chain_restart_gap_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/benzhi-project/ancient-quality-gate/internal/application"
	"github.com/benzhi-project/ancient-quality-gate/internal/persistence"
)

func TestAuditChainResumesAfterServiceRestart(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "quality-gate.db")
	ctx := context.Background()

	firstStore, err := persistence.Open(databasePath)
	if err != nil {
		t.Fatal(err)
	}
	firstService := application.New(firstStore, []byte("restart-test-key"))
	_, err = firstService.CreateBatch(ctx, application.CreateBatchCommand{
		Title: "四库全书", Edition: "影印本", Owner: "甲组", IdempotencyKey: "before-restart",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := firstStore.Close(); err != nil {
		t.Fatal(err)
	}

	secondStore, err := persistence.Open(databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer secondStore.Close()
	secondService := application.New(secondStore, []byte("restart-test-key"))
	_, err = secondService.CreateBatch(ctx, application.CreateBatchCommand{
		Title: "永乐大典", Edition: "校印本", Owner: "乙组", IdempotencyKey: "after-restart",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := secondService.VerifyAudit(ctx); err != nil {
		t.Fatalf("重启后新增事件必须延续已持久化的审计链: %v", err)
	}
}
