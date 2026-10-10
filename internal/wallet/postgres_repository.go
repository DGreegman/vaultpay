package wallet

import (
	"context"
	"errors"
	"fmt"

	"github.com/DGreegman/vaultpay/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrNotFound = errors.New("transaction: not found")

type PostgresRepository struct {
	queries *store.Queries
}

func NewPostgresRepository(queries *store.Queries) *PostgresRepository {
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

func (r *PostgresRepository)GetBalanceForUpdate(ctx context.Context, walletID uuid.UUID) (*Balance, error){

	row, err :=  r.queries.GetWalletBalanceForUpdate(ctx, toPgUUID(walletID))

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("wallet balance for update %w", err)
	}

	balance := &Balance{
		WalletID: 			walletID,
		AvailableBalance: 	row.AvailableBalance,
		HeldBalance: 		row.HeldBalance,
		UpdatedAt: 			row.UpdatedAt.Time,
	}

	return balance, nil

}

func (r *PostgresRepository) GetWalletForUpdate(ctx context.Context, walletID uuid.UUID) (*Wallet, error) {

	row, err := r.queries.GetWalletForUpdate(ctx, toPgUUID(walletID))

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get wallet for update: %w", err)
	}

	wallet := &Wallet {
		ID: 		fromPgUUID(row.ID),
		UserID: 	fromPgUUID(row.UserID),
		Currency: 	Currency(row.Currency),
		Status: 	Status(row.Status),

	}

	return wallet, nil
}


func toPgUUID(id uuid.UUID) pgtype.UUID {

	return pgtype.UUID{
		Bytes: id,
		Valid: true,
	}

}

func fromPgUUID(id pgtype.UUID) uuid.UUID {
    return uuid.UUID(id.Bytes)
}