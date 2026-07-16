-- A reusable trigger function that stamps updated_at = now() on any
-- row being updated. Defined once, attached to any table that has an
-- updated_at column. Written in PL/pgSQL (Postgres's procedural language).
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Attach the function to users. It runs BEFORE each UPDATE, so the
-- new timestamp is written as part of the same operation.
CREATE TRIGGER trg_users_set_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();