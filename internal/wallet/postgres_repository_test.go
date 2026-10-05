package wallet

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/DGreegman/vaultpay/internal/db"
	"github.com/DGreegman/vaultpay/internal/store"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

func TestPostgresRepository_GetBalanceForUpdate(t *testing.T) {
	ctx := context.Background()

	walletID := uuid.New()
	userID := uuid.New()

	availableBalance := int64(50000)
	heldBalance := int64(10000)

	updatedAt := time.Date(
		2026,
		time.September,
		18,
		10,
		30,
		0,
		0,
		time.UTC,
	)

	err := godotenv.Load("../../.env")
	if err != nil {
		t.Fatalf("failed to load .env: %v", err)
	}
	databaseURL := os.Getenv("DATABASE_URL")

	if databaseURL == "" {
		t.Fatalf("DATABASE_URL is required")
	}
	pool, err := db.New(ctx, db.DefaultConfig(databaseURL))

	if err != nil {
		t.Fatalf("failed to create db pool: %v", err)
	}

	t.Cleanup(func() {
		_, err := pool.Exec(ctx, `DELETE FROM wallet_balances WHERE wallet_id = $1`,walletID)

		if err != nil {
			t.Errorf("failed to cleanup wallet balance: %v", err)
		}

		_, err = pool.Exec(ctx, `DELETE FROM wallets WHERE id = $1`, walletID)
		if err != nil {
			t.Errorf("failed to cleanup wallet: %v", err)
		}

		_, err = pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
		if err != nil {
			t.Errorf("failed to cleanup user: %v", err)
		}
		pool.Close()
	})


	testEmail := fmt.Sprintf("test-%s@example.com", userID.String())
	_, err = pool.Exec(ctx, 
		`INSERT INTO users (
			id,
			email,
			password_hash
	)
	VALUES ($1, $2, $3)		
	`, userID, testEmail, "test-password-hash")

	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	_, err = pool.Exec(ctx, 
		`INSERT INTO wallets (
		id,
		user_id,
		currency
	)
		VALUES ($1, $2, $3)`, walletID, userID, "NGN")

	if err != nil {
		t.Fatalf("failed to create test wallet: %v", err)
	}

	_, err = pool.Exec(ctx, 
		`INSERT INTO wallet_balances (
			wallet_id,
			available_balance,
			held_balance,
			updated_at
		)
			VALUES($1, $2, $3, $4)
		`, walletID, availableBalance, heldBalance, updatedAt)

	if err != nil {
		t.Fatalf("failed to create test wallet balance: %v", err)
	}

	queries := store.New(pool)

	repo := NewPostgresRepository(queries)

	balance, err := repo.GetBalanceForUpdate(ctx, walletID)

	
	if err != nil {
		t.Fatalf("failed to get wallet balance: %v", err)
	}


	if balance.WalletID != walletID {
		t.Fatalf("expected wallet ID %v, got %v", walletID, balance.WalletID)
	}

	if balance.AvailableBalance != availableBalance {
		t.Fatalf("expected availbale balance %d, got %d", availableBalance, balance.AvailableBalance)
	}

	if balance.HeldBalance != heldBalance {
		t.Fatalf("expected held balance %d, got %d", heldBalance, balance.HeldBalance)
	}

	if !balance.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("expected updated %v, got %v", updatedAt, balance.UpdatedAt)
	}
	
}

func TestPostgresRepository_GetBalanceForUpdate_NotFound(t *testing.T){
	ctx := context.Background()
	walletID := uuid.New()

	err := godotenv.Load("../../.env")
	if err != nil {
		t.Fatalf("failed to load file .env: %v", err)
	}
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == ""{
		t.Fatalf("DATABASE_URL is required")
	}

	pool, err := db.New(ctx, db.DefaultConfig(databaseURL))
	if err != nil {
		t.Fatalf("failed to create db pool: %v", err)
	}

	defer pool.Close()

	

	queries := store.New(pool)
	repo := NewPostgresRepository(queries)

	balance, err := repo.GetBalanceForUpdate(ctx, walletID)

	if !errors.Is(err, ErrNotFound) {
    	t.Fatalf("expected ErrNotFound, got %v", err)
	}

	if balance != nil {
		t.Fatalf("expected nil balance, got %v", err)
	}
} 


func TestPostgresRepository_GetBalanceForUpdate_LocksRow(t *testing.T){
	ctx := context.Background()
	userID := uuid.New()
	walletID := uuid.New()
	walletBalance := int64(60000)
	
	err := godotenv.Load("../../.env")
	
	if err != nil {
		t.Fatalf("failed to load .env: %v", err)
	}
	databaseURL := os.Getenv("DATABASE_URL")

	if databaseURL == "" {
		t.Fatalf("DATABASE_URL is required")
	}
	pool, err := db.New(ctx, db.DefaultConfig(databaseURL))


	if err != nil {
		t.Fatalf("failed to create db pool: %v", err)
	}
	queries := store.New(pool)

	repo := NewPostgresRepository(queries)
	
	testEmail := fmt.Sprintf("test-%s@example.com", userID.String())

	_, err = pool.Exec(
		ctx,
		`INSERT INTO users(id, email, password_hash)
		VALUES($1, $2, $3)`,
		userID,
		testEmail,
		"test-password-hash",
	)

	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	_, err = pool.Exec(
		ctx,
		`INSERT INTO wallets (id, user_id, currency)
		VALUES($1, $2, $3)`,
		walletID,
		userID,
		"NGN",
	)

	if err != nil {
		t.Fatalf("failed to create test wallet: %v", err)
	}

	_, err = pool.Exec(
		ctx,
		`INSERT INTO wallet_balances (wallet_id, available_balance, held_balance)
		VALUES ($1, $2, $3)`,
		walletID,
		walletBalance,
		0,
	)

	if err != nil {
		t.Fatalf("failed to create test wallet balance: %v", err)
	}

	t.Cleanup(func() {
		_, err := pool.Exec(
			ctx, 
			`DELETE FROM wallet_balances WHERE wallet_id = $1`,
			walletID,
		)
		if err != nil {
			t.Errorf("failed to clean up wallet balance: %v", err)
		}

		_, err = pool.Exec(
			ctx, 
			`DELETE FROM wallets WHERE id = $1`,
			walletID,
		)

		if err != nil {
			t.Errorf("failed to cleanup wallet: %v", err)
		}

		_, err = pool.Exec(
			ctx,
			`DELETE FROM users WHERE id = $1`,
			userID,
		)

		if err != nil {
			t.Errorf("failed to cleanup user: %v", err)
		}

		pool.Close()
	})
	
	tx1, err := pool.Begin(ctx)


	if err != nil {
		t.Fatalf("failed to begin transaction 1: %v", err)
	}


	defer tx1.Rollback(ctx)

	repo1 := repo.WithTx(tx1)

	balance, err := repo1.GetBalanceForUpdate(ctx, walletID)

	if err != nil {
		t.Fatalf("failed to lock balance: %v", err)
	}

	if balance.WalletID != walletID {
		t.Fatalf("expected wallet ID %s, got %s", walletID, balance.WalletID)
	}

	if balance.AvailableBalance != walletBalance {
		t.Fatalf(
			"expected available balance %d, got %d",
			walletBalance,
			balance.AvailableBalance,
		)
	}

	tx2, err := pool.Begin(ctx)

	if err != nil {
		t.Fatalf("failed to open transaction 2: %v", err)
	}

	defer tx2.Rollback(ctx)

	done := make(chan error)

	go func() {
		_, err := tx2.Exec(ctx, `
		UPDATE wallet_balances
		SET available_balance = available_balance - 1000
		WHERE wallet_id = $1
		`, walletID)

		done <- err
	}()

	select {
	case err := <-done:
		t.Fatalf("transaction 2 completed before transaction 1 commited, err: %v", err)
	
	case <-time.After(100 * time.Millisecond):
		// Tx2 is still waitng for the lock, which is expected behavior.
	}

	err = tx1.Commit(ctx)

	if err != nil {
		t.Fatalf("failed to commit transaction 1: %v", err)
	}

	err = <-done
	
	if err != nil {
		t.Fatalf("transaction 2 failed: %v", err)
	}

	err = tx2.Commit(ctx)

	if err != nil {
		t.Fatalf("failed to commit transaction 2: %v", err)
	}

	var finalBalance int64 

	err = pool.QueryRow(ctx,
		`SELECT available_balance
		FROM wallet_balances
		WHERE wallet_id = $1`,
		walletID,
	).Scan(&finalBalance)

	if err != nil {
		t.Fatalf("failed to read final balance: %v", err)
	}

	if finalBalance != walletBalance - 1000 {
		t.Fatalf(
			"expected final balance %d, got %d",
			walletBalance-1000,
			finalBalance,
		)
	}
	
}