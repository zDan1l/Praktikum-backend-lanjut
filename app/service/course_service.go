package service

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"pemrograman-code/app/model"
	"pemrograman-code/app/repository"
	"pemrograman-code/helper"
)

// CourseService melayani endpoint KRS: daftar kelas, mendaftar, keluar, dan
// pindah kelas. Setiap perubahan KRS dicatat ke outbox DI DALAM transaction
// yang sama, sehingga notifikasi tidak pernah dibuat untuk pendaftaran yang
// batal dan tidak pernah hilang ketika Redis mati (Modul 11, A.3).
type CourseService struct {
	courses repository.CourseRepository
	outbox  *repository.OutboxRepository
	tx      *repository.TxManager
}

func NewCourseService(
	courses repository.CourseRepository,
	outbox *repository.OutboxRepository,
	tx *repository.TxManager,
) *CourseService {
	return &CourseService{courses: courses, outbox: outbox, tx: tx}
}

var (
	errCreditLimit = errors.New("batas SKS terlampaui")
	errNotEnrolled = errors.New("tidak terdaftar di kelas")
	errStudentGone = errors.New("akun mahasiswa tidak ditemukan")
)

func conflict(code, message string) *helper.AppError {
	return &helper.AppError{Status: fiber.StatusConflict, Code: code, Message: message}
}

// translateCourseError memetakan error dari transaction KRS menjadi response.
func translateCourseError(err error) error {
	switch {
	case errors.Is(err, errStudentGone):
		return helper.Unauthorized("akun tidak ditemukan")
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("kelas tidak ditemukan")
	case errors.Is(err, repository.ErrCourseFull):
		return conflict("COURSE_FULL", "kelas sudah penuh")
	case errors.Is(err, repository.ErrDuplicate):
		return conflict("ALREADY_ENROLLED", "Anda sudah terdaftar di kelas ini")
	case errors.Is(err, errCreditLimit):
		return conflict("CREDIT_LIMIT_EXCEEDED", "batas "+strconv.Itoa(MaxCreditsPerSemester)+" SKS per semester terlampaui")
	case errors.Is(err, errNotEnrolled):
		return conflict("NOT_ENROLLED", "Anda tidak terdaftar di kelas tersebut")
	case errors.Is(err, repository.ErrVersionConflict):
		return conflict("VERSION_CONFLICT", "kelas sudah diubah orang lain; muat ulang lalu coba lagi")
	case errors.Is(err, repository.ErrQuotaBelowEnrolled):
		return conflict("QUOTA_BELOW_ENROLLED", "kuota tidak boleh lebih kecil dari jumlah peserta")
	default:
		return helper.Internal(err)
	}
}

// ---------- GET /courses ----------

// List godoc
//
// @Summary Daftar seluruh kelas
// @Tags courses
// @Produce json
// @Success 200 {object} model.WebResponse{data=[]model.Course}
// @Failure 401 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /courses [get]
func (s *CourseService) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	courses, err := s.courses.List(ctx)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "daftar kelas berhasil diambil", courses)
}

// ---------- GET /courses/mine ----------

// Mine godoc
//
// @Summary KRS milik mahasiswa yang sedang login
// @Tags courses
// @Produce json
// @Success 200 {object} model.WebResponse{data=model.MyCourses}
// @Failure 401 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /courses/mine [get]
func (s *CourseService) Mine(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}
	courses, err := s.courses.ListByUser(ctx, current.UserID)
	if err != nil {
		return helper.Internal(err)
	}
	total := 0
	for _, course := range courses {
		total += course.Credits
	}
	return helper.Success(c, fiber.StatusOK, "KRS berhasil diambil", model.MyCourses{
		Courses: courses, TotalCredits: total, MaxCredits: MaxCreditsPerSemester,
	})
}

// ---------- GET /courses/:id ----------

// Get godoc
//
// @Summary Detail satu kelas
// @Tags courses
// @Produce json
// @Param id path int true "ID kelas"
// @Success 200 {object} model.WebResponse{data=model.Course}
// @Failure 400 {object} model.ErrorResponse "ID bukan angka positif"
// @Failure 404 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /courses/{id} [get]
func (s *CourseService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	courseID, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id kelas harus angka positif")
	}
	course, err := s.courses.FindByID(ctx, courseID)
	if err != nil {
		return translateCourseError(err)
	}
	return helper.Success(c, fiber.StatusOK, "detail kelas berhasil diambil", course)
}

// ---------- POST /courses/:id/enroll ----------

// Enroll godoc
//
// @Summary Mendaftar ke sebuah kelas
// @Description Mengambil satu kursi dan mencatat event enrollment.created ke outbox dalam SATU transaction.
// @Description Kode error 409: COURSE_FULL, ALREADY_ENROLLED, CREDIT_LIMIT_EXCEEDED.
// @Tags courses
// @Produce json
// @Param id path int true "ID kelas"
// @Success 201 {object} model.WebResponse{data=model.Course}
// @Header 201 {string} Location "URL kelas yang baru didaftari"
// @Failure 401 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 409 {object} model.ErrorResponse "Kuota penuh, sudah terdaftar, atau batas SKS terlampaui"
// @Security BearerAuth
// @Router /courses/{id}/enroll [post]
func (s *CourseService) Enroll(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}
	courseID, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id kelas harus angka positif")
	}

	var enrolled model.Course
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.courses.WithTx(tx)

		// 1. Kunci row mahasiswa: pendaftaran serentak milik mahasiswa yang
		//    sama mengantre di sini, sehingga pemeriksaan SKS di langkah 3
		//    selalu membaca jumlah SKS terbaru.
		if err := repo.LockStudent(ctx, current.UserID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return errStudentGone
			}
			return err
		}

		// 2. Kelasnya harus ada dan belum diambil.
		course, err := repo.FindByID(ctx, courseID)
		if err != nil {
			return err
		}
		already, err := repo.IsEnrolled(ctx, courseID, current.UserID)
		if err != nil {
			return err
		}
		if already {
			return repository.ErrDuplicate
		}

		// 3. Aturan batas SKS.
		taken, err := repo.TakenCredits(ctx, current.UserID)
		if err != nil {
			return err
		}
		if !WithinCreditLimit(taken, course.Credits) {
			return errCreditLimit
		}

		// 4. Ambil satu kursi secara atomik (row kelas di-lock).
		enrolled, err = repo.IncrementEnrolled(ctx, courseID)
		if err != nil {
			return err
		}
		if err := repo.InsertEnrollment(ctx, courseID, current.UserID); err != nil {
			return err
		}

		// 5. Catat event ke outbox memakai transaction yang SAMA: bila ada
		//    langkah yang gagal, catatan ini ikut dibatalkan.
		return s.outbox.WithTx(tx).Insert(ctx, model.EventEnrollmentCreated, model.EnrollmentEvent{
			UserID:     current.UserID,
			CourseID:   enrolled.ID,
			CourseCode: enrolled.Code,
			CourseName: enrolled.Name,
		})
	})
	if err != nil {
		return translateCourseError(err)
	}
	return helper.Created(c, "berhasil terdaftar di kelas", enrolled, "/api/v1/courses/"+strconv.Itoa(enrolled.ID))
}

// ---------- DELETE /courses/:id/enroll ----------

// Drop godoc
//
// @Summary Keluar dari sebuah kelas
// @Tags courses
// @Param id path int true "ID kelas"
// @Success 204 "Berhasil keluar; tanpa body"
// @Failure 401 {object} model.ErrorResponse
// @Failure 409 {object} model.ErrorResponse "NOT_ENROLLED"
// @Security BearerAuth
// @Router /courses/{id}/enroll [delete]
func (s *CourseService) Drop(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}
	courseID, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id kelas harus angka positif")
	}

	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.courses.WithTx(tx)

		if err := repo.DeleteEnrollment(ctx, courseID, current.UserID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return errNotEnrolled
			}
			return err
		}
		course, err := repo.DecrementEnrolled(ctx, courseID)
		if err != nil {
			return err
		}
		return s.outbox.WithTx(tx).Insert(ctx, model.EventEnrollmentDropped, model.EnrollmentEvent{
			UserID:     current.UserID,
			CourseID:   course.ID,
			CourseCode: course.Code,
			CourseName: course.Name,
		})
	})
	if err != nil {
		return translateCourseError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ---------- POST /enrollments/swap ----------

// Swap godoc
//
// @Summary Pindah kelas
// @Description Keluar dari kelas lama dan masuk ke kelas baru sebagai satu tindakan. Bila kelas baru penuh, mahasiswa tetap di kelas lama.
// @Tags courses
// @Accept json
// @Produce json
// @Param body body model.SwapRequest true "Kelas asal dan kelas tujuan"
// @Success 200 {object} model.WebResponse{data=model.Course}
// @Failure 401 {object} model.ErrorResponse
// @Failure 409 {object} model.ErrorResponse "COURSE_FULL, NOT_ENROLLED, CREDIT_LIMIT_EXCEEDED"
// @Failure 422 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /enrollments/swap [post]
func (s *CourseService) Swap(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}

	var req model.SwapRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	var to model.Course
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.courses.WithTx(tx)

		if err := repo.LockStudent(ctx, current.UserID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return errStudentGone
			}
			return err
		}

		fromCourse, err := repo.FindByID(ctx, req.FromCourseID)
		if err != nil {
			return err
		}
		toCourse, err := repo.FindByID(ctx, req.ToCourseID)
		if err != nil {
			return err
		}
		if already, err := repo.IsEnrolled(ctx, req.ToCourseID, current.UserID); err != nil {
			return err
		} else if already {
			return repository.ErrDuplicate
		}

		taken, err := repo.TakenCredits(ctx, current.UserID)
		if err != nil {
			return err
		}
		if CreditsAfterSwap(taken, fromCourse.Credits, toCourse.Credits) > MaxCreditsPerSemester {
			return errCreditLimit
		}

		// Keluar dari kelas asal.
		if err := repo.DeleteEnrollment(ctx, req.FromCourseID, current.UserID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return errNotEnrolled
			}
			return err
		}
		from, err := repo.DecrementEnrolled(ctx, req.FromCourseID)
		if err != nil {
			return err
		}

		// Masuk ke kelas tujuan. Bila penuh, SELURUH transaction batal dan
		// mahasiswa tetap di kelas asal.
		to, err = repo.IncrementEnrolled(ctx, req.ToCourseID)
		if err != nil {
			return err
		}
		if err := repo.InsertEnrollment(ctx, req.ToCourseID, current.UserID); err != nil {
			return err
		}

		// Satu event "pindah", bukan dua event "keluar" dan "masuk":
		// mahasiswa melakukan satu tindakan, maka ia menerima satu notifikasi.
		return s.outbox.WithTx(tx).Insert(ctx, model.EventEnrollmentSwapped, model.EnrollmentEvent{
			UserID:         current.UserID,
			CourseID:       to.ID,
			CourseCode:     to.Code,
			CourseName:     to.Name,
			FromCourseCode: from.Code,
			FromCourseName: from.Name,
		})
	})
	if err != nil {
		return translateCourseError(err)
	}
	return helper.Success(c, fiber.StatusOK, "berhasil pindah kelas", to)
}

// ---------- PATCH /courses/:id ----------
//
// Mengubah kuota dan mencatat event course.quota_changed ke outbox dalam satu
// transaction. Daftar peserta diambil sebagai snapshot di transaction yang
// sama dan disalin ke event (Modul 11, Tugas Mandiri D.2).

// UpdateQuota godoc
//
// @Summary Mengubah kuota kelas (admin)
// @Description Optimistic locking: kirim version yang terakhir dilihat. Membutuhkan permission course:manage.
// @Tags courses
// @Accept json
// @Produce json
// @Param id path int true "ID kelas"
// @Param body body model.UpdateQuotaRequest true "Kuota baru dan version"
// @Success 200 {object} model.WebResponse{data=model.Course}
// @Failure 403 {object} model.ErrorResponse
// @Failure 409 {object} model.ErrorResponse "VERSION_CONFLICT atau QUOTA_BELOW_ENROLLED"
// @Security BearerAuth
// @Router /courses/{id} [patch]
func (s *CourseService) UpdateQuota(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	courseID, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id kelas harus angka positif")
	}
	var req model.UpdateQuotaRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	var course model.Course
	err := s.tx.WithTx(ctx, func(tx repository.DBTX) error {
		repo := s.courses.WithTx(tx)

		var err error
		course, err = repo.UpdateQuota(ctx, courseID, req.Quota, req.Version)
		if err != nil {
			return err
		}
		participants, err := repo.ListParticipants(ctx, courseID)
		if err != nil {
			return err
		}
		return s.outbox.WithTx(tx).Insert(ctx, model.EventCourseQuotaChanged, model.EnrollmentEvent{
			CourseID:     course.ID,
			CourseCode:   course.Code,
			CourseName:   course.Name,
			NewQuota:     course.Quota,
			Participants: participants,
		})
	})
	if err != nil {
		return translateCourseError(err)
	}
	return helper.Success(c, fiber.StatusOK, "kuota kelas berhasil diperbarui", course)
}
