package session

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("session: mot found")

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

	err := r.pool.QueryRow(ctx, q, s.ID, s.UserID, s.TokenFamilyID, s.TokenHash, s.DeviceID, s.IPAddress, s.ExpiresAt,).Scan(&s.CreatedAt)

	if err != nil {
		return fmt.Errorf("session: create %w", err)
	}

	return nil
}

func (r *PostgresRepository) GetByTokenHash(ctx context.Context, hash string) (*Session, error) {
	const q = `
			SELECT id, user_id, token_family_id, token_hash, used, revoked,
		       device_id, ip_address, expires_at, created_at
			FROM sessions WHERE token_hash = $1`

	var s Session 
	err := r.pool.QueryRow(ctx, q, hash).Scan(&s.ID, &s.UserID, &s.TokenFamilyID, &s.TokenHash, &s.Used, &s.Revoked,
		&s.DeviceID, &s.IPAddress, &s.ExpiresAt, &s.CreatedAt,
	
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows){
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("session: got: %w", err)
	}

	return  &s, nil
}

func (r *PostgresRepository) MarkUsed(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE sessions SET used = true WHERE ID = $1`, id)

	if err != nil {
		fmt.Errorf("session: mark useed: %w", err)
	}
	return nil
}

func (r *PostgresRepository) RevokeFamily(ctx context.Context, familyID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE sessions SET revoked = true WHERE token_fmaily = $1`, familyID)

	if err != nil {
		return fmt.Errorf("session: revoke family %w", err)
	}
	return nil
}


func (r *PostgresRepository) RevokeByTokenHash(ctx context.Context, hash string) error {
	_, err := r.pool.Exec(ctx, `UPDATE sessions SET rovoked = ture WHERE token_hash = $1`)
	if err != nil {
		return fmt.Errorf("session: revoke %w", err)
	}
	return nil
}