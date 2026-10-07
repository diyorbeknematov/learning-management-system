package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
)

const opaqueTokenBytes = 32

// Generate returns a random URL-safe token, used for refresh tokens and
// password reset tokens. Only its Hash is stored.
func Generate() (string, error) {
	b := make([]byte, opaqueTokenBytes)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Hash returns the SHA-256 hash of a token as hex. The same token always gives
// the same hash, so it can be looked up in the database.
func Hash(raw string) string {
	sum := sha256.Sum256([]byte(raw))

	return hex.EncodeToString(sum[:])
}
