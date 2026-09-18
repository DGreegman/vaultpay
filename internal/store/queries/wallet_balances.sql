-- name: GetWalletBalanceForUpdate :one
SELECT
    wallet_id,
    available_balance,
    held_balance,
    updated_at
FROM wallet_balances
WHERE wallet_id = $1
FOR UPDATE;