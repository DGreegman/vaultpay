package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Sentinel errors the service and handlers can check with errors.Is,
// instead of inspecting raw database errors everywhere.
var (
	ErrNotFound   = errors.New("user: not found")
	ErrEmailTaken = errors.New("user: email already registered")
	ErrPhoneTaken = errors.New("user: phone already registered")
)

// PostgresRepository implements Repository against a pgx pool.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepostory wires the respoistory with its dependency (the pool)
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// Compile-time assertion that PostgresRepository satisfies Repository.
// If a method signature drifts, the build fails here — not at some
// call site far away.

var _ Reposistory = (*PostgresRepository)(nil)

func (r *PostgresRepository) Create(ctx context.Context, u *User) error {
	const q = `
		INSERT INTO users (id, email, phone, password_hash, role, kyc_tier, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at`

		err := r.pool.QueryRow(ctx, q, u.ID, u.Email, u.Phone, u.PasswordHash, u.Role, u.KYCTier, u.Status,).Scan(&u.CreatedAt, &u.UpdatedAt)

		if err != nil {
			return mapCreateError(err)
		}
		return  nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {

	const q = `
		SELECT id, email, phone, password_hash, role, kyc_tier, status, created_at, updated_at
		FROM users WHERE id = $1`

		return scanUser(r.pool.QueryRow(ctx, q, id))
}

func (r *PostgresRepository) GetByEmail(ctx context.Context, email string) (*User, error){
	const q = `
		SELECT id, email, phone, password_hash, role, kyc_tier, status, created_at, updated_at
		FROM users WHERE email = $1`

	return scanUser(r.pool.QueryRow(ctx, q, email))
	
}

// row is the small interface QueryRow satisfies, so scanUser can be
// reused by any single-row query above.
type row interface {
	Scan(dest ...any) error
}

func scanUser(rw row) (*User, error){
	var u User 
	err := rw.Scan(
		&u.ID, &u.Email, &u.Phone, &u.PasswordHash,
		&u.Role, &u.KYCTier, &u.Status, &u.CreatedAt, &u.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("user: scan: %w", err)
	}
	return &u, nil
}

// mapCreateError turns a Postgres unique-violation into a typed domain
// error, so callers never have to know Postgres error codes.

func mapCreateError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505"{
		switch pgErr.ConstraintName {
		case "users_email_key":
			return ErrEmailTaken
		case "users_phone_key":
			return ErrPhoneTaken
		}

	}
	return fmt.Errorf("user: create: %w", err)
}