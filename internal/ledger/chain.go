package ledger

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/benzhi-project/ancient-quality-gate/internal/persistence"
)

type auditWriter interface {
	LastAudit(context.Context) (persistence.AuditRecord, error)
	InsertAudit(context.Context, persistence.AuditRecord) error
}

type Chain struct {
	now func() time.Time
	mu  sync.Mutex
}

func NewChain(now func() time.Time) *Chain {
	if now == nil {
		now = time.Now
	}
	return &Chain{now: now}
}

func eventHash(r persistence.AuditRecord) string {
	b, _ := json.Marshal([]any{r.EventID, r.BatchID, r.EventType, r.Payload, r.PreviousHash, r.OccurredAt.UTC().Format(time.RFC3339Nano)})
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func (c *Chain) Append(ctx context.Context, w auditWriter, batchID, eventType string, payload any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	// 每次追加都从当前事务可见的审计表读取尾部哈希，而不是缓存上一次的结果。
	// 审计记录在事务内暂存后可能因事务回滚而被丢弃；缓存尾部会让后续事件指向
	// 从未提交的哈希，从而破坏审计链。从 writer（即当前事务）读取尾部可保证
	// PreviousHash 始终反映已提交（或本事务内已暂存）的真实状态。
	previous, err := w.LastAudit(ctx)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	r := persistence.AuditRecord{EventID: NewID("evt"), BatchID: batchID, EventType: eventType, Payload: string(raw), PreviousHash: previous.EventHash, OccurredAt: c.now().UTC()}
	r.EventHash = eventHash(r)
	return w.InsertAudit(ctx, r)
}

func Verify(records []persistence.AuditRecord) error {
	previous := ""
	for index, r := range records {
		if r.PreviousHash != previous {
			return fmt.Errorf("审计事件 %d 的前序哈希不匹配", index+1)
		}
		if eventHash(r) != r.EventHash {
			return fmt.Errorf("审计事件 %d 内容已变化", index+1)
		}
		previous = r.EventHash
	}
	return nil
}
