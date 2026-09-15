package service

import (
	"errors"
	"strconv"
	"strings"

	"pemrograman-code/app/model"
	"pemrograman-code/app/repository"
	"pemrograman-code/helper"

	"github.com/gofiber/fiber/v2"
)

// Template handler - copy file ini untuk resource baru, ganti model & validasi

type PrestasiHandler struct {
	repo repository.PrestasiRepository
}

func NewPrestasiHandler(repo repository.PrestasiRepository) *PrestasiHandler {
	return &PrestasiHandler{repo: repo}
}

func tampilError(c *fiber.Ctx, err error, pesanUmum string) error {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return helper.Fail(c, fiber.StatusNotFound, "prestasi tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return helper.Fail(c, fiber.StatusConflict, "id sudah dipakai")
	default:
		return helper.Fail(c, fiber.StatusInternalServerError, pesanUmum)
	}
}

func (r *PrestasiHandler) List(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()
	q := helper.ParsePrestasiQuery(c)
	prestasi, total, err := r.repo.FindAll(ctx, q)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal mengambil data prestasi")
	}
	return helper.SuccessList(c, "daftar prestasi berhasil diambil", prestasi, helper.NewMeta(q, total))
}

func (r *PrestasiHandler) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()
	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}
	prestasi, err := r.repo.FindByID(ctx, id)
	if err != nil {
		return tampilError(c, err, "gagal mengambil data prestasi")
	}
	return helper.Success(c, fiber.StatusOK, "prestasi ditemukan", prestasi)
}

func (r *PrestasiHandler) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()
	var req model.CreatePrestasiRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}
	if errs := ValidatePrestasiCreate(req); errs != nil {
		return helper.FailValidation(c, errs)
	}
	baru, err := r.repo.Create(ctx, model.Prestasi{
		ID_Student:   req.IDStudent,
		NamaPrestasi: strings.TrimSpace(req.NamaPrestasi),
		Juara:        req.Juara,
	})
	if err != nil {
		return tampilError(c, err, "gagal menyimpan prestasi")
	}
	return helper.Created(c, "prestasi berhasil dibuat", baru, "/api/v1/prestasi/"+strconv.Itoa(baru.ID))
}

func (r *PrestasiHandler) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()
	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}
	var req model.ReplacePrestasiRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}
	if errs := ValidatePrestasiReplace(req); errs != nil {
		return helper.FailValidation(c, errs)
	}
	hasil, err := r.repo.Update(ctx, model.Prestasi{
		ID:           id,
		ID_Student:   req.IDStudent,
		NamaPrestasi: strings.TrimSpace(req.NamaPrestasi),
		Juara:        req.Juara,
	})
	if err != nil {
		return tampilError(c, err, "gagal memperbarui prestasi")
	}
	return helper.Success(c, fiber.StatusOK, "prestasi berhasil diganti seluruhnya", hasil)
}

func (r *PrestasiHandler) Patch(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()
	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}
	var req model.PatchPrestasiRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}
	if req.IDStudent == nil && req.NamaPrestasi == nil && req.Juara == nil {
		return helper.Fail(c, fiber.StatusBadRequest, "tidak ada field yang diubah")
	}
	saatIni, err := r.repo.FindByID(ctx, id)
	if err != nil {
		return tampilError(c, err, "gagal mengambil data prestasi")
	}
	merged, errs := ApplyPrestasiPatch(saatIni, req)
	if errs != nil {
		return helper.FailValidation(c, errs)
	}
	hasil, err := r.repo.Update(ctx, merged)
	if err != nil {
		return tampilError(c, err, "gagal memperbarui prestasi")
	}
	return helper.Success(c, fiber.StatusOK, "prestasi berhasil diperbarui sebagian", hasil)
}

func (r *PrestasiHandler) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.ReqCtx(c)
	defer cancel()
	id, valid := helper.ParamID(c)
	if !valid {
		return helper.Fail(c, fiber.StatusBadRequest, "id harus berupa angka positif")
	}
	if err := r.repo.Delete(ctx, id); err != nil {
		return tampilError(c, err, "gagal menghapus prestasi")
	}
	return helper.NoContent(c)
}
