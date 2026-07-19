package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// refreshTokenTTL is how long a refresh token lives. Much longer than an
// access token, because its job is to avoid forcing re-login — but it is
// revocable (stored in the DB), which is what makes that safe.

const refreshTokenTTL = 7 * 24 * time.Hour

var (
	ErrInvalidRefreshToken = errors.New("session: invalid refresh token")
	ErrTokenResused        = errors.New("session: token reuse detected")
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}

}

// Issued is the result of issuing/rotating: the RAW token to hand the
// client, plus the session metadata. The raw token exists only here and
// in the client's hands — never in the database.
type Issued struct {
	RawToken string
	Session  *Session
}

// Issue creates a brand-new refresh token in a brand-new family (called
// at login).

func (s *Service) Issue(ctx context.Context, userID uuid.UUID, deviceID, ip *string) (*Issued, error) {
	return s.mint(ctx, userID, uuid.Must(uuid.NewV7()), deviceID, ip)
}

// Rotate validates a presented refresh token and issues its replacement
// in the same family. This is where theft is detected.
func (s *Service) Rotate(ctx context.Context, rawToken string, deviceID, ip *string) (*Issued, error) {

	hash := hashToken(rawToken)

	sess, err := s.repo.GetByTokenHash(ctx, hash)

	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, err
	}

	// THEFT DETECTION: a token that was already used is being presented
	// again. Either the legitimate holder or an attacker has a stale
	// token — we cannot tell which, so we revoke the ENTIRE family,
	// logging everyone out. The real user re-logs-in safely; the
	// attacker is locked out.
	if sess.Used {
		_ = s.repo.RevokeFamily(ctx, sess.TokenFamilyID)

		return nil, ErrTokenResused
	}
	if sess.Revoked {
		return nil, ErrInvalidRefreshToken
	}

	if time.Now().After(sess.ExpiresAt){
		return nil, ErrInvalidRefreshToken
	}

	return s.mint(ctx, sess.UserID, sess.TokenFamilyID, deviceID, ip)
} 

// Revoke kills a single refresh token (called at logout).
func(s *Service) Revoke(ctx context.Context, rawToken string) error {
	return s.repo.RevokeByTokenHash(ctx, hashToken(rawToken))
}

// mint generates a new random token, stores its hash in the given family,
// and returns the raw token.
func (s *Service) mint(ctx context.Context, userID, familyID uuid.UUID, deviceID, ip *string) (*Issued, error) {
	raw, err := generateToken()

	if err != nil {
		return nil, err 
	}

	sess := &Session{
		ID: uuid.Must(uuid.NewV7()),
		UserID: userID,
		TokenFamilyID: familyID,
		TokenHash: hashToken(raw),
		DeviceID: deviceID,
		IPAddress: ip,
		ExpiresAt: time.Now().Add(refreshTokenTTL),
	}

	if err := s.repo.Create(ctx, sess); err != nil {
		return nil, err
	}
	return &Issued{RawToken: raw, Session: sess}, nil
}

// generateToken produces 256 bits of cryptographic randomness as a
// URL-safe string. crypto/rand, never math/rand — a predictable token
// is a forgeable token.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("session: generate token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashToken returns the SHA-256 hex digest of a token. Fast and salt-free
// is correct here: the input is already high-entropy random, so there is
// nothing to brute-force and no need for bcrypt's deliberate slowness.
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}