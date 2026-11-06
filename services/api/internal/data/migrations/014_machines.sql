CREATE TABLE IF NOT EXISTS machines (
    id TEXT PRIMARY KEY,
    name TEXT,
    private_address TEXT NOT NULL,
    public_address TEXT,
    region TEXT,
    root_public_key TEXT,
    joined_k8s BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS machines_private_address_idx
    ON machines (private_address);
