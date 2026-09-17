-- Users: password WAJIB berupa hash bcrypt, bukan teks asli.
-- (Jika table users sudah ada dari pertemuan sebelumnya, bagian CREATE dilewati;
--  ALTER di bawah menambahkan role bila belum ada.)
CREATE TABLE IF NOT EXISTS users (
    id         SERIAL PRIMARY KEY,
    username   VARCHAR(50)  NOT NULL,
    email      VARCHAR(255) NOT NULL,
    password   VARCHAR(255) NOT NULL,
    is_active  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- Role disiapkan sekarang, dipakai untuk authorization di pertemuan 6.
ALTER TABLE users ADD COLUMN IF NOT EXISTS role VARCHAR(20) NOT NULL DEFAULT 'user';

CREATE UNIQUE INDEX IF NOT EXISTS users_username_key ON users (LOWER(username));
CREATE UNIQUE INDEX IF NOT EXISTS users_email_key ON users (LOWER(email));

-- Refresh token disimpan sebagai HASH: bila table ini bocor, penyerang
-- tetap tidak memperoleh token yang bisa langsung dipakai.
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         BIGSERIAL PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS refresh_tokens_user_id_idx ON refresh_tokens (user_id);
