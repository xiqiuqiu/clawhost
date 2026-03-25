package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

func GenerateSessionToken() (string, string, error) {
	raw, err := generateOpaqueToken(32)
	if err != nil {
		return "", "", err
	}
	return raw, hashToken(raw), nil
}

func HashToken(raw string) string {
	return hashToken(raw)
}

func generateOpaqueToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
