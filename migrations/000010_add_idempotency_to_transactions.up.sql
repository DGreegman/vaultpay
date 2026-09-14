ALTER TABLE transactions
    ADD COLUMN user_id UUID NOT NULL REFERENCES users(id),
    ADD COLUMN idempotency_key TEXT NOT NULL;

ALTER TABLE transactions
    ADD CONSTRAINT transactions_user_id_idempotency_key_unique
    UNIQUE (user_id, idempotency_key);