-- Case-insensitive text. Used for emails so that
-- 'Victor@example.com' and 'victor@example.com' cannot both exist.
CREATE EXTENSION IF NOT EXISTS citext;

-- Supported currencies. Deliberately closed: per PRD §4, VaultPay
-- supports per-currency wallets but NO cross-currency conversion.
CREATE TYPE currency AS ENUM ('NGN', 'USD');

-- Wallet lifecycle states.
--   active   - normal operation
--   frozen   - admin/fraud freeze; no money may move in or out
--   closed   - terminal; wallet is retired
CREATE TYPE wallet_status AS ENUM ('active', 'frozen', 'closed');