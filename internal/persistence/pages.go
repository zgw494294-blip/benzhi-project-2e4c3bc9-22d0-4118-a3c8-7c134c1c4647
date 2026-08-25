package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/benzhi-project/ancient-quality-gate/internal/domain"
)

func evidenceID(p domain.PageEvidence) string { return fmt.Sprintf("%s-r%d", p.PageID, p.Revision) }

func (t *Tx) InsertPage(ctx context.Context, p domain.PageEvidence) (string, error) {
	id := evidenceID(p)
	_, err := t.tx.ExecContext(ctx, `INSERT INTO pages(evidence_id,page_id,batch_id,sequence_no,image_digest,ocr_text,character_count,confidence,observed_at,revision) VALUES(?,?,?,?,?,?,?,?,?,?)`, id, p.PageID, p.BatchID, p.Sequence, p.ImageDigest, p.OCRText, p.CharacterCount, p.Confidence, p.ObservedAt.Format(time.RFC3339Nano), p.Revision)
	if err != nil {
		var coded interface{ Code() int }
		if errors.As(err, &coded) {
			switch coded.Code() {
			case 1555, 2067: // SQLITE_CONSTRAINT_PRIMARYKEY, SQLITE_CONSTRAINT_UNIQUE
				return "", domain.Conflict("页面证据修订已存在")
			}
		}
		return "", err
	}
	return id, nil
}

func scanPage(row interface{ Scan(...any) error }) (domain.PageEvidence, error) {
	var id string
	var p domain.PageEvidence
	var observed string
	err := row.Scan(&id, &p.PageID, &p.BatchID, &p.Sequence, &p.ImageDigest, &p.OCRText, &p.CharacterCount, &p.Confidence, &observed, &p.Revision)
	if errors.Is(err, sql.ErrNoRows) {
		return p, domain.NotFound("页面证据不存在")
	}
	if err != nil {
		return p, err
	}
	p.ObservedAt, err = time.Parse(time.RFC3339Nano, observed)
	return p, err
}

func (t *Tx) LatestPage(ctx context.Context, batchID, pageID string) (domain.PageEvidence, error) {
	return scanPage(t.tx.QueryRowContext(ctx, `SELECT evidence_id,page_id,batch_id,sequence_no,image_digest,ocr_text,character_count,confidence,observed_at,revision FROM pages WHERE batch_id=? AND page_id=? ORDER BY revision DESC LIMIT 1`, batchID, pageID))
}

func (t *Tx) GetEvidence(ctx context.Context, batchID, id string) (domain.PageEvidence, error) {
	return scanPage(t.tx.QueryRowContext(ctx, `SELECT evidence_id,page_id,batch_id,sequence_no,image_digest,ocr_text,character_count,confidence,observed_at,revision FROM pages WHERE batch_id=? AND evidence_id=?`, batchID, id))
}

func queryPages(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, batchID string) ([]domain.PageEvidence, error) {
	rows, err := q.QueryContext(ctx, `SELECT p.evidence_id,p.page_id,p.batch_id,p.sequence_no,p.image_digest,p.ocr_text,p.character_count,p.confidence,p.observed_at,p.revision FROM pages p JOIN (SELECT page_id,MAX(revision) revision FROM pages WHERE batch_id=? GROUP BY page_id) x ON p.page_id=x.page_id AND p.revision=x.revision WHERE p.batch_id=? ORDER BY p.sequence_no`, batchID, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.PageEvidence{}
	for rows.Next() {
		p, e := scanPage(rows)
		if e != nil {
			return nil, e
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (t *Tx) ListLatestPages(ctx context.Context, batchID string) ([]domain.PageEvidence, error) {
	return queryPages(ctx, t.tx, batchID)
}
func (s *Store) ListLatestPages(ctx context.Context, batchID string) ([]domain.PageEvidence, error) {
	return queryPages(ctx, s.db, batchID)
}
