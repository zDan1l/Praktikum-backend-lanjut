# Pemrograman Backend Lanjut

CRUD **Students + Prestasi** dengan **Fiber v2 + PostgreSQL (pgxpool) + Repository Pattern**.  
Base URL: `http://localhost:3000/api/v1`

## Stack & Struktur (sudah disederhanakan)
```
app/
  model/       -> entitas & DTO (Student, Prestasi, ListQuery, Meta, WebResponse)
  repository/  -> query SQL (common.go, student_repository.go, prestasi_repository.go)
  service/     -> handler + validasi murni (student_handler.go, prestasi_handler.go, validation.go)
config/        -> env, logger, app wiring
database/      -> pgxpool
helper/        -> request (ParseQuery + NewMeta) + response (Success/Fail)
middleware/    -> requestid, recover, helmet, cors, RequireJSON
route/         -> pendaftaran route (satu fungsi per resource)
migrations/    -> 001_create_students.sql, 002_create_prestasi.sql
main.go        -> wiring 5 langkah (lihat komentar di file)
```

## Cara Tambah Endpoint Baru (3 langkah, copy-paste)

Mau tambah tabel `buku`? Ikuti pola `student`/`prestasi`:

**1. Model** `app/model/buku.go`
```go
type Buku struct { ID int `json:"id"`; Judul string `json:"judul"`; CreatedAt *time.Time `json:"created_at"` }
type CreateBukuRequest struct { Judul string `json:"judul"` }
```

**2. Repository** `app/repository/buku_repository.go` (copy dari `student_repository.go`)
- ganti `kolomUrut`, `buildFilter`, `SELECT ... FROM buku`, `Scan`

**3. Handler + Validasi** `app/service/buku_handler.go` + `buku_validation.go` (copy dari `prestasi_handler.go`)
- ganti `ValidateBukuCreate`, handler `List/Get/Create/Replace/Patch/Delete`

**4. Wiring**
```go
// main.go tambah 2 baris:
bukuRepo := repository.NewBukuRepository(pool)
bukuHandler := service.NewBukuHandler(bukuRepo)
app := config.NewApp(logger, pool, studentHandler, prestasiHandler, bukuHandler)

// route/route.go tambah 1 baris di Setup:
registerBukuRoutes(api, bukuHandler)
func registerBukuRoutes(api fiber.Router, h *service.BukuHandler) {
    g := api.Group("/buku", middleware.RequireJSON)
    g.Get("/", h.List); g.Get("/:id", h.Get); g.Post("/", h.Create)
}
```

Sudah ada contoh lengkap di `prestasi` (full CRUD) bisa langsung di-copy.

## Variabel Environment
```bash
cp .env.example .env
# isi DB_PASSWORD
```
```
APP_PORT=3000
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=xxx
DB_NAME=praktikum_backend
DB_SSLMODE=disable
DB_MAX_CONNS=10
```

## Setup DB dari Nol
```bash
createdb praktikum_backend
psql -d praktikum_backend -f migrations/001_create_students.sql
psql -d praktikum_backend -f migrations/002_create_prestasi.sql
go run .
curl -i http://localhost:3000/api/v1/health # 200 jika DB hidup, 503 jika mati
```

## Kontrak API
| Metode | Endpoint | Body | Status |
|--------|----------|------|--------|
| GET | /health | - | 200 OK, 503 jika DB mati |
| GET | /students/ | query `page,limit,search,sort,order,is_active` | 200 + meta |
| GET | /students/:id | - | 200, 400, 404 |
| POST | /students/ | `nim,name,grade,is_active` | 201, 400, 415, 422, 409 |
| PUT | /students/:id | semua field | 200, 404, 409 |
| PATCH | /students/:id | salah satu field | 200, 400 |
| DELETE | /students/:id | - | 204, 404 |
| GET | /prestasi/ | `page,limit,search,sort,order` | 200 |
| GET | /prestasi/:id | - | 200, 404 |
| POST | /prestasi/ | `id_student,nama_prestasi,juara` | 201, 422 |
| PUT | /prestasi/:id | semua field | 200 |
| PATCH | /prestasi/:id | salah satu | 200 |
| DELETE | /prestasi/:id | - | 204 |

Validasi: `nim/name` tidak kosong, `grade 0-100`, `nama_prestasi` tidak kosong, `juara >=1`, `id_student >0`.

## Helper yang Memudahkan
- `helper.ParseStudentQuery(c)` / `ParsePrestasiQuery(c)` - sudah whitelist sort per tabel
- `helper.NewMeta(q, total)` - hitung pagination
- `helper.Success / SuccessList / Created / Fail / FailValidation` - response envelope
- `repository/common.go` - `orderDir` & `sortCol` biar tidak duplikat

## Development
```bash
go vet ./...
go test ./...
curl -i http://localhost:3000/api/v1/students/
curl -i -X POST -H "Content-Type: application/json" -d '{"nim":"22001","name":"Budi","grade":85,"is_active":true}' http://localhost:3000/api/v1/students/
curl -i -X POST -H "Content-Type: application/json" -d '{"id_student":1,"nama_prestasi":"CTF","juara":1}' http://localhost:3000/api/v1/prestasi/
```
