package wallet

import (
	"time"

	"github.com/google/uuid"
)

type Balance struct {
	WalletID         uuid.UUID
	AvailableBalance int64
	HeldBalance      int64
	UpdatedAt        time.Time
}
