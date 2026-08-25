package ledger

import (
	"sort"
	"time"

	"github.com/benzhi-project/ancient-quality-gate/internal/domain"
)

func BuildSnapshot(batch domain.DigitizationBatch, pages []domain.PageEvidence, issues []domain.QualityIssue, quality *domain.QualityResult, now time.Time) (domain.Snapshot, error) {
	sort.Slice(pages, func(i, j int) bool {
		if pages[i].Sequence == pages[j].Sequence {
			return pages[i].Revision < pages[j].Revision
		}
		return pages[i].Sequence < pages[j].Sequence
	})
	sort.Slice(issues, func(i, j int) bool { return issues[i].IssueID < issues[j].IssueID })
	manifest, err := domain.BuildManifest(batch, pages, issues)
	if err != nil {
		return domain.Snapshot{}, err
	}
	s := domain.Snapshot{Batch: batch, Pages: pages, Issues: issues, Quality: quality, Manifest: manifest, FrozenAt: now.UTC()}
	if err := domain.ValidateSnapshot(s); err != nil {
		return s, err
	}
	digest, err := domain.SnapshotDigest(s)
	if err != nil {
		return s, err
	}
	s.Digest = digest
	return s, nil
}
