package route

import (
	"context"
	"time"

	"pemrograman-code/app/service"
	"pemrograman-code/helper"
	"pemrograman-code/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Dependencies struct {
	Pool            *pgxpool.Pool
	JWT             *helper.JWTManager
	StudentHandler  *service.StudentHandler
	PrestasiHandler *service.PrestasiHandler
	AuthService     *service.AuthService
}

func Setup(app *fiber.App, deps Dependencies) {
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	api := app.Group("/api/v1")
	registerHealth(api, deps.Pool)
	registerAuthRoutes(api, deps.AuthService, deps.JWT)
	registerStudentRoutes(api, deps.StudentHandler, deps.JWT)
	registerPrestasiRoutes(api, deps.PrestasiHandler, deps.JWT)
}

// alias untuk kompatibilitas lama
func Register(app *fiber.App, deps Dependencies) {
	Setup(app, deps)
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

func registerAuthRoutes(api fiber.Router, auth *service.AuthService, jwt *helper.JWTManager) {
	g := api.Group("/auth", middleware.RequireJSON)
	g.Post("/register", auth.Register)
	g.Post("/login", middleware.LoginRateLimiter(), auth.Login)
	g.Post("/refresh", auth.Refresh)
	g.Post("/logout", auth.Logout)
	g.Get("/me", middleware.RequireAuth(jwt), auth.Me)
}

func registerStudentRoutes(api fiber.Router, h *service.StudentHandler, jwt *helper.JWTManager) {
	g := api.Group("/students", middleware.RequireJSON, middleware.RequireAuth(jwt))
	g.Get("/", h.List)
	g.Get("/:id", h.Get)
	g.Post("/", h.Create)
	g.Put("/:id", h.Replace)
	g.Patch("/:id", h.Patch)
	g.Delete("/:id", h.Delete)
}

func registerPrestasiRoutes(api fiber.Router, h *service.PrestasiHandler, jwt *helper.JWTManager) {
	g := api.Group("/prestasi", middleware.RequireJSON, middleware.RequireAuth(jwt))
	g.Get("/", h.List)
	g.Get("/:id", h.Get)
	g.Post("/", h.Create)
	g.Put("/:id", h.Replace)
	g.Patch("/:id", h.Patch)
	g.Delete("/:id", h.Delete)
}
