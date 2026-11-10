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

type PortfolioHandler struct {
	portfolios *repository.PortfolioRepository
	students   *repository.StudentRepository
}

func NewPortfolioHandler(
	portfolios *repository.PortfolioRepository,
	students *repository.StudentRepository,
) *PortfolioHandler {
	return &PortfolioHandler{portfolios: portfolios, students: students}
}

func translatePortfolioError(err error) error {
	switch {
	case errors.Is(err, repository.ErrPortfolioFull):
		return helper.Conflict("portofolio sudah berisi maksimal 20 proyek")
	case errors.Is(err, repository.ErrProjectNotFound):
		return helper.NotFound("proyek tidak ditemukan")
	case errors.Is(err, repository.ErrInvalidID):
		return helper.BadRequest("projectId harus berupa ObjectID 24 karakter heksadesimal")
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("portofolio belum dibuat")
	default:
		return helper.Internal(err)
	}
}

// pastikanSiswa menolak permintaan bila students.id tidak ada di PostgreSQL.
func (h *PortfolioHandler) pastikanSiswa(c *fiber.Ctx, id int) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	if _, err := h.students.FindByID(ctx, id); err != nil {
		return translateStudentError(err, "student")
	}
	return nil
}

func projectDariRequest(req model.ProjectRequest) (model.Project, bool) {
	technologies := make([]string, 0, len(req.Technologies))
	for _, teknologi := range req.Technologies {
		teknologi = strings.TrimSpace(teknologi)
		if teknologi == "" {
			return model.Project{}, false
		}
		technologies = append(technologies, teknologi)
	}
	return model.Project{
		Title:        strings.TrimSpace(req.Title),
		Technologies: technologies,
		Link:         strings.TrimSpace(req.Link),
		Year:         req.Year,
	}, true
}

func portfolioDariRequest(studentID int, req model.PortfolioRequest) (model.Portfolio, error) {
	portofolio := model.Portfolio{
		StudentID:    studentID,
		Headline:     strings.TrimSpace(req.Headline),
		Bio:          strings.TrimSpace(req.Bio),
		Projects:     make([]model.Project, 0, len(req.Projects)),
		Certificates: make([]model.Certificate, 0, len(req.Certificates)),
	}

	for i, reqProyek := range req.Projects {
		proyek, ok := projectDariRequest(reqProyek)
		if !ok {
			return model.Portfolio{}, helper.Validation(map[string]string{
				"projects[" + strconv.Itoa(i) + "].technologies": "wajib diisi",
			})
		}
		portofolio.Projects = append(portofolio.Projects, proyek)
	}

	for _, reqSertifikat := range req.Certificates {
		portofolio.Certificates = append(portofolio.Certificates, model.Certificate{
			Name:   strings.TrimSpace(reqSertifikat.Name),
			Issuer: strings.TrimSpace(reqSertifikat.Issuer),
			Date:   strings.TrimSpace(reqSertifikat.Date),
		})
	}

	return portofolio, nil
}

// Get godoc
//
// @Summary Portofolio milik seorang student
// @Tags portfolios
// @Produce json
// @Param id path int true "ID student"
// @Success 200 {object} model.WebResponse{data=model.Portfolio}
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse "Student tidak ada atau portofolio belum dibuat"
// @Security BearerAuth
// @Router /students/{id}/portfolio [get]
func (h *PortfolioHandler) Get(c *fiber.Ctx) error {
	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}
	if err := h.pastikanSiswa(c, id); err != nil {
		return err
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	portofolio, err := h.portfolios.FindByStudentID(ctx, id)
	if err != nil {
		return translatePortfolioError(err)
	}
	return helper.Success(c, fiber.StatusOK, "portofolio ditemukan", portofolio)
}

// Replace godoc
//
// @Summary Membuat atau mengganti portofolio student
// @Tags portfolios
// @Accept json
// @Produce json
// @Param id path int true "ID student"
// @Param body body model.PortfolioRequest true "Isi portofolio"
// @Success 200 {object} model.WebResponse{data=model.Portfolio}
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 422 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /students/{id}/portfolio [put]
func (h *PortfolioHandler) Replace(c *fiber.Ctx) error {
	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}
	if err := h.pastikanSiswa(c, id); err != nil {
		return err
	}

	var req model.PortfolioRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	portofolio, err := portfolioDariRequest(id, req)
	if err != nil {
		return err
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	hasil, err := h.portfolios.Upsert(ctx, portofolio)
	if err != nil {
		return translatePortfolioError(err)
	}
	return helper.Success(c, fiber.StatusOK, "portofolio berhasil disimpan", hasil)
}

// AddProject godoc
//
// @Summary Menambah satu proyek ke portofolio
// @Description Satu portofolio paling banyak berisi 20 proyek.
// @Tags portfolios
// @Accept json
// @Produce json
// @Param id path int true "ID student"
// @Param body body model.ProjectRequest true "Data proyek"
// @Success 201 {object} model.WebResponse{data=model.Portfolio}
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 409 {object} model.ErrorResponse "Portofolio sudah berisi 20 proyek"
// @Failure 422 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /students/{id}/portfolio/projects [post]
func (h *PortfolioHandler) AddProject(c *fiber.Ctx) error {
	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}
	if err := h.pastikanSiswa(c, id); err != nil {
		return err
	}

	var req model.ProjectRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	proyek, ok := projectDariRequest(req)
	if !ok {
		return helper.Validation(map[string]string{"technologies": "wajib diisi"})
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	hasil, err := h.portfolios.AddProject(ctx, id, proyek)
	if err != nil {
		return translatePortfolioError(err)
	}
	return helper.Created(c, "proyek berhasil ditambahkan", hasil,
		"/api/v1/students/"+strconv.Itoa(id)+"/portfolio")
}

// DeleteProject godoc
//
// @Summary Menghapus satu proyek dari portofolio
// @Tags portfolios
// @Param id path int true "ID student"
// @Param projectId path string true "ObjectID proyek (24 karakter heksadesimal)"
// @Success 204 "Berhasil dihapus; tanpa body"
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /students/{id}/portfolio/projects/{projectId} [delete]
func (h *PortfolioHandler) DeleteProject(c *fiber.Ctx) error {
	id, valid := helper.ParamID(c)
	if !valid {
		return helper.BadRequest("id harus berupa angka positif")
	}
	projectID := strings.TrimSpace(c.Params("projectId"))
	if projectID == "" {
		return helper.BadRequest("projectId wajib diisi")
	}

	if err := h.pastikanSiswa(c, id); err != nil {
		return err
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	if err := h.portfolios.DeleteProject(ctx, id, projectID); err != nil {
		return translatePortfolioError(err)
	}
	return helper.NoContent(c)
}

// TopTechnologies godoc
//
// @Summary Sepuluh teknologi yang paling banyak dipakai di seluruh portofolio
// @Tags portfolios
// @Produce json
// @Success 200 {object} model.WebResponse{data=[]model.TechnologyStat}
// @Failure 401 {object} model.ErrorResponse
// @Security BearerAuth
// @Router /portfolios/stats/technologies [get]
func (h *PortfolioHandler) TopTechnologies(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	stats, err := h.portfolios.TopTechnologies(ctx, 10)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK,
		"10 teknologi paling banyak dipakai", stats)
}
