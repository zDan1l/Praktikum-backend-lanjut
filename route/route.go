package route

import (
	"context"
	"time"

	"pemrograman-code/app/repository"
	"pemrograman-code/app/service"
	"pemrograman-code/helper"
	"pemrograman-code/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	Pool           *pgxpool.Pool
	JWT            *helper.JWTManager
	Users          *repository.UserRepository
	StudentHandler *service.StudentHandler
	AuthService    *service.AuthService
}

func Setup(app *fiber.App, deps Dependencies) {
	api := app.Group("/api/v1")

	registerHealth(api, deps.Pool)

	auth := api.Group("/auth", middleware.RequireJSON)
	auth.Post("/register", deps.AuthService.Register)
	auth.Post("/login", middleware.LoginRateLimiter(), deps.AuthService.Login)
	auth.Post("/refresh", deps.AuthService.Refresh)
	auth.Post("/logout", deps.AuthService.Logout)
	auth.Get("/me", middleware.RequireAuth(deps.JWT, deps.Users), deps.AuthService.Me)

	students := api.Group("/students", middleware.RequireJSON, middleware.RequireAuth(deps.JWT, deps.Users))
	students.Get("/", deps.StudentHandler.List)
	students.Get("/:id", deps.StudentHandler.Get)
	students.Post("/", deps.StudentHandler.Create)
	students.Put("/:id", deps.StudentHandler.Replace)
	students.Patch("/:id", deps.StudentHandler.Patch)
	students.Delete("/:id", deps.StudentHandler.Delete)
}

func registerHealth(api fiber.Router, pool *pgxpool.Pool) {
	api.Get("/health", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			return helper.Fail(c, fiber.StatusServiceUnavailable, "database tidak dapat dihubungi")
		}
		return helper.Success(c, fiber.StatusOK, "server dan database berjalan", fiber.Map{"timestamp": time.Now()})
	})
}
