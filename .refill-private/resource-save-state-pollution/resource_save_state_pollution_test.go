package resource_save_state_pollution_test

import (
	"context"
	"testing"

	"github.com/benzhi-project/ancient-quality-gate/internal/application"
	"github.com/benzhi-project/ancient-quality-gate/internal/persistence"
)

func TestRolledBackIdempotencyStateIsNotPublished(t *testing.T) {
	store, err := persistence.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service := application.New(store, []byte("private-test-key"))
	ctx := context.Background()

	_, err = store.DB().Exec(`CREATE TRIGGER fail_selected_idempotency
		BEFORE INSERT ON idempotency
		WHEN NEW.idempotency_key = 'rolled-back-key'
		BEGIN
			SELECT RAISE(FAIL, 'forced idempotency persistence failure');
		END`)
	if err != nil {
		t.Fatal(err)
	}

	command := application.CreateBatchCommand{
		Title: "四库全书", Edition: "影印本", Owner: "质检组", IdempotencyKey: "rolled-back-key",
	}
	if _, err = service.CreateBatch(ctx, command); err == nil {
		t.Fatal("注入的 idempotency 持久化失败应使事务回滚")
	}

	if _, err = service.CreateBatch(ctx, application.CreateBatchCommand{
		Title: "永乐大典", Edition: "影印本", Owner: "复核组", IdempotencyKey: "healthy-key",
	}); err != nil {
		t.Fatalf("后续健康事务应提交: %v", err)
	}
	if _, err = store.DB().Exec(`DROP TRIGGER fail_selected_idempotency`); err != nil {
		t.Fatal(err)
	}

	retried, err := service.CreateBatch(ctx, command)
	if err != nil {
		t.Fatalf("清除存储故障后的重试应成功: %v", err)
	}
	if _, err = service.GetBatch(ctx, retried.Batch.BatchID); err != nil {
		t.Fatalf("幂等重试返回了未持久化批次: %v", err)
	}
}
