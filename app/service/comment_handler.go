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

type CommentHandler struct {
	repo *repository.CommentRepository
}

func NewCommentHandler(repo *repository.CommentRepository) *CommentHandler {
	return &CommentHandler{repo: repo}
}

// Cek ownership ada di service layer (bukan middleware) karena keputusannya
// bergantung pada isi data: author_id komentar harus dibaca dulu dari DB.
func (h *CommentHandler) bolehUbah(comment model.Comment, user model.AuthUser) bool {
	return user.Role == "admin" || comment.AuthorID == user.UserID
}

func (h *CommentHandler) ambilKomentar(c *fiber.Ctx) (int, int, model.Comment, bool) {
	articleID, err := strconv.Atoi(c.Params("articleId"))
	if err != nil || articleID < 1 {
		return 0, 0, model.Comment{}, false
	}
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id < 1 {
		return 0, 0, model.Comment{}, false
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	comment, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return 0, 0, model.Comment{}, false
	}
	// Komentar harus berada di artikel yang dimaksud di URL.
	if comment.ArticleID != articleID {
		return 0, 0, model.Comment{}, false
	}
	return articleID, id, comment, true
}

// List godoc
//
// @Summary Daftar komentar sebuah artikel
// @Description Membutuhkan permission comment:read.
// @Tags comments
// @Produce json
// @Param articleId path int true "ID artikel"
// @Success 200 {object} model.WebResponse{data=[]model.Comment}
// @Failure 400 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse "Artikel tidak ditemukan"
// @Security BearerAuth
// @Router /articles/{articleId}/comments [get]
func (h *CommentHandler) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	articleID, err := strconv.Atoi(c.Params("articleId"))
	if err != nil || articleID < 1 {
		return helper.BadRequest("articleId harus berupa angka positif")
	}
	ada, err := h.repo.ArticleExists(ctx, articleID)
	if err != nil {
		return helper.Internal(err)
	}
	if !ada {
		return helper.NotFound("artikel tidak ditemukan")
	}

	comments, err := h.repo.ListByArticle(ctx, articleID)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "daftar komentar berhasil diambil", comments)
}

// Create godoc
//
// @Summary Menulis komentar pada sebuah artikel
// @Description Membutuhkan permission comment:create.
// @Tags comments
// @Accept json
// @Produce json
// @Param articleId path int true "ID artikel"
// @Param body body model.CommentRequest true "Isi komentar"
// @Success 201 {object} model.WebResponse{data=model.Comment}
// @Failure 400 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse "Artikel tidak ditemukan"
// @Failure 422 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /articles/{articleId}/comments [post]
func (h *CommentHandler) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	articleID, err := strconv.Atoi(c.Params("articleId"))
	if err != nil || articleID < 1 {
		return helper.BadRequest("articleId harus berupa angka positif")
	}

	var req model.CommentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		return helper.Validation(map[string]string{"content": "wajib diisi"})
	}

	ada, err := h.repo.ArticleExists(ctx, articleID)
	if err != nil {
		return helper.Internal(err)
	}
	if !ada {
		return helper.NotFound("artikel tidak ditemukan")
	}

	user, _ := helper.CurrentUser(c)
	comment, err := h.repo.Create(ctx, model.Comment{
		ArticleID: articleID,
		AuthorID:  user.UserID,
		Content:   req.Content,
	})
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Created(c, "komentar berhasil dibuat", comment,
		"/api/v1/articles/"+strconv.Itoa(articleID)+"/comments/"+strconv.Itoa(comment.ID))
}

// Update godoc
//
// @Summary Mengubah komentar
// @Description Membutuhkan permission comment:update. Selain admin, hanya pemilik komentar yang boleh mengubah.
// @Tags comments
// @Accept json
// @Produce json
// @Param articleId path int true "ID artikel"
// @Param id path int true "ID komentar"
// @Param body body model.CommentRequest true "Isi komentar baru"
// @Success 200 {object} model.WebResponse{data=model.Comment}
// @Failure 400 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse "Tidak punya permission atau bukan pemilik"
// @Failure 404 {object} model.ErrorResponse
// @Failure 422 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /articles/{articleId}/comments/{id} [put]
func (h *CommentHandler) Update(c *fiber.Ctx) error {
	_, id, comment, ok := h.ambilKomentar(c)
	if !ok {
		return helper.NotFound("komentar tidak ditemukan")
	}

	user, _ := helper.CurrentUser(c)
	if !h.bolehUbah(comment, user) {
		return helper.Forbidden("akses ditolak: bukan pemilik komentar ini")
	}

	var req model.CommentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		return helper.Validation(map[string]string{"content": "wajib diisi"})
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	// Update hanya boleh mengubah field content.
	hasil, err := h.repo.UpdateContent(ctx, id, req.Content)
	if errors.Is(err, repository.ErrNotFound) {
		return helper.NotFound("komentar tidak ditemukan")
	}
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "komentar berhasil diperbarui", hasil)
}

// Delete godoc
//
// @Summary Menghapus komentar
// @Description Membutuhkan permission comment:delete. Selain admin, hanya pemilik komentar yang boleh menghapus.
// @Tags comments
// @Param articleId path int true "ID artikel"
// @Param id path int true "ID komentar"
// @Success 204 "Berhasil dihapus; tanpa body"
// @Failure 403 {object} model.ErrorResponse "Tidak punya permission atau bukan pemilik"
// @Failure 404 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /articles/{articleId}/comments/{id} [delete]
func (h *CommentHandler) Delete(c *fiber.Ctx) error {
	_, id, comment, ok := h.ambilKomentar(c)
	if !ok {
		return helper.NotFound("komentar tidak ditemukan")
	}

	user, _ := helper.CurrentUser(c)
	if !h.bolehUbah(comment, user) {
		return helper.Forbidden("akses ditolak: bukan pemilik komentar ini")
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	if err := h.repo.Delete(ctx, id); err != nil {
		return helper.Internal(err)
	}
	return helper.NoContent(c)
}
