package repository

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"pemrograman-code/app/model"
)

const portfolioCollection = "student_portfolios"

var (
	ErrPortfolioFull   = errors.New("portofolio sudah berisi 20 proyek")
	ErrProjectNotFound = errors.New("proyek tidak ditemukan")
	ErrInvalidID       = errors.New("object id tidak valid")
)

// Dokumen internal repository. Hanya di sini tipe driver boleh muncul.
type portfolioDoc struct {
	ID           bson.ObjectID    `bson:"_id,omitempty"`
	StudentID    int              `bson:"student_id"`
	Headline     string           `bson:"headline"`
	Bio          string           `bson:"bio"`
	Projects     []projectDoc     `bson:"projects"`
	Certificates []certificateDoc `bson:"certificates"`
}

type projectDoc struct {
	ID           bson.ObjectID `bson:"_id"`
	Title        string        `bson:"title"`
	Technologies []string      `bson:"technologies"`
	Link         string        `bson:"link"`
	Year         int           `bson:"year"`
}

type certificateDoc struct {
	Name   string `bson:"name"`
	Issuer string `bson:"issuer"`
	Date   string `bson:"date"`
}

type technologyRow struct {
	Technology string `bson:"technology"`
	Count      int    `bson:"count"`
}

type PortfolioRepository struct {
	collection *mongo.Collection
}

func NewPortfolioRepository(db *mongo.Database) *PortfolioRepository {
	return &PortfolioRepository{collection: db.Collection(portfolioCollection)}
}

// EnsureIndexes menjaga "satu mahasiswa satu portofolio" di level database,
// bukan hanya di kode. Index technologies (multikey) mempercepat pencarian
// portofolio berdasarkan teknologi bila endpoint filter ditambahkan.
func (r *PortfolioRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "student_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_student_id"),
		},
		{
			Keys:    bson.D{{Key: "projects.technologies", Value: 1}},
			Options: options.Index().SetName("projects_technologies_idx"),
		},
	})
	if err != nil {
		return fmt.Errorf("membuat index student_portfolios: %w", err)
	}
	return nil
}

func (d portfolioDoc) toModel() model.Portfolio {
	p := model.Portfolio{
		StudentID:    d.StudentID,
		Headline:     d.Headline,
		Bio:          d.Bio,
		Projects:     make([]model.Project, 0, len(d.Projects)),
		Certificates: make([]model.Certificate, 0, len(d.Certificates)),
	}
	if !d.ID.IsZero() {
		p.ID = d.ID.Hex()
	}
	for _, pr := range d.Projects {
		tech := pr.Technologies
		if tech == nil {
			tech = []string{}
		}
		p.Projects = append(p.Projects, model.Project{
			ID:           pr.ID.Hex(),
			Title:        pr.Title,
			Technologies: tech,
			Link:         pr.Link,
			Year:         pr.Year,
		})
	}
	for _, c := range d.Certificates {
		p.Certificates = append(p.Certificates, model.Certificate{
			Name: c.Name, Issuer: c.Issuer, Date: c.Date,
		})
	}
	return p
}

func projectToDoc(p model.Project) (projectDoc, error) {
	id := bson.NewObjectID()
	if p.ID != "" {
		parsed, err := bson.ObjectIDFromHex(p.ID)
		if err != nil {
			return projectDoc{}, ErrInvalidID
		}
		id = parsed
	}
	tech := p.Technologies
	if tech == nil {
		tech = []string{}
	}
	return projectDoc{
		ID: id, Title: p.Title, Technologies: tech, Link: p.Link, Year: p.Year,
	}, nil
}

func portfolioToDoc(p model.Portfolio) (portfolioDoc, error) {
	doc := portfolioDoc{
		StudentID:    p.StudentID,
		Headline:     p.Headline,
		Bio:          p.Bio,
		Projects:     make([]projectDoc, 0, len(p.Projects)),
		Certificates: make([]certificateDoc, 0, len(p.Certificates)),
	}
	for _, pr := range p.Projects {
		project, err := projectToDoc(pr)
		if err != nil {
			return portfolioDoc{}, err
		}
		doc.Projects = append(doc.Projects, project)
	}
	for _, c := range p.Certificates {
		doc.Certificates = append(doc.Certificates, certificateDoc{
			Name: c.Name, Issuer: c.Issuer, Date: c.Date,
		})
	}
	return doc, nil
}

func (r *PortfolioRepository) FindByStudentID(ctx context.Context, studentID int) (model.Portfolio, error) {
	var doc portfolioDoc
	err := r.collection.FindOne(ctx, bson.M{"student_id": studentID}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return model.Portfolio{}, ErrNotFound
	}
	if err != nil {
		return model.Portfolio{}, fmt.Errorf("mengambil portofolio: %w", err)
	}
	return doc.toModel(), nil
}

// Upsert mengganti seluruh dokumen atau membuatnya bila belum ada.
func (r *PortfolioRepository) Upsert(ctx context.Context, p model.Portfolio) (model.Portfolio, error) {
	doc, err := portfolioToDoc(p)
	if err != nil {
		return model.Portfolio{}, err
	}

	filter := bson.M{"student_id": p.StudentID}
	_, err = r.collection.ReplaceOne(ctx, filter, doc, options.Replace().SetUpsert(true))
	// Dua PUT bersamaan bisa sama-sama upsert; yang kalah kena unique index.
	// Ulangi sebagai replace biasa karena dokumennya sudah ada.
	if mongo.IsDuplicateKeyError(err) {
		_, err = r.collection.ReplaceOne(ctx, filter, doc)
	}
	if err != nil {
		return model.Portfolio{}, fmt.Errorf("menyimpan portofolio: %w", err)
	}
	return r.FindByStudentID(ctx, p.StudentID)
}

// AddProject memakai pola kunci "projects.19" (seperti replies.49): filter
// hanya cocok bila slot ke-20 belum terisi, sehingga batas 20 proyek dijaga
// atomik oleh MongoDB tanpa read-then-write.
func (r *PortfolioRepository) AddProject(
	ctx context.Context, studentID int, p model.Project,
) (model.Portfolio, error) {
	doc, err := projectToDoc(p)
	if err != nil {
		return model.Portfolio{}, err
	}

	filter := bson.M{
		"student_id":  studentID,
		"projects.19": bson.M{"$exists": false},
	}
	result, err := r.collection.UpdateOne(ctx, filter,
		bson.M{"$push": bson.M{"projects": doc}})
	if err != nil {
		return model.Portfolio{}, fmt.Errorf("menambah proyek: %w", err)
	}
	if result.MatchedCount == 0 {
		// Bedakan portofolio tidak ada dengan batas 20 proyek tercapai.
		if _, err := r.FindByStudentID(ctx, studentID); err != nil {
			return model.Portfolio{}, err
		}
		return model.Portfolio{}, ErrPortfolioFull
	}
	return r.FindByStudentID(ctx, studentID)
}

// DeleteProject memakai $pull untuk menghapus proyek berdasarkan _id.
func (r *PortfolioRepository) DeleteProject(
	ctx context.Context, studentID int, projectID string,
) error {
	objectID, err := bson.ObjectIDFromHex(projectID)
	if err != nil {
		return ErrInvalidID
	}

	result, err := r.collection.UpdateOne(ctx,
		bson.M{"student_id": studentID},
		bson.M{"$pull": bson.M{"projects": bson.M{"_id": objectID}}})
	if err != nil {
		return fmt.Errorf("menghapus proyek: %w", err)
	}
	if result.MatchedCount == 0 {
		return ErrNotFound
	}
	if result.ModifiedCount == 0 {
		return ErrProjectNotFound
	}
	return nil
}

// TopTechnologies menjalankan dua $unwind (projects lalu technologies),
// menghitung kemunculan tiap teknologi, lalu mengambil 10 teratas.
func (r *PortfolioRepository) TopTechnologies(ctx context.Context, limit int) ([]model.TechnologyStat, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$unwind", Value: "$projects"}},
		{{Key: "$unwind", Value: "$projects.technologies"}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$projects.technologies"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		{{Key: "$sort", Value: bson.D{
			{Key: "count", Value: -1},
			{Key: "_id", Value: 1},
		}}},
		{{Key: "$limit", Value: limit}},
		{{Key: "$project", Value: bson.D{
			{Key: "_id", Value: 0},
			{Key: "technology", Value: "$_id"},
			{Key: "count", Value: 1},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("menghitung teknologi: %w", err)
	}
	defer cursor.Close(ctx)

	rows := []technologyRow{}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("membaca hasil agregasi: %w", err)
	}

	stats := make([]model.TechnologyStat, 0, len(rows))
	for _, row := range rows {
		stats = append(stats, model.TechnologyStat{
			Technology: row.Technology, Count: row.Count,
		})
	}
	return stats, nil
}
