# Pemrograman Backend Lanjut — API Student

Server: Fiber · Base URL: `http://localhost:3000/api/v1`

## Kontrak API

| Metode | Endpoint | Parameter | Contoh Body Permintaan | Status yang Mungkin Dikembalikan | Contoh Respons |
|--------|----------|-----------|------------------------|----------------------------------|----------------|
| `GET` | `/health` | — | — | `200` OK | `{"success":true,"message":"server berjalan","data":{"timestamp":"..."}}` |
| `GET` | `/students` | query: `page` (default 1), `limit` (default 10, maks 100), `search`, `sort` (`id`, `name`, `grade`), `order` (`asc`, `desc`), `is_active` (`true`/`false`) | — | `200` OK<br>`500` error server | `{"success":true,"message":"daftar student berhasil diambil","data":[{"ID":1,"Name":"Budi","Grade":85.5,"IsActive":true}],"meta":{"page":1,"limit":10,"total":1,"total_pages":1}}` |
| `GET` | `/students/:id` | path: `id` (angka positif) | — | `200` OK<br>`400` id tidak valid<br>`404` tidak ditemukan<br>`500` error server | `{"success":true,"message":"student ditemukan","data":{"ID":1,"Name":"Budi","Grade":85.5,"IsActive":true}}` |
| `POST` | `/students` | header: `Content-Type: application/json` | `{"name":"Budi","grade":85.5,"is_active":true}` | `201` Created<br>`400` body bukan JSON valid<br>`415` Content-Type bukan JSON<br>`422` validasi gagal<br>`500` error server | `{"success":true,"message":"student berhasil dibuat","data":{"ID":1,"Name":"Budi","Grade":85.5,"IsActive":true}}` + header `Location: /api/v1/students/1` |
| `PUT` | `/students/:id` | path: `id` (angka positif)<br>header: `Content-Type: application/json` | `{"name":"Budi","grade":90,"is_active":false}` | `200` OK<br>`400` id/body tidak valid<br>`404` tidak ditemukan<br>`415` Content-Type bukan JSON<br>`422` validasi gagal<br>`500` error server | `{"success":true,"message":"student berhasil diganti seluruhnya","data":{"ID":1,"Name":"Budi","Grade":90,"IsActive":false}}` |
| `PATCH` | `/students/:id` | path: `id` (angka positif)<br>header: `Content-Type: application/json` | `{"grade":95.5}` | `200` OK<br>`400` id/body tidak valid / tidak ada field diubah<br>`404` tidak ditemukan<br>`415` Content-Type bukan JSON<br>`422` validasi gagal<br>`500` error server | `{"success":true,"message":"student berhasil diperbarui sebagian","data":{"ID":1,"Name":"Budi","Grade":95.5,"IsActive":true}}` |
| `DELETE` | `/students/:id` | path: `id` (angka positif) | — | `204` No Content (tanpa body)<br>`400` id tidak valid<br>`404` tidak ditemukan<br>`500` error server | — |

### Catatan

- Body request: `name`, `grade`, `is_active` (snake_case).
- Response `data`: `ID`, `Name`, `Grade`, `IsActive` (PascalCase, `model.go:3` tanpa tag `json`).
- Gagal: `{"success":false,"message":"..."}` dan `422` ditambah `"errors":{"field":"pesan"}`.
