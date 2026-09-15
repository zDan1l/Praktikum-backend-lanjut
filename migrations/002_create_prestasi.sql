CREATE TABLE IF NOT EXISTS prestasi (
  id            SERIAL,
  id_student    INT NOT NULL REFERENCES students(id) ON DELETE CASCADE,
  nama_prestasi VARCHAR(255) NOT NULL,
  juara         INT NOT NULL CHECK (juara > 0),
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id, id_student)
);

-- Alternatif jika ingin PK tunggal (uncomment jika dibutuhkan):
-- CREATE TABLE IF NOT EXISTS prestasi (
--   id            SERIAL PRIMARY KEY,
--   id_student    INT NOT NULL REFERENCES students(id) ON DELETE CASCADE,
--   nama_prestasi VARCHAR(255) NOT NULL,
--   juara         INT NOT NULL CHECK (juara > 0),
--   created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
-- );

-- Perbaikan untuk DB yang sudah terlanjur dibuat tanpa DEFAULT/NOT NULL (agar scan time.Time tidak gagal saat NULL)
UPDATE prestasi SET created_at = NOW() WHERE created_at IS NULL;
ALTER TABLE prestasi ALTER COLUMN created_at SET DEFAULT NOW();
-- Jika ingin ketat, uncomment:
-- ALTER TABLE prestasi ALTER COLUMN created_at SET NOT NULL;

CREATE INDEX IF NOT EXISTS prestasi_nama_prestasi_idx ON prestasi (LOWER(nama_prestasi));
CREATE INDEX IF NOT EXISTS prestasi_id_student_idx ON prestasi (id_student);
