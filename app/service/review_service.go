package service

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"pemrograman-code/app/model"
	"pemrograman-code/app/repository"
	"pemrograman-code/helper"
)

// ReviewService melayani ulasan kelas. Ulasan disimpan di MongoDB, sedangkan
// kelas dan keikutsertaan mahasiswa berada di PostgreSQL. Keduanya diakses
// lewat INTERFACE repository, sehingga unit test dapat mengganti keduanya
// dengan fake tanpa mengubah satu baris pun kode di file ini.
type ReviewService struct {
	reviews repository.ReviewRepository
	courses repository.CourseRepository
	perms   *helper.PermissionSet
}

func NewReviewService(reviews repository.ReviewRepository, courses repository.CourseRepository,
	perms *helper.PermissionSet) *ReviewService {
	return &ReviewService{reviews: reviews, courses: courses, perms: perms}
}

const (
	defaultReviewLimit = 10
	maxReviewLimit     = 50
)

// translateReviewError memutuskan status code setiap kegagalan dari
// repository ulasan. errors.Is dipakai agar error yang sudah dibungkus
// (fmt.Errorf("...: %w", err)) tetap dikenali.
func translateReviewError(err error) error {
	switch {
	case errors.Is(err, repository.ErrInvalidID):
		return helper.BadRequest("id tidak valid")
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("ulasan atau kelas tidak ditemukan")
	case errors.Is(err, repository.ErrDuplicate):
		return conflict("ALREADY_REVIEWED", "Anda sudah menulis ulasan untuk kelas ini")
	case errors.Is(err, repository.ErrNotOwner):
		return helper.Forbidden("hanya penulis ulasan yang boleh melakukan ini")
	case errors.Is(err, repository.ErrOwnReview):
		return conflict("OWN_REVIEW", "tidak dapat memberi suara pada ulasan sendiri")
	case errors.Is(err, repository.ErrAlreadyVoted):
		return conflict("ALREADY_VOTED", "Anda sudah memberi suara pada ulasan ini")
	case errors.Is(err, repository.ErrReplyLimit):
		return conflict("REPLY_LIMIT", "jumlah balasan sudah mencapai batas")
	default:
		return helper.Internal(err)
	}
}

// ---------- POST /courses/:id/reviews ----------

// Create godoc
//
// @Summary Menulis ulasan untuk sebuah kelas
// @Description Hanya peserta kelas. Satu mahasiswa hanya boleh menulis satu ulasan per kelas.
// @Tags reviews
// @Accept json
// @Produce json
// @Param id path int true "ID kelas"
// @Param body body model.CreateReviewRequest true "Isi ulasan"
// @Success 201 {object} model.WebResponse{data=model.Review}
// @Failure 403 {object} model.ErrorResponse "NOT_ENROLLED"
// @Failure 409 {object} model.ErrorResponse "ALREADY_REVIEWED"
// @Failure 422 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /courses/{id}/reviews [post]
func (s *ReviewService) Create(c *fiber.Ctx) error {
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

	var req model.CreateReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	req.Comment = strings.TrimSpace(req.Comment)
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	if _, err := s.courses.FindByID(ctx, courseID); err != nil {
		return translateReviewError(err)
	}
	enrolled, err := s.courses.IsEnrolled(ctx, courseID, current.UserID)
	if err != nil {
		return helper.Internal(err)
	}
	if !enrolled {
		return &helper.AppError{Status: fiber.StatusForbidden, Code: "NOT_ENROLLED",
			Message: "hanya peserta kelas yang boleh menulis ulasan"}
	}

	review, err := s.reviews.Create(ctx, model.Review{
		CourseID: courseID, UserID: current.UserID, Username: current.Username,
		Rating: req.Rating, Comment: req.Comment, Tags: NormalizeTags(req.Tags),
	})
	if err != nil {
		return translateReviewError(err)
	}
	return helper.Created(c, "ulasan berhasil dibuat", review, "/api/v1/reviews/"+review.ID)
}

// ---------- GET /courses/:id/reviews ----------

// ListByCourse godoc
//
// @Summary Daftar ulasan sebuah kelas, terbaru lebih dulu
// @Description Cursor pagination: kirim meta.next_before sebagai parameter before untuk halaman berikutnya.
// @Tags reviews
// @Produce json
// @Param id path int true "ID kelas"
// @Param tag query string false "Saring berdasarkan tag"
// @Param before query string false "Cursor: id ulasan terakhir yang sudah dilihat"
// @Param limit query int false "Jumlah per halaman, 1 sampai 50" default(10)
// @Success 200 {object} model.ReviewPage
// @Failure 400 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /courses/{id}/reviews [get]
func (s *ReviewService) ListByCourse(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	courseID, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id kelas harus angka positif")
	}
	want := c.QueryInt("limit", defaultReviewLimit)
	if want < 1 || want > maxReviewLimit {
		return helper.BadRequest("limit harus antara 1 dan 50")
	}
	tag := strings.ToLower(strings.TrimSpace(c.Query("tag")))
	before := strings.TrimSpace(c.Query("before"))

	all, err := s.reviews.FindByCourse(ctx, courseID)
	if err != nil {
		return translateReviewError(err)
	}

	// all sudah terurut terbaru lebih dulu. Cursor "before" adalah id ulasan
	// terakhir yang sudah dilihat: halaman berikutnya dimulai SETELAH id itu.
	reviews := []model.Review{}
	passedCursor := before == ""
	hasMore := false
	for _, r := range all {
		if !passedCursor {
			passedCursor = r.ID == before
			continue
		}
		if tag != "" && !containsTag(r.Tags, tag) {
			continue
		}
		if len(reviews) == want {
			hasMore = true
			break
		}
		reviews = append(reviews, r)
	}
	if !passedCursor {
		return helper.BadRequest("cursor before tidak dikenal")
	}

	next := ""
	if hasMore {
		next = reviews[len(reviews)-1].ID
	}
	return c.JSON(model.ReviewPage{
		Success: true,
		Message: "daftar ulasan berhasil diambil",
		Data:    reviews,
		Meta:    model.ReviewPageMeta{Limit: want, HasMore: hasMore, NextBefore: next},
	})
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

// ---------- GET /courses/:id/reviews/summary ----------

// Summary godoc
//
// @Summary Ringkasan ulasan sebuah kelas
// @Description Jumlah ulasan, rata-rata rating, sebaran rating (indeks 0 = bintang 1), dan lima tag terbanyak.
// @Tags reviews
// @Produce json
// @Param id path int true "ID kelas"
// @Success 200 {object} model.WebResponse{data=model.ReviewSummary}
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse "Kelas tidak ditemukan"
// @Security BearerAuth
// @Router /courses/{id}/reviews/summary [get]
func (s *ReviewService) Summary(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	courseID, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id kelas harus angka positif")
	}
	if _, err := s.courses.FindByID(ctx, courseID); err != nil {
		return translateReviewError(err)
	}
	summary, err := s.reviews.Summary(ctx, courseID)
	if err != nil {
		return translateReviewError(err)
	}
	return helper.Success(c, fiber.StatusOK, "ringkasan ulasan berhasil diambil", summary)
}

// ---------- GET /reviews/:id ----------

// Get godoc
//
// @Summary Detail satu ulasan beserta balasannya
// @Tags reviews
// @Produce json
// @Param id path string true "ID ulasan (ObjectID)"
// @Success 200 {object} model.WebResponse{data=model.Review}
// @Failure 400 {object} model.ErrorResponse "ID bukan ObjectID yang valid"
// @Failure 404 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /reviews/{id} [get]
func (s *ReviewService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	review, err := s.reviews.FindByID(ctx, c.Params("id"))
	if err != nil {
		return translateReviewError(err)
	}
	return helper.Success(c, fiber.StatusOK, "ulasan berhasil diambil", review)
}

// ---------- PATCH /reviews/:id ----------

// Update godoc
//
// @Summary Mengubah ulasan milik sendiri
// @Description Field yang tidak dikirim tidak diubah. Hanya penulis ulasan yang boleh mengubah.
// @Tags reviews
// @Accept json
// @Produce json
// @Param id path string true "ID ulasan (ObjectID)"
// @Param body body model.UpdateReviewRequest true "Field yang diubah"
// @Success 200 {object} model.WebResponse{data=model.Review}
// @Failure 400 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse "Bukan penulis ulasan"
// @Failure 404 {object} model.ErrorResponse
// @Failure 422 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /reviews/{id} [patch]
func (s *ReviewService) Update(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}
	var req model.UpdateReviewRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if req.Comment != nil {
		trimmed := strings.TrimSpace(*req.Comment)
		req.Comment = &trimmed
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}
	if req.Rating == nil && req.Comment == nil && req.Tags == nil {
		return helper.Validation(map[string]string{"_": "minimal satu field harus diisi"})
	}

	var tags []string
	if req.Tags != nil {
		tags = NormalizeTags(*req.Tags)
	}
	review, err := s.reviews.Update(ctx, c.Params("id"), current.UserID, req, tags)
	if err != nil {
		return translateReviewError(err)
	}
	return helper.Success(c, fiber.StatusOK, "ulasan berhasil diperbarui", review)
}

// ---------- DELETE /reviews/:id ----------
//
// Penulis boleh menghapus ulasannya sendiri. Role dengan permission
// review:moderate boleh menghapus ulasan siapa pun.

// Delete godoc
//
// @Summary Menghapus ulasan
// @Description Penulis boleh menghapus ulasannya sendiri. Role dengan permission review:moderate boleh menghapus ulasan siapa pun.
// @Tags reviews
// @Param id path string true "ID ulasan (ObjectID)"
// @Success 204 "Berhasil dihapus; tanpa body"
// @Failure 400 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse "Bukan penulis dan bukan moderator"
// @Failure 404 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /reviews/{id} [delete]
func (s *ReviewService) Delete(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}
	asModerator := s.perms.Has(current.Role, "review:moderate")
	if err := s.reviews.Delete(ctx, c.Params("id"), current.UserID, asModerator); err != nil {
		return translateReviewError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ---------- POST /reviews/:id/helpful ----------

// Vote godoc
//
// @Summary Menandai ulasan sebagai membantu
// @Description Satu suara per mahasiswa per ulasan. Kode error 409: OWN_REVIEW, ALREADY_VOTED.
// @Tags reviews
// @Produce json
// @Param id path string true "ID ulasan (ObjectID)"
// @Success 200 {object} model.WebResponse{data=model.Review}
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 409 {object} model.ErrorResponse "OWN_REVIEW atau ALREADY_VOTED"
// @Security BearerAuth
// @Router /reviews/{id}/helpful [post]
func (s *ReviewService) Vote(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}
	review, err := s.reviews.Vote(ctx, c.Params("id"), current.UserID)
	if err != nil {
		return translateReviewError(err)
	}
	return helper.Success(c, fiber.StatusOK, "suara berhasil dicatat", review)
}

// ---------- POST /reviews/:id/replies ----------

// Reply godoc
//
// @Summary Membalas sebuah ulasan
// @Description Satu ulasan paling banyak memiliki 20 balasan.
// @Tags reviews
// @Accept json
// @Produce json
// @Param id path string true "ID ulasan (ObjectID)"
// @Param body body model.CreateReplyRequest true "Isi balasan"
// @Success 201 {object} model.WebResponse{data=model.Review}
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 409 {object} model.ErrorResponse "REPLY_LIMIT"
// @Failure 422 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /reviews/{id}/replies [post]
func (s *ReviewService) Reply(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	current, ok := helper.CurrentUser(c)
	if !ok {
		return helper.Unauthorized("belum terautentikasi")
	}
	var req model.CreateReplyRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	req.Body = strings.TrimSpace(req.Body)
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}
	review, err := s.reviews.AddReply(ctx, c.Params("id"), model.Reply{
		UserID: current.UserID, Username: current.Username, Body: req.Body,
	})
	if err != nil {
		return translateReviewError(err)
	}
	return helper.Success(c, fiber.StatusCreated, "balasan berhasil ditambahkan", review)
}
