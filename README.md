# Pemrograman Backend Lanjut — Modul 3 (Database & Repository)

CRUD Students dengan **Fiber v2 + PostgreSQL (pgxpool) + Repository Pattern**.  
Base URL: `http://localhost:3000/api/v1`

## Stack & Struktur
- **Go 1.22**, `github.com/gofiber/fiber/v2`, `github.com/jackc/pgx/v5`, `github.com/joho/godotenv`
- `main.go` — wiring `database.NewPool → repository.NewStudentRepository → NewStudentHandler` dan routing `/students` + `/health`
- `config/env.go` — `LoadEnv()`, `GetEnv()`, `GetEnvInt()` (fallback bila `.env` tidak ada)
- `database/postgres.go` — `NewPool()` dengan `pgxpool` (MaxConns 10, MinConns 2, Ping 5s; `Ping` juga dipakai di `/health`)
- `app/model/student.go` — `Student` (`id, nim, name, grade, is_active, created_at`), `ListQuery` (+`Offset()`), `Meta`, `WebResponse`, DTO `Create/Replace/Patch`
- `app/repository/student_repository.go` — interface `StudentRepository` (`FindAll, FindByID, Create, Update, Delete`) + `studentPostgresRepository` (whitelist `kolomUrut`, `buildFilter` dengan parameter, `isUniqueViolation 23505`, tanpa import `fiber`)
- `handler.go` — `StudentHandler` (depend pada interface) + `terjemahkanError` (404/409), validasi `nim/name/grade 0-100`
- `helper.go` — `ok/okList/created/noContent/fail/failValidation`, `parseListQuery` (whitelist `id,nim,name,grade,created_at`), `paramID`, `requireJSON`, `reqCtx` 5s
- `migrations/001_create_students.sql` — skema `students`

## Daftar Variabel Environment
Salin dari template (`.env` sudah di-`.gitignore` agar kredensial tidak ter-commit):
```bash
cp .env.example .env
```
Isi `.env` (lihat `.env.example`):
```
APP_PORT=3000
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=isi_kata_sandi_anda
DB_NAME=praktikum_backend
DB_SSLMODE=disable
DB_MAX_CONNS=10
```
| Variabel | Wajib | Default (`config/env.go`) | Penjelasan |
|----------|-------|----------------------------|------------|
| `APP_PORT` | tidak | `3000` | Port Fiber (`main.go:82`) |
| `DB_HOST` | tidak | `localhost` | Host PostgreSQL |
| `DB_PORT` | tidak | `5432` | Port PostgreSQL |
| `DB_USER` | tidak | `postgres` | User DB |
| `DB_PASSWORD` | ya | `""` | Password DB (kosong di example) |
| `DB_NAME` | tidak | `praktikum_backend` | Nama DB |
| `DB_SSLMODE` | tidak | `disable` | `disable` untuk lokal |
| `DB_MAX_CONNS` | tidak | `10` | Ukuran pool (`database/postgres.go:28`) |

## Cara Menyiapkan Basis Data dari Nol
Anggap Anda baru meng-klona repo dan tidak bisa bertanya:
```bash
# 1. clone & env
git clone <url-repo> && cd pemrograman-code
cp .env.example .env
# edit .env, isi DB_PASSWORD sesuai instalasi PostgreSQL lokal

# 2. buat database (butuh psql terinstall)
createdb praktikum_backend
# atau via psql: CREATE DATABASE praktikum_backend;

# 3. migrasi (hanya satu berkas saat ini)
psql -d praktikum_backend -f migrations/001_create_students.sql
# verifikasi:
# psql -d praktikum_backend -c "\d students"
# psql -d praktikum_backend -c "\di"

# 4. jalankan
go run .
# Server berjalan di http://localhost:3000  (main.go:82-85)
# cek health yang ikut Ping DB:
# curl -i http://localhost:3000/api/v1/health  -> 200 jika DB hidup, 503 jika DB mati
```

Skema (`migrations/001_create_students.sql` verbatim):
```sql
CREATE TABLE IF NOT EXISTS students (
  id         SERIAL PRIMARY KEY,
  nim        VARCHAR(20) NOT NULL,
  name       VARCHAR(100) NOT NULL,
  grade      NUMERIC(5,2) NOT NULL CHECK (grade >= 0 AND grade <= 100),
  is_active  BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX students_nim_key ON students (nim);
CREATE INDEX students_name_lower_idx ON students (LOWER(name));
CREATE INDEX students_is_active_idx ON students (is_active);
```
**Mengapa NIM unik dijaga DB bukan Go?**  
`UNIQUE INDEX` di DB bersifat atomik dan race-free: dua request `POST /students` dengan NIM sama yang datang bersamaan tidak bisa lolos jika hanya cek `SELECT` di Go (celah TOCTOU). DB menjadi single source of truth meski ada banyak instance aplikasi. Go cukup menangani error `23505` (`isUniqueViolation`) → `ErrDuplicate` → HTTP `409` (lihat `app/repository/student_repository.go:181` dan `handler.go:24`). Lebih sederhana: tidak perlu `SELECT` dulu.

**Indeks tambahan:** `LOWER(name)` mempercepat `ILIKE %search%` pada `List` (pencarian nama), dan `is_active` mempercepat `WHERE is_active = $1`. Tanpa indeks, pencarian harus full scan.

## Kontrak API

| Metode | Endpoint | Parameter | Contoh Body | Status | Contoh Respons |
|--------|----------|-----------|-------------|--------|----------------|
| `GET` | `/health` | — | — | `200` OK, `503` jika `pool.Ping` gagal (`main.go:60`) | `{"success":true,"message":"server dan database berjalan","data":{"timestamp":"..."}}` |
| `GET` | `/students/` | query: `page` (1), `limit` (10, max 100), `search` (ILIKE `name`), `sort` (`id`/`nim`/`name`/`grade`/`created_at`), `order` (`asc`/`desc`), `is_active` (`true`/`false`) — semua dipindah ke SQL (`LIMIT/OFFSET`, `COUNT(*)`, `ORDER BY whitelist`) | — | `200` OK | `{"success":true,"message":"daftar student berhasil diambil","data":[{"id":1,"nim":"22001","name":"Budi","grade":85.5,"is_active":true,"created_at":"..."}],"meta":{"page":1,"limit":10,"total":1,"total_pages":1}}` |
| `GET` | `/students/:id` | path `id` angka positif | — | `200` OK, `400` id tidak valid, `404` tidak ditemukan | `{"success":true,"message":"student ditemukan","data":{...}}` |
| `POST` | `/students/` | header `Content-Type: application/json` | `{"nim":"22001","name":"Budi","grade":85.5,"is_active":true}` | `201` Created + `Location: /api/v1/students/:id`, `400` JSON tidak valid, `415` bukan JSON, `422` validasi, `409` NIM dipakai | `{"success":true,"message":"student berhasil dibuat","data":{...}}` |
| `PUT` | `/students/:id` | path `id`, header JSON | `{"nim":"22001","name":"Budi","grade":90,"is_active":false}` | `200` OK, `400`/`404`/`409`/`415`/`422` | `{"success":true,"message":"student berhasil diganti seluruhnya","data":{...}}` |
| `PATCH` | `/students/:id` | path `id`, header JSON | `{"grade":95.5}` atau `{"name":"Ani"}` | `200` OK, `400` tidak ada field diubah, `404`/`409` | `{"success":true,"message":"student berhasil diperbarui sebagian","data":{...}}` |
| `DELETE` | `/students/:id` | path `id` | — | `204` No Content, `400`/`404` | — |
| `*` | `/*` | — | — | `404` endpoint tidak ditemukan (`main.go:78`) | `{"success":false,"message":"endpoint tidak ditemukan"}` |

### Catatan Validasi & Query DB Side
- `POST` wajib: `nim`/`name` tidak kosong (TrimSpace), `grade` 0–100 → `422` (`handler.go:Create`).
- `PUT` wajib semua field `nim`/`name`/`grade` 0–100 (`handler.go:Replace`).
- `PATCH` minimal satu dari `nim`/`name`/`grade`/`is_active` → `400` bila kosong, baca `FindByID` dulu lalu merge (`handler.go:Patch`).
- `requireJSON` (`helper.go:105`) → `415` jika bukan `application/json`.
- `paramID` (`helper.go:90`) → `400` jika bukan angka positif.
- `parseListQuery` (`helper.go:59`) whitelist `id,nim,name,grade,created_at`.
- `buildFilter` (`student_repository.go:31`) selalu `ILIKE $n` dan `WHERE is_active = $n` (parameter, bukan sambung string).
- Pagination: `SELECT COUNT(*) ... WHERE` sama dengan `SELECT ... LIMIT/OFFSET` untuk `meta.total`.
- Duplikasi NIM → `23505` → `ErrDuplicate` → `409`.

## Status HTTP dari Error Basis Data (untuk laporan, dengan screenshot)
| Status | Situasi yang harus dibuat | Cara Uji |
|--------|---------------------------|----------|
| `404` | GET/PUT/PATCH/DELETE id tidak ada | `curl -i http://localhost:3000/api/v1/students/9999` → `{"success":false,"message":"student tidak ditemukan"}` |
| `409` | NIM ganda saat Create/Update | Buat `nim=22001`, lalu `POST` lagi `nim=22001` → `409` |
| `503` (dipilih) | PostgreSQL dimatikan lalu API dipanggil | Matikan service `pg_ctl stop` / `net stop postgresql`, lalu `curl -i /health` → `503` |

**Mengapa `503` untuk DB mati?**  
`503 Service Unavailable` paling tepat karena server sendiri hidup tapi dependensi (DB) tidak tersedia — bersifat sementara dan bisa pulih tanpa perubahan kode. `health` memakai `pool.Ping` 2 detik (`main.go:61`), sehingga klien tahu untuk retry. `500` lebih generik untuk bug internal, bukan dependensi eksternal. Laporan menjelaskan pilihan ini (tidak ada jawaban benar, yang dinilai alasan).

## Contoh curl (semua endpoint)
```bash
curl -i http://localhost:3000/api/v1/health
curl -i "http://localhost:3000/api/v1/students/?page=1&limit=2&search=budi&sort=name&order=asc&is_active=true"
curl -i http://localhost:3000/api/v1/students/1
curl -i -X POST -H "Content-Type: application/json" -d '{"nim":"22001","name":"Budi","grade":85.5,"is_active":true}' http://localhost:3000/api/v1/students/
curl -i -X PUT -H "Content-Type: application/json" -d '{"nim":"22001","name":"Budi","grade":90,"is_active":false}' http://localhost:3000/api/v1/students/1
curl -i -X PATCH -H "Content-Type: application/json" -d '{"grade":95.5}' http://localhost:3000/api/v1/students/1
curl -i -X DELETE http://localhost:3000/api/v1/students/1
```
Lihat `curl.txt` untuk 24 varian lengkap (200/201/204/400/415/422/404/409/503).

## Laporan
Lihat `Laporan Praktikum Backend Lanjut - Modul 3 Database dan Repository.docx` (verbatim dari workspace) untuk skema, potongan `database/postgres.go`, `app/repository/student_repository.go`, `handler.go`/`helper.go`/`main.go` dan placeholder screenshot (ganti dengan screenshot yang menampilkan status HTTP).

## Lisensi
Praktikum — untuk keperluan akademik.
