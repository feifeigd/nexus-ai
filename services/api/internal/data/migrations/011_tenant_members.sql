CREATE TABLE IF NOT EXISTS tenant_members (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('owner', 'member')),
    created_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS tenant_members_tenant_user_idx
    ON tenant_members (tenant_id, user_id);

CREATE INDEX IF NOT EXISTS tenant_members_user_id_idx
    ON tenant_members (user_id);

-- 为既有租户的 owner 补建 owner 成员行。tenants.id 即租户 id，owner_id 即所有者用户 id。
INSERT INTO tenant_members (id, tenant_id, user_id, role, created_at)
SELECT 'tm_' || md5(id || ':' || owner_id), id, owner_id, 'owner', created_at
FROM tenants
ON CONFLICT (tenant_id, user_id) DO NOTHING;
