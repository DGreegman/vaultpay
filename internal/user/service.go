package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// bcryptCost is the work factor. Higher = slower = harder to brute-force.
// 12 is a sensible default in 2026: meaningfully stronger than bcrypt's
// default of 10, without making a login take an uncomfortable amount of time.
const bcryptCost = 12


// ErrInvalidCredentials is returned by Authenticate. It is deliberately
// vague: we never reveal whether the email or the password was wrong,
// so an attacker cannot use the error to discover which emails exist.
var ErrInvalidCredentials = errors.New("user: invalid credentials")


// Service holds the business logic for users. It depends on the
// Repository interface, so it can be tested with a fake implementation.

type Service struct {
	repo Reposistory
}

func NewService(repo Reposistory) *Service {
	return &Service{repo: repo}
}


// RegisterInput is what the service needs to create a user. It is the
// service's own input type — not the HTTP request and not the DB row.
type RegisterInput struct {
	Email    string
	Phone    *string
	Password string
}

// Register hashes the password, mints an ID, and persists the user.
func (s *Service) Register(ctx context.Context, in RegisterInput) (*User, error){
	email := normalizeEmail(in.Email)

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcryptCost)

	if err != nil {
		return nil, fmt.Errorf("user: hash password %w", err)
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("user: generate id: %w", err)
	}
	u := &User{
		ID :			id,
		Email: 			email,
		Phone: 			in.Phone,
		PasswordHash: 	string(hash),
		Role:         	RoleUser,   // everyone starts as a plain user
		KYCTier:      	0,          // unverified
		Status:       	StatusActive,

	}

	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err // already a typed domain error (ErrEmailTaken, etc.)
	}

	return u, nil
}


// Authenticate verifies an email/password pair and return the user
func(s *Service) Authenticate(ctx context.Context, email, password string) (*User, error) {
	u, err := s.repo.GetByEmail(ctx, normalizeEmail(email))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			// Run a dummy hash comparison anyway (see note below), then
			// return the same vague error as a wrong password

			_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil{
		return nil, ErrInvalidCredentials
	}
	return u, nil
}

// normalizeEmail trims whitespace. Case-insensitivity is handled by the
// citext column in the database, so we do not lowercase here — but we do
// trim, since a trailing space is never intended.
func normalizeEmail(email string) string {
	return strings.TrimSpace(email)
}

// dummyHash is a precomputed bcrypt hash used to keep Authenticate's
// timing roughly constant whether or not the email exists (see below).
var dummyHash = []byte("$2a$12$XNB.SMb/lwprcYtglIrNk.rzR5/zC4edQtiUxu5d2KA94nozzRiPu")