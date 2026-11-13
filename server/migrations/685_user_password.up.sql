-- The earlier self-hosted password build already had this table.
CREATE TABLE IF NOT EXISTS user_password (
    user_id UUID NOT NULL,
    password_hash TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
