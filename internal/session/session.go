package session

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Session is one refresh-token row. The raw token is never stored —
// only its hash. All fields mirror the sessions table.

type Session struct {
	ID				uuid.UUID
	UserID			uuid.UUID
	TokenFamilyID	uuid.UUID
	TokenHash		string
	Used			bool
	Revoked			bool
	DeviceID		*string
	IPAddress		*string
	ExpiresAt		time.Time
	CreatedAt		time.Time
}

// Repository is the persistence contract for sessions.

type Repository interface {
	Create(ctx context.Context, s *Session) error
	GetByTokenHash(ctx context.Context, hash string) (*Session, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error 
	RevokeFamily(ctx context.Context, familyID uuid.UUID) error 
	RevokeByTokenHash(ctx context.Context, hash string) error
}