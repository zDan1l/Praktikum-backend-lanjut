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

// ApplyPatch menggabungkan saja. Bentuk sudah diperiksa tag sebelum
// fungsi ini dipanggil.
func ApplyPatch(current model.Student, req model.PatchStudentRequest) model.Student {
	if req.NIM != nil {
		current.NIM = strings.TrimSpace(*req.NIM)
	}
	if req.Name != nil {
		current.Name = strings.TrimSpace(*req.Name)
	}
	if req.Grade != nil {
		current.Grade = *req.Grade
	}
	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}
	return current
}

func IsEmptyPatch(req model.PatchStudentRequest) bool {
	return req.NIM == nil && req.Name == nil && req.Grade == nil && req.IsActive == nil
}

// translateError menerjemahkan error repository menjadi AppError.
// Tidak kenal fiber.Ctx. Yang tidak dikenali menjadi Internal (fail closed).
func translateStudentError(err error, entity string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound(entity + " tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Conflict("nim sudah dipakai")
	default:
		return helper.Internal(err)
	}
}

func (h *StudentHandler) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	q, err := helper.ParseCursorQuery(c)
	if err != nil {
		return err
	}

	rows, err := h.repo.FindAfterCursor(ctx, q)
	if err != nil {
		return helper.Internal(err)
	}

	hasMore := len(rows) > q.Limit
	if hasMore {
		rows = rows[:q.Limit]
	}

	meta := &model.CursorMeta{Limit: q.Limit, HasMore: hasMore}
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		meta.NextCursor = helper.EncodeCursor(last.CreatedAt, last.ID)
	}

	return helper.SuccessCursor(c, "daftar student berhasil diambil", rows, meta)
}

func (h *StudentHandler) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}
	student, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return translateStudentError(err, "student")
	}
	return helper.Success(c, fiber.StatusOK, "student ditemukan", student)
}

func (h *StudentHandler) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	var req model.StudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}
	baru, err := h.repo.Create(ctx, model.Student{
		NIM:      strings.TrimSpace(req.NIM),
		Name:     strings.TrimSpace(req.Name),
		Grade:    req.Grade,
		IsActive: req.IsActive,
	})
	if err != nil {
		return translateStudentError(err, "student")
	}
	return helper.Created(c, "student berhasil dibuat", baru, "/api/v1/students/"+strconv.Itoa(baru.ID))
}

func (h *StudentHandler) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}
	var req model.StudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}
	hasil, err := h.repo.Update(ctx, model.Student{
		ID: id, NIM: strings.TrimSpace(req.NIM), Name: strings.TrimSpace(req.Name),
		Grade: req.Grade, IsActive: req.IsActive,
	})
	if err != nil {
		return translateStudentError(err, "student")
	}
	return helper.Success(c, fiber.StatusOK, "student berhasil diganti seluruhnya", hasil)
}

func (h *StudentHandler) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}
	var req model.PatchStudentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}
	if IsEmptyPatch(req) {
		return helper.BadRequest("tidak ada field yang diubah")
	}
	saatIni, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return translateStudentError(err, "student")
	}
	merged := ApplyPatch(saatIni, req)
	hasil, err := h.repo.Update(ctx, merged)
	if err != nil {
		return translateStudentError(err, "student")
	}
	return helper.Success(c, fiber.StatusOK, "student berhasil diperbarui sebagian", hasil)
}

func (h *StudentHandler) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}
	if err := h.repo.Delete(ctx, id); err != nil {
		return translateStudentError(err, "student")
	}
	return helper.NoContent(c)
}
