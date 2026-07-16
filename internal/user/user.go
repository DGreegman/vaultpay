package user

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Role and Status mirror the CHECK constraint and enum in the database.
// Defining them as typed constants here means the compiler helps us use
// valid values, and the DB is the final backstop.

type Role string

const (
	RoleUser 		Role = "user"
	RoleBusiness	Role = "business"
	RoleAdmin		Role = "admin"
	RoleOps			Role = "ops"
)

type Status string 
const (
	StatusActive		Status = "active"
	StatusSuspended		Status = "suspended"
	StatusDeleted		Status = "deleted"
)

// User is the domain model. It mirrors the users table, but it is the
// application's type — not a database row and not an API response.

type User struct {
	ID           uuid.UUID
	Email        string
	Phone        *string // nullable in the DB, so a pointer here
	PasswordHash string
	Role         Role
	KYCTier      int16
	Status       Status
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Repository is the persistence contract. The service depends on this
// interface, never on the concrete Postgres implementation — so the
// service can be tested with a fake, and the storage can change without
// touching business logic.
type Reposistory interface {
	Create(ctx context.Context, u *User) error 
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
}