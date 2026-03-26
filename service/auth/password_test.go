package auth

import "testing"

func TestHashPassword(t *testing.T) {
	hashed, err := HashPassword("secret-123")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hashed == "" {
		t.Fatal("expected hash to be non-empty")
	}
	if hashed == "secret-123" {
		t.Fatal("expected hash to differ from plain password")
	}
}

func TestVerifyPassword(t *testing.T) {
	hashed, err := HashPassword("secret-123")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if !VerifyPassword(hashed, "secret-123") {
		t.Fatal("expected password verification to succeed")
	}
	if VerifyPassword(hashed, "wrong-secret") {
		t.Fatal("expected password verification to fail for wrong password")
	}
}
