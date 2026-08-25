package application

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"

	"github.com/benzhi-project/ancient-quality-gate/internal/ledger"
	"github.com/benzhi-project/ancient-quality-gate/internal/persistence"
)

type credentialVerification struct {
	done     chan struct{}
	response VerifyResponse
	err      error
}

type Service struct {
	Store              *persistence.Store
	Chain              *ledger.Chain
	Signer             *ledger.Signer
	Now                func() time.Time
	verificationMu     sync.Mutex
	verificationFlight map[string]*credentialVerification
}

func New(store *persistence.Store, secret []byte) *Service {
	now := func() time.Time { return time.Now().UTC() }
	return &Service{
		Store:              store,
		Chain:              ledger.NewChain(now),
		Signer:             ledger.NewSigner(secret, now),
		Now:                now,
		verificationFlight: make(map[string]*credentialVerification),
	}
}
func requestHash(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func encode(v any) []byte { b, _ := json.Marshal(v); return b }
