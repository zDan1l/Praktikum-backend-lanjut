# Pemrograman Backend Lanjut — Pertemuan 6 (Authorization & RBAC)

API **CRUD Students + Auth (JWT) + Comments (RBAC)** dengan **Fiber v2 + PostgreSQL (pgxpool)**.
Base URL: `http://localhost:3000/api/v1`

## Struktur
```
main.go            -> wiring: env, logger, DB pool, JWT, repository, handler, route
config/config.go   -> env (.env), logger, pembuatan app Fiber
database/postgres.go -> pgxpool + ping
app/model/         -> entitas & DTO (Student, User, Auth, Comment, ListQuery, Meta, WebResponse)
app/repository/    -> query SQL (common.go, student, user, token, comment)
app/service/       -> handler + validasi + cek ownership (student_handler, auth_service, comment_handler)
helper/            -> response, request, jwt, security (bcrypt)
middleware/        -> recover, cors, logger, RequireJSON, RequireAuth, RequirePermission, rate limiter
route/route.go     -> pendaftaran semua endpoint
migrations/        -> 001_create_students.sql, 002_auth.sql, 003_rbac.sql, 004_articles_comments.sql
```

## Setup
```bash
cp .env.example .env
# isi DB_PASSWORD dan JWT_SECRET (openssl rand -hex 32, minimal 32 karakter)

createdb praktikum_backend
psql -d praktikum_backend -f migrations/001_create_students.sql
psql -d praktikum_backend -f migrations/002_auth.sql
psql -d praktikum_backend -f migrations/003_rbac.sql
psql -d praktikum_backend -f migrations/004_articles_comments.sql
go run .
curl -i http://localhost:3000/api/v1/health   # 200 jika DB hidup, 503 jika mati
```

## Variabel Environment
```
APP_PORT=3000
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=xxx
DB_NAME=praktikum_backend
DB_SSLMODE=disable
DB_MAX_CONNS=10
JWT_SECRET=hasil-openssl-rand-hex-32
JWT_ISSUER=praktikum-backend
JWT_ACCESS_TTL_MINUTES=15
JWT_REFRESH_TTL_DAYS=7
ALLOWED_ORIGINS=http://localhost:5173
```

## Kontrak API
| Metode | Endpoint | Keterangan | Status |
|--------|----------|-----------|--------|
| GET | /health | cek DB, publik | 200, 503 |
| POST | /auth/register | `username,email,password` | 201, 422, 409 |
| POST | /auth/login | `username,password` → token pair | 200, 401, 429 |
| POST | /auth/refresh | `refresh_token` → token pair baru (rotasi) | 200, 401 |
| POST | /auth/logout | cabut refresh_token | 200 |
| GET | /auth/me | profil (butuh access token) | 200, 401 |
| GET | /students/ | query `page,limit,search,sort,order,is_active` | 200, 401 |
| GET | /students/:id | - | 200, 400, 404 |
| POST | /students/ | `nim,name,grade,is_active` | 201, 422, 409 |
| PUT | /students/:id | semua field | 200, 404, 409 |
| PATCH | /students/:id | salah satu field | 200, 400 |
| DELETE | /students/:id | - | 204, 404 |
| GET | /articles/:articleId/comments | daftar komentar artikel (permission `read comment`) | 200, 401, 403, 404 |
| POST | /articles/:articleId/comments | `content` (permission `create comment`) | 201, 401, 403, 404, 422 |
| PUT | /articles/:articleId/comments/:id | ubah `content`, cuma pemilik/admin (permission `update comment`) | 200, 401, 403, 404, 422 |
| DELETE | /articles/:articleId/comments/:id | hapus, cuma pemilik/admin (permission `delete comment`) | 204, 401, 403, 404 |

Semua endpoint /students dan /articles/:articleId/comments membutuhkan header `Authorization: Bearer <access_token>`.

## RBAC (Tugas Mandiri Modul 6)
Skema pakai tabel relasional: `roles` → `role_permissions` → `permissions`, user merujuk role lewat `users.role_id`.

Distribusi permission pada resource `comment`:

| Role | create | read | update | delete |
|------|--------|------|--------|--------|
| admin | ya | ya | ya (siapapun) | ya (siapapun) |
| editor | ya | ya | ya (milik sendiri) | ya (milik sendiri) |
| viewer | ya | ya | tidak | tidak |

- **Authentication** (`middleware/auth.go`): verifikasi JWT lalu ambil user + role **terkini dari DB** (perubahan role langsung berlaku, token lama tidak membawa role basi).
- **Authorization** (`middleware/authorize.go`): cek `role_permissions` di DB. **Fail closed** — kalau DB error, akses tetap ditolak (403).
- **Ownership** (`app/service/comment_handler.go`): update/delete komentar harus pemilik; admin bisa milik siapapun. Update hanya mengubah field `content`.
- 401 = identitas tidak terverifikasi (token hilang/expired/invalid), 403 = identitas valid tapi tidak berhak.

Kenapa ownership dicek di service layer, bukan middleware? Karena keputusannya bergantung pada isi data (`author_id` komentar) yang harus dibaca dari DB dulu. Middleware hanya cocok untuk keputusan berbasis metadata request (siapa user, permission apa). Service layer memang sudah membaca datanya, jadi cukup sekali baca dan tidak mencampur tanggung jawab antara auth, authz, dan ownership.

### User uji (password: `password123`)
| Username | Role |
|----------|------|
| admin1 | admin |
| editor1 | editor |
| viewer1 | viewer |

### Tabel ekspektasi pengujian
| User | Aksi | Milik | Expected |
|------|------|-------|----------|
| tanpa token | semua | - | 401 |
| viewer | read comment | siapapun | 200 |
| viewer | create comment | - | 201 |
| viewer | update comment | milik sendiri | 403 (tidak punya permission update) |
| viewer | delete comment | - | 403 (tidak punya permission delete) |
| editor | update comment | milik sendiri | 200 |
| editor | update comment | milik orang lain | 403 (bukan pemilik) |
| editor | delete comment | milik sendiri | 204 |
| admin | update/delete comment | siapapun | 200 / 204 |

Contoh perintah pengujian lengkap ada di bagian bawah `curl-ex.txt`.

## Keamanan yang Diterapkan
- Password di-hash **bcrypt** (cost 12), tidak pernah keluar di JSON (`json:"-"`)
- **JWT** access token pendek (15 menit) + **refresh token** acak (7 hari, disimpan sebagai hash SHA-256, bisa dicabut, dirotasi tiap dipakai)
- **Rate limiter** login: 5x/menit per IP → 429 + Retry-After
- **Anti user enumeration**: pesan login selalu sama + hash palsu saat username tidak ada
- **Anti algorithm confusion**: algoritma JWT diperiksa eksplisit (HMAC saja)
- **Anti mass assignment**: role ditentukan server, tidak ada di DTO register
- CORS dibatasi origin, body dibatasi 1 MB, sort kolom di-whitelist

## Development
```bash
go vet ./...
```
Contoh pengujian manual lengkap ada di `curl-ex.txt`.
