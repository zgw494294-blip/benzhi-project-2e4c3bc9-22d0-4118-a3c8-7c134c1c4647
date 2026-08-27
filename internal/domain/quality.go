package domain

import (
	"math"
	"sort"
	"strings"
	"time"
)

const MinimumConfidence = 0.80
const MinimumCoverage = 0.95

func (p PageEvidence) Validate() error {
	if p.PageID == "" || p.BatchID == "" {
		return Invalid("pageID 和 batchID 不能为空", "pageID")
	}
	if p.Sequence <= 0 {
		return Invalid("sequence 必须为正数", "sequence")
	}
	if len(p.ImageDigest) < 8 {
		return Invalid("imageDigest 长度不足", "imageDigest")
	}
	if p.CharacterCount < 0 {
		return Invalid("characterCount 不能为负数", "characterCount")
	}
	if p.Confidence < 0 || p.Confidence > 1 {
		return Invalid("confidence 必须在 0 到 1 之间", "confidence")
	}
	return nil
}

func EvaluateQuality(batch DigitizationBatch, pages []PageEvidence, issues []QualityIssue, now time.Time) QualityResult {
	result := QualityResult{Passed: true, CheckedAt: now, Issues: []QualityIssue{}}
	if len(pages) == 0 {
		result.Passed = false
		result.Issues = append(result.Issues, QualityIssue{Category: "coverage", Severity: "critical", Description: "批次没有页面证据"})
		return result
	}
	unresolved := make(map[string]QualityIssue)
	for _, issue := range issues {
		if issue.ResolvedAt == nil {
			unresolved[issue.PageID+":"+issue.Category] = issue
		}
	}
	counts := make([]int, 0, len(pages))
	maxSequence := 0
	sequences := make(map[int]struct{}, len(pages))
	for _, page := range pages {
		if page.CharacterCount > 0 {
			counts = append(counts, page.CharacterCount)
		}
		if page.Sequence > maxSequence {
			maxSequence = page.Sequence
		}
		sequences[page.Sequence] = struct{}{}
	}
	sort.Ints(counts)
	medianCharacters := 0
	if len(counts) > 0 {
		medianCharacters = counts[len(counts)/2]
	}
	var confidence float64
	for _, page := range pages {
		confidence += page.Confidence
		if page.Confidence < MinimumConfidence {
			severity := "major"
			if page.Confidence < 0.6 {
				severity = "critical"
			}
			addAutomaticIssue(&result, unresolved, QualityIssue{BatchID: batch.BatchID, PageID: page.PageID, Category: "confidence", Severity: severity, Description: "页面 OCR 置信度未达到质量门槛"})
		}
		if page.CharacterCount <= 0 || characterDeviation(page.CharacterCount, medianCharacters) > 0.5 {
			addAutomaticIssue(&result, unresolved, QualityIssue{BatchID: batch.BatchID, PageID: page.PageID, Category: "character_count", Severity: "major", Description: "页面识别字符数与批次中位数偏差超过 50%"})
		}
	}
	if maxSequence > 0 {
		result.Coverage = float64(len(sequences)) / float64(maxSequence)
	}
	result.AverageConfidence = confidence / float64(len(pages))
	for _, existing := range issues {
		if existing.ResolvedAt == nil {
			key := existing.PageID + ":" + existing.Category
			if _, wasAutomatic := unresolved[key]; wasAutomatic && !containsIssue(result.Issues, existing.IssueID) {
				result.Issues = append(result.Issues, existing)
			}
		}
	}
	if result.Coverage < MinimumCoverage {
		result.Passed = false
		addAutomaticIssue(&result, unresolved, QualityIssue{BatchID: batch.BatchID, Category: "coverage", Severity: "major", Description: "页面序号覆盖率未达到 95%"})
	}
	if result.AverageConfidence < MinimumConfidence {
		result.Passed = false
	}
	if len(result.Issues) > 0 {
		result.Passed = false
	}
	sort.Slice(result.Issues, func(i, j int) bool { return strings.Compare(result.Issues[i].Severity, result.Issues[j].Severity) < 0 })
	return result
}

func characterDeviation(value, median int) float64 {
	if median == 0 {
		if value == 0 {
			return 0
		}
		return 1
	}
	return math.Abs(float64(value-median)) / float64(median)
}

func addAutomaticIssue(result *QualityResult, unresolved map[string]QualityIssue, issue QualityIssue) {
	if existing, ok := unresolved[issue.PageID+":"+issue.Category]; ok {
		if !containsIssue(result.Issues, existing.IssueID) {
			result.Issues = append(result.Issues, existing)
		}
		return
	}
	result.Issues = append(result.Issues, issue)
}

func containsIssue(issues []QualityIssue, id string) bool {
	if id == "" {
		return false
	}
	for _, issue := range issues {
		if issue.IssueID == id {
			return true
		}
	}
	return false
}

func ValidatePageRegistration(page PageEvidence, existing []PageEvidence) error {
	if err := page.Validate(); err != nil {
		return err
	}
	for _, other := range existing {
		if other.PageID == page.PageID {
			return Conflict("pageID 已经登记")
		}
		if other.Sequence == page.Sequence {
			return Conflict("sequence 已经被其他页面使用")
		}
	}
	return nil
}

func ValidateIssue(issue QualityIssue) error {
	if issue.IssueID == "" || issue.BatchID == "" {
		return Invalid("issueID 和 batchID 不能为空", "issueID")
	}
	if issue.Severity != "critical" && issue.Severity != "major" && issue.Severity != "minor" {
		return Invalid("severity 不合法", "severity")
	}
	if issue.Description == "" {
		return Invalid("description 不能为空", "description")
	}
	return nil
}

func ResolveIssue(issue *QualityIssue, evidence PageEvidence, evidenceID string, now time.Time) error {
	if issue.ResolvedAt != nil {
		return Conflict("问题已经解决")
	}
	if evidence.BatchID != issue.BatchID {
		return Invalid("整改证据不属于当前批次", "evidenceID")
	}
	if evidence.PageID != issue.PageID {
		return Invalid("整改证据页面不匹配", "evidenceID")
	}
	if evidenceID == "" {
		return Invalid("correctedEvidenceID 不能为空", "correctedEvidenceID")
	}
	issue.CorrectedEvidenceID = evidenceID
	issue.Disposition = "resolved"
	issue.ResolvedAt = &now
	return nil
}
