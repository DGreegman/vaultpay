package session

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound      = errors.New("session: not found")
	ErrNotConsumable = errors.New("session: token not consumable")
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {

	return &PostgresRepository{
		pool: pool,
	}
}

var _ Repository = (*PostgresRepository)(nil)

func (r *PostgresRepository) Create(ctx context.Context, s *Session) error {
	const q = `
		INSERT INTO sessions (id, user_id, token_family_id, token_hash, device_id, ip_address, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at`

	err := r.pool.QueryRow(ctx, q, s.ID, s.UserID, s.TokenFamilyID, s.TokenHash, s.DeviceID, s.IPAddress, s.ExpiresAt).Scan(&s.CreatedAt)

	if err != nil {
		return fmt.Errorf("session: create %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetByTokenHash(ctx context.Context, hash string) (*Session, error) {
	const q = `
			SELECT id, user_id, token_family_id, token_hash, used, revoked,
		       device_id, ip_address::text, expires_at, created_at
			FROM sessions WHERE token_hash = $1`

	var s Session
	err := r.pool.QueryRow(ctx, q, hash).Scan(&s.ID, &s.UserID, &s.TokenFamilyID, &s.TokenHash, &s.Used, &s.Revoked,
		&s.DeviceID, &s.IPAddress, &s.ExpiresAt, &s.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("session: got: %w", err)
	}

	return &s, nil
}

// ConsumeAndCreate claims a refresh token and inserts its replacement
// inside one transaction. Either both happen or neither does — a token is
// never burnt without its successor existing.

func (r *PostgresRepository) ConsumeAndCreate(ctx context.Context, oldHash string, next *Session)  error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return  fmt.Errorf("session: begin: %w", err)
	}
	// Safety net: if we return before Commit for any reason, this undoes
	// everything. After a successful Commit it is a no-op, so it is always
	// safe to defer unconditionally.
	defer tx.Rollback(ctx)

	const consumeQ = `
			UPDATE sessions
		SET used = true
		WHERE token_hash = $1
		  AND used = false
		  AND revoked = false
		  AND expires_at > now()
		RETURNING id, user_id, token_family_id, token_hash, used, revoked,device_id, ip_address::text, expires_at, created_at`

	var old Session

	err = tx.QueryRow(ctx, consumeQ, oldHash).Scan(&old.ID, &old.UserID, &old.TokenFamilyID, &old.TokenHash, &old.Used, &old.Revoked, &old.DeviceID, &old.IPAddress, &old.ExpiresAt, &old.CreatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return  ErrNotConsumable
		}
		return fmt.Errorf("session: consume %w", err)
	}

	const createQ = `
		INSERT INTO sessions (id, user_id, token_family_id, token_hash, device_id, ip_address, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at`
	err = tx.QueryRow(ctx, createQ, next.ID, next.UserID, next.TokenFamilyID, next.TokenHash,
		next.DeviceID, next.IPAddress, next.ExpiresAt).Scan(&next.CreatedAt)
	if err != nil {
		return fmt.Errorf("session: create replacement: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("session: commit: %w", err)
	}
	return nil
}

func (r *PostgresRepository) RevokeFamily(ctx context.Context, familyID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE sessions SET revoked = true WHERE token_family_id = $1`, familyID)

	if err != nil {
		return fmt.Errorf("session: revoke family %w", err)
	}
	return nil
}

