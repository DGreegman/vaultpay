-- Reverse order: the index and table depend on the type, so they
-- go first, and the type is dropped last.
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS user_status;