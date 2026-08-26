package idempotency_body_cache_scope_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/benzhi-project/ancient-quality-gate/internal/application"
	"github.com/benzhi-project/ancient-quality-gate/internal/httpapi"
	"github.com/benzhi-project/ancient-quality-gate/internal/persistence"
)

func TestIdempotencyBodyCacheIsScopedToOperation(t *testing.T) {
	store, err := persistence.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	app := application.New(store, []byte("private-reproduction-secret"))
	first, err := app.CreateBatch(context.Background(), application.CreateBatchCommand{
		Title: "第一批", Edition: "甲本", Owner: "甲组", IdempotencyKey: "create-first-batch",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := app.CreateBatch(context.Background(), application.CreateBatchCommand{
		Title: "第二批", Edition: "乙本", Owner: "乙组", IdempotencyKey: "create-second-batch",
	})
	if err != nil {
		t.Fatal(err)
	}

	handler := httpapiHandler(app)
	postPage(t, handler, first.Batch.BatchID, `{"pageID":"page-first","sequence":1,"imageDigest":"digest-first","ocrText":"甲","characterCount":1,"confidence":0.99,"expectedVersion":1}`)
	postPage(t, handler, second.Batch.BatchID, `{"pageID":"page-second","sequence":2,"imageDigest":"digest-second","ocrText":"乙","characterCount":1,"confidence":0.98,"expectedVersion":1}`)

	pages, err := store.ListLatestPages(context.Background(), second.Batch.BatchID)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 1 || pages[0].PageID != "page-second" || pages[0].Sequence != 2 {
		t.Fatalf("第二个操作复用了其他批次的请求体: %+v", pages)
	}
}

type requestHandler interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}

func httpapiHandler(app *application.Service) requestHandler {
	return httpapi.New(app).Handler()
}

func postPage(t *testing.T, handler requestHandler, batchID, body string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/batches/"+batchID+"/pages", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "shared-cross-batch-key")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusCreated {
		var response any
		_ = json.Unmarshal(recorder.Body.Bytes(), &response)
		t.Fatalf("登记页面失败: status=%d body=%v", recorder.Code, response)
	}
}
