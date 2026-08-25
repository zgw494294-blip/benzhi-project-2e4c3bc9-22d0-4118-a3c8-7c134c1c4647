package domain

func (b DigitizationBatch) CanAddEvidence() bool {
	return b.Status == StatusDraft || b.Status == StatusEvidence
}
func (b DigitizationBatch) CanReviseEvidence() bool {
	return b.Status == StatusDraft || b.Status == StatusEvidence || b.Status == StatusReview || b.Status == StatusRejected
}
func (b DigitizationBatch) CanQualityCheck() bool {
	return b.Status == StatusEvidence || b.Status == StatusReview
}
func (b DigitizationBatch) CanReview() bool {
	return b.Status == StatusReview || b.Status == StatusRejected
}
func (b DigitizationBatch) CanFreeze() bool { return b.Status == StatusApproved }

func (b *DigitizationBatch) MarkEvidence() error {
	if !b.CanAddEvidence() {
		return StateError("当前批次状态不允许登记证据")
	}
	b.Status = StatusEvidence
	b.Version++
	return nil
}

func (b *DigitizationBatch) MarkEvidenceRevision() error {
	if !b.CanReviseEvidence() {
		return StateError("当前批次状态不允许修订 OCR 证据")
	}
	if b.Status == StatusDraft {
		b.Status = StatusEvidence
	}
	b.Version++
	return nil
}

func (b *DigitizationBatch) BeginReview() error {
	if !b.CanQualityCheck() {
		return StateError("当前批次状态不允许质量检查")
	}
	b.Status = StatusReview
	b.Version++
	return nil
}

func (b *DigitizationBatch) Approve() error {
	if b.Status != StatusReview {
		return StateError("只有待复核批次才能批准")
	}
	b.Status = StatusApproved
	b.Version++
	return nil
}

func (b *DigitizationBatch) Reject() error {
	if b.Status != StatusReview {
		return StateError("只有待复核批次才能退回")
	}
	b.Status = StatusRejected
	b.Version++
	return nil
}

func (b *DigitizationBatch) Freeze() error {
	if !b.CanFreeze() {
		return StateError("只有批准批次才能冻结")
	}
	b.Status = StatusFrozen
	b.Version++
	return nil
}

func (b DigitizationBatch) Validate() error {
	if b.BatchID == "" {
		return Invalid("batchID 不能为空", "batchID")
	}
	if b.Title == "" {
		return Invalid("title 不能为空", "title")
	}
	if b.Edition == "" {
		return Invalid("edition 不能为空", "edition")
	}
	if b.Owner == "" {
		return Invalid("owner 不能为空", "owner")
	}
	return nil
}
