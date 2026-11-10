package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"pemrograman-code/app/model"
)

const (
	reviewCollection = "course_reviews"
	// MaxRepliesPerReview membatasi panjang array replies agar dokumen tidak
	// tumbuh tanpa batas (batas ukuran dokumen MongoDB 16 MB).
	MaxRepliesPerReview = 20
)

var (
	ErrNotOwner     = errors.New("bukan pemilik ulasan")
	ErrOwnReview    = errors.New("tidak dapat memberi suara pada ulasan sendiri")
	ErrAlreadyVoted = errors.New("sudah pernah memberi suara")
	ErrReplyLimit   = errors.New("jumlah balasan sudah mencapai batas")
)

// ReviewRepository adalah kontrak penyimpanan ulasan kelas.
//
// FindByCourse sengaja mengembalikan SELURUH ulasan sebuah kelas, terbaru
// lebih dulu, lalu penyaringan tag dan cursor dilakukan di service. Jumlah
// ulasan per kelas dibatasi oleh aturan "satu mahasiswa satu ulasan" dan oleh
// kuota kelas (puluhan, bukan jutaan), sehingga cara ini aman — dan membuat
// fake di unit test cukup meniru satu method sederhana.
type ReviewRepository interface {
	Create(ctx context.Context, r model.Review) (model.Review, error)
	FindByCourse(ctx context.Context, courseID int) ([]model.Review, error)
	FindByID(ctx context.Context, id string) (model.Review, error)
	Update(ctx context.Context, id string, userID int, req model.UpdateReviewRequest, tags []string) (model.Review, error)
	Delete(ctx context.Context, id string, userID int, asModerator bool) error
	Vote(ctx context.Context, id string, userID int) (model.Review, error)
	AddReply(ctx context.Context, id string, reply model.Reply) (model.Review, error)
	Summary(ctx context.Context, courseID int) (model.ReviewSummary, error)
}

// Dokumen internal repository. Hanya di sini tipe driver boleh muncul.
type reviewDoc struct {
	ID            bson.ObjectID `bson:"_id,omitempty"`
	CourseID      int           `bson:"course_id"`
	UserID        int           `bson:"user_id"`
	Username      string        `bson:"username"`
	Rating        int           `bson:"rating"`
	Comment       string        `bson:"comment"`
	Tags          []string      `bson:"tags"`
	HelpfulVoters []int         `bson:"helpful_voters"`
	HelpfulCount  int           `bson:"helpful_count"`
	Replies       []replyDoc    `bson:"replies"`
	CreatedAt     time.Time     `bson:"created_at"`
	UpdatedAt     time.Time     `bson:"updated_at"`
}

type replyDoc struct {
	ID        bson.ObjectID `bson:"_id"`
	UserID    int           `bson:"user_id"`
	Username  string        `bson:"username"`
	Body      string        `bson:"body"`
	CreatedAt time.Time     `bson:"created_at"`
}

func (d reviewDoc) toModel() model.Review {
	r := model.Review{
		ID: d.ID.Hex(), CourseID: d.CourseID, UserID: d.UserID, Username: d.Username,
		Rating: d.Rating, Comment: d.Comment, Tags: d.Tags, HelpfulCount: d.HelpfulCount,
		Replies: make([]model.Reply, 0, len(d.Replies)), CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
	if r.Tags == nil {
		r.Tags = []string{}
	}
	for _, rp := range d.Replies {
		r.Replies = append(r.Replies, model.Reply{
			ID: rp.ID.Hex(), UserID: rp.UserID, Username: rp.Username, Body: rp.Body, CreatedAt: rp.CreatedAt,
		})
	}
	return r
}

type reviewRepository struct {
	collection *mongo.Collection
}

func NewReviewRepository(db *mongo.Database) ReviewRepository {
	return &reviewRepository{collection: db.Collection(reviewCollection)}
}

// EnsureReviewIndexes menjaga "satu mahasiswa satu ulasan per kelas" di level
// database lewat unique index course_user_unique, bukan hanya di kode.
func EnsureReviewIndexes(ctx context.Context, db *mongo.Database) error {
	_, err := db.Collection(reviewCollection).Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "course_id", Value: 1}, {Key: "user_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("course_user_unique"),
		},
		{
			Keys:    bson.D{{Key: "course_id", Value: 1}, {Key: "_id", Value: -1}},
			Options: options.Index().SetName("course_newest"),
		},
	})
	if err != nil {
		return fmt.Errorf("membuat index ulasan: %w", err)
	}
	return nil
}

func parseObjectID(id string) (bson.ObjectID, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return bson.ObjectID{}, ErrInvalidID
	}
	return oid, nil
}

func (r *reviewRepository) Create(ctx context.Context, rv model.Review) (model.Review, error) {
	now := time.Now().UTC().Truncate(time.Millisecond)
	doc := reviewDoc{
		ID: bson.NewObjectID(), CourseID: rv.CourseID, UserID: rv.UserID, Username: rv.Username,
		Rating: rv.Rating, Comment: rv.Comment, Tags: rv.Tags,
		HelpfulVoters: []int{}, Replies: []replyDoc{}, CreatedAt: now, UpdatedAt: now,
	}
	if doc.Tags == nil {
		doc.Tags = []string{}
	}
	if _, err := r.collection.InsertOne(ctx, doc); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return model.Review{}, ErrDuplicate
		}
		return model.Review{}, fmt.Errorf("menyimpan ulasan: %w", err)
	}
	return doc.toModel(), nil
}

func (r *reviewRepository) FindByCourse(ctx context.Context, courseID int) ([]model.Review, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"course_id": courseID},
		options.Find().SetSort(bson.D{{Key: "_id", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("mengambil ulasan: %w", err)
	}
	defer cursor.Close(ctx)

	result := []model.Review{}
	for cursor.Next(ctx) {
		var doc reviewDoc
		if err := cursor.Decode(&doc); err != nil {
			return nil, fmt.Errorf("membaca ulasan: %w", err)
		}
		result = append(result, doc.toModel())
	}
	return result, cursor.Err()
}

func (r *reviewRepository) findDoc(ctx context.Context, oid bson.ObjectID) (reviewDoc, error) {
	var doc reviewDoc
	err := r.collection.FindOne(ctx, bson.M{"_id": oid}).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return reviewDoc{}, ErrNotFound
	}
	if err != nil {
		return reviewDoc{}, fmt.Errorf("mengambil ulasan: %w", err)
	}
	return doc, nil
}

func (r *reviewRepository) FindByID(ctx context.Context, id string) (model.Review, error) {
	oid, err := parseObjectID(id)
	if err != nil {
		return model.Review{}, err
	}
	doc, err := r.findDoc(ctx, oid)
	if err != nil {
		return model.Review{}, err
	}
	return doc.toModel(), nil
}

// updateOne menjalankan FindOneAndUpdate bersyarat. Bila syarat tidak
// terpenuhi, classify dipanggil dengan dokumen terkini untuk menentukan
// alasan penolakannya.
func (r *reviewRepository) updateOne(ctx context.Context, oid bson.ObjectID, filter bson.M, update bson.M,
	classify func(doc reviewDoc) error) (model.Review, error) {
	filter["_id"] = oid
	var doc reviewDoc
	err := r.collection.FindOneAndUpdate(ctx, filter, update,
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&doc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		current, findErr := r.findDoc(ctx, oid)
		if findErr != nil {
			return model.Review{}, findErr
		}
		return model.Review{}, classify(current)
	}
	if err != nil {
		return model.Review{}, fmt.Errorf("mengubah ulasan: %w", err)
	}
	return doc.toModel(), nil
}

func (r *reviewRepository) Update(ctx context.Context, id string, userID int, req model.UpdateReviewRequest, tags []string) (model.Review, error) {
	oid, err := parseObjectID(id)
	if err != nil {
		return model.Review{}, err
	}
	set := bson.M{"updated_at": time.Now().UTC().Truncate(time.Millisecond)}
	if req.Rating != nil {
		set["rating"] = *req.Rating
	}
	if req.Comment != nil {
		set["comment"] = *req.Comment
	}
	if tags != nil {
		set["tags"] = tags
	}
	return r.updateOne(ctx, oid, bson.M{"user_id": userID}, bson.M{"$set": set},
		func(reviewDoc) error { return ErrNotOwner })
}

func (r *reviewRepository) Delete(ctx context.Context, id string, userID int, asModerator bool) error {
	oid, err := parseObjectID(id)
	if err != nil {
		return err
	}
	filter := bson.M{"_id": oid}
	if !asModerator {
		filter["user_id"] = userID
	}
	res, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("menghapus ulasan: %w", err)
	}
	if res.DeletedCount == 0 {
		if _, err := r.findDoc(ctx, oid); err != nil {
			return err
		}
		return ErrNotOwner
	}
	return nil
}

// Vote menambah satu suara "membantu". Syarat ditulis di filter agar
// pemeriksaan dan perubahan terjadi dalam SATU operasi atomik: dua request
// serentak dari user yang sama tidak dapat sama-sama lolos.
func (r *reviewRepository) Vote(ctx context.Context, id string, userID int) (model.Review, error) {
	oid, err := parseObjectID(id)
	if err != nil {
		return model.Review{}, err
	}
	return r.updateOne(ctx, oid,
		bson.M{"user_id": bson.M{"$ne": userID}, "helpful_voters": bson.M{"$ne": userID}},
		bson.M{"$addToSet": bson.M{"helpful_voters": userID}, "$inc": bson.M{"helpful_count": 1}},
		func(doc reviewDoc) error {
			if doc.UserID == userID {
				return ErrOwnReview
			}
			return ErrAlreadyVoted
		})
}

func (r *reviewRepository) AddReply(ctx context.Context, id string, reply model.Reply) (model.Review, error) {
	oid, err := parseObjectID(id)
	if err != nil {
		return model.Review{}, err
	}
	doc := replyDoc{
		ID: bson.NewObjectID(), UserID: reply.UserID, Username: reply.Username,
		Body: reply.Body, CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
	}
	// "replies.N tidak ada" berarti panjang array masih kurang dari N+1.
	limitField := fmt.Sprintf("replies.%d", MaxRepliesPerReview-1)
	return r.updateOne(ctx, oid,
		bson.M{limitField: bson.M{"$exists": false}},
		bson.M{"$push": bson.M{"replies": doc}},
		func(reviewDoc) error { return ErrReplyLimit })
}

func (r *reviewRepository) Summary(ctx context.Context, courseID int) (model.ReviewSummary, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"course_id": courseID}}},
		{{Key: "$facet", Value: bson.M{
			"stats": bson.A{bson.M{"$group": bson.M{
				"_id": nil, "count": bson.M{"$sum": 1}, "avg": bson.M{"$avg": "$rating"},
			}}},
			"ratings": bson.A{bson.M{"$group": bson.M{"_id": "$rating", "n": bson.M{"$sum": 1}}}},
			"tags": bson.A{
				bson.M{"$unwind": "$tags"},
				bson.M{"$group": bson.M{"_id": "$tags", "n": bson.M{"$sum": 1}}},
				bson.M{"$sort": bson.D{{Key: "n", Value: -1}, {Key: "_id", Value: 1}}},
				bson.M{"$limit": 5},
			},
		}}},
	}
	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return model.ReviewSummary{}, fmt.Errorf("meringkas ulasan: %w", err)
	}
	defer cursor.Close(ctx)

	var rows []struct {
		Stats []struct {
			Count int     `bson:"count"`
			Avg   float64 `bson:"avg"`
		} `bson:"stats"`
		Ratings []struct {
			Rating int `bson:"_id"`
			N      int `bson:"n"`
		} `bson:"ratings"`
		Tags []struct {
			Tag string `bson:"_id"`
			N   int    `bson:"n"`
		} `bson:"tags"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return model.ReviewSummary{}, fmt.Errorf("membaca ringkasan ulasan: %w", err)
	}

	summary := model.ReviewSummary{CourseID: courseID, TopTags: []model.TagCount{}}
	if len(rows) == 0 {
		return summary, nil
	}
	if len(rows[0].Stats) > 0 {
		summary.Count = rows[0].Stats[0].Count
		summary.AverageRating = float64(int(rows[0].Stats[0].Avg*100+0.5)) / 100
	}
	for _, rt := range rows[0].Ratings {
		if rt.Rating >= 1 && rt.Rating <= 5 {
			summary.Distribution[rt.Rating-1] = rt.N
		}
	}
	for _, tg := range rows[0].Tags {
		summary.TopTags = append(summary.TopTags, model.TagCount{Tag: tg.Tag, Count: tg.N})
	}
	return summary, nil
}
