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
	ErrTokenReused         = errors.New("session: token reuse detected")
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

	// Read the immutable identity fields (user, family) so we can build the
	// replacement. These never change for a given token_hash, so staleness
	// is impossible here. The mutable state is verified atomically below.
	current, err := s.repo.GetByTokenHash(ctx, hash)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	raw, next, err := s.newSession(current.UserID, current.TokenFamilyID, deviceID, ip)

	if err != nil {
		return  nil, err
	}

	// Claim the old token and persist the one atomically. this is the only step that authorizes the rotation 
	err = s.repo.ConsumeAndCreate(ctx, hash, next)
	if err == nil {
		return &Issued{RawToken: raw, Session: next}, nil
	}
	if !errors.Is(err, ErrNotConsumable) {
		return nil, err
	}

	// Refused, Re-read for diagonisis - `current` is now known to be out of date, since something changed between it and the UPDATE
	latest, getErr := s.repo.GetByTokenHash(ctx, hash)
	if getErr != nil {
		return nil, ErrInvalidRefreshToken
	}
	if latest.Used {
		if err := s.repo.RevokeFamily(ctx, latest.TokenFamilyID); err != nil {
			return nil, fmt.Errorf("session: revoke family: %w", err)
		}
		return nil, ErrTokenReused
	}
	// Revoked or expired - nothing suspicious just not valid
	return nil, ErrInvalidRefreshToken
}

// Revoke kills a single refresh token (called at logout).
func (s *Service) Revoke(ctx context.Context, rawToken string) error {
	return s.repo.RevokeByTokenHash(ctx, hashToken(rawToken))
}

// mint creates and persists a brand-new session. Used at login.
func (s *Service) mint(ctx context.Context, userID, familyID uuid.UUID, deviceID, ip *string) (*Issued, error) {
	raw, sess, err := s.newSession(userID, familyID, deviceID, ip)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, sess); err != nil {
		return nil, err
	}
	return &Issued{RawToken: raw, Session: sess}, nil	
}

// newSession builds an unsaved session and its raw token. It touches no
// database — persisting it is the caller's job, because login and rotation
// need to persist it in different ways.
func (s *Service) newSession(userID, familyID uuid.UUID, deviceID, ip *string) (string, *Session, error) {
	raw, err := generateToken()
	if err != nil {
		return "", nil, err
	}
	return raw, &Session{
		ID:		uuid.Must(uuid.NewV7()),
		UserID: userID,
		TokenFamilyID: familyID,
		TokenHash: hashToken(raw),
		DeviceID: deviceID,
		IPAddress: ip,
		ExpiresAt: time.Now().Add(refreshTokenTTL),

	}, nil
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