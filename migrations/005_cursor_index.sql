-- Urutan kolom HARUS sama dengan ORDER BY query (created_at DESC, id DESC).
-- Bila tidak, PostgreSQL tidak dapat melompat ke posisi cursor.
CREATE INDEX IF NOT EXISTS students_created_at_id_desc_idx
ON students (created_at DESC, id DESC);
