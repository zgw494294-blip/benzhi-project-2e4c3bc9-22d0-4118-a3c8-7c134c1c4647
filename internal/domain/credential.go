package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func CanonicalSnapshot(snapshot Snapshot) ([]byte, error) {
	copy := snapshot
	copy.Digest = ""
	return json.Marshal(copy)
}

func SnapshotDigest(snapshot Snapshot) (string, error) {
	b, err := CanonicalSnapshot(snapshot)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
