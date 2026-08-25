package ledger

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/benzhi-project/ancient-quality-gate/internal/persistence"
)

type auditWriter interface {
	LastAudit(context.Context) (persistence.AuditRecord, error)
	InsertAudit(context.Context, persistence.AuditRecord) error
}

type Chain struct{ now func() time.Time }

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
