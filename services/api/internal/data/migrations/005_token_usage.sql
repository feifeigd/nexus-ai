CREATE TABLE IF NOT EXISTS token_usages (
    id BIGSERIAL PRIMARY KEY,
    thread_id TEXT NOT NULL,
    run_id TEXT NOT NULL,
    prompt_tokens BIGINT NOT NULL,
    completion_tokens BIGINT NOT NULL,
    total_tokens BIGINT NOT NULL,
    judge_tokens BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS token_usages_created_at_idx
    ON token_usages (created_at DESC);

CREATE INDEX IF NOT EXISTS token_usages_thread_id_created_at_idx
    ON token_usages (thread_id, created_at);
