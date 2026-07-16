-- Account lifecycle states. Closed and stable, so an enum fits.
--   active    - normal operation
--   suspended - temporarily blocked (fraud hold, admin action)
--   deleted   - soft-deleted; row retained for audit/history
CREATE TYPE user_status AS ENUM ('active', 'suspended', 'deleted');

-- The users table holds identity and credentials only.
-- Profile data (display name, avatar, preferences) lives elsewhere,
-- so that auth queries stay narrow and the password hash sits in a
-- table with the smallest possible write surface.

-- Roles use TEXT + CHECK rather than an enum: RBAC roles churn over
-- time, and a CHECK constraint can be dropped and recreated cleanly,
-- whereas an enum value can never be removed once added.
CREATE TABLE users (
    id             UUID         PRIMARY KEY,
    email          CITEXT       NOT NULL UNIQUE,
    phone          TEXT         UNIQUE,
    password_hash  TEXT         NOT NULL,

    role           TEXT         NOT NULL DEFAULT 'user'
                   CHECK (role IN ('user', 'business', 'admin', 'ops')),

    -- KYC tier is ordinal (tier 2 > tier 1), so it is a smallint we
    -- can compare with >=, not an enum. 0 means unverified.
    kyc_tier       SMALLINT     NOT NULL DEFAULT 0
                   CHECK (kyc_tier BETWEEN 0 AND 3),

    -- Account status is a closed, stable set, so an enum is a good fit.
    status         user_status  NOT NULL DEFAULT 'active',

    created_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);
