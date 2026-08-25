package domain

import "time"

type BatchStatus string

const (
	StatusDraft    BatchStatus = "draft"
	StatusEvidence BatchStatus = "evidence"
	StatusReview   BatchStatus = "review"
	StatusApproved BatchStatus = "approved"
	StatusFrozen   BatchStatus = "frozen"
	StatusRejected BatchStatus = "rejected"
)

type DigitizationBatch struct {
	BatchID   string      `json:"batchID"`
	Title     string      `json:"title"`
	Edition   string      `json:"edition"`
	Owner     string      `json:"owner"`
	Status    BatchStatus `json:"status"`
	Version   int64       `json:"version"`
	CreatedAt time.Time   `json:"createdAt"`
	UpdatedAt time.Time   `json:"updatedAt"`
}

type PageEvidence struct {
	PageID         string    `json:"pageID"`
	BatchID        string    `json:"batchID"`
	Sequence       int       `json:"sequence"`
	ImageDigest    string    `json:"imageDigest"`
	OCRText        string    `json:"ocrText"`
	CharacterCount int       `json:"characterCount"`
	Confidence     float64   `json:"confidence"`
	ObservedAt     time.Time `json:"observedAt"`
	Revision       int       `json:"revision"`
}

type QualityIssue struct {
	IssueID             string     `json:"issueID"`
	BatchID             string     `json:"batchID"`
	PageID              string     `json:"pageID"`
	Category            string     `json:"category"`
	Severity            string     `json:"severity"`
	Description         string     `json:"description"`
	Disposition         string     `json:"disposition"`
	CorrectedEvidenceID string     `json:"correctedEvidenceID,omitempty"`
	ResolvedAt          *time.Time `json:"resolvedAt,omitempty"`
}

type ReleaseCredential struct {
	CredentialID   string     `json:"credentialID"`
	BatchID        string     `json:"batchID"`
	SnapshotDigest string     `json:"snapshotDigest"`
	IssuedTo       string     `json:"issuedTo"`
	IssuedAt       time.Time  `json:"issuedAt"`
	Signature      string     `json:"signature"`
	RevokedAt      *time.Time `json:"revokedAt,omitempty"`
}

type QualityResult struct {
	Passed            bool           `json:"passed"`
	Coverage          float64        `json:"coverage"`
	AverageConfidence float64        `json:"averageConfidence"`
	Issues            []QualityIssue `json:"issues"`
	CheckedAt         time.Time      `json:"checkedAt"`
}

type ReviewDecision struct {
	Approved   bool      `json:"approved"`
	Reviewer   string    `json:"reviewer"`
	Comment    string    `json:"comment"`
	ReviewedAt time.Time `json:"reviewedAt"`
}

type Snapshot struct {
	Batch    DigitizationBatch `json:"batch"`
	Pages    []PageEvidence    `json:"pages"`
	Issues   []QualityIssue    `json:"issues"`
	Quality  *QualityResult    `json:"quality,omitempty"`
	Manifest ReleaseManifest   `json:"manifest"`
	Digest   string            `json:"digest"`
	FrozenAt time.Time         `json:"frozenAt"`
}

type ManifestEntry struct {
	PageID         string  `json:"pageID"`
	Sequence       int     `json:"sequence"`
	Revision       int     `json:"revision"`
	ImageDigest    string  `json:"imageDigest"`
	EvidenceDigest string  `json:"evidenceDigest"`
	Confidence     float64 `json:"confidence"`
}

type ReleaseManifest struct {
	BatchID           string          `json:"batchID"`
	BatchVersion      int64           `json:"batchVersion"`
	PageCount         int             `json:"pageCount"`
	CharacterCount    int             `json:"characterCount"`
	OpenIssueCount    int             `json:"openIssueCount"`
	CriticalCount     int             `json:"criticalCount"`
	AverageConfidence float64         `json:"averageConfidence"`
	Entries           []ManifestEntry `json:"entries"`
}
