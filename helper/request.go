package helper

import (
	"context"
	"strconv"
	"strings"
	"time"

	"pemrograman-code/app/model"

	"github.com/gofiber/fiber/v2"
)

// RequestContext kasih timeout 5 detik untuk tiap query DB.
// Pakai di handler: ctx, cancel := helper.ReqCtx(c); defer cancel()
func RequestContext(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.UserContext(), 5*time.Second)
}

// ReqCtx alias biar kompatibel dengan kode lama
func ReqCtx(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return RequestContext(c)
}

// ParamID ambil :id dari URL, pastikan angka positif
func ParamID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}

// Whitelist kolom yang boleh di-sort per resource
var studentSortWhitelist = map[string]bool{
	"id": true, "nim": true, "name": true, "grade": true, "created_at": true,
}
var prestasiSortWhitelist = map[string]bool{
	"id": true, "id_student": true, "nama_prestasi": true, "juara": true, "created_at": true,
}

// allowedSort gabungan untuk ParseListQuery generik (backward compat)
var allowedSort = map[string]bool{
	"id": true, "nim": true, "name": true, "grade": true, "created_at": true,
	"id_student": true, "nama_prestasi": true, "juara": true,
}

// parseList pakai whitelist tertentu
func parseList(c *fiber.Ctx, whitelist map[string]bool) model.ListQuery {
	q := model.ListQuery{
		Page:   c.QueryInt("page", 1),
		Limit:  c.QueryInt("limit", 10),
		Search: strings.TrimSpace(c.Query("search")),
		Sort:   c.Query("sort", "id"),
		Order:  strings.ToLower(c.Query("order", "asc")),
	}
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 {
		q.Limit = 10
	}
	if q.Limit > 100 {
		q.Limit = 100
	}
	if !whitelist[q.Sort] {
		q.Sort = "id"
	}
	if q.Order != "desc" {
		q.Order = "asc"
	}
	if raw := c.Query("is_active"); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil {
			q.IsActive = &v
		}
	}
	return q
}

// ParseListQuery generik (gabungan student+prestasi). Untuk kode baru, pakai yang spesifik di bawah.
func ParseListQuery(c *fiber.Ctx) model.ListQuery {
	return parseList(c, allowedSort)
}

// ParseStudentQuery khusus untuk /students
func ParseStudentQuery(c *fiber.Ctx) model.ListQuery {
	return parseList(c, studentSortWhitelist)
}

// ParsePrestasiQuery khusus untuk /prestasi
func ParsePrestasiQuery(c *fiber.Ctx) model.ListQuery {
	return parseList(c, prestasiSortWhitelist)
}

// NewMeta hitung meta pagination dari total & limit (bulat ke atas)
// Contoh pakai di handler: meta := helper.NewMeta(q, total)
func NewMeta(q model.ListQuery, total int) *model.Meta {
	pages := 0
	if q.Limit > 0 {
		pages = (total + q.Limit - 1) / q.Limit
	}
	return &model.Meta{Page: q.Page, Limit: q.Limit, Total: total, TotalPages: pages}
}
