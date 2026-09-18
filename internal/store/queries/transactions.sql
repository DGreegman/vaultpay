-- name: GetTransactionByID :one
SELECT
    id,
    user_id,
    idempotency_key,
    type,
    status,
    amount,
    currency,
    source_wallet_id,
    destination_wallet_id,
    created_at
FROM transactions
WHERE id = $1;


-- name: CreateTransaction :one
INSERT INTO transactions (
    id,
    user_id,
    idempotency_key,
    type,
    status,
    amount,
    currency,
    source_wallet_id,
    destination_wallet_id
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9
)
ON CONFLICT (user_id, idempotency_key)
DO NOTHING
RETURNING
    id,
    user_id,
    idempotency_key,
    type,
    status,
    amount,
    currency,
    source_wallet_id,
    destination_wallet_id,
    created_at;


-- name: GetTransactionByIdempotencyKey :one
SELECT
    id,
    user_id,
    idempotency_key,
    type,
    status,
    amount,
    currency,
    source_wallet_id,
    destination_wallet_id,
    created_at
FROM transactions
WHERE user_id = $1
  AND idempotency_key = $2;


