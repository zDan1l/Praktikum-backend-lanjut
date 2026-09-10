package service

import (
	"errors"

	"pemrograman-code/app/model"
	"pemrograman-code/app/repository"
	"pemrograman-code/helper"

	"github.com/gofiber/fiber/v2"
)

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
	q := helper.ParseListQuery(c)
	prestasi, total, err := r.repo.FindAll(ctx, q)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal mengambil data prestasi")
	}
	totalPages := 0
	if q.Limit > 0 {
		totalPages = (total + q.Limit - 1) / q.Limit
	}
	return helper.OkList(c, "daftar prestasi berhasil diambil", prestasi, &model.Meta{
		Page: q.Page, Limit: q.Limit, Total: total, TotalPages: totalPages,
	})
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
	return helper.Ok(c, "prestasi ditemukan", prestasi)
}

