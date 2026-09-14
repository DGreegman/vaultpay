ALTER TABLE transactions
    DROP CONSTRAINT transactions_user_id_idempotency_key_unique;

ALTER TABLE transactions
    DROP COLUMN user_id,
    DROP COLUMN idempotency_key;