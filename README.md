# Pemrograman Backend Lanjut — Modul 3 (Database & Repository)

CRUD Users dengan **Fiber v2 + PostgreSQL (pgxpool) + Repository Pattern**.  
Base URL: `http://localhost:3000/api/v1`

## Stack & Struktur
- **Go 1.22**, `github.com/gofiber/fiber/v2`, `github.com/jackc/pgx/v5`, `github.com/joho/godotenv`
- `main.go` — wiring `database.NewPool → repository.NewUserRepository → NewUserHandler` dan routing
- `config/env.go` — `LoadEnv()`, `GetEnv()`, `GetEnvInt()` (fallback bila `.env` tidak ada)
- `database/postgres.go` — `NewPool()` dengan `pgxpool` (MaxConns 10, MinConns 2, Ping 5s)
- `app/model/user.go` — `User`, `ListQuery` (+`Offset()`), `Meta`, `WebResponse`, DTO `Create/Replace/Patch`
- `app/repository/user-repository.go` — interface `UserRepository` + `userPostgresRepository` (whitelist `kolomUrut`, `buildFilter`, `FindAll/FindByID/Create/Update/Delete`)
- `handler.go` — `UserHandler` (depend pada interface) + `terjemahkanError` (404/409)
- `helper.go` — `ok/okList/created/noContent/fail/failValidation`, `parseListQuery`, `paramID`, `requireJSON`, `reqCtx`
- `migrations/001_user_migrations.sql` — skema `users`

## Prasyarat
PostgreSQL 15, DB `praktikum_backend`, Go 1.22+.

## Setup
```bash
# 1. env
cp .env.example .env
# isi DB_PASSWORD, DB_USER, dll. Contoh:
# APP_PORT=3000
# DB_HOST=localhost
# DB_PORT=5432
# DB_USER=postgres
# DB_PASSWORD=admin
# DB_NAME=praktikum_backend
# DB_SSLMODE=disable
# DB_MAX_CONNS=10

# 2. migrasi
createdb praktikum_backend
psql -d praktikum_backend -f migrations/001_user_migrations.sql

# 3. jalankan
go run .
# Server berjalan di http://localhost:3000  (main.go:82)
```

Skema (`migrations/001_user_migrations.sql`):
```sql
CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  username VARCHAR(50) NOT NULL,
  email VARCHAR(255) NOT NULL,
  password VARCHAR(255) NOT NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX users_username_lower_key ON users (LOWER(username));
CREATE INDEX users_email_lower_idx ON users (LOWER(email));
```

## Kontrak API

| Metode | Endpoint | Parameter | Contoh Body | Status | Contoh Respons |
|--------|----------|-----------|-------------|--------|----------------|
| `GET` | `/health` | — | — | `200` OK, `503` DB down | `{"success":true,"message":"server dan database berjalan","data":{"timestamp":"..."}}` |
| `GET` | `/users/` | query: `page` (1), `limit` (10, max 100), `search` (ILIKE username/email), `sort` (`id`/`username`/`email`/`created_at`), `order` (`asc`/`desc`), `is_active` (`true`/`false`) | — | `200` OK | `{"success":true,"message":"daftar user berhasil diambil","data":[...],"meta":{"page":1,"limit":10,"total":1,"total_pages":1}}` |
| `GET` | `/users/:id` | path `id` angka positif | — | `200` OK, `400` id tidak valid, `404` tidak ditemukan | `{"success":true,"message":"user ditemukan","data":{"id":1,"username":"budi","email":"budi@mail.com","password":"...","is_active":true,"created_at":"..."}}` |
| `POST` | `/users/` | header `Content-Type: application/json` | `{"username":"budi","email":"budi@mail.com","password":"12345678"}` | `201` Created + `Location`, `400` JSON tidak valid, `415` bukan JSON, `422` validasi, `409` username dipakai | `{"success":true,"message":"user berhasil dibuat","data":{...}}` |
| `PUT` | `/users/:id` | path `id`, header JSON | `{"username":"budi2","email":"budi2@mail.com","is_active":false}` | `200` OK, `400`/`404`/`409`/`415`/`422` | `{"success":true,"message":"user berhasil diganti seluruhnya","data":{...}}` |
| `PATCH` | `/users/:id` | path `id`, header JSON | `{"email":"baru@mail.com"}` atau `{"is_active":true}` | `200` OK, `400` tidak ada field diubah, `404`/`409` | `{"success":true,"message":"user berhasil diperbarui sebagian","data":{...}}` |
| `DELETE` | `/users/:id` | path `id` | — | `204` No Content, `400`/`404` | — (tanpa body) |
| `*` | `/*` | — | — | `404` endpoint tidak ditemukan | `{"success":false,"message":"endpoint tidak ditemukan"}` |

### Catatan Validasi
- `POST` wajib: `username` tidak kosong (TrimSpace), `email` mengandung `@`, `password` ≥8 karakter → `422` dengan `{"errors":{"field":"pesan"}}` (`handler.go:69`).
- `PUT` wajib semua field `username`/`email` (`handler.go:108`).
- `PATCH` minimal satu dari `username`/`email`/`is_active` → `400` bila kosong (`handler.go:136`), baca dulu `FindByID` lalu merge.
- `requireJSON` (`helper.go:105`) menolak `POST`/`PUT`/`PATCH` tanpa `application/json` → `415`.
- `paramID` (`helper.go:90`) menolak id bukan angka positif → `400`.
- `parseListQuery` (`helper.go:59`) whitelist sort `id,username,email,created_at`; fallback `id`.
- `buildFilter` (`app/repository/user-repository.go:47`) selalu parameterized `ILIKE $n` (anti SQL injection).
- Duplikasi username ditangani DB `LOWER(username)` → `23505` → `ErrDuplicate` → `409` (`handler.go:24`).

## Contoh curl
```bash
curl -i http://localhost:3000/api/v1/health
curl -i http://localhost:3000/api/v1/users/?page=1&limit=2&search=budi&sort=username&order=asc&is_active=true
curl -i http://localhost:3000/api/v1/users/1
curl -i -X POST -H "Content-Type: application/json" -d '{"username":"budi","email":"budi@mail.com","password":"12345678"}' http://localhost:3000/api/v1/users/
curl -i -X PUT -H "Content-Type: application/json" -d '{"username":"budi2","email":"budi2@mail.com","is_active":false}' http://localhost:3000/api/v1/users/1
curl -i -X PATCH -H "Content-Type: application/json" -d '{"email":"baru@mail.com"}' http://localhost:3000/api/v1/users/1
curl -i -X DELETE http://localhost:3000/api/v1/users/1
```

## Laporan
Lihat `Laporan Praktikum Backend Lanjut - Modul 3 Database dan Repository.docx` (full verbatim dari kode workspace, hitam-putih) untuk skema, potongan `database/postgres.go`, `app/repository/user-repository.go`, `handler.go`/`helper.go`/`main.go` dan placeholder screenshot pengujian (ganti dengan screenshot yang menampilkan status HTTP).

## Lisensi
Praktikum — untuk keperluan akademik.
