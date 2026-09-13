-- A wallet is a per-user, per-currency account for holding funds.
-- Wallets are never physically deleted; 'closed' is the terminal lifecycle state.


CREATE TABLE wallets (
    id          UUID            PRIMARY KEY,
    user_id     UUID            NOT NULL REFERENCES users(id),
    currency    currency        NOT NULL,
    status      wallet_status   NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ     NOT NULL DEFAULT now(),

    -- A user can have at most one wallet for each currency.
    UNIQUE (user_id, currency)
);

CREATE TRIGGER trg_wallets_set_updated_at
    BEFORE UPDATE ON wallets
    FOR EACH ROW 
    EXECUTE FUNCTION set_updated_at();