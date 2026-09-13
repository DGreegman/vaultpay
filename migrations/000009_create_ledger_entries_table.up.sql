-- Ledger entries represent the accounting effect of a transaction
-- on a wallet. Ledger entries are append-only and should never be mutated.

CREATE TYPE ledger_entry_type AS ENUM (
    'debit',
    'credit'
);

CREATE TABLE ledger_entries (
    id             UUID              PRIMARY KEY,
    transaction_id UUID              NOT NULL REFERENCES transactions(id),
    wallet_id      UUID              NOT NULL REFERENCES wallets(id),
    entry_type     ledger_entry_type  NOT NULL,
    amount         BIGINT            NOT NULL
        CHECK (amount > 0),
    currency       currency          NOT NULL,
    created_at     TIMESTAMPTZ       NOT NULL DEFAULT now()
);

CREATE INDEX idx_ledger_entries_transaction
    ON ledger_entries (transaction_id);

CREATE INDEX idx_ledger_entries_wallet
    ON ledger_entries (wallet_id);