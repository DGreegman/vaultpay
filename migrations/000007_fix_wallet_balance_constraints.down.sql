ALTER TABLE wallet_balances
    DROP CONSTRAINT wallet_balances_held_balance_check;

ALTER TABLE wallet_balances
    ADD CONSTRAINT wallet_balances_held_balance_check
    CHECK (held_balance > 0);