package ledger

import (
	"crypto/rand"
	"encoding/hex"
)

func NewID(prefix string) string {
	var raw [10]byte
	_, _ = rand.Read(raw[:])
	return prefix + "_" + hex.EncodeToString(raw[:])
}
