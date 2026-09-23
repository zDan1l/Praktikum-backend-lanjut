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

func (h *CommentHandler) List(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	articleID, err := strconv.Atoi(c.Params("articleId"))
	if err != nil || articleID < 1 {
		return helper.Fail(c, fiber.StatusBadRequest, "articleId harus berupa angka positif")
	}
	ada, err := h.repo.ArticleExists(ctx, articleID)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal mengecek artikel")
	}
	if !ada {
		return helper.Fail(c, fiber.StatusNotFound, "artikel tidak ditemukan")
	}

	comments, err := h.repo.ListByArticle(ctx, articleID)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal mengambil komentar")
	}
	return helper.Success(c, fiber.StatusOK, "daftar komentar berhasil diambil", comments)
}

func (h *CommentHandler) Create(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	articleID, err := strconv.Atoi(c.Params("articleId"))
	if err != nil || articleID < 1 {
		return helper.Fail(c, fiber.StatusBadRequest, "articleId harus berupa angka positif")
	}

	var req model.CommentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		return helper.FailValidation(c, map[string]string{"content": "wajib diisi"})
	}

	ada, err := h.repo.ArticleExists(ctx, articleID)
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal mengecek artikel")
	}
	if !ada {
		return helper.Fail(c, fiber.StatusNotFound, "artikel tidak ditemukan")
	}

	user, _ := helper.CurrentUser(c)
	comment, err := h.repo.Create(ctx, model.Comment{
		ArticleID: articleID,
		AuthorID:  user.UserID,
		Content:   req.Content,
	})
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal menyimpan komentar")
	}
	return helper.Created(c, "komentar berhasil dibuat", comment,
		"/api/v1/articles/"+strconv.Itoa(articleID)+"/comments/"+strconv.Itoa(comment.ID))
}

func (h *CommentHandler) Update(c *fiber.Ctx) error {
	_, id, comment, ok := h.ambilKomentar(c)
	if !ok {
		return helper.Fail(c, fiber.StatusNotFound, "komentar tidak ditemukan")
	}

	user, _ := helper.CurrentUser(c)
	if !h.bolehUbah(comment, user) {
		return helper.Fail(c, fiber.StatusForbidden, "akses ditolak: bukan pemilik komentar ini")
	}

	var req model.CommentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.Fail(c, fiber.StatusBadRequest, "body harus berupa JSON yang valid")
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		return helper.FailValidation(c, map[string]string{"content": "wajib diisi"})
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	// Update hanya boleh mengubah field content.
	hasil, err := h.repo.UpdateContent(ctx, id, req.Content)
	if errors.Is(err, repository.ErrNotFound) {
		return helper.Fail(c, fiber.StatusNotFound, "komentar tidak ditemukan")
	}
	if err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal memperbarui komentar")
	}
	return helper.Success(c, fiber.StatusOK, "komentar berhasil diperbarui", hasil)
}

func (h *CommentHandler) Delete(c *fiber.Ctx) error {
	_, id, comment, ok := h.ambilKomentar(c)
	if !ok {
		return helper.Fail(c, fiber.StatusNotFound, "komentar tidak ditemukan")
	}

	user, _ := helper.CurrentUser(c)
	if !h.bolehUbah(comment, user) {
		return helper.Fail(c, fiber.StatusForbidden, "akses ditolak: bukan pemilik komentar ini")
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	if err := h.repo.Delete(ctx, id); err != nil {
		return helper.Fail(c, fiber.StatusInternalServerError, "gagal menghapus komentar")
	}
	return helper.NoContent(c)
}
