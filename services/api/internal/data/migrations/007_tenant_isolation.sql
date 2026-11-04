ALTER TABLE users ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT '';
ALTER TABLE token_usages ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT '';
ALTER TABLE documents ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT '';
ALTER TABLE chunks ADD COLUMN IF NOT EXISTS tenant_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS token_usages_tenant_id_created_at_idx
    ON token_usages (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS documents_tenant_id_created_at_idx
    ON documents (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS chunks_tenant_id_idx
    ON chunks (tenant_id);
