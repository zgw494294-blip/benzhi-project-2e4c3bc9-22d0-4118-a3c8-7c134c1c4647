package domain

import "fmt"

func ValidateSnapshot(snapshot Snapshot) error {
	if snapshot.Batch.Status != StatusFrozen {
		return StateError("冻结快照中的批次状态不是 frozen")
	}
	if snapshot.Batch.BatchID != snapshot.Manifest.BatchID {
		return Invalid("快照批次与发布清单不匹配", "manifest.batchID")
	}
	if snapshot.Batch.Version != snapshot.Manifest.BatchVersion {
		return Invalid("快照版本与发布清单不匹配", "manifest.batchVersion")
	}
	if snapshot.Quality == nil || !snapshot.Quality.Passed {
		return QualityError("冻结快照没有通过质量门禁")
	}
	if err := ValidateReleaseManifest(snapshot.Manifest); err != nil {
		return err
	}
	rebuilt, err := BuildManifest(snapshot.Batch, snapshot.Pages, snapshot.Issues)
	if err != nil {
		return err
	}
	if rebuilt.PageCount != snapshot.Manifest.PageCount ||
		rebuilt.CharacterCount != snapshot.Manifest.CharacterCount ||
		rebuilt.OpenIssueCount != snapshot.Manifest.OpenIssueCount ||
		rebuilt.CriticalCount != snapshot.Manifest.CriticalCount ||
		len(rebuilt.Entries) != len(snapshot.Manifest.Entries) {
		return Conflict("冻结快照内容与发布清单统计不一致")
	}
	for index := range rebuilt.Entries {
		if rebuilt.Entries[index] != snapshot.Manifest.Entries[index] {
			return Conflict(fmt.Sprintf("冻结快照第 %d 项证据摘要不一致", index+1))
		}
	}
	return nil
}

func ValidateCredential(credential ReleaseCredential) error {
	if credential.CredentialID == "" {
		return Invalid("credentialID 不能为空", "credentialID")
	}
	if credential.BatchID == "" {
		return Invalid("凭据缺少 batchID", "batchID")
	}
	if len(credential.SnapshotDigest) != 64 {
		return Invalid("snapshotDigest 格式不合法", "snapshotDigest")
	}
	if credential.IssuedTo == "" {
		return Invalid("issuedTo 不能为空", "issuedTo")
	}
	if credential.IssuedAt.IsZero() {
		return Invalid("issuedAt 不能为空", "issuedAt")
	}
	if len(credential.Signature) != 64 {
		return Invalid("signature 格式不合法", "signature")
	}
	return nil
}
