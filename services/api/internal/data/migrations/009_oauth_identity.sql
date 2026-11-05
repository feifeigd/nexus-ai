ALTER TABLE users ADD COLUMN IF NOT EXISTS oauth_provider TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS oauth_subject TEXT;

UPDATE users
    SET oauth_provider = 'github', oauth_subject = github_id
    WHERE github_id IS NOT NULL AND github_id != '';

CREATE UNIQUE INDEX IF NOT EXISTS users_oauth_provider_subject_idx
    ON users (oauth_provider, oauth_subject);

ALTER TABLE users DROP COLUMN IF EXISTS github_id;
