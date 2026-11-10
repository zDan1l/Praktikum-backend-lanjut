-- Migration 010: SKS, kuota, dan version pada courses (fondasi Modul 9 & 12).
-- psql -U postgres -d praktikum_backend -f migrations/010_credits_version.sql
--
-- Modul 12 menguji aturan batas 24 SKS dan optimistic locking kuota kelas.
-- Keduanya membutuhkan tiga hal yang belum ada di tabel courses:
--   credits : bobot SKS kelas;
--   quota   : nama kolom yang dipakai modul (sebelumnya capacity);
--   version : penanda optimistic locking untuk PATCH /courses/:id.
--
-- Seluruh perintah ditulis agar aman dijalankan berulang kali, karena
-- TestMain menjalankan ulang seluruh migration sebelum integration test.

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'courses' AND column_name = 'capacity'
    ) THEN
        ALTER TABLE courses RENAME COLUMN capacity TO quota;
    END IF;
END $$;

ALTER TABLE courses ADD COLUMN IF NOT EXISTS credits INTEGER NOT NULL DEFAULT 3;
ALTER TABLE courses ADD COLUMN IF NOT EXISTS version INTEGER NOT NULL DEFAULT 1;

ALTER TABLE courses DROP CONSTRAINT IF EXISTS courses_credits_check;
ALTER TABLE courses ADD CONSTRAINT courses_credits_check CHECK (credits BETWEEN 1 AND 6);

-- Bobot SKS kelas contoh dari migration 007.
UPDATE courses SET credits = 4 WHERE code IN ('IF201', 'IF305') AND credits <> 4;
UPDATE courses SET credits = 2 WHERE code = 'IF204' AND credits <> 2;

-- Index untuk menjumlahkan SKS seorang mahasiswa (aturan batas 24 SKS).
CREATE INDEX IF NOT EXISTS enrollments_user_idx ON enrollments (user_id);

-- Permission baru dengan model (action, resource) milik repo ini. Di kode,
-- keduanya dibaca sebagai "resource:action": course:manage dan review:moderate.
INSERT INTO permissions (action, resource) VALUES
    ('manage',   'course'),
    ('moderate', 'review')
ON CONFLICT (action, resource) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p
  ON (p.action, p.resource) IN (('manage', 'course'), ('moderate', 'review'))
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;
