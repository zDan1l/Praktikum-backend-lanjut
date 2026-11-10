package route

import (
	"context"
	"time"

	"pemrograman-code/app/repository"
	"pemrograman-code/app/service"
	"pemrograman-code/helper"
	"pemrograman-code/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/swagger"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Dependencies berisi seluruh yang dibutuhkan route. Field yang tidak diisi
// boleh nil: test hanya mengisi bagian yang ia uji (Modul 12, Langkah 3 & 8).
type Dependencies struct {
	Pool             *pgxpool.Pool
	Mongo            *mongo.Client
	JWT              *helper.JWTManager
	Users            repository.UserRepository
	Permissions      *helper.PermissionSet
	StudentHandler   *service.StudentHandler
	PortfolioHandler *service.PortfolioHandler
	AuthService      *service.AuthService
	CommentHandler   *service.CommentHandler

	// Modul 11: KRS dan notifikasi background job.
	CourseService       *service.CourseService
	NotificationService *service.NotificationService

	// Modul 10/12: ulasan kelas (MongoDB).
	ReviewService *service.ReviewService

	// EnableSwagger menyalakan Swagger UI di /swagger. Di production
	// sebaiknya dimatikan: dokumentasi lengkap adalah peta gratis bagi
	// penyerang.
	EnableSwagger bool
}

func Register(app *fiber.App, deps Dependencies) {
	if deps.EnableSwagger {
		// Swagger UI: http://localhost:3000/swagger/index.html
		app.Get("/swagger/*", swagger.HandlerDefault)
	}

	api := app.Group("/api/v1")

	registerHealth(api, deps.Pool, deps.Mongo)

	requireAuth := middleware.RequireAuth(deps.JWT, deps.Users)

	auth := api.Group("/auth", middleware.RequireJSON)
	auth.Post("/register", deps.AuthService.Register)
	auth.Post("/login", middleware.LoginRateLimiter(), deps.AuthService.Login)
	auth.Post("/refresh", deps.AuthService.Refresh)
	auth.Post("/logout", deps.AuthService.Logout)
	auth.Get("/me", requireAuth, deps.AuthService.Me)

	students := api.Group("/students", middleware.RequireJSON, requireAuth)
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

	portfolios := api.Group("/portfolios", middleware.RequireJSON, requireAuth)
	portfolios.Get("/stats/technologies", deps.PortfolioHandler.TopTechnologies)

	// Tugas mandiri modul 6: comments dengan RBAC.
	// authenticate dulu (401 kalau token bermasalah), baru authorize (403 kalau
	// role tidak punya permission), kepemilikan komentar dicek di service layer.
	comments := api.Group("/articles/:articleId/comments", middleware.RequireJSON, requireAuth)
	comments.Get("/", middleware.RequirePermission(deps.Pool, "read", "comment"), deps.CommentHandler.List)
	comments.Post("/", middleware.RequirePermission(deps.Pool, "create", "comment"), deps.CommentHandler.Create)
	comments.Put("/:id", middleware.RequirePermission(deps.Pool, "update", "comment"), deps.CommentHandler.Update)
	comments.Delete("/:id", middleware.RequirePermission(deps.Pool, "delete", "comment"), deps.CommentHandler.Delete)

	// --- KRS: setiap perubahan dicatat ke outbox dalam transaction ---
	// /mine didaftarkan sebelum /:id agar tidak tertangkap sebagai id.
	courses := api.Group("/courses", middleware.RequireJSON, requireAuth)
	courses.Get("/", deps.CourseService.List)
	courses.Get("/mine", deps.CourseService.Mine)
	courses.Get("/:id", deps.CourseService.Get)
	courses.Patch("/:id", middleware.RequireAllowed(deps.Permissions, "course:manage"), deps.CourseService.UpdateQuota)
	courses.Post("/:id/enroll", deps.CourseService.Enroll)
	courses.Delete("/:id/enroll", deps.CourseService.Drop)

	// --- ulasan kelas (MongoDB) ---
	courses.Get("/:id/reviews", deps.ReviewService.ListByCourse)
	courses.Post("/:id/reviews", deps.ReviewService.Create)
	courses.Get("/:id/reviews/summary", deps.ReviewService.Summary)

	reviews := api.Group("/reviews", middleware.RequireJSON, requireAuth)
	reviews.Get("/:id", deps.ReviewService.Get)
	reviews.Patch("/:id", deps.ReviewService.Update)
	reviews.Delete("/:id", deps.ReviewService.Delete)
	reviews.Post("/:id/helpful", deps.ReviewService.Vote)
	reviews.Post("/:id/replies", deps.ReviewService.Reply)

	enrollments := api.Group("/enrollments", middleware.RequireJSON, requireAuth)
	enrollments.Post("/swap", deps.CourseService.Swap)

	// --- notifikasi (diisi oleh worker, dibaca lewat API) ---
	notifications := api.Group("/notifications", middleware.RequireJSON, requireAuth)
	notifications.Get("/mine", deps.NotificationService.Mine)

	admin := api.Group("/admin", middleware.RequireJSON, requireAuth)
	admin.Get("/queue", middleware.RequireAllowed(deps.Permissions, "queue:monitor"), deps.NotificationService.Monitor)
}

func registerHealth(api fiber.Router, pool *pgxpool.Pool, mongoClient *mongo.Client) {
	api.Get("/health", func(c *fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.UserContext(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			return helper.ServiceUnavailable("PostgreSQL tidak dapat dihubungi")
		}
		if mongoClient != nil {
			if err := mongoClient.Ping(ctx, nil); err != nil {
				return helper.ServiceUnavailable("MongoDB tidak dapat dihubungi")
			}
		}
		return helper.Success(c, fiber.StatusOK, "server, PostgreSQL, dan MongoDB berjalan", fiber.Map{"timestamp": time.Now()})
	})
}
