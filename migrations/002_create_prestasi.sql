-- Tabel prestasi (sudah ada atau baru)
CREATE TABLE IF NOT EXISTS prestasi (
  id            SERIAL,
  id_student    INT NOT NULL,
  nama_prestasi VARCHAR(255) NOT NULL,
  juara         INT NOT NULL CHECK (juara > 0),
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id, id_student)
);

-- Perbaikan untuk DB yang sudah terlanjur dibuat tanpa SERIAL/DEFAULT
-- (agar POST /prestasi tidak 500 'null value in column id')
CREATE SEQUENCE IF NOT EXISTS prestasi_id_seq;
ALTER TABLE prestasi ALTER COLUMN id SET DEFAULT nextval('prestasi_id_seq');
SELECT setval('prestasi_id_seq', COALESCE((SELECT MAX(id) FROM prestasi), 0));

-- Perbaikan agar scan tidak gagal saat NULL
UPDATE prestasi SET created_at = NOW() WHERE created_at IS NULL;
ALTER TABLE prestasi ALTER COLUMN created_at SET DEFAULT NOW();

CREATE INDEX IF NOT EXISTS prestasi_nama_prestasi_idx ON prestasi (LOWER(nama_prestasi));
CREATE INDEX IF NOT EXISTS prestasi_id_student_idx ON prestasi (id_student);

-- Jika butuh foreign key (opsional, bisa gagal jika data yatim):
-- ALTER TABLE prestasi ADD CONSTRAINT prestasi_id_student_fkey
--   FOREIGN KEY (id_student) REFERENCES students(id) ON DELETE CASCADE;
