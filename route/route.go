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
	Pool             *pgxpool.Pool
	JWT              *helper.JWTManager
	Users            *repository.UserRepository
	StudentHandler   *service.StudentHandler
	PortfolioHandler *service.PortfolioHandler
	AuthService      *service.AuthService
	CommentHandler   *service.CommentHandler

	// Modul 11: KRS dan notifikasi background job.
	CourseService       *service.CourseService
	NotificationService *service.NotificationService
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

	// Tugas mandiri modul 7: portofolio student (PostgreSQL + MongoDB).
	students.Get("/:id/portfolio", deps.PortfolioHandler.Get)
	students.Put("/:id/portfolio", deps.PortfolioHandler.Replace)
	students.Post("/:id/portfolio/projects", deps.PortfolioHandler.AddProject)
	students.Delete("/:id/portfolio/projects/:projectId", deps.PortfolioHandler.DeleteProject)

	portfolios := api.Group("/portfolios", middleware.RequireJSON, middleware.RequireAuth(deps.JWT, deps.Users))
	portfolios.Get("/stats/technologies", deps.PortfolioHandler.TopTechnologies)

	// Tugas mandiri modul 6: comments dengan RBAC.
	// authenticate dulu (401 kalau token bermasalah), baru authorize (403 kalau
	// role tidak punya permission), kepemilikan komentar dicek di service layer.
	comments := api.Group("/articles/:articleId/comments", middleware.RequireJSON, middleware.RequireAuth(deps.JWT, deps.Users))
	comments.Get("/", middleware.RequirePermission(deps.Pool, "read", "comment"), deps.CommentHandler.List)
	comments.Post("/", middleware.RequirePermission(deps.Pool, "create", "comment"), deps.CommentHandler.Create)
	comments.Put("/:id", middleware.RequirePermission(deps.Pool, "update", "comment"), deps.CommentHandler.Update)
	comments.Delete("/:id", middleware.RequirePermission(deps.Pool, "delete", "comment"), deps.CommentHandler.Delete)

	// --- KRS (Modul 11): setiap perubahan dicatat ke outbox dalam transaction ---
	courses := api.Group("/courses", middleware.RequireJSON, middleware.RequireAuth(deps.JWT, deps.Users))
	courses.Post("/:id/enroll", deps.CourseService.Enroll)
	courses.Delete("/:id/enroll", deps.CourseService.Drop)
	courses.Patch("/:id", deps.CourseService.UpdateQuota)

	enrollments := api.Group("/enrollments", middleware.RequireJSON, middleware.RequireAuth(deps.JWT, deps.Users))
	enrollments.Post("/swap", deps.CourseService.Swap)

	// --- notifikasi (diisi oleh worker, dibaca lewat API) ---
	notifications := api.Group("/notifications", middleware.RequireJSON, middleware.RequireAuth(deps.JWT, deps.Users))
	notifications.Get("/mine", deps.NotificationService.Mine)

	admin := api.Group("/admin", middleware.RequireJSON, middleware.RequireAuth(deps.JWT, deps.Users))
	admin.Get("/queue", middleware.RequirePermission(deps.Pool, "monitor", "queue"), deps.NotificationService.Monitor)
}

func registerHealth(api fiber.Router, pool *pgxpool.Pool) {
	api.Get("/health", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			return helper.ServiceUnavailable("database tidak dapat dihubungi")
		}
		return helper.Success(c, fiber.StatusOK, "server dan database berjalan", fiber.Map{"timestamp": time.Now()})
	})
}
