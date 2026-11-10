package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"pemrograman-code/app/model"
)

var (
	// ErrCourseFull dikembalikan IncrementEnrolled bila kursi kelas sudah habis.
	ErrCourseFull = errors.New("kelas sudah penuh")
	// ErrVersionConflict: version yang dikirim sudah tidak sama dengan
	// version di database (kelas diubah orang lain lebih dulu).
	ErrVersionConflict = errors.New("version kelas sudah berubah")
	// ErrQuotaBelowEnrolled: kuota baru lebih kecil dari jumlah peserta.
	ErrQuotaBelowEnrolled = errors.New("kuota lebih kecil dari jumlah peserta")
)

// CourseRepository adalah kontrak akses data kelas dan KRS.
//
// Service bergantung pada interface ini, bukan pada struct PostgreSQL, sehingga
// unit test dapat memasang fake (Modul 12, Langkah 2) tanpa mengubah service.
type CourseRepository interface {
	// WithTx mengembalikan repository yang terikat pada transaction tx.
	WithTx(tx DBTX) CourseRepository

	List(ctx context.Context) ([]model.Course, error)
	ListByUser(ctx context.Context, userID int) ([]model.Course, error)
	FindByID(ctx context.Context, id int) (model.Course, error)
	IsEnrolled(ctx context.Context, courseID, userID int) (bool, error)

	// LockStudent mengunci row mahasiswa sampai transaction selesai.
	LockStudent(ctx context.Context, userID int) error
	TakenCredits(ctx context.Context, userID int) (int, error)

	IncrementEnrolled(ctx context.Context, courseID int) (model.Course, error)
	DecrementEnrolled(ctx context.Context, courseID int) (model.Course, error)
	InsertEnrollment(ctx context.Context, courseID, userID int) error
	DeleteEnrollment(ctx context.Context, courseID, userID int) error

	UpdateQuota(ctx context.Context, courseID, quota, version int) (model.Course, error)
	ListParticipants(ctx context.Context, courseID int) ([]int, error)
}

type courseRepository struct{ db DBTX }

func NewCourseRepository(db DBTX) CourseRepository { return &courseRepository{db: db} }

func (r *courseRepository) WithTx(tx DBTX) CourseRepository { return &courseRepository{db: tx} }

const courseColumns = `id, code, name, credits, quota, enrolled_count, version`

func scanCourse(row pgx.Row) (model.Course, error) {
	var c model.Course
	err := row.Scan(&c.ID, &c.Code, &c.Name, &c.Credits, &c.Quota, &c.EnrolledCount, &c.Version)
	return c, err
}

func (r *courseRepository) queryCourses(ctx context.Context, sql string, args ...any) ([]model.Course, error) {
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("mengambil daftar kelas: %w", err)
	}
	defer rows.Close()

	courses := []model.Course{}
	for rows.Next() {
		c, err := scanCourse(rows)
		if err != nil {
			return nil, fmt.Errorf("membaca row kelas: %w", err)
		}
		courses = append(courses, c)
	}
	return courses, rows.Err()
}

func (r *courseRepository) List(ctx context.Context) ([]model.Course, error) {
	return r.queryCourses(ctx, `SELECT `+courseColumns+` FROM courses ORDER BY id`)
}

func (r *courseRepository) ListByUser(ctx context.Context, userID int) ([]model.Course, error) {
	return r.queryCourses(ctx, `
		SELECT c.id, c.code, c.name, c.credits, c.quota, c.enrolled_count, c.version
		FROM enrollments e
		JOIN courses c ON c.id = e.course_id
		WHERE e.user_id = $1
		ORDER BY c.id`, userID)
}

func (r *courseRepository) FindByID(ctx context.Context, id int) (model.Course, error) {
	c, err := scanCourse(r.db.QueryRow(ctx,
		`SELECT `+courseColumns+` FROM courses WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Course{}, ErrNotFound
	}
	if err != nil {
		return model.Course{}, fmt.Errorf("mengambil kelas: %w", err)
	}
	return c, nil
}

func (r *courseRepository) IsEnrolled(ctx context.Context, courseID, userID int) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM enrollments WHERE course_id = $1 AND user_id = $2)`,
		courseID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("memeriksa enrollment: %w", err)
	}
	return exists, nil
}

// LockStudent mengunci row mahasiswa dengan SELECT ... FOR UPDATE.
//
// Aturan batas 24 SKS adalah pola check-then-act (Modul 9): hitung SKS yang
// sudah diambil, lalu tambah kelas bila masih cukup. Tanpa lock, sepuluh
// request serentak dari mahasiswa yang SAMA dapat membaca jumlah SKS yang sama
// lalu semuanya lolos. FOR UPDATE membuat request kedua menunggu sampai
// transaction pertama selesai, sehingga ia membaca jumlah SKS yang terbaru.
//
// Lock dipasang pada row users, bukan pada row courses: yang dijaga adalah
// total SKS SATU mahasiswa, yang tersebar di banyak kelas berbeda.
func (r *courseRepository) LockStudent(ctx context.Context, userID int) error {
	var id int
	err := r.db.QueryRow(ctx,
		`SELECT id FROM users WHERE id = $1 FOR UPDATE`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("mengunci mahasiswa: %w", err)
	}
	return nil
}

// TakenCredits menjumlahkan SKS seluruh kelas yang sedang diambil mahasiswa.
func (r *courseRepository) TakenCredits(ctx context.Context, userID int) (int, error) {
	var total int
	err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(c.credits), 0)
		FROM enrollments e
		JOIN courses c ON c.id = e.course_id
		WHERE e.user_id = $1`, userID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("menghitung SKS: %w", err)
	}
	return total, nil
}

// IncrementEnrolled mengambil satu kursi secara atomik. UPDATE bersyarat
// me-lock row kelas sampai transaction selesai, sehingga dua mahasiswa yang
// mendaftar bersamaan tidak dapat melewati kuota.
func (r *courseRepository) IncrementEnrolled(ctx context.Context, courseID int) (model.Course, error) {
	c, err := scanCourse(r.db.QueryRow(ctx, `
		UPDATE courses
		SET enrolled_count = enrolled_count + 1
		WHERE id = $1 AND enrolled_count < quota
		RETURNING `+courseColumns, courseID))
	if errors.Is(err, pgx.ErrNoRows) {
		// Tidak ada baris ter-update: entah kelasnya tidak ada, entah penuh.
		return model.Course{}, r.classifyNoSeat(ctx, courseID)
	}
	if err != nil {
		return model.Course{}, fmt.Errorf("mengambil kursi: %w", err)
	}
	return c, nil
}

func (r *courseRepository) classifyNoSeat(ctx context.Context, courseID int) error {
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
func (r *courseRepository) DecrementEnrolled(ctx context.Context, courseID int) (model.Course, error) {
	c, err := scanCourse(r.db.QueryRow(ctx, `
		UPDATE courses
		SET enrolled_count = GREATEST(enrolled_count - 1, 0)
		WHERE id = $1
		RETURNING `+courseColumns, courseID))
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
func (r *courseRepository) InsertEnrollment(ctx context.Context, courseID, userID int) error {
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
func (r *courseRepository) DeleteEnrollment(ctx context.Context, courseID, userID int) error {
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

// UpdateQuota mengganti kuota dengan optimistic locking: UPDATE hanya berhasil
// bila version masih sama dengan yang dikirim admin dan kuota baru tidak lebih
// kecil dari jumlah peserta. Setiap perubahan menaikkan version.
func (r *courseRepository) UpdateQuota(ctx context.Context, courseID, quota, version int) (model.Course, error) {
	c, err := scanCourse(r.db.QueryRow(ctx, `
		UPDATE courses
		SET quota = $2, version = version + 1
		WHERE id = $1 AND version = $3 AND enrolled_count <= $2
		RETURNING `+courseColumns, courseID, quota, version))
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Course{}, r.classifyQuotaRejection(ctx, courseID, version)
	}
	if err != nil {
		return model.Course{}, fmt.Errorf("mengubah kuota kelas: %w", err)
	}
	return c, nil
}

func (r *courseRepository) classifyQuotaRejection(ctx context.Context, courseID, version int) error {
	current, err := r.FindByID(ctx, courseID)
	if err != nil {
		return err // ErrNotFound atau error database
	}
	if current.Version != version {
		return ErrVersionConflict
	}
	return ErrQuotaBelowEnrolled
}

// ListParticipants mengembalikan id seluruh mahasiswa yang terdaftar di kelas,
// urut menurut user_id. Dipakai notifikasi perubahan kuota.
func (r *courseRepository) ListParticipants(ctx context.Context, courseID int) ([]int, error) {
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
