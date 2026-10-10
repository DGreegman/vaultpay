-- name: CreateWallet :one
INSERT INTO wallets (
    id,
    user_id,
    currency
)
VALUES (
    $1,
    $2,
    $3
)
RETURNING
    id,
    user_id,
    currency,
    status,
    created_at,
    updated_at;


-- name: GetWalletForUpdate :one
SELECT
    id,
    user_id,
    currency,
    status
FROM wallets
WHERE id = $1
FOR UPDATE;