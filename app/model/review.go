package model

import "time"

// Review adalah ulasan seorang mahasiswa terhadap sebuah kelas. Disimpan di
// MongoDB karena bentuknya dokumen: tag, daftar balasan, dan daftar pemberi
// suara melekat pada ulasannya.
type Review struct {
	ID           string    `json:"id"`
	CourseID     int       `json:"course_id"`
	UserID       int       `json:"user_id"`
	Username     string    `json:"username"`
	Rating       int       `json:"rating"`
	Comment      string    `json:"comment"`
	Tags         []string  `json:"tags"`
	HelpfulCount int       `json:"helpful_count"`
	Replies      []Reply   `json:"replies"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Reply adalah balasan terhadap sebuah ulasan.
type Reply struct {
	ID        string    `json:"id"`
	UserID    int       `json:"user_id"`
	Username  string    `json:"username"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateReviewRequest adalah body POST /courses/:id/reviews.
type CreateReviewRequest struct {
	Rating  int      `json:"rating" validate:"required,min=1,max=5"`
	Comment string   `json:"comment" validate:"required,min=10,max=1000"`
	Tags    []string `json:"tags" validate:"max=5,dive,max=30"`
}

// UpdateReviewRequest adalah body PATCH /reviews/:id. Field yang tidak
// dikirim tidak diubah.
type UpdateReviewRequest struct {
	Rating  *int      `json:"rating,omitempty" validate:"omitnil,min=1,max=5"`
	Comment *string   `json:"comment,omitempty" validate:"omitnil,min=10,max=1000"`
	Tags    *[]string `json:"tags,omitempty" validate:"omitnil,max=5,dive,max=30"`
}

// CreateReplyRequest adalah body POST /reviews/:id/replies.
type CreateReplyRequest struct {
	Body string `json:"body" validate:"required,min=2,max=500"`
}

// ReviewSummary adalah ringkasan ulasan sebuah kelas.
type ReviewSummary struct {
	CourseID      int        `json:"course_id"`
	Count         int        `json:"count"`
	AverageRating float64    `json:"average_rating"`
	Distribution  [5]int     `json:"distribution"` // indeks 0 = bintang 1
	TopTags       []TagCount `json:"top_tags"`
}

// TagCount adalah jumlah kemunculan satu tag.
type TagCount struct {
	Tag   string `json:"tag"`
	Count int    `json:"count"`
}
