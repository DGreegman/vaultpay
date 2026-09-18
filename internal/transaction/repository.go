package transaction

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	CreateTransaction(ctx context.Context, params CreateTransactionParams) (*Transaction, bool, error)
	GetTransactionByIdempotencyKey(
		ctx context.Context,
		userID uuid.UUID,
		idempotencyKey string,

	) (*Transaction, error)
}

type TransactionType string

const (
	TransactionTypeTransfer   TransactionType = "transfer"
	TransactionTypeDeposit    TransactionType = "deposit"
	TransactionTypeWithdrawal TransactionType = "withdrawal"
)

type TransactionStatus string

const (
	TransactionStatusPending TransactionStatus = "pending"
	TransactionStatusSuccess TransactionStatus = "success"
	TransactionStatusFailed  TransactionStatus = "failed"
)

type Currency string

const (
	CurrencyNGN Currency = "NGN"
	CurrencyUSD Currency = "USD"
)

type Transaction struct {
	ID                  uuid.UUID
	UserID              uuid.UUID
	IdempotencyKey      string
	Type                TransactionType
	Status              TransactionStatus
	Amount              int64
	Currency            Currency
	SourceWalletID      *uuid.UUID
	DestinationWalletID *uuid.UUID
	CreatedAt           time.Time
}

type CreateTransactionParams struct {
	ID                  uuid.UUID
	UserID              uuid.UUID
	IdempotencyKey      string
	Type                TransactionType
	Status              TransactionStatus
	Amount              int64
	Currency            Currency
	SourceWalletID      *uuid.UUID
	DestinationWalletID *uuid.UUID
}
