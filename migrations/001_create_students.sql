CREATE TABLE IF NOT EXISTS students (
  id         SERIAL PRIMARY KEY,
  nim        VARCHAR(20) NOT NULL,
  name       VARCHAR(100) NOT NULL,
  grade      NUMERIC(5,2) NOT NULL CHECK (grade >= 0 AND grade <= 100),
  is_active  BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- NIM wajib unik. Keunikan dijaga basis data (UNIQUE) karena:
-- 1) Atomic & race-condition free: dua request bersamaan tidak bisa lolos cek Go yang terpisah SELECT lalu INSERT.
-- 2) Single source of truth: DB menjamin konsistensi meski ada banyak instance aplikasi.
-- 3) Lebih sederhana: Go tidak perlu SELECT dulu, cukup tangani error 23505 -> 409.
CREATE UNIQUE INDEX IF NOT EXISTS students_nim_key
  ON students (nim);

-- Indeks tambahan untuk mempercepat pencarian ILIKE pada nama dan filter is_active.
-- Pencarian pada pertemuan 2 dilakukan di Go (loop slice), kini di SQL dengan ILIKE.
CREATE INDEX IF NOT EXISTS students_name_lower_idx
  ON students (LOWER(name));

CREATE INDEX IF NOT EXISTS students_is_active_idx
  ON students (is_active);
