-- A transaction represents a financial operation.
-- Ledger entries will record the accounting effects of the transaction.

CREATE TYPE transaction_type AS ENUM (
    'transfer',
    'deposit',
    'withdrawal'
);

CREATE TYPE transaction_status AS ENUM (
    'pending',
    'success',
    'failed'
);

CREATE TABLE transactions (
    id                    UUID              PRIMARY KEY,

    type                  transaction_type  NOT NULL,

    status                transaction_status NOT NULL DEFAULT 'pending',

    amount                BIGINT            NOT NULL
        CHECK (amount > 0),

    currency              currency          NOT NULL,

    source_wallet_id      UUID REFERENCES wallets(id),

    destination_wallet_id UUID REFERENCES wallets(id),

    created_at            TIMESTAMPTZ       NOT NULL DEFAULT now(),

    -- Enforce which wallet references are valid for each transaction type.
    CHECK (
        (type = 'transfer'
            AND source_wallet_id IS NOT NULL
            AND destination_wallet_id IS NOT NULL)
        OR
        (type = 'deposit'
            AND source_wallet_id IS NULL
            AND destination_wallet_id IS NOT NULL)
        OR
        (type = 'withdrawal'
            AND source_wallet_id IS NOT NULL
            AND destination_wallet_id IS NULL)
    ),

    -- A wallet cannot transfer money to itself.
    CHECK (
        source_wallet_id IS NULL
        OR destination_wallet_id IS NULL
        OR source_wallet_id <> destination_wallet_id
    )
);

CREATE INDEX idx_transactions_source_wallet
    ON transactions (source_wallet_id);

CREATE INDEX idx_transactions_destination_wallet
    ON transactions (destination_wallet_id);