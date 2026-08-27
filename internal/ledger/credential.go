package ledger

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/benzhi-project/ancient-quality-gate/internal/domain"
)

type Signer struct {
	secret []byte
	now    func() time.Time
}

func NewSigner(secret []byte, now func() time.Time) *Signer {
	if len(secret) == 0 {
		secret = []byte("benzhi-local-ancient-book-release-key")
	}
	if now == nil {
		now = time.Now
	}
	return &Signer{secret: append([]byte(nil), secret...), now: now}
}
func credentialMessage(c domain.ReleaseCredential) string {
	return strings.Join([]string{c.CredentialID, c.BatchID, c.SnapshotDigest, c.IssuedTo, c.IssuedAt.UTC().Format(time.RFC3339Nano)}, "|")
}
func (s *Signer) signature(c domain.ReleaseCredential) string {
	m := hmac.New(sha256.New, s.secret)
	_, _ = m.Write([]byte(credentialMessage(c)))
	return hex.EncodeToString(m.Sum(nil))
}
func (s *Signer) Issue(batchID, digest, issuedTo string) domain.ReleaseCredential {
	c := domain.ReleaseCredential{CredentialID: NewID("cred"), BatchID: batchID, SnapshotDigest: digest, IssuedTo: issuedTo, IssuedAt: s.now().UTC()}
	c.Signature = s.signature(c)
	return c
}
func (s *Signer) Verify(c domain.ReleaseCredential, snapshot domain.Snapshot) (bool, string) {
	if err := domain.ValidateCredential(c); err != nil {
		return false, err.Error()
	}
	if c.RevokedAt != nil {
		return false, "凭据已撤销"
	}
	if err := domain.ValidateSnapshot(snapshot); err != nil {
		return false, err.Error()
	}
	actualDigest, err := domain.SnapshotDigest(snapshot)
	if err != nil {
		return false, "无法重新计算冻结快照摘要"
	}
	if actualDigest != snapshot.Digest {
		return false, "冻结快照内容摘要无效"
	}
	if snapshot.Digest != c.SnapshotDigest {
		return false, "冻结快照摘要不匹配"
	}
	if !hmac.Equal([]byte(c.Signature), []byte(s.signature(c))) {
		return false, "凭据签名无效"
	}
	return true, "凭据有效"
}
