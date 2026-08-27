package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/benzhi-project/ancient-quality-gate/internal/application"
	"github.com/benzhi-project/ancient-quality-gate/internal/httpapi"
	"github.com/benzhi-project/ancient-quality-gate/internal/persistence"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:19081", "HTTP 监听地址")
	selfcheck := flag.Bool("selfcheck", false, "运行有界自检")
	dbPath := flag.String("db", "file:quality-gate.db?_pragma=foreign_keys(1)", "SQLite 数据库路径")
	flag.Parse()
	if port := os.Getenv("PORT"); port != "" && *addr == "127.0.0.1:19081" {
		*addr = "127.0.0.1:" + port
	}
	if *selfcheck {
		*dbPath = ":memory:"
	}
	store, err := persistence.Open(*dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer store.Close()
	app := application.New(store, nil)
	server := httpapi.New(app)
	if *selfcheck {
		if err := runSelfcheck(app); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Println("selfcheck passed")
		return
	}
	srv := &http.Server{Addr: *addr, Handler: server.Handler(), ReadHeaderTimeout: 5 * time.Second}
	fmt.Printf("古籍质量放行服务监听 %s\n", *addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runSelfcheck(app *application.Service) error {
	ctx := context.Background()
	batch, e := app.CreateBatch(ctx, application.CreateBatchCommand{Title: "本草纲目", Edition: "明刻本", Owner: "自检用户", IdempotencyKey: "selfcheck-batch"})
	if e != nil {
		return e
	}
	for i := 1; i <= 20; i++ {
		r, e := app.AddPage(ctx, application.AddPageCommand{BatchID: batch.Batch.BatchID, PageID: fmt.Sprintf("p-%02d", i), Sequence: i, ImageDigest: fmt.Sprintf("digest-%08d", i), OCRText: "天地玄黄", CharacterCount: 4, Confidence: 0.99, ExpectedVersion: batch.Batch.Version, IdempotencyKey: fmt.Sprintf("selfcheck-page-%02d", i)})
		if e != nil {
			return e
		}
		batch.Batch = r.Batch
	}
	q, e := app.QualityCheck(ctx, application.QualityCommand{BatchID: batch.Batch.BatchID, ExpectedVersion: batch.Batch.Version, IdempotencyKey: "selfcheck-quality"})
	if e != nil {
		return e
	}
	if !q.Result.Passed {
		return fmt.Errorf("自检质量门禁未通过")
	}
	review, e := app.Review(ctx, application.ReviewCommand{BatchID: batch.Batch.BatchID, Approved: true, Reviewer: "专家", Comment: "自检批准", ExpectedVersion: q.Batch.Version, IdempotencyKey: "selfcheck-review"})
	if e != nil {
		return e
	}
	f, e := app.Freeze(ctx, application.FreezeCommand{BatchID: batch.Batch.BatchID, IssuedTo: "发布组", ExpectedVersion: review.Batch.Version, IdempotencyKey: "selfcheck-freeze"})
	if e != nil {
		return e
	}
	v, e := app.VerifyCredential(ctx, f.Credential.CredentialID)
	if e != nil {
		return e
	}
	if !v.Valid {
		return fmt.Errorf("自检凭据验真失败: %s", v.Reason)
	}
	return app.VerifyAudit(ctx)
}
