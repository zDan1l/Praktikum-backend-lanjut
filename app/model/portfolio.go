package model

// Portfolio adalah dokumen student_portfolios di MongoDB. Tipe di sini
// sengaja bebas dari tipe driver (tidak ada primitive.ObjectID / bson.M);
// konversi ke dokumen Mongo hanya terjadi di repository.
type Portfolio struct {
	ID           string        `json:"id,omitempty"`
	StudentID    int           `json:"student_id"`
	Headline     string        `json:"headline"`
	Bio          string        `json:"bio"`
	Projects     []Project     `json:"projects"`
	Certificates []Certificate `json:"certificates"`
}

type Project struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Technologies []string `json:"technologies"`
	Link         string   `json:"link"`
	Year         int      `json:"year"`
}

type Certificate struct {
	Name   string `json:"name"`
	Issuer string `json:"issuer"`
	Date   string `json:"date"`
}

// PortfolioRequest dipakai PUT /students/:id/portfolio. Body menggantikan
// seluruh dokumen portofolio (headline, bio, projects, certificates).
type PortfolioRequest struct {
	Headline     string               `json:"headline" validate:"required,min=3,max=150"`
	Bio          string               `json:"bio" validate:"max=2000"`
	Projects     []ProjectRequest     `json:"projects" validate:"max=20,dive"`
	Certificates []CertificateRequest `json:"certificates" validate:"dive"`
}

type ProjectRequest struct {
	Title        string   `json:"title" validate:"required,min=3,max=150"`
	Technologies []string `json:"technologies" validate:"required,min=1,max=20,dive,required,max=50"`
	Link         string   `json:"link" validate:"omitempty,url,max=500"`
	Year         int      `json:"year" validate:"required,min=1900,max=2100"`
}

type CertificateRequest struct {
	Name   string `json:"name" validate:"required,min=3,max=150"`
	Issuer string `json:"issuer" validate:"required,min=2,max=150"`
	Date   string `json:"date" validate:"required,datetime=2006-01-02"`
}

type TechnologyStat struct {
	Technology string `json:"technology"`
	Count      int    `json:"count"`
}
