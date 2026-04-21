package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// CPFHasher implements application CPF hashing using a configured secret key.
type CPFHasher struct {
	key string
}

// NewCPFHasher creates a new keyed CPF hasher.
func NewCPFHasher(key string) *CPFHasher {
	return &CPFHasher{key: key}
}

// Hash converts a CPF into a deterministic opaque hash.
func (h *CPFHasher) Hash(cpf string) string {
	return HashCPF(cpf, h.key)
}

// HashCPF produces a deterministic, irreversible HMAC-SHA256 hash of a CPF.
// Uses a secret key to prevent rainbow-table attacks (only ~200M valid CPFs exist).
func HashCPF(cpf, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(cpf))
	return hex.EncodeToString(mac.Sum(nil))
}
