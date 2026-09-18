package transaction

import (
	"context"
	"errors"
	"fmt"

	"github.com/DGreegman/vaultpay/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrNotFound = errors.New("transaction: not found")

type PostgresRepository struct {
	queries *store.Queries
}

func NewPostgresRespository(queries *store.Queries) *PostgresRepository {
	return &PostgresRepository{
		queries: queries,
	}
}

func (r *PostgresRepository) WithTx(tx pgx.Tx) *PostgresRepository {

	return &PostgresRepository{
		queries: r.queries.WithTx(tx),
	}

}

var _ Repository = (*PostgresRepository)(nil)

func (r *PostgresRepository) CreateTransaction(ctx context.Context, params CreateTransactionParams) (*Transaction, bool, error) {

	row, err := r.queries.CreateTransaction(ctx, store.CreateTransactionParams{
		ID:                  toPgUUID(params.ID),
		UserID:              toPgUUID(params.ID),
		IdempotencyKey:      params.IdempotencyKey,
		Type:                store.TransactionType(params.Type),
		Status:              store.TransactionStatus(params.Status),
		Amount:              params.Amount,
		Currency:            store.Currency(params.Currency),
		SourceWalletID:      toNullablePgUUID(params.SourceWalletID),
		DestinationWalletID: toNullablePgUUID(params.DestinationWalletID),
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			existing, err := r.GetTransactionByIdempotencyKey(ctx, params.UserID, params.IdempotencyKey)

			if err != nil {
				return nil, false, err
			}
			return existing, false, nil
		}

		return nil, false, fmt.Errorf("transaction: create %w", err)
	}

	transaction := &Transaction{
		ID:                  fromPgUUID(row.ID),
		UserID:              fromPgUUID(row.UserID),
		IdempotencyKey:      row.IdempotencyKey,
		Type:                TransactionType(row.Type),
		Status:              TransactionStatus(row.Status),
		Amount:              row.Amount,
		Currency:            Currency(row.Currency),
		SourceWalletID:      fromNullablePgUUID(row.SourceWalletID),
		DestinationWalletID: fromNullablePgUUID(row.SourceWalletID),
		CreatedAt:           row.CreatedAt.Time,
	}

	return transaction, true, nil
}

func (r *PostgresRepository) GetTransactionByIdempotencyKey(ctx context.Context, userID uuid.UUID, idempotencyKey string) (*Transaction, error) {

	row, err := r.queries.GetTransactionByIdempotencyKey(ctx, store.GetTransactionByIdempotencyKeyParams{
		UserID:         toPgUUID(userID),
		IdempotencyKey: idempotencyKey,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("transaction: get by idempotency key: %w", err)
	}

	transaction := &Transaction{
		ID:                  fromPgUUID(row.ID),
		UserID:              fromPgUUID(row.UserID),
		IdempotencyKey:      row.IdempotencyKey,
		Type:                TransactionType(row.Type),
		Status:              TransactionStatus(row.Status),
		Amount:              row.Amount,
		Currency:            Currency(row.Currency),
		SourceWalletID:      fromNullablePgUUID(row.SourceWalletID),
		DestinationWalletID: fromNullablePgUUID(row.DestinationWalletID),
	}

	return transaction, nil
}
