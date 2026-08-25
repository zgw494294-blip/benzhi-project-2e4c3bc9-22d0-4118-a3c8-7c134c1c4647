package persistence

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/benzhi-project/ancient-quality-gate/internal/domain"
)

func (t *Tx) InsertBatch(ctx context.Context, b domain.DigitizationBatch) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO batches(batch_id,title,edition,owner,status,version,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, b.BatchID, b.Title, b.Edition, b.Owner, b.Status, b.Version, b.CreatedAt.Format(time.RFC3339Nano), b.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return domain.Conflict("batchID 已存在")
	}
	return nil
}

func scanBatch(row interface{ Scan(...any) error }) (domain.DigitizationBatch, error) {
	var b domain.DigitizationBatch
	var status, created, updated string
	err := row.Scan(&b.BatchID, &b.Title, &b.Edition, &b.Owner, &status, &b.Version, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return b, domain.NotFound("批次不存在")
	}
	if err != nil {
		return b, err
	}
	b.Status = domain.BatchStatus(status)
	b.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return b, err
	}
	b.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	return b, err
}

func (t *Tx) GetBatch(ctx context.Context, id string) (domain.DigitizationBatch, error) {
	return scanBatch(t.tx.QueryRowContext(ctx, `SELECT batch_id,title,edition,owner,status,version,created_at,updated_at FROM batches WHERE batch_id=?`, id))
}

func (s *Store) GetBatch(ctx context.Context, id string) (domain.DigitizationBatch, error) {
	return scanBatch(s.db.QueryRowContext(ctx, `SELECT batch_id,title,edition,owner,status,version,created_at,updated_at FROM batches WHERE batch_id=?`, id))
}

func (t *Tx) UpdateBatch(ctx context.Context, b domain.DigitizationBatch, expected int64) error {
	res, err := t.tx.ExecContext(ctx, `UPDATE batches SET status=?,version=?,updated_at=? WHERE batch_id=? AND version=?`, b.Status, b.Version, b.UpdatedAt.Format(time.RFC3339Nano), b.BatchID, expected)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return domain.VersionConflict("expectedVersion 与当前版本不一致")
	}
	return nil
}
