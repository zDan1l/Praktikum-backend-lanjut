package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5/pgxpool"

	"pemrograman-code/app/repository"
	"pemrograman-code/app/service"
	"pemrograman-code/config"
	"pemrograman-code/database"
	"pemrograman-code/helper"
	"pemrograman-code/route"
)

const minSecretLength = 32

func main() {
	config.LoadEnv()
	logger := config.NewLogger()

	// Satu program, tiga cara menjalankan:
	//   go run . -role=api      hanya melayani HTTP
	//   go run . -role=worker   hanya relay outbox dan worker queue
	//   go run . -role=all      keduanya dalam satu proses (bawaan)
	// Di production, api dan worker dijalankan sebagai proses terpisah agar
	// masing-masing dapat diperbanyak sesuai bebannya.
	role := flag.String("role", config.GetEnv("APP_ROLE", "all"), "api | worker | all")
	flag.Parse()
	runAPI := *role == "api" || *role == "all"
	runWorker := *role == "worker" || *role == "all"
	if !runAPI && !runWorker {
		logger.Error("role tidak dikenal", slog.String("role", *role))
		os.Exit(1)
	}

	pool, err := database.NewPool(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	// ctx dibatalkan ketika Ctrl+C ditekan atau sistem mengirim SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var app *fiber.App
	if runAPI {
		app = startAPI(logger, pool)
		if app == nil {
			os.Exit(1)
		}
	}

	var stopWorker func()
	if runWorker {
		stopWorker = startWorker(ctx, logger, pool)
		if stopWorker == nil {
			os.Exit(1)
		}
	}

	logger.Info("aplikasi berjalan", slog.String("role", *role))
	<-ctx.Done()

	// Urutan berhenti: pintu masuk ditutup lebih dulu (HTTP), baru kemudian
	// pekerja di belakangnya.
	if app != nil {
		if err := app.ShutdownWithTimeout(10 * time.Second); err != nil {
			logger.Error("shutdown HTTP tidak rapi", slog.String("error", err.Error()))
		}
	}
	if stopWorker != nil {
		stopWorker()
	}
	logger.Info("aplikasi berhenti dengan rapi")
}

// startAPI menyiapkan seluruh kebutuhan HTTP. Mengembalikan nil bila gagal.
func startAPI(logger *slog.Logger, pool *pgxpool.Pool) *fiber.App {
	jwtSecret := config.GetEnv("JWT_SECRET", "")
	if len(jwtSecret) < minSecretLength {
		logger.Error("JWT_SECRET tidak diisi atau terlalu pendek",
			slog.Int("minimal_karakter", minSecretLength))
		return nil
	}

	mongoClient, mongoDB, err := database.NewMongo(context.Background())
	if err != nil {
		logger.Error("gagal terhubung ke MongoDB", slog.String("error", err.Error()))
		return nil
	}

	portfolioRepo := repository.NewPortfolioRepository(mongoDB)
	indexCtx, cancelIndex := context.WithTimeout(context.Background(), 10*time.Second)
	err = portfolioRepo.EnsureIndexes(indexCtx)
	if err == nil {
		err = repository.EnsureReviewIndexes(indexCtx, mongoDB)
	}
	cancelIndex()
	if err != nil {
		logger.Error("gagal menyiapkan index MongoDB", slog.String("error", err.Error()))
		return nil
	}

	// Pemetaan role -> permission dibaca sekali saat menyala.
	rawPerms, err := repository.NewRoleRepository(pool).LoadPermissions(context.Background())
	if err != nil {
		logger.Error("gagal memuat permission", slog.String("error", err.Error()))
		return nil
	}
	permissions := helper.NewPermissionSet(rawPerms)

	jwtManager := helper.NewJWTManager(
		jwtSecret,
		config.GetEnv("JWT_ISSUER", "praktikum-backend"),
		time.Duration(config.GetEnvInt("JWT_ACCESS_TTL_MINUTES", 15))*time.Minute,
	)

	userRepo := repository.NewUserRepository(pool)
	studentRepo := repository.NewStudentRepository(pool)
	txManager := repository.NewTxManager(pool)
	courseRepo := repository.NewCourseRepository(pool)
	outboxRepo := repository.NewOutboxRepository(pool)
	notifRepo := repository.NewNotificationRepository(pool)

	// Inspector hanya MEMBACA keadaan queue untuk endpoint admin. Ia tidak
	// menghubungi Redis saat dibuat, sehingga API tetap dapat menyala
	// walaupun Redis sedang mati.
	inspector := asynq.NewInspector(config.RedisOpt())

	authService := service.NewAuthService(
		userRepo,
		repository.NewTokenRepository(pool),
		jwtManager,
		time.Duration(config.GetEnvInt("JWT_REFRESH_TTL_DAYS", 7))*24*time.Hour,
	)

	app := config.NewApp(logger, route.Dependencies{
		Pool:                pool,
		Mongo:               mongoClient,
		JWT:                 jwtManager,
		Users:               userRepo,
		Permissions:         permissions,
		StudentHandler:      service.NewStudentHandler(studentRepo),
		PortfolioHandler:    service.NewPortfolioHandler(portfolioRepo, studentRepo),
		AuthService:         authService,
		CommentHandler:      service.NewCommentHandler(repository.NewCommentRepository(pool)),
		CourseService:       service.NewCourseService(courseRepo, outboxRepo, txManager),
		NotificationService: service.NewNotificationService(notifRepo, outboxRepo, inspector),
		ReviewService:       service.NewReviewService(repository.NewReviewRepository(mongoDB), courseRepo, permissions),
	})

	// Sumber daya milik API ditutup setelah server HTTP berhenti.
	app.Hooks().OnShutdown(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = inspector.Close()
		return mongoClient.Disconnect(ctx)
	})

	port := config.GetEnv("APP_PORT", "3000")
	go func() {
		if err := app.Listen(":" + port); err != nil {
			logger.Error("server berhenti", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()
	logger.Info("HTTP berjalan", slog.String("port", port))
	return app
}
