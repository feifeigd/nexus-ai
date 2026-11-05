CREATE TABLE IF NOT EXISTS tenants (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    owner_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS tenants_owner_id_idx
    ON tenants (owner_id, created_at DESC);

-- 为既有用户（每用户一租户，tenant_id = 用户 id）补建个人租户，复用用户 id 作为
-- 租户 id，保证 users.tenant_id 仍指向有效租户。
INSERT INTO tenants (id, name, owner_id, created_at)
SELECT id, username || ' 的个人空间', id, created_at
FROM users
ON CONFLICT (id) DO NOTHING;
