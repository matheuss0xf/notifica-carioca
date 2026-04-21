package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// ValidateSignature verifies an HMAC-SHA256 signature in the format "sha256=<hex>".
// Uses constant-time comparison to prevent timing attacks.
func ValidateSignature(body []byte, signature, secret string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	expectedHex := signature[7:] // strip "sha256=" prefix

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	computedHex := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expectedHex), []byte(computedHex))
}

// ComputeSignature generates an HMAC-SHA256 signature for the given body.
// Returns in the format "sha256=<hex>". Used by tests.
func ComputeSignature(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return fmt.Sprintf("sha256=%s", hex.EncodeToString(mac.Sum(nil)))
}
