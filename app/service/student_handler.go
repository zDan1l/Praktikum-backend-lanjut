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

// List godoc
//
// @Summary Daftar student dengan cursor pagination
// @Description Kirim meta.next_cursor sebagai parameter cursor untuk halaman berikutnya. Header Accept: text/csv menghasilkan CSV.
// @Tags students
// @Produce json
// @Produce text/csv
// @Param limit query int false "Jumlah per halaman, 1 sampai 100" default(10)
// @Param search query string false "Cari berdasarkan nama atau NIM"
// @Param is_active query bool false "Saring berdasarkan status aktif"
// @Param cursor query string false "Cursor dari meta.next_cursor"
// @Success 200 {object} model.WebCursorResponse{data=[]model.Student}
// @Failure 400 {object} model.ErrorResponse "Cursor atau limit tidak valid"
// @Failure 401 {object} model.ErrorResponse
// @Failure 406 {object} model.ErrorResponse "Format pada header Accept tidak didukung"
// @Security BearerAuth
// @Router /students [get]
func (h *StudentHandler) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	// Format dipilih SEBELUM query. Format tak dikenal tidak membebani DB.
	format, err := helper.Negotiate(c, helper.FormatJSON, helper.FormatCSV)
	if err != nil {
		return err
	}

	q, err := helper.ParseCursorQuery(c)
	if err != nil {
		return err
	}

	rows, err := h.repo.FindAfterCursor(ctx, q)
	if err != nil {
		return helper.Internal(err)
	}

	if format == helper.FormatCSV {
		return helper.WriteStudentsCSV(c, rows)
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

// Get godoc
//
// @Summary Detail satu student
// @Tags students
// @Produce json
// @Param id path int true "ID student"
// @Success 200 {object} model.WebResponse{data=model.Student}
// @Failure 400 {object} model.ErrorResponse "ID bukan angka positif"
// @Failure 404 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /students/{id} [get]
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

// Create godoc
//
// @Summary Menambah student
// @Description NIM wajib 10 digit angka dan unik.
// @Tags students
// @Accept json
// @Produce json
// @Param body body model.StudentRequest true "Data student"
// @Success 201 {object} model.WebResponse{data=model.Student}
// @Header 201 {string} Location "URL student yang baru dibuat"
// @Failure 400 {object} model.ErrorResponse
// @Failure 409 {object} model.ErrorResponse "NIM sudah dipakai"
// @Failure 422 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /students [post]
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

// Replace godoc
//
// @Summary Mengganti seluruh data student
// @Tags students
// @Accept json
// @Produce json
// @Param id path int true "ID student"
// @Param body body model.StudentRequest true "Data student lengkap"
// @Success 200 {object} model.WebResponse{data=model.Student}
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 409 {object} model.ErrorResponse "NIM sudah dipakai"
// @Failure 422 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /students/{id} [put]
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

// Patch godoc
//
// @Summary Mengubah sebagian data student
// @Description Field yang tidak dikirim tidak diubah.
// @Tags students
// @Accept json
// @Produce json
// @Param id path int true "ID student"
// @Param body body model.PatchStudentRequest true "Field yang diubah"
// @Success 200 {object} model.WebResponse{data=model.Student}
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 409 {object} model.ErrorResponse "NIM sudah dipakai"
// @Failure 422 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /students/{id} [patch]
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

// Delete godoc
//
// @Summary Menghapus student
// @Tags students
// @Param id path int true "ID student"
// @Success 204 "Berhasil dihapus; tanpa body"
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /students/{id} [delete]
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
