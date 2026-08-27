package persistence

import (
	"context"
	"database/sql"
	"time"

	"github.com/benzhi-project/ancient-quality-gate/internal/domain"
)

func (t *Tx) InsertIssue(ctx context.Context, i domain.QualityIssue) error {
	_, err := t.tx.ExecContext(ctx, `INSERT INTO issues(issue_id,batch_id,page_id,category,severity,description,disposition,corrected_evidence_id,resolved_at) VALUES(?,?,?,?,?,?,?,?,NULL)`, i.IssueID, i.BatchID, i.PageID, i.Category, i.Severity, i.Description, i.Disposition, i.CorrectedEvidenceID)
	return err
}

func scanIssue(row interface{ Scan(...any) error }) (domain.QualityIssue, error) {
	var i domain.QualityIssue
	var resolved sql.NullString
	err := row.Scan(&i.IssueID, &i.BatchID, &i.PageID, &i.Category, &i.Severity, &i.Description, &i.Disposition, &i.CorrectedEvidenceID, &resolved)
	if err != nil {
		return i, err
	}
	if resolved.Valid {
		parsed, e := time.Parse(time.RFC3339Nano, resolved.String)
		if e != nil {
			return i, e
		}
		i.ResolvedAt = &parsed
	}
	return i, nil
}

func (t *Tx) GetIssue(ctx context.Context, id string) (domain.QualityIssue, error) {
	i, err := scanIssue(t.tx.QueryRowContext(ctx, `SELECT issue_id,batch_id,page_id,category,severity,description,disposition,corrected_evidence_id,resolved_at FROM issues WHERE issue_id=?`, id))
	if err == sql.ErrNoRows {
		return i, domain.NotFound("质量问题不存在")
	}
	return i, err
}

func (t *Tx) UpdateIssue(ctx context.Context, i domain.QualityIssue) error {
	var resolved any
	if i.ResolvedAt != nil {
		resolved = i.ResolvedAt.Format(time.RFC3339Nano)
	}
	_, err := t.tx.ExecContext(ctx, `UPDATE issues SET disposition=?,corrected_evidence_id=?,resolved_at=? WHERE issue_id=?`, i.Disposition, i.CorrectedEvidenceID, resolved, i.IssueID)
	return err
}

func queryIssues(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, batchID string) ([]domain.QualityIssue, error) {
	rows, err := q.QueryContext(ctx, `SELECT issue_id,batch_id,page_id,category,severity,description,disposition,corrected_evidence_id,resolved_at FROM issues WHERE batch_id=? ORDER BY issue_id`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []domain.QualityIssue{}
	for rows.Next() {
		i, e := scanIssue(rows)
		if e != nil {
			return nil, e
		}
		result = append(result, i)
	}
	return result, rows.Err()
}
func (t *Tx) ListIssues(ctx context.Context, id string) ([]domain.QualityIssue, error) {
	return queryIssues(ctx, t.tx, id)
}
func (s *Store) ListIssues(ctx context.Context, id string) ([]domain.QualityIssue, error) {
	return queryIssues(ctx, s.db, id)
}
