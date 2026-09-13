-- Each wallet has exactly one balance row.
-- wallet_id is both the primary key and foreign key, giving us
-- a shared-primary-key 1-to-1 relationship with wallets.

CREATE TABLE wallet_balances (
    wallet_id               UUID        PRIMARY KEY REFERENCES wallets(id),
    available_balance       BIGINT      NOT NULL DEFAULT 0
        CHECK (available_balance >= 0),
    held_balance            BIGINT      NOT NULL DEFAULT 0
        CHECK (held_balance >= 0),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_wallet_balances_set_updated_at
    BEFORE UPDATE ON wallet_balances
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();