package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"pemrograman-code/app/model"
)

// ErrCourseFull dikembalikan IncrementEnrolled bila kursi kelas sudah habis.
var ErrCourseFull = errors.New("kelas sudah penuh")

type CourseRepository struct{ db DBTX }

func NewCourseRepository(db DBTX) *CourseRepository { return &CourseRepository{db: db} }

// WithTx mengembalikan repository yang terikat pada transaction tx, sehingga
// seluruh operasinya berjalan di dalam transaction yang sama.
func (r *CourseRepository) WithTx(tx DBTX) *CourseRepository { return &CourseRepository{db: tx} }

func (r *CourseRepository) FindByID(ctx context.Context, id int) (model.Course, error) {
	var c model.Course
	err := r.db.QueryRow(ctx,
		`SELECT id, code, name, capacity, enrolled_count FROM courses WHERE id = $1`, id,
	).Scan(&c.ID, &c.Code, &c.Name, &c.Capacity, &c.EnrolledCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Course{}, ErrNotFound
	}
	if err != nil {
		return model.Course{}, fmt.Errorf("mengambil kelas: %w", err)
	}
	return c, nil
}

// IncrementEnrolled mengambil satu kursi secara atomik. UPDATE me-lock row
// kelas sampai transaction selesai (seperti Modul 9), sehingga dua mahasiswa
// yang mendaftar bersamaan tidak dapat melewati batas capacity. Mengembalikan
// keadaan kelas terbaru.
func (r *CourseRepository) IncrementEnrolled(ctx context.Context, courseID int) (model.Course, error) {
	var c model.Course
	err := r.db.QueryRow(ctx, `
		UPDATE courses
		SET enrolled_count = enrolled_count + 1
		WHERE id = $1 AND enrolled_count < capacity
		RETURNING id, code, name, capacity, enrolled_count`, courseID,
	).Scan(&c.ID, &c.Code, &c.Name, &c.Capacity, &c.EnrolledCount)
	if errors.Is(err, pgx.ErrNoRows) {
		// Tidak ada baris ter-update: entah kelasnya tidak ada, entah penuh.
		return model.Course{}, r.classifyNoSeat(ctx, courseID)
	}
	if err != nil {
		return model.Course{}, fmt.Errorf("mengambil kursi: %w", err)
	}
	return c, nil
}

func (r *CourseRepository) classifyNoSeat(ctx context.Context, courseID int) error {
	var exists bool
	if err := r.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM courses WHERE id = $1)`, courseID,
	).Scan(&exists); err != nil {
		return fmt.Errorf("memeriksa kelas: %w", err)
	}
	if !exists {
		return ErrNotFound
	}
	return ErrCourseFull
}

// DecrementEnrolled melepas satu kursi. GREATEST menjaga counter tidak pernah
// negatif walaupun ada inkonsistensi data.
func (r *CourseRepository) DecrementEnrolled(ctx context.Context, courseID int) (model.Course, error) {
	var c model.Course
	err := r.db.QueryRow(ctx, `
		UPDATE courses
		SET enrolled_count = GREATEST(enrolled_count - 1, 0)
		WHERE id = $1
		RETURNING id, code, name, capacity, enrolled_count`, courseID,
	).Scan(&c.ID, &c.Code, &c.Name, &c.Capacity, &c.EnrolledCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Course{}, ErrNotFound
	}
	if err != nil {
		return model.Course{}, fmt.Errorf("melepas kursi: %w", err)
	}
	return c, nil
}

// InsertEnrollment mencatat KRS. Pelanggaran UNIQUE (user_id, course_id)
// dipetakan menjadi ErrDuplicate, yang oleh service dijadikan 409.
func (r *CourseRepository) InsertEnrollment(ctx context.Context, courseID, userID int) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO enrollments (user_id, course_id) VALUES ($1, $2)`, userID, courseID)
	if isUniqueViolation(err) {
		return ErrDuplicate
	}
	if err != nil {
		return fmt.Errorf("mencatat enrollment: %w", err)
	}
	return nil
}

// DeleteEnrollment menghapus KRS. Mengembalikan ErrNotFound bila mahasiswa
// memang belum terdaftar di kelas itu.
func (r *CourseRepository) DeleteEnrollment(ctx context.Context, courseID, userID int) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM enrollments WHERE user_id = $1 AND course_id = $2`, userID, courseID)
	if err != nil {
		return fmt.Errorf("menghapus enrollment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateQuota mengganti kuota kelas dan mengembalikan keadaan terbarunya.
// Dipakai PATCH /courses/:id dan tugas mandiri D.2.
func (r *CourseRepository) UpdateQuota(ctx context.Context, courseID, capacity int) (model.Course, error) {
	var c model.Course
	err := r.db.QueryRow(ctx, `
		UPDATE courses SET capacity = $2
		WHERE id = $1
		RETURNING id, code, name, capacity, enrolled_count`, courseID, capacity,
	).Scan(&c.ID, &c.Code, &c.Name, &c.Capacity, &c.EnrolledCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Course{}, ErrNotFound
	}
	if err != nil {
		return model.Course{}, fmt.Errorf("mengubah kuota kelas: %w", err)
	}
	return c, nil
}

// ListParticipants mengembalikan id seluruh mahasiswa yang terdaftar di kelas,
// urut menurut user_id. Dipakai tugas mandiri D.2 untuk mengirim notifikasi
// ke semua peserta ketika kuota berubah.
func (r *CourseRepository) ListParticipants(ctx context.Context, courseID int) ([]int, error) {
	rows, err := r.db.Query(ctx,
		`SELECT user_id FROM enrollments WHERE course_id = $1 ORDER BY user_id`, courseID)
	if err != nil {
		return nil, fmt.Errorf("mengambil peserta kelas: %w", err)
	}
	defer rows.Close()

	ids := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("membaca peserta kelas: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
