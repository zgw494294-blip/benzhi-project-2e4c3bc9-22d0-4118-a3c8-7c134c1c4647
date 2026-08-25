package persistence

import (
	"context"
	"database/sql"
	"time"
)

type AuditRecord struct {
	Sequence     int64
	EventID      string
	BatchID      string
	EventType    string
	Payload      string
	PreviousHash string
	EventHash    string
	OccurredAt   time.Time
}

func scanAudit(row interface{ Scan(...any) error }) (AuditRecord, error) {
	var r AuditRecord
	var occurred string
	err := row.Scan(&r.Sequence, &r.EventID, &r.BatchID, &r.EventType, &r.Payload, &r.PreviousHash, &r.EventHash, &occurred)
	if err != nil {
		return r, err
	}
	r.OccurredAt, err = time.Parse(time.RFC3339Nano, occurred)
	return r, err
}
func (t *Tx) LastAudit(ctx context.Context) (AuditRecord, error) {
	r, e := scanAudit(t.tx.QueryRowContext(ctx, `SELECT sequence_no,event_id,batch_id,event_type,payload,previous_hash,event_hash,occurred_at FROM audit_events ORDER BY sequence_no DESC LIMIT 1`))
	if e == sql.ErrNoRows {
		return AuditRecord{}, nil
	}
	return r, e
}
func (t *Tx) InsertAudit(ctx context.Context, r AuditRecord) error {
	_, e := t.tx.ExecContext(ctx, `INSERT INTO audit_events(event_id,batch_id,event_type,payload,previous_hash,event_hash,occurred_at) VALUES(?,?,?,?,?,?,?)`, r.EventID, r.BatchID, r.EventType, r.Payload, r.PreviousHash, r.EventHash, r.OccurredAt.Format(time.RFC3339Nano))
	return e
}
func (s *Store) Audits(ctx context.Context) ([]AuditRecord, error) {
	rows, e := s.db.QueryContext(ctx, `SELECT sequence_no,event_id,batch_id,event_type,payload,previous_hash,event_hash,occurred_at FROM audit_events ORDER BY sequence_no`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []AuditRecord{}
	for rows.Next() {
		r, e := scanAudit(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
