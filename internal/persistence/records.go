package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/benzhi-project/ancient-quality-gate/internal/domain"
)

func (t *Tx) InsertQuality(ctx context.Context, id, batchID string, r domain.QualityResult) error {
	b, _ := json.Marshal(r)
	_, e := t.tx.ExecContext(ctx, `INSERT INTO quality_checks(check_id,batch_id,passed,coverage,average_confidence,checked_at,payload) VALUES(?,?,?,?,?,?,?)`, id, batchID, r.Passed, r.Coverage, r.AverageConfidence, r.CheckedAt.Format(time.RFC3339Nano), string(b))
	return e
}
func (t *Tx) LatestQuality(ctx context.Context, batchID string) (*domain.QualityResult, error) {
	var raw string
	err := t.tx.QueryRowContext(ctx, `SELECT payload FROM quality_checks WHERE batch_id=? ORDER BY checked_at DESC LIMIT 1`, batchID).Scan(&raw)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var r domain.QualityResult
	err = json.Unmarshal([]byte(raw), &r)
	return &r, err
}
func (t *Tx) InsertReview(ctx context.Context, id, batchID string, r domain.ReviewDecision) error {
	_, e := t.tx.ExecContext(ctx, `INSERT INTO reviews(review_id,batch_id,approved,reviewer,comment,reviewed_at) VALUES(?,?,?,?,?,?)`, id, batchID, r.Approved, r.Reviewer, r.Comment, r.ReviewedAt.Format(time.RFC3339Nano))
	return e
}
func (t *Tx) InsertSnapshot(ctx context.Context, s domain.Snapshot) error {
	b, _ := json.Marshal(s)
	_, e := t.tx.ExecContext(ctx, `INSERT INTO snapshots(batch_id,digest,frozen_at,payload) VALUES(?,?,?,?)`, s.Batch.BatchID, s.Digest, s.FrozenAt.Format(time.RFC3339Nano), string(b))
	return e
}
func cloneSnapshot(s domain.Snapshot) domain.Snapshot {
	out := s
	if s.Pages != nil {
		out.Pages = make([]domain.PageEvidence, len(s.Pages))
		copy(out.Pages, s.Pages)
	}
	if s.Issues != nil {
		out.Issues = make([]domain.QualityIssue, len(s.Issues))
		copy(out.Issues, s.Issues)
		for i := range out.Issues {
			if s.Issues[i].ResolvedAt != nil {
				t := *s.Issues[i].ResolvedAt
				out.Issues[i].ResolvedAt = &t
			}
		}
	}
	if s.Quality != nil {
		q := *s.Quality
		if q.Issues != nil {
			q.Issues = make([]domain.QualityIssue, len(s.Quality.Issues))
			copy(q.Issues, s.Quality.Issues)
			for i := range q.Issues {
				if s.Quality.Issues[i].ResolvedAt != nil {
					t := *s.Quality.Issues[i].ResolvedAt
					q.Issues[i].ResolvedAt = &t
				}
			}
		}
		out.Quality = &q
	}
	if s.Manifest.Entries != nil {
		out.Manifest.Entries = make([]domain.ManifestEntry, len(s.Manifest.Entries))
		copy(out.Manifest.Entries, s.Manifest.Entries)
	}
	return out
}
func (s *Store) GetSnapshot(ctx context.Context, batchID string) (domain.Snapshot, error) {
	s.snapshotMu.RLock()
	cached, ok := s.snapshotCache[batchID]
	s.snapshotMu.RUnlock()
	if ok {
		return cloneSnapshot(cached), nil
	}

	var out domain.Snapshot
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT payload FROM snapshots WHERE batch_id=?`, batchID).Scan(&raw)
	if err == sql.ErrNoRows {
		return out, domain.NotFound("冻结快照不存在")
	}
	if err != nil {
		return out, err
	}
	err = json.Unmarshal([]byte(raw), &out)
	if err == nil {
		s.snapshotMu.Lock()
		s.snapshotCache[batchID] = out
		s.snapshotMu.Unlock()
	}
	return cloneSnapshot(out), err
}
func (t *Tx) InsertCredential(ctx context.Context, c domain.ReleaseCredential) error {
	_, e := t.tx.ExecContext(ctx, `INSERT INTO credentials(credential_id,batch_id,snapshot_digest,issued_to,issued_at,signature,revoked_at) VALUES(?,?,?,?,?,?,NULL)`, c.CredentialID, c.BatchID, c.SnapshotDigest, c.IssuedTo, c.IssuedAt.Format(time.RFC3339Nano), c.Signature)
	return e
}
func (s *Store) GetCredential(ctx context.Context, id string) (domain.ReleaseCredential, error) {
	var c domain.ReleaseCredential
	var issued string
	var revoked sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT credential_id,batch_id,snapshot_digest,issued_to,issued_at,signature,revoked_at FROM credentials WHERE credential_id=?`, id).Scan(&c.CredentialID, &c.BatchID, &c.SnapshotDigest, &c.IssuedTo, &issued, &c.Signature, &revoked)
	if err == sql.ErrNoRows {
		return c, domain.NotFound("发布凭据不存在")
	}
	if err != nil {
		return c, err
	}
	c.IssuedAt, err = time.Parse(time.RFC3339Nano, issued)
	if revoked.Valid {
		v, e := time.Parse(time.RFC3339Nano, revoked.String)
		if e != nil {
			return c, e
		}
		c.RevokedAt = &v
	}
	return c, err
}
