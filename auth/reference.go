package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// SessionReference возвращает безопасный opaque reference для transport/session binding.
func SessionReference(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}
