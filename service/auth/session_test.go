package auth

import "testing"

func TestGenerateSessionToken(t *testing.T) {
	raw, hashed, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken returned error: %v", err)
	}
	if raw == "" {
		t.Fatal("expected raw session token")
	}
	if hashed == "" {
		t.Fatal("expected hashed session token")
	}
	if raw == hashed {
		t.Fatal("expected hashed token to differ from raw token")
	}
}
