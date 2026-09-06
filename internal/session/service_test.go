package session

import (
	"strings"
	"testing"
)

func TestHashTokenIsDeterminitic(t *testing.T){

	const raw = "some-refresh-token"

	if hashToken(raw) != hashToken(raw) {
		t.Fatal("hashToken is not detrminitic: lookups by would fail")
	}
}

func TestHashTokenDistinguishesInputs(t *testing.T){
	if hashToken("token-a") == hashToken("token-b"){
		t.Fatal("different tokens produced the same hash")
	}
}

// The raw token must never be recoverable from what we store. This is the
// property that makes a leaked sessions table useless to an attacker.
func TestHashTokenDoesNotContainRawToken(t *testing.T){
	const raw = "super-secret-token"
	got := hashToken(raw)

	if strings.Contains(got, raw) {
		t.Fatalf("hash contains the raw token: %q", got)
	}

	if len(got) != 64 {
		t.Fatalf("expected 64 hex chars from SHA-256, got %d", len(got))
	}
}