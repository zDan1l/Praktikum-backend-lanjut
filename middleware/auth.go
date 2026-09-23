package middleware

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"pemrograman-code/app/model"
	"pemrograman-code/app/repository"
	"pemrograman-code/helper"
)

func RequireAuth(jwtManager *helper.JWTManager, users *repository.UserRepository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		header := c.Get(fiber.HeaderAuthorization)
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			c.Set("WWW-Authenticate", `Bearer realm="api"`)
			return helper.Fail(c, fiber.StatusUnauthorized, "header Authorization tidak ada atau salah bentuk")
		}

		claims, err := jwtManager.Parse(strings.TrimSpace(parts[1]))
		if err != nil {
			c.Set("WWW-Authenticate", `Bearer realm="api"`)
			if errors.Is(err, helper.ErrExpiredToken) {
				return helper.Fail(c, fiber.StatusUnauthorized, "access token kedaluwarsa")
			}
			return helper.Fail(c, fiber.StatusUnauthorized, "access token tidak valid")
		}

		// Ambil user + role terkini dari DB, bukan cuma dari isi JWT,
		// supaya perubahan role setelah token terbit langsung berlaku.
		ctx, cancel := helper.RequestContext(c)
		defer cancel()
		user, err := users.FindByID(ctx, claims.UserID)
		if err != nil {
			return helper.Fail(c, fiber.StatusUnauthorized, "user tidak ditemukan")
		}
		if !user.IsActive {
			return helper.Fail(c, fiber.StatusForbidden, "akun dinonaktifkan")
		}

		c.Locals(helper.LocalsAuthUser, model.AuthUser{
			UserID: user.ID, Username: user.Username, RoleID: user.RoleID, Role: user.Role,
		})
		return c.Next()
	}
}

func LoginRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        5,
		Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			c.Set("Retry-After", "60")
			return helper.Fail(c, fiber.StatusTooManyRequests, "terlalu banyak percobaan login, coba lagi dalam satu menit")
		},
	})
}
