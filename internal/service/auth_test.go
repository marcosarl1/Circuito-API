package service

import (
	"strings"
	"testing"
	"time"
)

func TestPasswordRoundTrip(t *testing.T) {
	hash, err := HashPassword("correta-123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == "correta-123" || !strings.HasPrefix(hash, "$2a$") {
		t.Fatal("expected bcrypt hash, got plaintext or unknown format")
	}
	if err := VerifyPassword(hash, "correta-123"); err != nil {
		t.Fatalf("valid password rejected: %v", err)
	}
	if err := VerifyPassword(hash, "errada"); err == nil {
		t.Fatal("wrong password accepted")
	}
}

func TestAccessTokenRoundTrip(t *testing.T) {
	now := time.Now()
	raw, err := IssueAccessToken("test-secret-32-chars-minimum-here", "user-1", now)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	subject, err := ParseAccessToken("test-secret-32-chars-minimum-here", raw, now)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if subject != "user-1" {
		t.Fatalf("subject: %q", subject)
	}
}

func TestAccessTokenRejectsExpiryAndTampering(t *testing.T) {
	now := time.Now()
	raw, err := IssueAccessToken("test-secret-32-chars-minimum-here", "user-1", now)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	if _, err := ParseAccessToken("test-secret-32-chars-minimum-here", raw, now.Add(AccessTokenTTL+time.Minute)); err == nil {
		t.Fatal("expired token accepted")
	}
	if _, err := ParseAccessToken("outro-secret", raw, now); err == nil {
		t.Fatal("wrong secret accepted")
	}
	if _, err := ParseAccessToken("test-secret-32-chars-minimum-here", raw+"x", now); err == nil {
		t.Fatal("tampered token accepted")
	}
}

func TestRefreshTokenHashes(t *testing.T) {
	raw, hash, err := NewRefreshToken()
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	if len(raw) != 64 {
		t.Fatalf("raw length: %d", len(raw))
	}
	if hash == raw {
		t.Fatal("hash must differ from raw token")
	}
	if HashRefreshToken(raw) != hash {
		t.Fatal("hash not deterministic")
	}
}
