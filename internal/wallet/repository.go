package wallet

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	GetBalanceForUpdate(ctx context.Context, walletID uuid.UUID) (*Balance, error)
	GetWalletForUpdate(ctx context.Context, walletID uuid.UUID) (*Wallet, error)
}
