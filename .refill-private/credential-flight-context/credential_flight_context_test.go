package credentialflightcontext

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/benzhi-project/ancient-quality-gate/internal/application"
	"github.com/benzhi-project/ancient-quality-gate/internal/domain"
	"github.com/benzhi-project/ancient-quality-gate/internal/ledger"
	"github.com/benzhi-project/ancient-quality-gate/internal/persistence"
)

type observedContext struct {
	context.Context
	observed chan struct{}
	once     sync.Once
}

func newObservedContext(ctx context.Context) *observedContext {
	return &observedContext{Context: ctx, observed: make(chan struct{})}
}

func (c *observedContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.observed) })
	return c.Context.Done()
}

func (c *observedContext) Err() error {
	c.once.Do(func() { close(c.observed) })
	return c.Context.Err()
}

type verificationResult struct {
	response application.VerifyResponse
	err      error
}

func TestConcurrentVerificationKeepsIndependentContext(t *testing.T) {
	store, err := persistence.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	service := application.New(store, []byte("private-test-signing-key"))
	credential := seedCredential(t, service)

	blocker, err := store.DB().Begin()
	if err != nil {
		t.Fatal(err)
	}
	blockerReleased := false
	t.Cleanup(func() {
		if !blockerReleased {
			_ = blocker.Rollback()
		}
	})

	leaderBase, cancelLeader := context.WithCancel(context.Background())
	leaderCtx := newObservedContext(leaderBase)
	leaderResult := make(chan verificationResult, 1)
	go func() {
		response, verifyErr := service.VerifyCredential(leaderCtx, credential.CredentialID)
		leaderResult <- verificationResult{response: response, err: verifyErr}
	}()
	<-leaderCtx.observed

	followerCtx := newObservedContext(context.Background())
	followerResult := make(chan verificationResult, 1)
	go func() {
		response, verifyErr := service.VerifyCredential(followerCtx, credential.CredentialID)
		followerResult <- verificationResult{response: response, err: verifyErr}
	}()
	<-followerCtx.observed

	cancelLeader()
	leader := <-leaderResult
	if leader.err == nil {
		t.Fatal("已取消的首个验真请求应返回 context 错误")
	}
	if err := blocker.Rollback(); err != nil {
		t.Fatal(err)
	}
	blockerReleased = true

	follower := <-followerResult
	if follower.err != nil {
		t.Fatalf("健康的并发验真请求继承了其他请求的取消: %v", follower.err)
	}
	if !follower.response.Valid {
		t.Fatalf("健康的并发验真请求应验证成功: %+v", follower.response)
	}
}

func seedCredential(t *testing.T, service *application.Service) domain.ReleaseCredential {
	t.Helper()
	now := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	batch := domain.DigitizationBatch{
		BatchID: "batch-context-flight", Title: "永乐大典", Edition: "影印本", Owner: "质检组",
		Status: domain.StatusFrozen, Version: 5, CreatedAt: now, UpdatedAt: now,
	}
	page := domain.PageEvidence{
		PageID: "page-1", BatchID: batch.BatchID, Sequence: 1, ImageDigest: "sha256-image-1",
		OCRText: "天地玄黄", CharacterCount: 4, Confidence: 0.99, ObservedAt: now, Revision: 1,
	}
	quality := &domain.QualityResult{Passed: true, Coverage: 1, AverageConfidence: 0.99, CheckedAt: now}
	snapshot, err := ledger.BuildSnapshot(batch, []domain.PageEvidence{page}, nil, quality, now)
	if err != nil {
		t.Fatal(err)
	}
	credential := service.Signer.Issue(batch.BatchID, snapshot.Digest, "发布组")
	err = service.Store.Transact(context.Background(), func(tx *persistence.Tx) error {
		if insertErr := tx.InsertSnapshot(context.Background(), snapshot); insertErr != nil {
			return insertErr
		}
		return tx.InsertCredential(context.Background(), credential)
	})
	if err != nil {
		t.Fatal(err)
	}
	return credential
}
