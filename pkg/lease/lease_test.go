package lease

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"
)

func generateTestSeed(t *testing.T) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return base64.StdEncoding.EncodeToString(priv.Seed())
}

func TestSignAndVerify(t *testing.T) {
	seed := generateTestSeed(t)
	signer, err := NewSigner(seed, 8*time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	verifier, err := NewVerifier(signer.PublicKeyBase64())
	if err != nil {
		t.Fatal(err)
	}

	l := signer.Issue("test-uuid-123")

	if !verifier.Verify(l) {
		t.Fatal("valid lease rejected")
	}
}

func TestExpiredLease(t *testing.T) {
	seed := generateTestSeed(t)
	// Create with 0 validity so it expires immediately
	signer, _ := NewSigner(seed, 0)
	verifier, _ := NewVerifier(signer.PublicKeyBase64())

	l := signer.Issue("test-uuid")
	// Lease expires at issuance (validity=0), so ExpiresAt == IssuedAt
	if verifier.Verify(l) {
		t.Fatal("expired lease accepted")
	}
}

func TestTamperedLease(t *testing.T) {
	seed := generateTestSeed(t)
	signer, _ := NewSigner(seed, 8*time.Hour)
	verifier, _ := NewVerifier(signer.PublicKeyBase64())

	l := signer.Issue("test-uuid")
	l.InstallUUID = "different-uuid" // tamper
	if verifier.Verify(l) {
		t.Fatal("tampered lease accepted")
	}
}

func TestWrongKey(t *testing.T) {
	seed1 := generateTestSeed(t)
	seed2 := generateTestSeed(t)

	signer, _ := NewSigner(seed1, 8*time.Hour)
	otherSigner, _ := NewSigner(seed2, 8*time.Hour)
	verifier, _ := NewVerifier(otherSigner.PublicKeyBase64())

	l := signer.Issue("test-uuid")
	if verifier.Verify(l) {
		t.Fatal("lease signed with wrong key accepted")
	}
}

func TestInvalidSeed(t *testing.T) {
	_, err := NewSigner("not-valid-base64!!!", 8*time.Hour)
	if err == nil {
		t.Fatal("expected error for invalid seed")
	}

	_, err = NewSigner(base64.StdEncoding.EncodeToString([]byte("tooshort")), 8*time.Hour)
	if err == nil {
		t.Fatal("expected error for wrong seed length")
	}
}

func TestInvalidPublicKey(t *testing.T) {
	_, err := NewVerifier("not-valid-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid public key")
	}

	_, err = NewVerifier(base64.StdEncoding.EncodeToString([]byte("short")))
	if err == nil {
		t.Fatal("expected error for wrong key length")
	}
}

func TestLeaseValid(t *testing.T) {
	l := Lease{
		IssuedAt:  time.Now().Add(-1 * time.Hour).Unix(),
		ExpiresAt: time.Now().Add(1 * time.Hour).Unix(),
	}
	if !l.Valid() {
		t.Fatal("current lease should be valid")
	}

	l.ExpiresAt = time.Now().Add(-1 * time.Minute).Unix()
	if l.Valid() {
		t.Fatal("past lease should be invalid")
	}
}
