// Package lease implements Ed25519-signed fallback leases.
//
// The energy-server signs a lease with each price response. The energy-app
// stores the lease and, when the server becomes unreachable, checks the
// lease before activating local fallback scrapers. This prevents users
// from gaming the system by blocking the energy server via firewall to
// force indefinite local scraping of third-party sources.
//
// Key distribution:
//   - Private key (seed): energy-server environment variable LEASE_PRIVATE_KEY
//   - Public key: compiled into energy-app binary
package lease

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"time"
)

// DefaultValidity is how long a lease remains valid after issuance.
const DefaultValidity = 8 * time.Hour

// Lease authorizes an installation to use local fallback for a limited time.
type Lease struct {
	InstallUUID string `json:"install_uuid"`
	IssuedAt    int64  `json:"issued_at"`  // Unix timestamp
	ExpiresAt   int64  `json:"expires_at"` // Unix timestamp
	Signature   string `json:"signature"`  // Base64-encoded Ed25519 signature
}

// Valid checks whether the lease is current (not expired).
func (l Lease) Valid() bool {
	now := time.Now().Unix()
	return now >= l.IssuedAt && now < l.ExpiresAt
}

// message returns the canonical byte string that is signed/verified.
func (l Lease) message() []byte {
	return []byte(fmt.Sprintf("%s:%d:%d", l.InstallUUID, l.IssuedAt, l.ExpiresAt))
}

// Signer creates new leases. Used by the energy-server only.
type Signer struct {
	key      ed25519.PrivateKey
	validity time.Duration
}

// NewSigner creates a signer from a base64-encoded Ed25519 seed (32 bytes).
func NewSigner(seedBase64 string, validity time.Duration) (*Signer, error) {
	seed, err := base64.StdEncoding.DecodeString(seedBase64)
	if err != nil {
		return nil, fmt.Errorf("decode seed: %w", err)
	}
	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("invalid seed length: got %d, want %d", len(seed), ed25519.SeedSize)
	}
	return &Signer{
		key:      ed25519.NewKeyFromSeed(seed),
		validity: validity,
	}, nil
}

// PublicKeyBase64 returns the base64-encoded public key for embedding in clients.
func (s *Signer) PublicKeyBase64() string {
	pub := s.key.Public().(ed25519.PublicKey)
	return base64.StdEncoding.EncodeToString(pub)
}

// Issue creates a signed lease for the given installation UUID.
func (s *Signer) Issue(installUUID string) Lease {
	now := time.Now()
	l := Lease{
		InstallUUID: installUUID,
		IssuedAt:    now.Unix(),
		ExpiresAt:   now.Add(s.validity).Unix(),
	}
	sig := ed25519.Sign(s.key, l.message())
	l.Signature = base64.StdEncoding.EncodeToString(sig)
	return l
}

// Verifier checks lease signatures. Used by the energy-app with an embedded public key.
type Verifier struct {
	key ed25519.PublicKey
}

// NewVerifier creates a verifier from a base64-encoded Ed25519 public key (32 bytes).
func NewVerifier(pubBase64 string) (*Verifier, error) {
	pub, err := base64.StdEncoding.DecodeString(pubBase64)
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}
	if len(pub) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key length: got %d, want %d", len(pub), ed25519.PublicKeySize)
	}
	return &Verifier{key: ed25519.PublicKey(pub)}, nil
}

// Verify checks that the lease has a valid signature and is not expired.
func (v *Verifier) Verify(l Lease) bool {
	if !l.Valid() {
		return false
	}
	sig, err := base64.StdEncoding.DecodeString(l.Signature)
	if err != nil {
		return false
	}
	return ed25519.Verify(v.key, l.message(), sig)
}
