ALTER TABLE users ADD COLUMN IF NOT EXISTS github_id TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS users_github_id_idx
    ON users (github_id);
