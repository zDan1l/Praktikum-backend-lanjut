-- Migration 007: kelas (courses) dan pendaftaran (enrollments) — fondasi KRS.
-- psql -U postgres -d praktikum_backend -f migrations/007_courses.sql
--
-- Modul 11 membangun notifikasi KRS di atas tabel-tabel ini. Repo ini belum
-- pernah memilikinya (sebelumnya berhenti di portofolio), jadi dibuat di sini
-- dalam bentuk minimal yang cukup untuk Langkah panduan dan uji Bagian C.

-- ---------------------------------------------------------------------
-- courses — satu baris per kelas.
--
-- enrolled_count disimpan sebagai counter agar pengambilan kursi dapat
-- dilakukan atomik lewat satu UPDATE (lihat IncrementEnrolled). capacity
-- membatasi jumlah peserta.
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS courses (
    id             SERIAL PRIMARY KEY,
    code           VARCHAR(20)  UNIQUE NOT NULL,
    name           VARCHAR(120) NOT NULL,
    capacity       INTEGER NOT NULL DEFAULT 40 CHECK (capacity >= 0),
    enrolled_count INTEGER NOT NULL DEFAULT 0 CHECK (enrolled_count >= 0)
);

-- ---------------------------------------------------------------------
-- enrollments — KRS seorang mahasiswa pada sebuah kelas.
--
-- UNIQUE (user_id, course_id) mencegah pendaftaran ganda pada kelas yang
-- sama: INSERT kedua ditolak 23505 dan dipetakan menjadi 409 ALREADY_ENROLLED.
-- ---------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS enrollments (
    id         BIGSERIAL PRIMARY KEY,
    user_id    INTEGER NOT NULL REFERENCES users(id)   ON DELETE CASCADE,
    course_id  INTEGER NOT NULL REFERENCES courses(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT enrollments_unique UNIQUE (user_id, course_id)
);

CREATE INDEX IF NOT EXISTS enrollments_course_idx ON enrollments (course_id);

-- Kelas contoh dengan id tetap 1..9 agar cocok dengan uji Bagian C.
-- Uji C.1 menyebut kelas 2 = "IF102 Basis Data Lanjut" dan kelas 4 =
-- "IF302 Keamanan Informasi"; keduanya dikunci di sini.
INSERT INTO courses (id, code, name, capacity) VALUES
    (1, 'IF101', 'Algoritma dan Pemrograman', 40),
    (2, 'IF102', 'Basis Data Lanjut',         40),
    (3, 'IF201', 'Jaringan Komputer',          40),
    (4, 'IF302', 'Keamanan Informasi',         40),
    (5, 'IF203', 'Sistem Operasi',             40),
    (6, 'IF204', 'Pemrograman Web',            40),
    (7, 'IF305', 'Kecerdasan Buatan',          40),
    (8, 'IF306', 'Komputasi Awan',             40),
    (9, 'IF307', 'Rekayasa Perangkat Lunak',   40)
ON CONFLICT (id) DO NOTHING;

-- Sinkronkan sequence agar INSERT berikutnya tidak bentrok dengan id manual.
SELECT setval(pg_get_serial_sequence('courses', 'id'), (SELECT MAX(id) FROM courses));
