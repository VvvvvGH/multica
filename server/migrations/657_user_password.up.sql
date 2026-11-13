CREATE TABLE user_password (
    user_id UUID NOT NULL,
    password_hash TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
