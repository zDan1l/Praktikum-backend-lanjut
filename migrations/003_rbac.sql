-- RBAC (Role Based Access Control) sesuai Modul 6.
-- role -> permission dipetakan lewat tabel role_permissions,
-- sehingga kebijakan akses bisa diubah tanpa mengubah kode.

CREATE TABLE IF NOT EXISTS roles (
    id   SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL
);

CREATE TABLE IF NOT EXISTS permissions (
    id       SERIAL PRIMARY KEY,
    action   VARCHAR(50) NOT NULL,
    resource VARCHAR(50) NOT NULL,
    UNIQUE (action, resource)
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id       INTEGER NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id INTEGER NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- users sekarang merujuk role lewat role_id (FK ke roles),
-- kolom role VARCHAR lama diganti.
ALTER TABLE users ADD COLUMN IF NOT EXISTS role_id INTEGER REFERENCES roles(id);

INSERT INTO roles (name) VALUES ('admin'), ('editor'), ('viewer')
ON CONFLICT (name) DO NOTHING;

-- Permission untuk resource comment (tugas mandiri modul 6).
INSERT INTO permissions (action, resource) VALUES
    ('create', 'comment'),
    ('read',   'comment'),
    ('update', 'comment'),
    ('delete', 'comment')
ON CONFLICT (action, resource) DO NOTHING;

-- admin: semua permission pada comment.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;

-- editor: create, read, update, delete comment
-- (update & delete dibatasi milik sendiri, dicek di service layer).
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p
WHERE r.name = 'editor'
ON CONFLICT DO NOTHING;

-- viewer: hanya create dan read comment.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.action IN ('create', 'read') AND p.resource = 'comment'
WHERE r.name = 'viewer'
ON CONFLICT DO NOTHING;

-- User lama yang belum punya role dijadikan viewer,
-- user baru akan selalu diberi role viewer oleh kode register.
UPDATE users
SET role_id = (SELECT id FROM roles WHERE name = 'viewer')
WHERE role_id IS NULL;

ALTER TABLE users ALTER COLUMN role_id SET NOT NULL;

ALTER TABLE users DROP COLUMN IF EXISTS role;
