package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

func EvidenceDigest(page PageEvidence) (string, error) {
	canonical := struct {
		PageID         string  `json:"pageID"`
		BatchID        string  `json:"batchID"`
		Sequence       int     `json:"sequence"`
		ImageDigest    string  `json:"imageDigest"`
		OCRText        string  `json:"ocrText"`
		CharacterCount int     `json:"characterCount"`
		Confidence     float64 `json:"confidence"`
		Revision       int     `json:"revision"`
	}{page.PageID, page.BatchID, page.Sequence, page.ImageDigest, page.OCRText, page.CharacterCount, page.Confidence, page.Revision}
	raw, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("编码证据摘要: %w", err)
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

func BuildManifest(batch DigitizationBatch, pages []PageEvidence, issues []QualityIssue) (ReleaseManifest, error) {
	manifest := ReleaseManifest{
		BatchID:      batch.BatchID,
		BatchVersion: batch.Version,
		Entries:      make([]ManifestEntry, 0, len(pages)),
	}
	seenSequence := make(map[int]string, len(pages))
	for _, page := range pages {
		if page.BatchID != batch.BatchID {
			return manifest, Invalid("发布清单包含其他批次的页面", "pages")
		}
		if existing, ok := seenSequence[page.Sequence]; ok && existing != page.PageID {
			return manifest, Conflict("发布清单包含重复页面序号")
		}
		seenSequence[page.Sequence] = page.PageID
		digest, err := EvidenceDigest(page)
		if err != nil {
			return manifest, err
		}
		manifest.PageCount++
		manifest.CharacterCount += page.CharacterCount
		manifest.AverageConfidence += page.Confidence
		manifest.Entries = append(manifest.Entries, ManifestEntry{
			PageID:         page.PageID,
			Sequence:       page.Sequence,
			Revision:       page.Revision,
			ImageDigest:    page.ImageDigest,
			EvidenceDigest: digest,
			Confidence:     page.Confidence,
		})
	}
	if manifest.PageCount > 0 {
		manifest.AverageConfidence /= float64(manifest.PageCount)
	}
	for _, issue := range issues {
		if issue.BatchID != batch.BatchID {
			return manifest, Invalid("发布清单包含其他批次的问题", "issues")
		}
		if issue.ResolvedAt == nil {
			manifest.OpenIssueCount++
			if issue.Severity == "critical" {
				manifest.CriticalCount++
			}
		}
	}
	sort.Slice(manifest.Entries, func(i, j int) bool {
		if manifest.Entries[i].Sequence == manifest.Entries[j].Sequence {
			return manifest.Entries[i].PageID < manifest.Entries[j].PageID
		}
		return manifest.Entries[i].Sequence < manifest.Entries[j].Sequence
	})
	return manifest, nil
}

func ValidateReleaseManifest(manifest ReleaseManifest) error {
	if manifest.BatchID == "" || manifest.BatchVersion <= 0 {
		return Invalid("发布清单缺少批次身份", "manifest")
	}
	if manifest.PageCount == 0 || manifest.PageCount != len(manifest.Entries) {
		return Invalid("发布清单页面数量不一致", "manifest.pageCount")
	}
	if manifest.CharacterCount <= 0 {
		return QualityError("发布清单没有有效识别字符")
	}
	if manifest.OpenIssueCount != 0 || manifest.CriticalCount != 0 {
		return QualityError("发布清单仍包含未关闭问题")
	}
	if manifest.AverageConfidence < MinimumConfidence {
		return QualityError("发布清单平均置信度未达到门槛")
	}
	for index, entry := range manifest.Entries {
		if entry.PageID == "" || entry.Sequence <= 0 || entry.Revision <= 0 || entry.EvidenceDigest == "" {
			return Invalid(fmt.Sprintf("发布清单第 %d 项不完整", index+1), "manifest.entries")
		}
		if index > 0 && manifest.Entries[index-1].Sequence >= entry.Sequence {
			return Conflict("发布清单页面序号没有严格递增")
		}
	}
	return nil
}
