package helper

import (
	"context"
	"strconv"
	"strings"
	"time"

	"pemrograman-code/app/model"

	"github.com/gofiber/fiber/v2"
)

// RequestContext memberi batas 5 detik untuk query DB di satu request.
func RequestContext(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.UserContext(), 5*time.Second)
}

// ParamID mengambil :id dari URL, pastikan angka positif.
func ParamID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, false
	}
	return id, true
}

// kolom yang boleh dipakai untuk ?sort= (whitelist, cegah SQL injection)
var sortWhitelist = map[string]bool{
	"id": true, "nim": true, "name": true, "grade": true, "created_at": true,
}

// ParseQuery membaca query param list: page, limit, search, sort, order, is_active.
func ParseQuery(c *fiber.Ctx) model.ListQuery {
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
	if !sortWhitelist[q.Sort] {
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

// NewMeta menghitung meta pagination (total halaman dibulatkan ke atas).
func NewMeta(q model.ListQuery, total int) *model.Meta {
	pages := 0
	if q.Limit > 0 {
		pages = (total + q.Limit - 1) / q.Limit
	}
	return &model.Meta{Page: q.Page, Limit: q.Limit, Total: total, TotalPages: pages}
}
