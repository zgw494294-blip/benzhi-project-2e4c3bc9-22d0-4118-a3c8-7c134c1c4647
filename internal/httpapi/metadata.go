package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/benzhi-project/ancient-quality-gate/internal/domain"
)

type mutationMetadata struct {
	RequestID       string
	IdempotencyKey  string
	ExpectedVersion int64
}

func parseMutationMetadata(r *http.Request, bodyVersion int64, versionRequired bool) (mutationMetadata, error) {
	meta := mutationMetadata{
		RequestID:       requestID(r),
		IdempotencyKey:  strings.TrimSpace(r.Header.Get("Idempotency-Key")),
		ExpectedVersion: bodyVersion,
	}
	if len(meta.RequestID) > 128 {
		return meta, domain.Invalid("requestId 长度不能超过 128", "requestId")
	}
	if meta.IdempotencyKey == "" {
		return meta, domain.Invalid("Idempotency-Key 不能为空", "idempotencyKey")
	}
	if len(meta.IdempotencyKey) > 128 {
		return meta, domain.Invalid("Idempotency-Key 长度不能超过 128", "idempotencyKey")
	}
	if value := strings.TrimSpace(r.Header.Get("Expected-Version")); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return meta, domain.Invalid("Expected-Version 必须是正整数", "expectedVersion")
		}
		meta.ExpectedVersion = parsed
	}
	if versionRequired && meta.ExpectedVersion <= 0 {
		return meta, domain.Invalid("expectedVersion 必须是正整数", "expectedVersion")
	}
	return meta, nil
}
