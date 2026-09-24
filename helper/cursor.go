package helper

import (
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"pemrograman-code/app/model"
)

// ErrInvalidCursor diterjemahkan menjadi 400 oleh service.
var ErrInvalidCursor = errors.New("cursor tidak valid")

// EncodeCursor mengubah penanda menjadi string aman URL.
// Base64 adalah PENGKODEAN, bukan enkripsi.
func EncodeCursor(createdAt time.Time, id int) string {
	raw := strconv.FormatInt(createdAt.UTC().UnixNano(), 10) +
		"|" + strconv.Itoa(id)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor membaca kembali penanda. Rusak berarti 400.
func DecodeCursor(encoded string) (model.Cursor, error) {
	if strings.TrimSpace(encoded) == "" {
		return model.Cursor{}, ErrInvalidCursor
	}

	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return model.Cursor{}, ErrInvalidCursor
	}

	parts := strings.Split(string(decoded), "|")
	if len(parts) != 2 {
		return model.Cursor{}, ErrInvalidCursor
	}

	nanos, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return model.Cursor{}, ErrInvalidCursor
	}

	id, err := strconv.Atoi(parts[1])
	if err != nil || id < 1 {
		return model.Cursor{}, ErrInvalidCursor
	}

	return model.Cursor{CreatedAt: time.Unix(0, nanos).UTC(), ID: id}, nil
}

// ParseCursorQuery membaca limit/cursor/search/is_active.
// Limit dibatasi agar 1000 tidak membebani database.
func ParseCursorQuery(c *fiber.Ctx) (model.CursorQuery, error) {
	q := model.CursorQuery{Limit: c.QueryInt("limit", 10)}
	if q.Limit < 1 {
		q.Limit = 10
	}
	if q.Limit > 50 {
		q.Limit = 50
	}
	q.Search = strings.TrimSpace(c.Query("search"))
	if raw := c.Query("is_active"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return model.CursorQuery{}, BadRequest("is_active harus true atau false")
		}
		q.IsActive = &v
	}
	if raw := strings.TrimSpace(c.Query("cursor")); raw != "" {
		cur, err := DecodeCursor(raw)
		if err != nil {
			return model.CursorQuery{}, BadRequest("cursor tidak valid")
		}
		q.After = &cur
	}
	return q, nil
}
