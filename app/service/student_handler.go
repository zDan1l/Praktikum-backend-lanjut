package service

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"pemrograman-code/app/model"
	"pemrograman-code/app/repository"
	"pemrograman-code/helper"
)

type StudentHandler struct {
	repo *repository.StudentRepository
}

func NewStudentHandler(repo *repository.StudentRepository) *StudentHandler {
	return &StudentHandler{repo: repo}
}

// ===== validasi (murni tanpa Fiber, gampang di-test) =====

func ValidateStudent(req model.StudentRequest) map[string]string {
	errs := map[string]string{}
	if strings.TrimSpace(req.NIM) == "" {
		errs["nim"] = "wajib diisi"
	}
	if strings.TrimSpace(req.Name) == "" {
		errs["name"] = "wajib diisi"
	}
	if req.Grade < 0 || req.Grade > 100 {
		errs["grade"] = "harus 0 - 100"
	}
	if len(errs) == 0 {
		return nil
	}
	return errs
}

// ApplyPatch menerapkan perubahan PATCH pada student lama (merge),
// lalu memvalidasi hasilnya.
func ApplyPatch(current model.Student, req model.PatchStudentRequest) (model.Student, map[string]string) {
	if req.NIM != nil {
		if strings.TrimSpace(*req.NIM) == "" {
			return current, map[string]string{"nim": "tidak boleh kosong"}
		}
		current.NIM = strings.TrimSpace(*req.NIM)
	}
	if req.Name != nil {
		if strings.TrimSpace(*req.Name) == "" {
			return current, map[string]string{"name": "tidak boleh kosong"}
		}
		current.Name = strings.TrimSpace(*req.Name)
	}
	if req.Grade != nil {
		if *req.Grade < 0 || *req.Grade > 100 {
			return current, map[string]string{"grade": "harus 0 - 100"}
		}
		current.Grade = *req.Grade
	}
	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}
	return current, nil
}

func IsEmptyPatch(req model.PatchStudentRequest) bool {
	return req.NIM == nil && req.Name == nil && req.Grade == nil && req.IsActive == nil
}

// ===== handler =====

// failFromError menerjemahkan error repository jadi response HTTP.
func failFromError(c *fiber.Ctx, err error, pesanUmum string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.Fail(c, fiber.StatusNotFound, "student tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Fail(c, fiber.StatusConflict, "nim sudah dipakai")
	default:
		return helper.Fail(c, fiber.StatusInternalServerError, pesanUmum)
	}
}

func (h *StudentHandler) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	q := helper.ParseQuery(c)
	students, total, err := h.repo.FindAll(ctx, q)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal mengambil data student")
	}
	return helper.SuccessList(c, "daftar student berhasil diambil", students, helper.NewMeta(q, total))
}

func (h *StudentHandler) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}
	student, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return failFromError(c, err, "gagal mengambil data student")
	}
	return helper.Success(c, fiber.StatusOK, "student ditemukan", student)
}

func (h *StudentHandler) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.StudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}
	if errs := ValidateStudent(req); errs != nil {
		return helper.FailValidation(c, errs)
	}
	baru, err := h.repo.Create(ctx, model.Student{
		NIM:      strings.TrimSpace(req.NIM),
		Name:     strings.TrimSpace(req.Name),
		Grade:    req.Grade,
		IsActive: req.IsActive,
	})
	if err != nil {
		return failFromError(c, err, "gagal menyimpan student")
	}
	return helper.Created(c, "student berhasil dibuat", baru, "/api/v1/students/"+strconv.Itoa(baru.ID))
}

func (h *StudentHandler) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}
	var req model.StudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}
	if errs := ValidateStudent(req); errs != nil {
		return helper.FailValidation(c, errs)
	}
	hasil, err := h.repo.Update(ctx, model.Student{
		ID: id, NIM: strings.TrimSpace(req.NIM), Name: strings.TrimSpace(req.Name),
		Grade: req.Grade, IsActive: req.IsActive,
	})
	if err != nil {
		return failFromError(c, err, "gagal memperbarui student")
	}
	return helper.Success(c, fiber.StatusOK, "student berhasil diganti seluruhnya", hasil)
}

func (h *StudentHandler) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}
	var req model.PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}
	if IsEmptyPatch(req) {
		return helper.Fail(c, fiber.StatusBadRequest, "tidak ada field yang diubah")
	}
	saatIni, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return failFromError(c, err, "gagal mengambil data student")
	}
	merged, errs := ApplyPatch(saatIni, req)
	if errs != nil {
		return helper.FailValidation(c, errs)
	}
	hasil, err := h.repo.Update(ctx, merged)
	if err != nil {
		return failFromError(c, err, "gagal memperbarui student")
	}
	return helper.Success(c, fiber.StatusOK, "student berhasil diperbarui sebagian", hasil)
}

func (h *StudentHandler) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}
	if err := h.repo.Delete(ctx, id); err != nil {
		return failFromError(c, err, "gagal menghapus student")
	}
	return helper.NoContent(c)
}
