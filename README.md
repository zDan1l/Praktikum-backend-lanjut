# Pemrograman Backend Lanjut — Pertemuan 5 (Authentication & Security)

API **CRUD Students + Auth (JWT)** dengan **Fiber v2 + PostgreSQL (pgxpool)**.
Base URL: `http://localhost:3000/api/v1`

## Struktur
```
main.go            -> wiring: env, logger, DB pool, JWT, repository, handler, route
config/config.go   -> env (.env), logger, pembuatan app Fiber
database/postgres.go -> pgxpool + ping
app/model/         -> entitas & DTO (Student, User, Auth, ListQuery, Meta, WebResponse)
app/repository/    -> query SQL (common.go, student, user, token)
app/service/       -> handler + validasi (student_handler.go, auth_service.go, auth_rules.go)
helper/            -> response, request, jwt, security (bcrypt)
middleware/        -> recover, cors, logger, RequireJSON, RequireAuth, rate limiter
route/route.go     -> pendaftaran semua endpoint
migrations/        -> 001_create_students.sql, 002_auth.sql
```

## Setup
```bash
cp .env.example .env
# isi DB_PASSWORD dan JWT_SECRET (openssl rand -hex 32, minimal 32 karakter)

createdb praktikum_backend
psql -d praktikum_backend -f migrations/001_create_students.sql
psql -d praktikum_backend -f migrations/002_auth.sql
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

Semua endpoint /students membutuhkan header `Authorization: Bearer <access_token>`.

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
