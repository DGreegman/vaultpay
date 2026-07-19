-- A session represents one refresh-token lineage from a single login.
-- Refresh tokens rotate: each use issues a new token and marks the old
-- one used. All tokens from one login share a token_family_id, so that
-- detecting a reused (stale) token lets us revoke the entire family.

CREATE TABLE sessions (
    id               UUID         PRIMARY KEY,
    user_id          UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- All refresh tokens descended from one login share this id.
    -- Reuse of a rotated token triggers revocation of the whole family.
    token_family_id  UUID         NOT NULL,

    -- SHA-256 of the refresh token, never the token itself. A leaked
    -- sessions table must not expose usable credentials.
    token_hash       TEXT         NOT NULL UNIQUE,

    -- Rotation bookkeeping: a token that has been used to mint a new one
    -- is marked used. Presenting a used token again is the theft signal.
    used             BOOLEAN      NOT NULL DEFAULT false,

    -- Revocation: set true on logout, or on whole-family revocation.
    revoked          BOOLEAN      NOT NULL DEFAULT false,

    -- Captured for audit and anomaly detection (CBN device-binding, §7.1).
    device_id        TEXT,
    ip_address       INET,

    expires_at       TIMESTAMPTZ  NOT NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now()
);


-- Revoke or inspect an entire family in one query.
CREATE INDEX idx_sessions_family ON sessions (token_family_id);

-- "All active sessions for this user" (e.g. a logout-everywhere feature).
CREATE INDEX idx_sessions_user ON sessions (user_id);