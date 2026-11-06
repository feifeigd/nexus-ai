-- 更新 role 枚举：staff → admin，新增 auditor（审核员）、readonly（只读）。
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
UPDATE users SET role = 'admin' WHERE role = 'staff';
ALTER TABLE users ADD CONSTRAINT users_role_check
    CHECK (role IN ('admin', 'auditor', 'readonly', 'user'));
