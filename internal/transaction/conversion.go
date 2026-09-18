package transaction

import (
	"github.com/DGreegman/vaultpay/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func toPgUUID(id uuid.UUID) pgtype.UUID {

	return pgtype.UUID{
		Bytes: id,
		Valid: true,
	}

}

func toNullablePgUUID(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{
			Valid: false,
		}
	}

	return pgtype.UUID{
		Bytes: *id,
		Valid: true,
	}
}

func fromPgUUID(id pgtype.UUID) uuid.UUID {
	return uuid.UUID(id.Bytes)
}

func fromNullablePgUUID(id pgtype.UUID) *uuid.UUID {
	if !id.Valid {
		return nil
	}

	value := uuid.UUID(id.Bytes)
	return &value
}

func mapTransaction(row store.CreateTransactionRow) *Transaction {
	return &Transaction{
		ID:                  fromPgUUID(row.ID),
		UserID:              fromPgUUID(row.UserID),
		IdempotencyKey:      row.IdempotencyKey,
		Type:                TransactionType(row.Type),
		Status:              TransactionStatus(row.Status),
		Amount:              row.Amount,
		Currency:            Currency(row.Currency),
		SourceWalletID:      fromNullablePgUUID(row.SourceWalletID),
		DestinationWalletID: fromNullablePgUUID(row.DestinationWalletID),
		CreatedAt:           row.CreatedAt.Time,
	}
}
