package context_transaction_poison_test

import (
	"context"
	"testing"

	"github.com/benzhi-project/ancient-quality-gate/internal/application"
	"github.com/benzhi-project/ancient-quality-gate/internal/persistence"
)

func TestCanceledTransactionDoesNotPoisonLaterRequests(t *testing.T) {
	store, err := persistence.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	firstRequest, cancel := context.WithCancel(context.Background())
	if err := store.Transact(firstRequest, func(*persistence.Tx) error { return nil }); err != nil {
		t.Fatalf("首个事务应当成功: %v", err)
	}
	cancel()

	service := application.New(store, []byte("context-poison-test-key"))
	created, err := service.CreateBatch(context.Background(), application.CreateBatchCommand{
		Title: "后续批次", Edition: "刻本", Owner: "质检组", IdempotencyKey: "healthy-request",
	})
	if err != nil {
		t.Fatalf("取消请求不得污染后续健康请求: %v", err)
	}
	if created.Batch.BatchID == "" {
		t.Fatal("健康请求应返回已创建批次")
	}
}
