# Pemrograman Backend Lanjut (Portofolio Student: PostgreSQL + MongoDB)

API **CRUD Students + Auth (JWT) + Comments (RBAC) + Portofolio (MongoDB)** dengan **Fiber v2 + PostgreSQL (pgxpool) + MongoDB**.
Base URL: `http://localhost:3000/api/v1`

## Struktur
```
main.go            -> wiring: env, logger, DB pool, JWT, Mongo, repository, handler, route
config/config.go   -> env (.env), logger, pembuatan app Fiber
database/postgres.go -> pgxpool + ping
database/mongo.go  -> koneksi MongoDB + ping
app/model/         -> entitas & DTO (Student, Portfolio, User, Auth, Comment, ListQuery, Meta, WebResponse)
app/repository/    -> query SQL/Mongo (common.go, student, user, token, comment, portfolio)
app/service/       -> handler + validasi + cek ownership (student_handler, auth_service, comment_handler, portfolio_handler)
helper/            -> response, request, jwt, security (bcrypt)
middleware/        -> recover, cors, logger, RequireJSON, RequireAuth, RequirePermission, rate limiter
route/route.go     -> pendaftaran semua endpoint
migrations/        -> 001_create_students.sql ... 010_credits_version.sql, 006_student_portfolios.js
tests/integration/ -> integration test (build tag integration) terhadap praktikum_backend_test
docs/              -> hasil go tool swag init (docs.go, swagger.json, swagger.yaml)
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

MongoDB harus hidup di `MONGO_URI` (bawaan `mongodb://localhost:27017`). Index
`student_portfolios` dibuat otomatis saat startup; versi manualnya ada di
`migrations/006_student_portfolios.js` (`mongosh praktikum_backend migrations/006_student_portfolios.js`).

## Menjalankan Seluruh Sistem dengan Docker (Modul 12)
Cukup Docker; Go, PostgreSQL, MongoDB, dan Redis tidak perlu dipasang.
```bash
cp .env.example .env            # isi DB_PASSWORD dan JWT_SECRET (wajib)
docker compose up -d --build    # api, worker, postgres, mongo, redis
docker compose ps               # tunggu api berstatus healthy
curl http://localhost:8080/api/v1/health
```
- API di port **8080** (diteruskan ke 3000 di container). Database dan Redis
  hanya dapat dihubungi dari dalam network compose.
- Migration `migrations/*.sql` dijalankan image postgres **hanya** saat volume
  `pgdata` masih kosong. Migration baru: `docker compose exec -T postgres psql -U postgres -d praktikum_backend < migrations/0xx.sql`.
- `docker compose down` mempertahankan data; `docker compose down -v` menghapus seluruh volume.
- Menjadikan akun admin: `docker compose exec postgres psql -U postgres -d praktikum_backend -c "UPDATE users SET role_id = (SELECT id FROM roles WHERE name='admin') WHERE username='admin';"`

## Dokumentasi API (Swagger)
```bash
go tool swag init               # WAJIB diulang setiap anotasi berubah
```
Dengan `SWAGGER_ENABLED=true`, Swagger UI tersedia di
`http://localhost:3000/swagger/index.html` (atau port 8080 via compose).
Klik **Authorize** lalu isi `Bearer <access_token>`. Bawaannya mati (404),
karena dokumentasi lengkap di production adalah peta gratis bagi penyerang.
Folder `docs/` hasil swag **di-commit** karena diimpor saat compile.

## Testing (Modul 12)
```bash
go test ./...                                   # unit + component test, tanpa database
go test -shuffle=on -count=5 ./app/service/     # mendeteksi ketergantungan urutan
go test -coverprofile=coverage.out ./app/service/ && go tool cover -func=coverage.out

# integration test: butuh database TERPISAH yang namanya berakhiran _test
psql -U postgres -c "CREATE DATABASE praktikum_backend_test;"
echo 'TEST_DATABASE_URL=postgres://postgres:<password>@localhost:5432/praktikum_backend_test?sslmode=disable' > .env.test
go test -tags=integration -count=1 -v ./tests/integration/
```
`TestMain` menjalankan ulang seluruh migration (semuanya idempoten) dan
menolak berjalan pada database yang namanya tidak berakhiran `_test`.
Race detector (`-race`) butuh gcc 64-bit; di Windows tanpa gcc dapat dijalankan
di container: `docker run --rm -v "$PWD:/src" -w /src golang:1.26 go test -race ./app/service/`.

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
MONGO_URI=mongodb://localhost:27017
MONGO_DB=praktikum_backend
MONGO_TIMEOUT_SECONDS=10
JWT_SECRET=hasil-openssl-rand-hex-32
JWT_ISSUER=praktikum-backend
JWT_ACCESS_TTL_MINUTES=15
JWT_REFRESH_TTL_DAYS=7
ALLOWED_ORIGINS=http://localhost:5173
REDIS_ADDR=localhost:6379
SWAGGER_ENABLED=false
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
| GET | /students/:id/portfolio | portofolio milik student | 200, 400, 401, 404 |
| PUT | /students/:id/portfolio | buat/ganti seluruh portofolio (upsert) | 200, 400, 401, 404, 422 |
| POST | /students/:id/portfolio/projects | tambah proyek (maks 20, atomik) | 201, 400, 401, 404, 409, 422 |
| DELETE | /students/:id/portfolio/projects/:projectId | hapus proyek (`$pull`) | 204, 400, 401, 404 |
| GET | /portfolios/stats/technologies | 10 teknologi teratas (aggregation) | 200, 401 |
| GET | /articles/:articleId/comments | daftar komentar artikel (permission `read comment`) | 200, 401, 403, 404 |
| POST | /articles/:articleId/comments | `content` (permission `create comment`) | 201, 401, 403, 404, 422 |
| PUT | /articles/:articleId/comments/:id | ubah `content`, cuma pemilik/admin (permission `update comment`) | 200, 401, 403, 404, 422 |
| DELETE | /articles/:articleId/comments/:id | hapus, cuma pemilik/admin (permission `delete comment`) | 204, 401, 403, 404 |

Semua endpoint /students dan /articles/:articleId/comments membutuhkan header `Authorization: Bearer <access_token>`.

## Portofolio Student (Tugas Mandiri Modul 7)
Data mahasiswa tetap di PostgreSQL (`students`); portofolio disimpan di MongoDB
collection `student_portfolios`. Setiap mahasiswa punya tepat **satu** dokumen:

```json
{
  "student_id": 9,
  "headline": "Backend Engineer",
  "bio": "Suka Go dan MongoDB",
  "projects": [
    {
      "id": "6ae119d42f520bc3d8561611",
      "title": "API Portofolio",
      "technologies": ["Go", "Fiber", "MongoDB"],
      "link": "https://example.com/api",
      "year": 2026
    }
  ],
  "certificates": [
    {"name": "Backend Fundamental", "issuer": "Dicoding", "date": "2025-08-17"}
  ]
}
```

- **Model bebas tipe driver**: `app/model/portfolio.go` hanya memakai tipe Go
  (`string`, `int`, slice). Konversi `bson.ObjectID`/dokumen Mongo hanya ada di
  `app/repository/portfolio_repository.go`.
- **Satu mahasiswa satu portofolio** dijaga unique index `uniq_student_id`
  (`student_id: 1, unique: true`), bukan hanya pengecekan kode. Ada juga index
  multikey `projects_technologies_idx` untuk pencarian berdasarkan teknologi.
- **PUT** memverifikasi `students.id` ke PostgreSQL lebih dulu (404 bila tidak
  ada), lalu melakukan `ReplaceOne` + `upsert`. Bila dua PUT bersamaan membuat
  dokumen baru, duplikat unique index ditangani dengan replace ulang.
- **POST proyek** memakai pola `projects.19` (analog `replies.49`): filter update
  hanya cocok bila slot ke-20 belum terisi, sehingga batas 20 proyek atomik.
  Bila penuh → 409 `CONFLICT`.
- **DELETE proyek** memakai `$pull` berdasarkan `_id` proyek. `projectId` wajib
  ObjectID 24 karakter heksadesimal (400 bila tidak valid, 404 bila tidak ada).
- **Stats** menjalankan aggregation: `$unwind: "$projects"` → `$unwind:
  "$projects.technologies"` → `$group` hitung kemunculan → `$sort` → `$limit 10`.
- Seluruh endpoint portofolio membutuhkan `Authorization: Bearer <access_token>`
  seperti endpoint `/students`.

Contoh:
```bash
TOKEN="PASTE_ACCESS_TOKEN"

curl -s -X PUT localhost:3000/api/v1/students/9/portfolio \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"headline":"Backend Engineer","bio":"Suka Go","projects":[{"title":"API Portofolio","technologies":["Go","MongoDB"],"link":"https://example.com","year":2026}],"certificates":[{"name":"Backend Fundamental","issuer":"Dicoding","date":"2025-08-17"}]}'

curl -s localhost:3000/api/v1/students/9/portfolio -H "Authorization: Bearer $TOKEN"

curl -s -X POST localhost:3000/api/v1/students/9/portfolio/projects \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"title":"CLI Todo","technologies":["Go","Redis"],"link":"","year":2024}'

curl -s -X DELETE localhost:3000/api/v1/students/9/portfolio/projects/OBJECTID \
  -H "Authorization: Bearer $TOKEN"

curl -s localhost:3000/api/v1/portfolios/stats/technologies \
  -H "Authorization: Bearer $TOKEN"
```

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
