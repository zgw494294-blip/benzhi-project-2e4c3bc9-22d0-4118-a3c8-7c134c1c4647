package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/benzhi-project/ancient-quality-gate/internal/application"
	"github.com/benzhi-project/ancient-quality-gate/internal/persistence"
)

func TestEndToEndHTTP(t *testing.T) {
	store, e := persistence.Open(":memory:")
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	srv := New(application.New(store, nil)).Handler()
	post := func(path string, body any, version int64, key string) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		r := httptest.NewRequest("POST", path, bytes.NewReader(b))
		r.Header.Set("Expected-Version", fmtInt(version))
		r.Header.Set("Idempotency-Key", key)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, r)
		return w
	}
	w := post("/v1/batches", map[string]string{"title": "测试", "edition": "初版", "owner": "甲"}, 0, "b")
	if w.Code != 201 {
		t.Fatalf("create %d %s", w.Code, w.Body.String())
	}
	var created application.BatchResult
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	for i := 1; i <= 20; i++ {
		w = post("/v1/batches/"+created.Batch.BatchID+"/pages", map[string]any{"pageID": fmtInt(int64(i)), "sequence": i, "imageDigest": "digest-123456", "ocrText": "天地玄黄", "characterCount": 4, "confidence": 0.99}, created.Batch.Version, fmtInt(int64(i)))
		if w.Code != 201 {
			t.Fatalf("page %d %d", i, w.Code)
		}
		_ = json.Unmarshal(w.Body.Bytes(), &created)
	}
	w = post("/v1/batches/"+created.Batch.BatchID+"/quality-check", map[string]any{}, created.Batch.Version, "q")
	if w.Code != 200 {
		t.Fatalf("quality %d %s", w.Code, w.Body.String())
	}
}
func fmtInt(v int64) string { return strconv.FormatInt(v, 10) }
